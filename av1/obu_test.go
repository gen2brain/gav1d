package av1

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testDataDir(t *testing.T) string {
	dir := os.Getenv("DAV1D_TEST_DATA")
	if dir == "" {
		t.Skip("set DAV1D_TEST_DATA")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("test data not available at %s", dir)
	}

	return dir
}

func ivfFrames(t *testing.T, path string) [][]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 32 || string(data[:4]) != "DKIF" {
		t.Fatalf("%s: not an IVF file", path)
	}

	var frames [][]byte
	for p := 32; p+12 <= len(data); {
		n := int(binary.LittleEndian.Uint32(data[p:]))
		p += 12
		if n < 0 || p+n > len(data) {
			break
		}
		frames = append(frames, data[p:p+n])
		p += n
	}

	return frames
}

func TestOBUParity(t *testing.T) {
	dir := testDataDir(t)

	want := readFixture(t, "obu.txt")

	var got bytes.Buffer
	w := bufio.NewWriterSize(&got, 1<<20)

	for _, line := range strings.Split(string(want), "\n") {
		name, ok := strings.CutPrefix(line, "FILE ")
		if !ok {
			continue
		}

		w.WriteString("FILE " + name + "\n")

		var d Decoder
		d.headersOnly = true
		d.seqHdrHook = func(s *sequenceHeader) { dumpSeqHdr(w, s) }
		d.frameHdrHook = func(h *frameHeader) { dumpFrameHdr(w, h) }

		for _, tu := range ivfFrames(t, filepath.Join(dir, name)) {
			for len(tu) > 0 {
				n, err := d.parseOBU(tu)
				if err != nil || n <= 0 {
					break
				}
				tu = tu[n:]
			}
		}
	}
	w.Flush()

	if bytes.Equal(got.Bytes(), want) {
		t.Logf("%d lines match", bytes.Count(want, []byte("\n")))

		return
	}

	gotLines := strings.Split(got.String(), "\n")
	wantLines := strings.Split(string(want), "\n")
	file := ""
	for i := range max(len(gotLines), len(wantLines)) {
		var g, wl string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			wl = wantLines[i]
		}
		if strings.HasPrefix(wl, "FILE ") {
			file = wl
		}
		if g != wl {
			t.Fatalf("%s\nline %d:\n got %q\nwant %q", file, i+1, g, wl)
		}
	}
	t.Fatalf("output lengths differ: got %d lines, want %d", len(gotLines), len(wantLines))
}
