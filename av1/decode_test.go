package av1

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unsafe"
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

func decodeIVF(t *testing.T, path string, threads int) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 32 || string(data[:4]) != "DKIF" {
		t.Fatalf("%s: not an IVF file", path)
	}

	d := Decoder{Threads: threads}

	return decodeIVFWith(t, &d, path)
}

// decodeIVFWith decodes into a decoder the caller has already set up.
func decodeIVFWith(t *testing.T, d *Decoder, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer

	for _, chunk := range ivfChunks(data) {
		pics, err := d.DecodeOBUs(chunk)
		if err != nil {
			t.Fatalf("%s: decode: %v", filepath.Base(path), err)
		}
		for _, pic := range pics {
			planeRows(pic, &out)
			pic.Release()
		}
	}

	return out.Bytes()
}

// poisonedScratch fills a task scratch with a byte no field would hold, so a
// decoder that reads any part of one before writing it cannot match dav1d.
func poisonedScratch[P pixel]() *scratch[P] {
	s := new(scratch[P])
	b := unsafe.Slice((*byte)(unsafe.Pointer(s)), unsafe.Sizeof(*s))
	for i := range b {
		b[i] = 0xa5
	}

	return s
}

// TestScratchReuse decodes with a scratch pool that is already full of dirty
// buffers, which is what every frame after the first gets once the pool is
// warm. The streams still have to match dav1d.
func TestScratchReuse(t *testing.T) {
	dir := "testdata/streams"

	f, err := os.Open(filepath.Join(dir, "md5.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		name, want := fields[0], fields[1]

		t.Run(name, func(t *testing.T) {
			d := Decoder{Threads: 1}
			for range 8 {
				d.tc.Put(poisonedScratch[uint8]())
				d.tc.Put(poisonedScratch[uint16]())
			}

			sum := md5.Sum(decodeIVFWith(t, &d, filepath.Join(dir, name)))
			if hex.EncodeToString(sum[:]) != want {
				t.Fatalf("md5 = %s, want %s", hex.EncodeToString(sum[:]), want)
			}
		})
	}
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
			for _, threads := range []int{1, 0} {
				got := decodeIVF(t, filepath.Join(dir, name), threads)
				sum := md5.Sum(got)
				if hex.EncodeToString(sum[:]) != want {
					t.Fatalf("threads=%d: md5 = %s, want %s (%d bytes)",
						threads, hex.EncodeToString(sum[:]), want, len(got))
				}
			}
		})
		n++
	}

	if n == 0 {
		t.Fatal("no streams listed")
	}
}

// TestFrameParallel checks that keeping frames in flight decodes to the same
// bytes as one frame at a time, at several tile and frame thread counts.
func TestFrameParallel(t *testing.T) {
	dir := "testdata/streams"

	names, err := filepath.Glob(filepath.Join(dir, "*.ivf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no streams found")
	}

	for _, path := range names {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			chunks := ivfChunks(data)
			want := streamMD5(t, chunks, 1, 1)

			for _, tiles := range []int{1, 0} {
				for _, frames := range []int{2, 4, 0} {
					got := streamMD5(t, chunks, tiles, frames)
					if got != want {
						t.Fatalf("tiles=%d frames=%d: md5 = %s, want %s",
							tiles, frames, got, want)
					}
				}
			}
		})
	}
}

func streamMD5(t *testing.T, chunks [][]byte, tiles, frames int) string {
	t.Helper()

	var out bytes.Buffer
	d := Decoder{Threads: tiles, FrameThreads: frames}
	emit := func(pic *Picture) { planeRows(pic, &out) }
	if err := decodeStreaming(&d, chunks, emit); err != nil {
		t.Fatalf("tiles=%d frames=%d: %v", tiles, frames, err)
	}
	sum := md5.Sum(out.Bytes())

	return hex.EncodeToString(sum[:])
}

func ivfChunks(data []byte) [][]byte {
	var chunks [][]byte

	for p := 32; p+12 <= len(data); {
		n := int(binary.LittleEndian.Uint32(data[p:]))
		p += 12
		if n < 0 || p+n > len(data) {
			break
		}
		chunks = append(chunks, data[p:p+n])
		p += n
	}

	return chunks
}

// TestDecodeExternal decodes an arbitrary stream and compares against a live
// dav1d run. Set STREAM_PATH to use it.
func TestDecodeExternal(t *testing.T) {
	path := os.Getenv("STREAM_PATH")
	if path == "" {
		t.Skip("set STREAM_PATH")
	}

	got := decodeIVF(t, path, 0)
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

// decodeStreaming feeds the decoder the way a player would, keeping several
// temporal units in flight instead of draining each one.
func decodeStreaming(d *Decoder, chunks [][]byte, emit func(*Picture)) error {
	depth := max(d.FrameThreads, 1)

	for i, chunk := range chunks {
		if err := d.Send(chunk); err != nil {
			return err
		}
		if i < depth {
			continue
		}
		pic, err := d.Receive()
		if err != nil {
			return err
		}
		if pic != nil {
			emit(pic)
		}
	}

	for {
		pic, err := d.Receive()
		if err != nil {
			return err
		}
		if pic == nil {
			return nil
		}
		emit(pic)
	}
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
		chunks = ivfChunks(data)
	} else if chunks = obuChunks(data); chunks == nil {
		return "unknown container"
	}

	var out bytes.Buffer
	var d Decoder
	if n, _ := strconv.Atoi(os.Getenv("FRAME_THREADS")); n != 0 {
		d.FrameThreads = n
		emit := func(pic *Picture) { planeRows(pic, &out) }
		if err := decodeStreaming(&d, chunks, emit); err != nil {
			return "decode: " + err.Error()
		}
	} else {
		for _, chunk := range chunks {
			pics, err := d.DecodeOBUs(chunk)
			if err != nil {
				return "decode: " + err.Error()
			}
			for _, pic := range pics {
				planeRows(pic, &out)
			}
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

// BenchmarkDecode decodes a stream at a range of thread counts. Set STREAM_PATH.
func BenchmarkDecode(b *testing.B) {
	path := os.Getenv("STREAM_PATH")
	if path == "" {
		b.Skip("set STREAM_PATH")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}

	chunks := ivfChunks(data)

	counts := []int{1, 2, 4, 8, 0}

	for _, n := range counts {
		b.Run("tiles="+countName(n), func(b *testing.B) {
			var pixels int64
			for b.Loop() {
				d := Decoder{Threads: n, FrameThreads: 1}
				for _, chunk := range chunks {
					pics, err := d.DecodeOBUs(chunk)
					if err != nil {
						b.Fatal(err)
					}
					for _, pic := range pics {
						pixels += int64(pic.Width * pic.Height)
						pic.Release()
					}
				}
			}
			b.ReportMetric(float64(pixels)/b.Elapsed().Seconds()/1e6, "Mpixel/s")
		})
	}

	stream := func(b *testing.B, tiles, frames int) {
		var pixels int64
		emit := func(pic *Picture) {
			pixels += int64(pic.Width * pic.Height)
			pic.Release()
		}
		for b.Loop() {
			d := Decoder{Threads: tiles, FrameThreads: frames}
			if err := decodeStreaming(&d, chunks, emit); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(pixels)/b.Elapsed().Seconds()/1e6, "Mpixel/s")
	}

	for _, n := range counts {
		b.Run("frames="+countName(n), func(b *testing.B) { stream(b, 1, n) })
	}

	for _, n := range counts {
		b.Run("both="+countName(n), func(b *testing.B) { stream(b, n, n) })
	}
}

func countName(n int) string {
	if n == 0 {
		return "max"
	}

	return strconv.Itoa(n)
}

func firstLine(v any) string {
	s := strings.TrimSpace(strings.SplitN(fmt.Sprint(v), "\n", 2)[0])
	if len(s) > 90 {
		s = s[:90]
	}

	return s
}

// TestFrameSizeLimit checks the guard that keeps a corrupt header from asking
// for an unbounded allocation.
func TestFrameSizeLimit(t *testing.T) {
	huge := &frameHeader{width: [2]int{100000, 100000}, height: 100000}
	hd := &frameHeader{width: [2]int{1920, 1920}, height: 1080}

	for _, c := range []struct {
		name  string
		limit int
		hdr   *frameHeader
		want  error
	}{
		{"default rejects huge", 0, huge, ErrFrameTooLarge},
		{"default allows 1080p", 0, hd, nil},
		{"negative is unlimited", -1, huge, nil},
		{"explicit limit", 1000, &frameHeader{width: [2]int{100, 100}, height: 11}, ErrFrameTooLarge},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := Decoder{FrameSizeLimit: c.limit}
			d.frameHdr = c.hdr
			if err := d.checkFrameSize(); !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}
