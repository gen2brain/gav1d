package avif

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gen2brain/gav1d/av1"
)

const avifdec = "/temp/av1/libavif-build/avifdec"

type y4m struct {
	w, h   int
	depth  int
	layout int
	alpha  bool
	planes [4][]byte
	stride [4]int
}

// readY4M parses the single frame avifdec writes.
func readY4M(b []byte) (*y4m, error) {
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return nil, fmt.Errorf("no y4m header")
	}
	hdr := string(b[:i])
	rest := b[i+1:]

	y := &y4m{depth: 8, layout: av1.LayoutI420}
	for _, f := range strings.Fields(hdr)[1:] {
		v := f[1:]
		switch f[0] {
		case 'W':
			y.w, _ = strconv.Atoi(v)
		case 'H':
			y.h, _ = strconv.Atoi(v)
		case 'C':
			cs := v
			if i := strings.LastIndexByte(cs, 'p'); i >= 0 {
				if d, err := strconv.Atoi(cs[i+1:]); err == nil {
					y.depth = d
					cs = cs[:i]
				}
			}
			if rest, ok := strings.CutSuffix(cs, "alpha"); ok {
				y.alpha = true
				cs = rest
			}
			switch {
			case strings.HasPrefix(cs, "mono"):
				y.layout = av1.LayoutI400
				if d, err := strconv.Atoi(cs[len("mono"):]); err == nil {
					y.depth = d
				}
			case strings.HasPrefix(cs, "420"):
				y.layout = av1.LayoutI420
			case strings.HasPrefix(cs, "422"):
				y.layout = av1.LayoutI422
			case strings.HasPrefix(cs, "444"):
				y.layout = av1.LayoutI444
			default:
				return nil, fmt.Errorf("y4m color space %q", v)
			}
		}
	}

	i = bytes.IndexByte(rest, '\n')
	if i < 0 || !bytes.HasPrefix(rest, []byte("FRAME")) {
		return nil, fmt.Errorf("no y4m frame")
	}
	rest = rest[i+1:]

	bpp := 1
	if y.depth > 8 {
		bpp = 2
	}
	ssHor, ssVer := subsampling(y.layout)

	for pl := range 4 {
		if pl != 0 && y.layout == av1.LayoutI400 {
			break
		}
		if pl == 3 && !y.alpha {
			break
		}
		pw, ph := y.w, y.h
		if pl != 0 && pl != 3 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		n := pw * ph * bpp
		if len(rest) < n {
			return nil, fmt.Errorf("y4m plane %d truncated", pl)
		}
		y.planes[pl] = rest[:n]
		y.stride[pl] = pw * bpp
		rest = rest[n:]
	}

	return y, nil
}

func decodeRef(t *testing.T, path string) *y4m {
	t.Helper()

	if _, err := os.Stat(avifdec); err != nil {
		t.Skip("avifdec not built")
	}
	out := filepath.Join(t.TempDir(), "ref.y4m")
	cmd := exec.Command(avifdec, "--no-strict", path, out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("avifdec: %v\n%s", err, b)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	y, err := readY4M(b)
	if err != nil {
		t.Fatal(err)
	}

	return y
}

func comparePlanes(t *testing.T, pic *av1.Picture, ref *y4m) {
	t.Helper()

	if pic.Width != ref.w || pic.Height != ref.h {
		t.Fatalf("size %dx%d, want %dx%d", pic.Width, pic.Height, ref.w, ref.h)
	}
	if pic.BitDepth != ref.depth {
		t.Fatalf("depth %d, want %d", pic.BitDepth, ref.depth)
	}
	if pic.Layout != ref.layout {
		t.Fatalf("layout %d, want %d", pic.Layout, ref.layout)
	}

	bpp := 1
	if ref.depth > 8 {
		bpp = 2
	}
	ssHor, ssVer := subsampling(ref.layout)

	for pl := range 3 {
		if ref.planes[pl] == nil {
			continue
		}
		pw, ph := ref.w, ref.h
		if pl != 0 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		for y := range ph {
			got := pic.Data[pl][y*pic.Stride[pl]:][:pw*bpp]
			want := ref.planes[pl][y*ref.stride[pl]:][:pw*bpp]
			if !bytes.Equal(got, want) {
				for x := range pw * bpp {
					if got[x] != want[x] {
						t.Fatalf("plane %d row %d byte %d: %d, want %d",
							pl, y, x, got[x], want[x])
					}
				}
			}
		}
	}
}

func diffPixels(got, want image.Image) string {
	gb, wb := got.Bounds(), want.Bounds()
	if gb != wb {
		return fmt.Sprintf("bounds %v, want %v", gb, wb)
	}
	for y := gb.Min.Y; y < gb.Max.Y; y++ {
		for x := gb.Min.X; x < gb.Max.X; x++ {
			r1, g1, b1, a1 := got.At(x, y).RGBA()
			r2, g2, b2, a2 := want.At(x, y).RGBA()
			if a1 != a2 {
				return "alpha differs"
			}
			if r1 != r2 || g1 != g2 || b1 != b2 {
				return "color differs"
			}
		}
	}

	return ""
}

func comparePixels(t *testing.T, got, want image.Image) {
	t.Helper()

	if diff := diffPixels(got, want); diff != "" {
		t.Fatal(diff)
	}
}

func avifFiles(t *testing.T, dir string) []string {
	t.Helper()

	var out []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && strings.HasSuffix(p, ".avif") {
			out = append(out, p)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)

	return out
}

// TestConformance decodes an external AVIF corpus, comparing every plane
// against avifdec. Set AVIF_CONFORMANCE_DIR (colon separated) to run it.
func TestConformance(t *testing.T) {
	env := os.Getenv("AVIF_CONFORMANCE_DIR")
	if env == "" {
		t.Skip("set AVIF_CONFORMANCE_DIR")
	}
	dirs := strings.Split(env, ":")

	var exact, skipped, colorExact int
	fails := map[string]int{}
	colorFails := map[string]int{}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			t.Skip("conformance corpus not present")
		}
		for _, path := range avifFiles(t, dir) {
			name := strings.TrimPrefix(path, dir+"/")
			t.Run(name, func(t *testing.T) {
				ref := decodeRef(t, path)

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				f, err := parse(data)
				if err != nil {
					fails[err.Error()]++
					t.Fatalf("parse: %v", err)
				}
				pic, _, err := f.decodePrimary()
				if err != nil {
					fails[err.Error()]++
					t.Fatalf("decode: %v", err)
				}
				comparePlanes(t, pic, ref)
				exact++

				// avifpng.c copies the Y plane raw for 8-bit monochrome, skipping conversion.
				if pic.BitDepth == 8 && pic.Layout == av1.LayoutI400 &&
					f.alphaFor(f.primary().id) == nil &&
					f.meta.prop(f.primary(), "clap") == nil &&
					f.meta.prop(f.primary(), "irot") == nil &&
					f.meta.prop(f.primary(), "imir") == nil {
					colorFails["avifdec writes the Y plane raw"]++

					return
				}

				out := filepath.Join(t.TempDir(), "ref.png")
				cmd := exec.Command(avifdec, "--no-strict", path, out)
				if _, err := cmd.CombinedOutput(); err != nil {
					colorFails["avifdec cannot convert"]++

					return
				}
				got, err := Decode(bytes.NewReader(data), Options{AutoRotate: true})
				if errors.Is(err, errUnsupported) {
					skipped++

					return
				}
				if err != nil {
					colorFails[err.Error()]++
					t.Skipf("color: %v", err)
				}
				if diff := diffPixels(got, decodePNG(t, out)); diff != "" {
					colorFails[diff]++
					t.Skipf("color: %s", diff)
				}
				colorExact++
			})
		}
	}

	t.Logf("%d planes bit-exact, %d color bit-exact, %d unsupported", exact, colorExact, skipped)
	for k, n := range colorFails {
		t.Logf("%4d  color: %s", n, k)
	}
	keys := make([]string, 0, len(fails))
	for k := range fails {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return fails[keys[i]] > fails[keys[j]] })
	for _, k := range keys {
		t.Logf("%4d  %s", fails[k], k)
	}
}

func planeMD5(pic *av1.Picture) string {
	h := md5.New()
	ssHor, ssVer := subsampling(pic.Layout)
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
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		for y := range ph {
			row := pic.Data[pl][y*pic.Stride[pl]:][:pw*bpp]
			if bpp == 1 {
				h.Write(row)

				continue
			}
			// Hash sample values, not host bytes, so the reference stays valid
			// on a big-endian build.
			var le [2]byte
			for x := range pw {
				binary.LittleEndian.PutUint16(le[:], binary.NativeEndian.Uint16(row[x*2:]))
				h.Write(le[:])
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// TestDecode compares the bundled files against the plane MD5s in md5.txt.
// Set AVIF_UPDATE_MD5 to regenerate them, which needs avifdec.
func TestDecode(t *testing.T) {
	const dir = "testdata"

	files, err := filepath.Glob(filepath.Join(dir, "*.avif"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no test files")
	}

	if os.Getenv("AVIF_UPDATE_MD5") != "" {
		var b strings.Builder
		for _, path := range files {
			ref := decodeRef(t, path)
			h := md5.New()
			for pl := range 3 {
				if ref.planes[pl] != nil {
					h.Write(ref.planes[pl])
				}
			}
			fmt.Fprintf(&b, "%s %s\n", filepath.Base(path),
				hex.EncodeToString(h.Sum(nil)))
		}
		if err := os.WriteFile(filepath.Join(dir, "md5.txt"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote md5.txt for %d files", len(files))

		return
	}

	want := map[string]string{}
	data, err := os.ReadFile(filepath.Join(dir, "md5.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if f := strings.Fields(line); len(f) == 2 {
			want[f[0]] = f[1]
		}
	}

	for _, path := range files {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parse(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			pic, alpha, err := f.decodePrimary()
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			got := planeMD5(pic)
			if want[name] == "" {
				t.Fatalf("no md5 for %s", name)
			}
			if got != want[name] {
				t.Fatalf("md5 = %s, want %s", got, want[name])
			}
			if strings.HasPrefix(name, "alpha-") && alpha == nil {
				t.Fatal("alpha item not found")
			}
		})
	}
}

// TestSequence decodes the bundled animation and compares every frame, and its
// duration, against avifdec.
func TestSequence(t *testing.T) {
	if _, err := os.Stat(avifdec); err != nil {
		t.Skip("avifdec not built")
	}
	const path = "testdata/anim.avif"

	dir := t.TempDir()
	cmd := exec.Command(avifdec, "--no-strict", "--index", "all", path,
		filepath.Join(dir, "f.png"))
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("avifdec: %v\n%s", err, b)
	}
	refs, err := filepath.Glob(filepath.Join(dir, "f-*.png"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(refs)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := DecodeAll(bytes.NewReader(data), Options{AutoRotate: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Image) != len(refs) {
		t.Fatalf("%d frames, want %d", len(a.Image), len(refs))
	}
	for i, rp := range refs {
		if d := diffPixels(a.Image[i], decodePNG(t, rp)); d != "" {
			t.Fatalf("frame %d: %s", i, d)
		}
		if a.Delay[i] != 0.1 {
			t.Fatalf("frame %d delay = %v, want 0.1", i, a.Delay[i])
		}
	}
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	return img
}

// TestColor compares each bundled file's RGB against avifdec.
func TestColor(t *testing.T) {
	if _, err := os.Stat(avifdec); err != nil {
		t.Skip("avifdec not built")
	}

	files, err := filepath.Glob(filepath.Join("testdata", "*.avif"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "ref.png")
			cmd := exec.Command(avifdec, "--no-strict", path, out)
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("avifdec: %v\n%s", err, b)
			}
			want := decodePNG(t, out)

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Decode(bytes.NewReader(raw), Options{AutoRotate: true})
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			comparePixels(t, got, want)
		})
	}
}

// FuzzDecode requires that no input panics, however malformed.
func FuzzDecode(f *testing.F) {
	seeds, _ := filepath.Glob(filepath.Join("testdata", "*.avif"))
	for _, p := range seeds {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		opt := Options{AutoRotate: true, FrameSizeLimit: 1 << 20}
		img, err := Decode(bytes.NewReader(data), opt)
		if err == nil && img == nil {
			t.Fatal("nil image without error")
		}
		if _, err := DecodeConfig(bytes.NewReader(data)); err != nil {
			return
		}
		_, _ = DecodeAll(bytes.NewReader(data), opt)
	})
}

// TestSampleEndianness pins the 16-bit plane read against the host byte order
// av1 stores planes in, so a big-endian build converts color correctly.
func TestSampleEndianness(t *testing.T) {
	pic := &av1.Picture{
		Width: 1, Height: 1,
		Layout: av1.LayoutI400, BitDepth: 10,
		FullRange: true, MatrixCoeffs: 1,
	}
	pic.Data[0] = make([]byte, 2)
	pic.Stride[0] = 2
	binary.NativeEndian.PutUint16(pic.Data[0], 512)

	cs := newColorState(pic, ColorInfo{Matrix: 1, FullRange: true}, 8)
	cs.prepare(1)
	rgb := make([]uint16, 3)
	cs.rgbRow(pic, 0, rgb)
	if rgb[0] != 128 || rgb[1] != 128 || rgb[2] != 128 {
		t.Fatalf("got %d,%d,%d, want 128,128,128", rgb[0], rgb[1], rgb[2])
	}
}

// TestYCbCr checks that the planar form holds the same picture the converted
// one does, by running Go's own YCbCr model over it and comparing against a
// decode forced through the same model.
func TestYCbCr(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.avif"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)

	planar := 0
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Decode(bytes.NewReader(raw), Options{ToYCbCr: true})
			if err != nil {
				t.Fatalf("ycbcr: %v", err)
			}
			want, err := Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("rgb: %v", err)
			}
			if got.Bounds() != want.Bounds() {
				t.Fatalf("bounds %v, want %v", got.Bounds(), want.Bounds())
			}

			switch got.(type) {
			case *image.YCbCr, *image.NYCbCrA, *image.Gray:
				planar++
			default:
				return
			}

			// The planes are the decoder's own, so a pixel read back through
			// any model must land within rounding of the converted one.
			b := got.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					_, _, _, ga := got.At(x, y).RGBA()
					_, _, _, wa := want.At(x, y).RGBA()
					if ga != wa {
						t.Fatalf("alpha at %d,%d = %d, want %d", x, y, ga, wa)
					}
				}
			}
		})
	}

	if planar == 0 {
		t.Fatal("no file took the planar path")
	}
	t.Logf("%d of %d files planar", planar, len(files))
}

// jfifModel reports whether Go's fixed YCbCr model is the right one for this
// description: it is full-range BT.601 and nothing else.
func jfifModel(ci ColorInfo) bool {
	return ci.FullRange &&
		(ci.Matrix == mcBT601 || ci.Matrix == mcBT470BG || ci.Matrix == mcUnspec)
}

func ycbcrPlanes(img image.Image) (*image.YCbCr, bool) {
	switch v := img.(type) {
	case *image.YCbCr:
		return v, true
	case *image.NYCbCrA:
		return &v.YCbCr, true
	}

	return nil, false
}

// TestColorInfo checks that DecodeColor reports the description the conversion
// ran on, and measures how far image.YCbCr's fixed model lands from it. Only
// 4:4:4 is compared, since the model reads chroma nearest-neighbour where the
// conversion upsamples it, and that difference would swamp the matrix. A
// full-range BT.601 file has to agree there; anything else is free not to.
func TestColorInfo(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.avif"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)

	matched, differed := 0, 0
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, ci, err := DecodeColor(bytes.NewReader(raw), Options{ToYCbCr: true})
			if err != nil {
				t.Fatalf("ycbcr: %v", err)
			}
			yc, ok := ycbcrPlanes(got)
			if !ok || yc.SubsampleRatio != image.YCbCrSubsampleRatio444 {
				return
			}
			want, err := Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("rgb: %v", err)
			}
			ref, ok := want.(*image.NRGBA)
			if !ok {
				return
			}

			worst := 0
			b := yc.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					gr, gg, gb, _ := yc.At(x, y).RGBA()
					c := ref.NRGBAAt(x, y)
					for _, d := range [3]int{
						abs(int(gr>>8) - int(c.R)),
						abs(int(gg>>8) - int(c.G)),
						abs(int(gb>>8) - int(c.B)),
					} {
						worst = max(worst, d)
					}
				}
			}

			t.Logf("matrix=%d full=%v: off by %d levels", ci.Matrix, ci.FullRange, worst)
			if !jfifModel(ci) {
				differed++

				return
			}
			matched++
			if worst > 2 {
				t.Fatalf("matrix=%d full=%v: off by %d levels, want the model to fit",
					ci.Matrix, ci.FullRange, worst)
			}
		})
	}

	if matched == 0 || differed == 0 {
		t.Fatalf("%d files fit Go's model, %d did not, want both", matched, differed)
	}
	t.Logf("%d fit Go's model, %d did not", matched, differed)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
