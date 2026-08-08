package av1

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func planeRows(pic *Picture, w *bytes.Buffer) {
	ssHor := b2i(pic.Layout != pixelLayoutI444)
	ssVer := b2i(pic.Layout == pixelLayoutI420)
	bpp := 1
	if pic.BitDepth > 8 {
		bpp = 2
	}

	for pl := range 3 {
		if pic.Data[pl] == nil {
			continue
		}
		pw, ph := pic.Width, pic.Height
		if pl != 0 {
			pw = (pw + ssHor) >> ssHor
			ph = (ph + ssVer) >> ssVer
		}
		for y := range ph {
			off := y * pic.Stride[pl]
			w.Write(pic.Data[pl][off : off+pw*bpp])
		}
	}
}

func decodeIVF(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 32 || string(data[:4]) != "DKIF" {
		t.Fatalf("%s: not an IVF file", path)
	}

	var out bytes.Buffer
	var d Decoder

	for p := 32; p+12 <= len(data); {
		n := int(binary.LittleEndian.Uint32(data[p:]))
		p += 12
		if n < 0 || p+n > len(data) {
			break
		}
		pics, err := d.DecodeOBUs(data[p : p+n])
		p += n
		if err != nil {
			t.Fatalf("%s: decode: %v", filepath.Base(path), err)
		}
		for _, pic := range pics {
			planeRows(pic, &out)
		}
	}

	return out.Bytes()
}

// TestDecodeStreams decodes intra-only streams encoded with all post-filters
// disabled, so reconstruction output equals dav1d's final output.
func TestDecodeStreams(t *testing.T) {
	dir := "testdata/streams"

	f, err := os.Open(filepath.Join(dir, "md5.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		name, want := fields[0], fields[1]

		t.Run(name, func(t *testing.T) {
			got := decodeIVF(t, filepath.Join(dir, name))
			sum := md5.Sum(got)
			if hex.EncodeToString(sum[:]) != want {
				t.Fatalf("md5 = %s, want %s (%d bytes)",
					hex.EncodeToString(sum[:]), want, len(got))
			}
			t.Logf("%d bytes, md5 %s", len(got), want)
		})
		n++
	}

	if n == 0 {
		t.Fatal("no streams listed")
	}
}

// TestDecodeExternal decodes an arbitrary stream and compares against a live
// dav1d run. Set STREAM_PATH to use it.
func TestDecodeExternal(t *testing.T) {
	path := os.Getenv("STREAM_PATH")
	if path == "" {
		t.Skip("set STREAM_PATH")
	}

	got := decodeIVF(t, path)
	want := dav1dYUV(t, path)

	if len(got) != len(want) {
		t.Fatalf("got %d bytes, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got %d, want %d", i, got[i], want[i])
		}
	}
	t.Logf("%d bytes bit-exact", len(got))
}

func dav1dYUV(t *testing.T, path string) []byte {
	t.Helper()

	bin := "/temp/av1/dav1d/build/tools/dav1d"
	if _, err := os.Stat(bin); err != nil {
		t.Skip("dav1d binary not built")
	}
	tmp := filepath.Join(t.TempDir(), "ref.yuv")
	cmd := exec.Command(bin, "-i", path, "-o", tmp, "--muxer", "yuv")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dav1d: %v\n%s", err, out)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

// TestConformance decodes the dav1d conformance suite and reports how many
// streams are bit-exact. Set CONFORMANCE_DIR to run it.
func TestConformance(t *testing.T) {
	dir := os.Getenv("CONFORMANCE_DIR")
	if dir == "" {
		t.Skip("set CONFORMANCE_DIR")
	}

	var streams []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() &&
			(strings.HasSuffix(p, ".ivf") || strings.HasSuffix(p, ".obu")) {
			streams = append(streams, p)
		}

		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(streams)

	var exact, mismatch int
	fails := map[string]int{}

	for _, p := range streams {
		name := strings.TrimPrefix(p, dir)
		res := conformanceOne(p)
		switch res {
		case "":
			exact++
		default:
			mismatch++
			key := res
			if i := strings.IndexByte(key, '\n'); i >= 0 {
				key = key[:i]
			}
			if len(key) > 90 {
				key = key[:90]
			}
			fails[key]++
			if os.Getenv("CONFORMANCE_VERBOSE") != "" || fails[key] <= 2 {
				t.Logf("%s: %s", name, key)
			}
		}
	}

	t.Logf("%d/%d bit-exact, %d mismatched", exact, len(streams), mismatch)
	keys := make([]string, 0, len(fails))
	for k := range fails {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return fails[keys[i]] > fails[keys[j]] })
	for _, k := range keys {
		t.Logf("%4d  %s", fails[k], k)
	}
}

func uleb(data []byte) (v uint64, n int) {
	for n < len(data) && n < 8 {
		b := data[n]
		v |= uint64(b&0x7f) << (7 * n)
		n++
		if b&0x80 == 0 {
			return v, n
		}
	}

	return 0, -1
}

// obuChunks splits a section 5 or Annex-B bitstream into runs to hand to DecodeOBUs.
func obuChunks(data []byte) [][]byte {
	var chunks [][]byte

	if tu := section5Chunks(data); tu != nil {
		return tu
	}

	for len(data) > 0 {
		tuSize, n := uleb(data)
		if n < 0 || uint64(len(data)-n) < tuSize {
			return nil
		}
		tu := data[n : n+int(tuSize)]
		data = data[n+int(tuSize):]

		for len(tu) > 0 {
			fuSize, n := uleb(tu)
			if n < 0 || uint64(len(tu)-n) < fuSize {
				return nil
			}
			fu := tu[n : n+int(fuSize)]
			tu = tu[n+int(fuSize):]

			for len(fu) > 0 {
				obuSize, n := uleb(fu)
				if n < 0 || uint64(len(fu)-n) < obuSize {
					return nil
				}
				chunks = append(chunks, fu[n:n+int(obuSize)])
				fu = fu[n+int(obuSize):]
			}
		}
	}

	return chunks
}

func section5Chunks(data []byte) [][]byte {
	var chunks [][]byte
	start := 0

	for p := 0; p < len(data); {
		if data[p]&0x80 != 0 {
			return nil
		}
		typ := data[p] >> 3 & 0xf
		hasExtension := data[p]>>2&1 != 0
		hasSize := data[p]>>1&1 != 0
		if !hasSize {
			return nil
		}
		hdr := p + 1
		if hasExtension {
			hdr++
		}
		if hdr > len(data) {
			return nil
		}
		size, n := uleb(data[hdr:])
		if n < 0 || uint64(len(data)-hdr-n) < size {
			return nil
		}
		if typ == obuTd && p > start {
			chunks = append(chunks, data[start:p])
			start = p
		}
		p = hdr + n + int(size)
	}

	if start < len(data) {
		chunks = append(chunks, data[start:])
	}

	return chunks
}

func conformanceOne(path string) (res string) {
	defer func() {
		if r := recover(); r != nil {
			res = "panic: " + firstLine(r)
		}
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		return "read: " + err.Error()
	}

	var chunks [][]byte
	if len(data) >= 32 && string(data[:4]) == "DKIF" {
		for p := 32; p+12 <= len(data); {
			n := int(binary.LittleEndian.Uint32(data[p:]))
			p += 12
			if n < 0 || p+n > len(data) {
				break
			}
			chunks = append(chunks, data[p:p+n])
			p += n
		}
	} else if chunks = obuChunks(data); chunks == nil {
		return "unknown container"
	}

	var out bytes.Buffer
	var d Decoder
	for _, chunk := range chunks {
		pics, err := d.DecodeOBUs(chunk)
		if err != nil {
			return "decode: " + err.Error()
		}
		for _, pic := range pics {
			planeRows(pic, &out)
		}
	}

	bin := "/temp/av1/dav1d/build/tools/dav1d"
	cmd := exec.Command(bin, "-i", path, "-o", "-", "--muxer", "md5", "--filmgrain", "1")
	ref, err := cmd.Output()
	if err != nil {
		return "dav1d: " + err.Error()
	}
	sum := md5.Sum(out.Bytes())
	if !strings.Contains(string(ref), hex.EncodeToString(sum[:])) {
		return "md5 mismatch"
	}

	return ""
}

func firstLine(v any) string {
	s := strings.TrimSpace(strings.SplitN(fmt.Sprint(v), "\n", 2)[0])
	if len(s) > 90 {
		s = s[:90]
	}

	return s
}
