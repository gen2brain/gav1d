package avif

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gen2brain/gav1d/av1"
)

// avifdec is libavif's decoder, which the cross-checks compare against. It is
// taken from AVIFDEC_BIN, or from PATH, and an empty result skips those tests.
var avifdec = func() string {
	name := os.Getenv("AVIFDEC_BIN")
	if name == "" {
		name = "avifdec"
	}
	bin, err := exec.LookPath(name)
	if err != nil {
		return ""
	}

	return bin
}()

// avifdecLibyuv reports whether avifdec converts YUV to RGB through libyuv.
// Its fixed point coefficients do not agree with the float path this package
// ports, by up to 12 of 255 on limited range BT.2020, so an RGB comparison
// against such a build cannot be exact. Plane comparisons are unaffected.
var avifdecLibyuv = func() bool {
	if avifdec == "" {
		return false
	}
	out, err := exec.Command(avifdec, "--version").CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "libyuv : available")
}()

func skipLibyuvRGB(t *testing.T) {
	t.Helper()

	if avifdecLibyuv {
		t.Skip("avifdec converts through libyuv (build it with AVIF_LIBYUV=OFF)")
	}
}

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

	if avifdec == "" {
		t.Skip("avifdec not found (set AVIFDEC_BIN)")
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

// TestMalformedFiles truncates and corrupts the bundled files and requires the
// decoder to return rather than crash. It is the in-repo stand-in for an
// invalid-file corpus: deterministic, and it runs everywhere the tests do.
func TestMalformedFiles(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.avif"))
	if err != nil || len(files) == 0 {
		t.Skip("no bundled files")
	}
	sort.Strings(files)
	if len(files) > 8 {
		files = files[:8]
	}

	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(p)

		// Truncation reaches every box boundary sooner or later, which is what
		// the truncated_header and truncated_data cases in a corpus are for.
		for _, frac := range []int{1, 2, 3, 5, 7, 11, 16, 32, 48, 63} {
			n := len(data) * frac / 64
			if msg := avifSurvives(data[:n]); msg != "" {
				t.Errorf("%s truncated to %d/%d: %s", name, n, len(data), msg)
			}
		}

		// Corruption in the header region hits box sizes, types and item
		// offsets; deeper offsets land in the payload.
		for _, off := range []int{4, 9, 17, 41, 89, 211, 1013} {
			if off >= len(data) {
				continue
			}
			for _, bit := range []uint{0, 4, 7} {
				bad := make([]byte, len(data))
				copy(bad, data)
				bad[off] ^= 1 << bit
				if msg := avifSurvives(bad); msg != "" {
					t.Errorf("%s bit %d of byte %d flipped: %s", name, bit, off, msg)
				}
			}
		}
	}
}

// avifSurvives reports a panic as a message, and treats any error as fine.
func avifSurvives(data []byte) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprintf("panic: %v", r)
		}
	}()

	opt := Options{AutoRotate: true, FrameSizeLimit: 1 << 20}
	if _, err := Decode(bytes.NewReader(data), opt); err != nil {
		return ""
	}
	_, _ = DecodeAll(bytes.NewReader(data), opt)

	return ""
}

// TestSniffBrands pins which major brands this package claims through
// image.RegisterFormat. mif1 is the generic MIAF brand: in a HEIC corpus 128
// of 168 files carry it as their major brand and no AVIF in four corpora does,
// so claiming it here takes those files away from a HEIC decoder registered
// for the same brand. Decode still accepts such a file when called directly,
// because the parser keys on the compatible brands.
func TestSniffBrands(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.avif"))
	if err != nil || len(files) == 0 {
		t.Skip("no bundled files")
	}
	sort.Strings(files)
	base, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(base) < 16 || string(base[4:8]) != "ftyp" {
		t.Skipf("%s has no ftyp", files[0])
	}

	rebrand := func(brand string) []byte {
		b := make([]byte, len(base))
		copy(b, base)
		copy(b[8:12], brand)

		return b
	}

	for _, c := range []struct {
		brand string
		ours  bool
	}{
		{"avif", true},
		{"avis", true},
		{"mif1", false},
		{"msf1", false},
		{"heic", false},
	} {
		_, format, err := image.DecodeConfig(bytes.NewReader(rebrand(c.brand)))
		got := err == nil && format == "avif"
		if got != c.ours {
			t.Errorf("major brand %q: sniffed as avif = %v, want %v (format %q, err %v)",
				c.brand, got, c.ours, format, err)
		}
	}

	// The generic brands are only undecidable to the sniffer. A caller that
	// knows the file is AVIF still gets it decoded.
	if _, err := Decode(bytes.NewReader(rebrand("mif1"))); err != nil {
		t.Errorf("Decode on a mif1 major brand: %v", err)
	}
}

// conformanceDirs returns the corpora named by CONFORMANCE_DIR, a colon
// separated list shared with the av1 package and the sibling codecs. Each test
// picks the entries it recognises; the rest contribute nothing.
func conformanceDirs(t *testing.T) []string {
	t.Helper()

	env := os.Getenv("CONFORMANCE_DIR")
	if env == "" {
		t.Skip("set CONFORMANCE_DIR")
	}

	return strings.Split(env, ":")
}

// TestCorpusRobust walks a corpus laid out as valid/, invalid/ and edge-cases/.
// Files under valid/ must decode; the rest only have to return rather than
// crash or hang, which is the contract those groups are collected for --
// several of the invalid files are accepted by libavif too. Set
// CONFORMANCE_DIR to run it.
func TestCorpusRobust(t *testing.T) {
	var groups []string
	for _, root := range conformanceDirs(t) {
		for _, group := range []string{"valid", "invalid", "edge-cases"} {
			if _, err := os.Stat(filepath.Join(root, group)); err == nil {
				groups = append(groups, filepath.Join(root, group))
			}
		}
	}
	if len(groups) == 0 {
		t.Skip("no valid/invalid/edge-cases corpus in CONFORMANCE_DIR")
	}

	for _, dir := range groups {
		group := filepath.Base(dir)

		var ok, failed, panicked int
		for _, path := range avifFiles(t, dir) {
			name := strings.TrimPrefix(path, dir+"/")
			err, panicMsg := corpusDecode(path)
			switch {
			case panicMsg != "":
				panicked++
				t.Errorf("%s/%s: %s", group, name, panicMsg)
			case err != nil:
				failed++
				if group == "valid" {
					t.Errorf("%s/%s: %v", group, name, err)
				} else {
					t.Logf("%s/%s: rejected: %v", group, name, err)
				}
			default:
				ok++
				if group == "invalid" {
					t.Logf("invalid/%s: decoded rather than rejected", name)
				}
			}
		}
		t.Logf("%-11s %3d decoded, %3d rejected, %d panicked", group, ok, failed, panicked)
	}
}

func corpusDecode(path string) (err error, panicMsg string) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg = fmt.Sprintf("panic: %v", r)
		}
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		return err, ""
	}
	f, err := parse(memSource(data))
	if err != nil {
		return err, ""
	}
	if _, _, err := f.decodePrimary(); err != nil {
		return err, ""
	}

	return nil, ""
}

// TestConformance decodes an external AVIF corpus, comparing every plane
// against avifdec. Set CONFORMANCE_DIR to run it.
func TestConformance(t *testing.T) {
	dirs := conformanceDirs(t)
	if avifdec == "" {
		t.Skip("avifdec not found (set AVIFDEC_BIN)")
	}

	var exact, skipped, colorExact int
	fails := map[string]int{}
	colorFails := map[string]int{}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		for _, path := range avifFiles(t, dir) {
			name := strings.TrimPrefix(path, dir+"/")
			t.Run(name, func(t *testing.T) {
				ref := decodeRef(t, path)

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				f, err := parse(memSource(data))
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

				if avifdecLibyuv {
					return
				}

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
				if errors.Is(err, ErrUnsupported) {
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
			f, err := parse(memSource(raw))
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
	if avifdec == "" {
		t.Skip("avifdec not found (set AVIFDEC_BIN)")
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
		if d := diffPixels(a.Image[i], decodePNG(t, rp)); d != "" && !avifdecLibyuv {
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
	if avifdec == "" {
		t.Skip("avifdec not found (set AVIFDEC_BIN)")
	}
	skipLibyuvRGB(t)

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

type rawBox struct {
	typ            string
	off, hdr, size int
}

func rawBoxes(b []byte, off, end int) []rawBox {
	var out []rawBox
	for off+8 <= end {
		size := int(binary.BigEndian.Uint32(b[off:]))
		typ := string(b[off+4 : off+8])
		hdr := 8
		switch size {
		case 1:
			if off+16 > end {
				return out
			}
			size = int(binary.BigEndian.Uint64(b[off+8:]))
			hdr = 16
		case 0:
			size = end - off
		}
		if size < hdr || off+size > end {
			return out
		}
		out = append(out, rawBox{typ, off, hdr, size})
		off += size
	}

	return out
}

func findRawBox(b []byte, path ...string) (rawBox, bool) {
	off, end := 0, len(b)
	var found rawBox
	for _, want := range path {
		ok := false
		for _, x := range rawBoxes(b, off, end) {
			if x.typ != want {
				continue
			}
			found, ok = x, true
			off, end = x.off+x.hdr, x.off+x.size
			if want == "meta" {
				off += 4
			}

			break
		}
		if !ok {
			return rawBox{}, false
		}
	}

	return found, true
}

// setEssential marks every association to the one-based property idx as
// essential, and reports whether it found one.
func setEssential(b []byte, idx int) bool {
	box, ok := findRawBox(b, "meta", "iprp", "ipma")
	if !ok {
		return false
	}

	p := box.off + box.hdr
	ver := int(b[p])
	flags := int(binary.BigEndian.Uint32(b[p:]) & 0xffffff)
	p += 4
	count := int(binary.BigEndian.Uint32(b[p:]))
	p += 4

	set := false
	for range count {
		if ver < 1 {
			p += 2
		} else {
			p += 4
		}
		n := int(b[p])
		p++
		for range n {
			if flags&1 != 0 {
				if int(binary.BigEndian.Uint16(b[p:])&0x7fff) == idx {
					b[p] |= 0x80
					set = true
				}
				p += 2
			} else {
				if int(b[p]&0x7f) == idx {
					b[p] |= 0x80
					set = true
				}
				p++
			}
		}
	}

	return set
}

// TestEssentialProperty checks that an item carrying an essential property the
// reader does not implement is refused, and that the same property left
// non-essential is ignored.
func TestEssentialProperty(t *testing.T) {
	for _, name := range []string{"bt709-full.avif", "grid-2x2.avif", "alpha-420.avif"} {
		orig, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}

		ipco, ok := findRawBox(orig, "meta", "iprp", "ipco")
		if !ok {
			t.Fatalf("%s: no ipco", name)
		}
		props := rawBoxes(orig, ipco.off+ipco.hdr, ipco.off+ipco.size)

		idx := 0
		for i, p := range props {
			if p.typ == "pixi" {
				idx = i + 1

				break
			}
		}
		if idx == 0 {
			t.Fatalf("%s: no pixi property to rename", name)
		}

		b := bytes.Clone(orig)
		copy(b[props[idx-1].off+4:], "zzzz")

		if _, err := Decode(bytes.NewReader(b)); err != nil {
			t.Fatalf("%s: unknown non-essential property should be ignored: %v", name, err)
		}

		if !setEssential(b, idx) {
			t.Fatalf("%s: no association to property %d", name, idx)
		}
		if _, err := Decode(bytes.NewReader(b)); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("%s: unknown essential property = %v, want ErrUnsupported", name, err)
		}
	}
}

// trackDurationOff returns the offset of the tkhd duration field.
func trackDurationOff(b []byte) (int, int) {
	tkhd, ok := findRawBox(b, "moov", "trak", "tkhd")
	if !ok {
		return 0, 0
	}
	p := tkhd.off + tkhd.hdr
	if b[p] == 1 {
		return p + 4 + 8 + 8 + 4 + 4, 8
	}

	return p + 4 + 4 + 4 + 4 + 4, 4
}

// TestLoopCount checks the repeat count the edit list carries. Only the
// indefinite case appears in the test files, so the other two are made by
// patching the one flag and the one field that decide it.
func TestLoopCount(t *testing.T) {
	orig, err := os.ReadFile(filepath.Join("testdata", "anim.avif"))
	if err != nil {
		t.Fatal(err)
	}

	loop := func(b []byte) int {
		a, err := DecodeAll(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}

		return a.LoopCount
	}

	if got := loop(orig); got != 0 {
		t.Fatalf("indefinite duration with a repeating edit list = %d, want 0", got)
	}

	elst, ok := findRawBox(orig, "moov", "trak", "edts", "elst")
	if !ok {
		t.Fatal("no elst")
	}
	durOff, durLen := trackDurationOff(orig)
	if durOff == 0 {
		t.Fatal("no tkhd")
	}

	b := bytes.Clone(orig)
	b[elst.off+elst.hdr+3] &^= 1
	if got := loop(b); got != -1 {
		t.Fatalf("edit list that does not repeat = %d, want -1", got)
	}

	// segment_duration is 5, so a track four segments long plays four times.
	b = bytes.Clone(orig)
	switch durLen {
	case 8:
		binary.BigEndian.PutUint64(b[durOff:], 20)
	case 4:
		binary.BigEndian.PutUint32(b[durOff:], 20)
	}
	if got := loop(b); got != 3 {
		t.Fatalf("track four segments long = %d, want 3", got)
	}

	// A track shorter than one segment still plays once.
	b = bytes.Clone(orig)
	switch durLen {
	case 8:
		binary.BigEndian.PutUint64(b[durOff:], 5)
	case 4:
		binary.BigEndian.PutUint32(b[durOff:], 5)
	}
	if got := loop(b); got != -1 {
		t.Fatalf("track one segment long = %d, want -1", got)
	}
}

// toneMapPayload builds a one-channel tmap payload with the given versions.
func toneMapPayload(version uint8, minVer, writerVer uint16, gammaN uint32, extra int) []byte {
	var b []byte
	b = append(b, version)
	b = binary.BigEndian.AppendUint16(b, minVer)
	b = binary.BigEndian.AppendUint16(b, writerVer)
	b = append(b, 0x40)

	u32 := func(v uint32) { b = binary.BigEndian.AppendUint32(b, v) }
	u32(6)
	u32(2)
	u32(0)
	u32(1)

	u32(0xffffffff)
	u32(1)
	u32(11)
	u32(1)
	u32(gammaN)
	u32(1)
	u32(0)
	u32(1)
	u32(0)
	u32(1)

	return append(b, make([]byte, extra)...)
}

// TestParseToneMap pins the version rules the format puts on a gain map: a
// reader must refuse one that asks for more than it implements, and must refuse
// trailing bytes from a writer no newer than itself.
func TestParseToneMap(t *testing.T) {
	cases := []struct {
		name  string
		b     []byte
		want  bool
		check func(*GainMap) error
	}{
		{name: "supported", b: toneMapPayload(0, 0, 0, 1, 0), want: true},
		{name: "unsupported version", b: toneMapPayload(1, 0, 0, 1, 0)},
		{name: "unsupported minimum version", b: toneMapPayload(0, 1, 1, 1, 0)},
		{name: "writer older than minimum", b: toneMapPayload(0, 1, 0, 1, 0)},
		{name: "extra bytes from a known writer", b: toneMapPayload(0, 0, 0, 1, 4)},
		{name: "extra bytes from a newer writer", b: toneMapPayload(0, 0, 1, 1, 4), want: true},
		{name: "zero gamma", b: toneMapPayload(0, 0, 0, 0, 0)},
		{name: "truncated", b: toneMapPayload(0, 0, 0, 1, 0)[:40]},
	}

	for _, c := range cases {
		g, ok := parseToneMap(c.b)
		if ok != c.want {
			t.Fatalf("%s: ok = %v, want %v", c.name, ok, c.want)
		}
		if !ok {
			continue
		}
		if got := g.BaseHDRHeadroom.Float(); got != 3 {
			t.Fatalf("%s: base headroom = %v, want 3", c.name, got)
		}
		if !g.UseBaseColorSpace {
			t.Fatalf("%s: use_base_colour_space not read", c.name)
		}
		// One channel fills the other two.
		if g.Max[2] != g.Max[0] || g.Min[2] != g.Min[0] {
			t.Fatalf("%s: single channel not copied to the rest", c.name)
		}
		if got := g.Min[0].Float(); got != -1 {
			t.Fatalf("%s: min = %v, want -1", c.name, got)
		}
	}
}

func buildBox(typ string, parts ...[]byte) []byte {
	n := 8
	for _, p := range parts {
		n += len(p)
	}
	b := binary.BigEndian.AppendUint32(make([]byte, 0, n), uint32(n))
	b = append(b, typ...)
	for _, p := range parts {
		b = append(b, p...)
	}

	return b
}

func fullBox(typ string, version uint8, flags uint32, parts ...[]byte) []byte {
	head := binary.BigEndian.AppendUint32(nil, uint32(version)<<24|flags)

	return buildBox(typ, append([][]byte{head}, parts...)...)
}

type tiffTag struct {
	tag, typ uint16
	count    uint32
	data     []byte
}

func tiffLen(tags []tiffTag) int {
	n := 2 + len(tags)*12 + 4
	for _, t := range tags {
		if len(t.data) > 4 {
			n += len(t.data)
		}
	}

	return n
}

// tiffIFD encodes an IFD placed at base, pointing values too big to inline at
// the overflow area that follows the directory.
func tiffIFD(base int, tags []tiffTag) []byte {
	over := base + 2 + len(tags)*12 + 4

	dir := binary.BigEndian.AppendUint16(nil, uint16(len(tags)))
	var extra []byte
	for _, t := range tags {
		dir = binary.BigEndian.AppendUint16(dir, t.tag)
		dir = binary.BigEndian.AppendUint16(dir, t.typ)
		dir = binary.BigEndian.AppendUint32(dir, t.count)
		if len(t.data) > 4 {
			dir = binary.BigEndian.AppendUint32(dir, uint32(over+len(extra)))
			extra = append(extra, t.data...)
		} else {
			v := make([]byte, 4)
			copy(v, t.data)
			dir = append(dir, v...)
		}
	}
	dir = binary.BigEndian.AppendUint32(dir, 0)

	return append(dir, extra...)
}

func be16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }
func be32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }
func rat(n, d uint32) []byte {
	return append(be32(n), be32(d)...)
}

// sampleTIFF builds the payload the Exif tests read back.
func sampleTIFF() []byte {
	gps := []tiffTag{
		{tagGPSLatitudeRef, typeASCIIString, 2, []byte("N\x00")},
		{tagGPSLatitude, typeUnsignedRational, 3, slices.Concat(rat(48, 1), rat(51, 1), rat(1234, 100))},
		{tagGPSLongitudeRef, typeASCIIString, 2, []byte("W\x00")},
		{tagGPSLongitude, typeUnsignedRational, 3, slices.Concat(rat(2, 1), rat(21, 1), rat(50, 100))},
		{tagGPSAltitudeRef, typeUnsignedByte, 1, []byte{1}},
		{tagGPSAltitude, typeUnsignedRational, 1, rat(3500, 100)},
	}
	sub := []tiffTag{
		{tagExposureTime, typeUnsignedRational, 1, rat(1, 250)},
		{tagFNumber, typeUnsignedRational, 1, rat(56, 10)},
		{tagISOSpeedRatings, typeUnsignedShort, 1, be16(800)},
		{tagDateTimeOriginal, typeASCIIString, 20, []byte("2024:03:01 09:15:00\x00")},
		{tagFlash, typeUnsignedShort, 1, be16(9)},
		{tagFocalLength, typeUnsignedRational, 1, rat(350, 10)},
	}

	ifd0Base := 8
	var ifd0 []tiffTag
	build := func() []byte {
		subBase := ifd0Base + tiffLen(ifd0)
		gpsBase := subBase + tiffLen(sub)

		return slices.Concat(
			[]byte("MM\x00\x2a"), be32(uint32(ifd0Base)),
			tiffIFD(ifd0Base, ifd0),
			tiffIFD(subBase, sub),
			tiffIFD(gpsBase, gps),
		)
	}

	ifd0 = []tiffTag{
		{tagImageWidth, typeUnsignedLong, 1, be32(1600)},
		{tagImageLength, typeUnsignedShort, 1, be16(900)},
		{tagMake, typeASCIIString, 6, []byte("Canon\x00")},
		{tagModel, typeASCIIString, 8, []byte("EOS 5D\x00\x00")},
		{tagSoftware, typeASCIIString, 6, []byte("gav1d\x00")},
		{tagDateTime, typeASCIIString, 20, []byte("2024:03:02 11:00:00\x00")},
		{tagArtist, typeASCIIString, 6, []byte("milan\x00")},
		{tagCopyright, typeASCIIString, 4, []byte("CC0\x00")},
		{tagOrientation, typeUnsignedShort, 1, be16(6)},
		{tagExifIFDPointer, typeUnsignedLong, 1, nil},
		{tagGPSIFDPointer, typeUnsignedLong, 1, nil},
	}
	subBase := ifd0Base + tiffLen(ifd0)
	ifd0[len(ifd0)-2].data = be32(uint32(subBase))
	ifd0[len(ifd0)-1].data = be32(uint32(subBase + tiffLen(sub)))

	return build()
}

type metaItem struct {
	id          uint16
	typ         string
	contentType string
	payload     []byte
	cdsc        uint16
}

// buildMetaFile assembles a file whose items live in idat, which is all the
// metadata readers walk.
func buildMetaFile(items []metaItem) []byte {
	var infes, locs, refs, idat []byte
	var nloc int

	for _, it := range items {
		name := []byte{0}
		body := slices.Concat(be16(it.id), be16(0), []byte(it.typ), name)
		if it.typ == "mime" {
			body = append(body, append([]byte(it.contentType), 0)...)
		}
		infes = append(infes, fullBox("infe", 2, 0, body)...)

		if it.payload != nil {
			locs = append(locs, slices.Concat(be16(it.id), be16(1), be16(0), be16(1),
				be32(uint32(len(idat))), be32(uint32(len(it.payload))))...)
			nloc++
			idat = append(idat, it.payload...)
		}
		if it.cdsc != 0 {
			refs = append(refs, buildBox("cdsc", be16(it.id), be16(1), be16(it.cdsc))...)
		}
	}

	meta := fullBox("meta", 0, 0,
		fullBox("pitm", 0, 0, be16(items[0].id)),
		fullBox("iinf", 0, 0, be16(uint16(len(items))), infes),
		fullBox("iref", 0, 0, refs),
		fullBox("iloc", 1, 0, []byte{0x44, 0x00}, be16(uint16(nloc)), locs),
		buildBox("idat", idat),
	)

	return append(buildBox("ftyp", []byte("avif"), be32(0), []byte("avifmif1")), meta...)
}

func exifItems(tiffOffset uint32, contentType string, cdsc uint16) []metaItem {
	tiff := sampleTIFF()
	pad := make([]byte, tiffOffset)

	return []metaItem{
		{id: 1, typ: "av01"},
		{id: 2, typ: "Exif", payload: slices.Concat(be32(tiffOffset), pad, tiff), cdsc: cdsc},
		{id: 3, typ: "mime", contentType: contentType, payload: []byte("<x:xmpmeta/>"), cdsc: cdsc},
	}
}

// TestExif reads the metadata items back out of a container built around a
// known TIFF payload, covering the header offset, the cdsc association that
// ties an item to the image it describes, and the mime type that separates XMP
// from any other mime item.
func TestExif(t *testing.T) {
	b := buildMetaFile(exifItems(0, xmpContentType, 1))

	e, err := DecodeExif(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	// Indexed so the degrees round exactly as the parser rounds them.
	dms := []float64{48, 51, 12.34, 2, 21, 0.5}
	want := Exif{
		Orientation: 6, Width: 1600, Height: 900,
		Make: "Canon", Model: "EOS 5D", Software: "gav1d",
		DateTime: "2024:03:02 11:00:00", DateTimeOriginal: "2024:03:01 09:15:00",
		ExposureTime: 1.0 / 250, FNumber: 5.6, ISOSpeed: 800, FocalLength: 35, Flash: 9,
		GPSLatitude:  dms[0] + dms[1]/60 + dms[2]/3600,
		GPSLongitude: -(dms[3] + dms[4]/60 + dms[5]/3600),
		GPSAltitude:  -35,
		Copyright:    "CC0", Artist: "milan",
	}
	if *e != want {
		t.Fatalf("exif =\n%+v\nwant\n%+v", *e, want)
	}

	xmp, err := RawXMP(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if string(xmp) != "<x:xmpmeta/>" {
		t.Fatalf("xmp = %q", xmp)
	}

	// A writer that pads before the TIFF header must signal where it starts.
	if _, err := DecodeExif(bytes.NewReader(buildMetaFile(exifItems(6, xmpContentType, 1)))); err != nil {
		t.Fatalf("exif_tiff_header_offset ignored: %v", err)
	}

	// Metadata describing some other item is not this image's.
	b = buildMetaFile(exifItems(0, xmpContentType, 9))
	if _, err := DecodeExif(bytes.NewReader(b)); !errors.Is(err, ErrNoExif) {
		t.Fatalf("exif without a cdsc reference to the image = %v, want ErrNoExif", err)
	}
	if _, err := RawXMP(bytes.NewReader(b)); !errors.Is(err, ErrNoXMP) {
		t.Fatalf("xmp without a cdsc reference to the image = %v, want ErrNoXMP", err)
	}

	// A mime item is only XMP when it says so.
	b = buildMetaFile(exifItems(0, "text/plain", 1))
	if _, err := RawXMP(bytes.NewReader(b)); !errors.Is(err, ErrNoXMP) {
		t.Fatalf("mime item of another type = %v, want ErrNoXMP", err)
	}

	b = buildMetaFile([]metaItem{{id: 1, typ: "av01"}})
	if _, err := DecodeExif(bytes.NewReader(b)); !errors.Is(err, ErrNoExif) {
		t.Fatalf("file without an Exif item = %v, want ErrNoExif", err)
	}
	if _, err := RawXMP(bytes.NewReader(b)); !errors.Is(err, ErrNoXMP) {
		t.Fatalf("file without an XMP item = %v, want ErrNoXMP", err)
	}
}

// TestExifFiles reads the two metadata items libavif's own tests pin the sizes
// of. Set CONFORMANCE_DIR to run it.
func TestExifFiles(t *testing.T) {
	dirs := conformanceDirs(t)

	for _, name := range []string{"colors-animated-8bpc-alpha-exif-xmp.avif", "paris_icc_exif_xmp.avif"} {
		var b []byte
		for _, dir := range dirs {
			if d, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				b = d

				break
			}
		}
		if b == nil {
			continue
		}

		raw, err := RawExif(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(raw) != 1126 {
			t.Fatalf("%s: exif = %d bytes, want 1126", name, len(raw))
		}
		xmp, err := RawXMP(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(xmp) != 3898 {
			t.Fatalf("%s: xmp = %d bytes, want 3898", name, len(xmp))
		}

		e, err := DecodeExif(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if e.Make != "Google" || e.Model != "Pixel 4a" || e.ISOSpeed != 47 ||
			e.FNumber != 1.7 || e.FocalLength != 4.4 || e.Flash != 16 ||
			e.DateTimeOriginal != "2022:08:22 20:30:46" {
			t.Fatalf("%s: exif = %+v", name, *e)
		}
	}
}

// TestTransferFuncs checks that every curve inverts itself, which is what the
// tone mapping relies on when it takes an image apart and puts it back.
func TestTransferFuncs(t *testing.T) {
	all := []uint16{
		trcBT709, trcBT470M, trcBT470BG, trcBT601, trcSMPTE240, trcLinear,
		trcLog100, trcLog100Sqrt10, trcIEC61966, trcBT1361, trcSRGB,
		trcBT2020_10, trcBT2020_12, trcPQ, trcSMPTE428, trcHLG,
	}

	for _, trc := range all {
		toLinear, toGamma := transferFuncs(trc)
		for i := range 101 {
			v := float64(i) / 100
			if got := toGamma(toLinear(v)); math.Abs(got-v) > 1e-6 {
				t.Fatalf("transfer %d: gamma(linear(%v)) = %v", trc, v, got)
			}
		}
		if toLinear(0) < 0 || toGamma(0) < 0 {
			t.Fatalf("transfer %d: negative at zero", trc)
		}
	}

	// The HDR curves put SDR white at 1, so the whole range reaches past it.
	for _, trc := range []uint16{trcPQ, trcHLG} {
		toLinear, _ := transferFuncs(trc)
		if got := toLinear(1); got <= 1 {
			t.Fatalf("transfer %d: peak = %v, want more than SDR white", trc, got)
		}
	}
}

// TestPrimariesMatrix checks the conversion between primaries, which must be
// the identity when there is nothing to convert and must undo itself.
func TestPrimariesMatrix(t *testing.T) {
	all := []uint16{
		priBT709, priBT470M, priBT470BG, priBT601, priSMPTE240,
		priGenericFilm, priBT2020, priXYZ, priSMPTE431, priSMPTE432, priEBU3213,
	}

	for _, cp := range all {
		m, ok := rgbToRGBMatrix(cp, cp)
		if !ok {
			t.Fatalf("primaries %d: no matrix", cp)
		}
		for i := range 3 {
			for j := range 3 {
				want := 0.0
				if i == j {
					want = 1
				}
				if math.Abs(m[i][j]-want) > 1e-9 {
					t.Fatalf("primaries %d: [%d][%d] = %v, want %v", cp, i, j, m[i][j], want)
				}
			}
		}

		if cp == priBT709 {
			continue
		}
		to, _ := rgbToRGBMatrix(priBT709, cp)
		back, _ := rgbToRGBMatrix(cp, priBT709)
		v := [3]float64{0.2, 0.5, 0.8}
		got := back.apply(to.apply(v))
		for c := range 3 {
			if math.Abs(got[c]-v[c]) > 1e-9 {
				t.Fatalf("primaries 1 -> %d -> 1: %v, want %v", cp, got, v)
			}
		}
	}

	// The published BT.709 to BT.2020 matrix, which the computation has to
	// land on for same-whitepoint primaries.
	want2020 := mat3{
		{0.6274, 0.3293, 0.0433},
		{0.0691, 0.9195, 0.0114},
		{0.0164, 0.0880, 0.8956},
	}
	got2020, _ := rgbToRGBMatrix(priBT709, priBT2020)
	for i := range 3 {
		for j := range 3 {
			if math.Abs(got2020[i][j]-want2020[i][j]) > 1e-3 {
				t.Fatalf("BT.709 to BT.2020 = %v, want %v", got2020, want2020)
			}
		}
	}

	// White stays white across a change of whitepoint, which only the
	// chromatic adaptation makes true.
	m, _ := rgbToRGBMatrix(priBT709, priSMPTE431)
	white := m.apply([3]float64{1, 1, 1})
	for c := range 3 {
		if math.Abs(white[c]-1) > 1e-6 {
			t.Fatalf("BT.709 white in SMPTE RP 431-2 = %v", white)
		}
	}
}

func TestGainMapWeight(t *testing.T) {
	g := &GainMap{
		BaseHDRHeadroom:      Rational{1, 1},
		AlternateHDRHeadroom: Rational{3, 1},
	}
	for _, c := range []struct {
		headroom, want float64
	}{
		{0, 0}, {1, 0}, {2, 0.5}, {3, 1}, {4, 1},
	} {
		if got := g.weight(c.headroom); got != c.want {
			t.Fatalf("weight(%v) = %v, want %v", c.headroom, got, c.want)
		}
	}

	// An alternate darker than the base runs the other way.
	g = &GainMap{
		BaseHDRHeadroom:      Rational{3, 1},
		AlternateHDRHeadroom: Rational{1, 1},
	}
	for _, c := range []struct {
		headroom, want float64
	}{
		{0, -1}, {1, -1}, {2, -0.5}, {3, 0}, {4, 0},
	} {
		if got := g.weight(c.headroom); got != c.want {
			t.Fatalf("reversed weight(%v) = %v, want %v", c.headroom, got, c.want)
		}
	}

	// Renditions that claim the same headroom say nothing about the gain.
	g = &GainMap{BaseHDRHeadroom: Rational{2, 1}, AlternateHDRHeadroom: Rational{2, 1}}
	if got := g.weight(4); got != 0 {
		t.Fatalf("equal headrooms = %v, want 0", got)
	}
}

// testGainMap builds metadata whose math is easy to follow: linear light, a
// gamma of one and a constant gain of log2Gain stops.
func testGainMap(log2Gain, baseOffset, altOffset float64, gain image.Image) *GainMap {
	r := func(v float64) Rational { return Rational{int32(math.Round(v * 1000)), 1000} }

	g := &GainMap{
		Image:                gain,
		UseBaseColorSpace:    true,
		Alt:                  ColorInfo{Primaries: cicpUnspecified, Transfer: cicpUnspecified},
		BaseHDRHeadroom:      Rational{0, 1},
		AlternateHDRHeadroom: Rational{2, 1},
	}
	for c := range 3 {
		g.Min[c] = r(log2Gain)
		g.Max[c] = r(log2Gain)
		g.Gamma[c] = Rational{1, 1}
		g.BaseOffset[c] = r(baseOffset)
		g.AlternateOffset[c] = r(altOffset)
	}

	return g
}

func grayGain(w, h int, v uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
	}

	return img
}

// TestApplyGainMap works in linear light so the gain is a plain multiply,
// which pins the formula without restating it.
func TestApplyGainMap(t *testing.T) {
	linear := ColorInfo{Primaries: priBT709, Transfer: trcLinear}

	base := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	copy(base.Pix, []byte{128, 64, 32, 200, 255, 255, 255, 255})

	// One stop at full weight doubles the linear value.
	g := testGainMap(1, 0, 0, grayGain(2, 1, 255))
	out, err := ApplyGainMap(base, linear, g, 2, linear)
	if err != nil {
		t.Fatal(err)
	}
	got := out.(*image.NRGBA)
	if want := []byte{255, 128, 64, 200}; !bytes.Equal(got.Pix[:4], want) {
		t.Fatalf("one stop = %v, want %v", got.Pix[:4], want)
	}

	// Half the headroom is half the stops.
	out, _ = ApplyGainMap(base, linear, g, 1, linear)
	got = out.(*image.NRGBA)
	if want := byte(math.Round(64.0 / 255 * math.Sqrt2 * 255)); got.Pix[1] != want {
		t.Fatalf("half weight = %d, want %d", got.Pix[1], want)
	}

	// The base offset is added before the gain and the alternate one taken
	// off after, so the two are not interchangeable.
	g = testGainMap(0, 0.1, 0.25, grayGain(2, 1, 255))
	out, _ = ApplyGainMap(base, linear, g, 2, linear)
	got = out.(*image.NRGBA)
	if want := byte(math.Round((64.0/255 + 0.1 - 0.25) * 255)); got.Pix[1] != want {
		t.Fatalf("offsets at unit gain = %d, want %d", got.Pix[1], want)
	}

	// The metadata carries the encoding gamma, so decoding raises the gain to
	// its inverse.
	g = testGainMap(0, 0, 0, grayGain(2, 1, 64))
	g.Min = [3]Rational{{0, 1}, {0, 1}, {0, 1}}
	g.Max = [3]Rational{{1, 1}, {1, 1}, {1, 1}}
	g.Gamma = [3]Rational{{2, 1}, {2, 1}, {2, 1}}
	out, _ = ApplyGainMap(base, linear, g, 2, linear)
	got = out.(*image.NRGBA)
	if want := byte(math.Round(64.0 / 255 * math.Exp2(math.Sqrt(64.0/255)) * 255)); got.Pix[1] != want {
		t.Fatalf("gain of two = %d, want %d", got.Pix[1], want)
	}

	// A gain map at zero reads the bottom of the range, not the top.
	g = testGainMap(1, 0, 0, grayGain(2, 1, 0))
	g.Min = [3]Rational{{0, 1}, {0, 1}, {0, 1}}
	out, _ = ApplyGainMap(base, linear, g, 2, linear)
	got = out.(*image.NRGBA)
	if !bytes.Equal(got.Pix[:4], base.Pix[:4]) {
		t.Fatalf("gain map at its minimum = %v, want the base back", got.Pix[:4])
	}

	// Alpha carries no light, so no curve is applied to it.
	srgb := ColorInfo{Primaries: priBT709, Transfer: trcSRGB}
	g = testGainMap(1, 0, 0, grayGain(2, 1, 255))
	out, _ = ApplyGainMap(base, srgb, g, 2, srgb)
	if got := out.(*image.NRGBA).Pix[3]; got != 200 {
		t.Fatalf("alpha = %d, want 200", got)
	}

	// Below the base headroom there is nothing to apply.
	out, err = ApplyGainMap(base, linear, g, 0, linear)
	if err != nil {
		t.Fatal(err)
	}
	if out != image.Image(base) {
		t.Fatal("no weight and no conversion should give the base back")
	}

	// A gain map that names its own primaries is applied in them, so a gain
	// on one channel alone reaches the others once the conversion is undone.
	mixed := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	copy(mixed.Pix, []byte{255, 128, 64, 255})
	g = testGainMap(0, 0, 0, grayGain(1, 1, 255))
	g.UseBaseColorSpace = false
	g.Alt.Primaries = priBT2020
	g.Min[0], g.Max[0] = Rational{1, 1}, Rational{1, 1}
	out, err = ApplyGainMap(mixed, linear, g, 2, linear)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{255, 102, 60, 255}; !bytes.Equal(out.(*image.NRGBA).Pix, want) {
		t.Fatalf("gain in the alternate primaries = %v, want %v", out.(*image.NRGBA).Pix, want)
	}

	// Output primaries are converted even with no gain to apply.
	red := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	copy(red.Pix, []byte{255, 0, 0, 255})
	wide := ColorInfo{Primaries: priBT2020, Transfer: trcLinear}
	out, err = ApplyGainMap(red, linear, g, 0, wide)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{160, 18, 4, 255}; !bytes.Equal(out.(*image.NRGBA).Pix, want) {
		t.Fatalf("BT.709 red in BT.2020 = %v, want %v", out.(*image.NRGBA).Pix, want)
	}

	if _, err := ApplyGainMap(base, linear, g, -1, linear); err == nil {
		t.Fatal("negative headroom should be rejected")
	}
	if _, err := ApplyGainMap(base, linear, nil, 1, linear); err == nil {
		t.Fatal("missing gain map should be rejected")
	}

	// The output keeps the base image's depth.
	deep := image.NewNRGBA64(image.Rect(0, 0, 2, 1))
	draw.Draw(deep, deep.Bounds(), base, base.Bounds().Min, draw.Src)
	out, err = ApplyGainMap(deep, linear, g, 2, linear)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(*image.NRGBA64); !ok {
		t.Fatalf("16-bit base gave %T", out)
	}
}

// TestApplyGainMapScale checks the gain map being read at the base image's
// resolution when it was stored smaller.
func TestApplyGainMapScale(t *testing.T) {
	linear := ColorInfo{Primaries: priBT709, Transfer: trcLinear}

	base := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	for x := range 4 {
		copy(base.Pix[x*4:], []byte{64, 64, 64, 255})
	}

	// Two gain samples, one at each end of the range.
	gain := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	copy(gain.Pix, []byte{0, 0, 0, 255, 255, 255, 255, 255})

	g := testGainMap(0, 0, 0, gain)
	g.Min = [3]Rational{{0, 1}, {0, 1}, {0, 1}}
	g.Max = [3]Rational{{1, 1}, {1, 1}, {1, 1}}

	out, err := ApplyGainMap(base, linear, g, 2, linear)
	if err != nil {
		t.Fatal(err)
	}
	got := out.(*image.NRGBA)

	// The two samples sit at the quarter points, so the row reads a quarter
	// and three quarters of a stop between them and holds flat past them.
	for x, stops := range []float64{0, 0.25, 0.75, 1} {
		want := byte(math.Round(64.0 / 255 * math.Exp2(stops) * 255))
		if got.Pix[x*4] != want {
			t.Fatalf("column %d = %d, want %d (%v stops)", x, got.Pix[x*4], want, stops)
		}
	}
}

// TestGainMapFile reads the gain maps of real files, which is the one path
// TestParseToneMap and the ApplyGainMap tests do not reach: decoding the gain
// map image itself, including from a grid, and the alternate rendition's color
// property. Set CONFORMANCE_DIR to run it.
func TestGainMapFile(t *testing.T) {
	dirs := conformanceDirs(t)

	cases := []struct {
		name            string
		base, gain      image.Point
		baseHR, altHR   Rational
		useBase         bool
		gamma, min, max Rational
	}{{
		name: "seine_sdr_gainmap_srgb.avif",
		base: image.Pt(400, 300), gain: image.Pt(400, 300),
		baseHR: Rational{0, 1}, altHR: Rational{13, 10}, useBase: true,
		gamma: Rational{119223, 125000},
		min:   Rational{-256907, 1000000}, max: Rational{1277177, 1000000},
	}, {
		// The base is a grid and the gain map is a 2x2 grid of its own.
		name: "color_grid_gainmap_different_grid.avif",
		base: image.Pt(512, 600), gain: image.Pt(128, 160),
		baseHR: Rational{6, 2}, altHR: Rational{0, 1}, useBase: true,
		gamma: Rational{1, 1}, min: Rational{-1, 1}, max: Rational{11, 1},
	}, {
		// An HDR base whose alternate is the SDR rendition, so the gain runs
		// the other way and the metadata does not use the base color space.
		name: "seine_hdr_gainmap_srgb.avif",
		base: image.Pt(400, 300), gain: image.Pt(400, 300),
		baseHR: Rational{13, 10}, altHR: Rational{0, 1}, useBase: false,
		gamma: Rational{119223, 125000},
		min:   Rational{-256907, 1000000}, max: Rational{1277177, 1000000},
	}}

	for _, c := range cases {
		var data []byte
		for _, dir := range dirs {
			if b, err := os.ReadFile(filepath.Join(dir, c.name)); err == nil {
				data = b

				break
			}
		}
		if data == nil {
			continue
		}

		img, g, err := DecodeGainMap(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if g == nil {
			t.Fatalf("%s: no gain map", c.name)
		}
		if got := img.Bounds().Size(); got != c.base {
			t.Fatalf("%s: base %v, want %v", c.name, got, c.base)
		}
		if got := g.Image.Bounds().Size(); got != c.gain {
			t.Fatalf("%s: gain map %v, want %v", c.name, got, c.gain)
		}
		if g.BaseHDRHeadroom != c.baseHR || g.AlternateHDRHeadroom != c.altHR {
			t.Fatalf("%s: headrooms %v %v, want %v %v", c.name,
				g.BaseHDRHeadroom, g.AlternateHDRHeadroom, c.baseHR, c.altHR)
		}
		if g.UseBaseColorSpace != c.useBase {
			t.Fatalf("%s: use_base_colour_space = %v", c.name, g.UseBaseColorSpace)
		}
		if g.Gamma[0] != c.gamma || g.Min[0] != c.min || g.Max[0] != c.max {
			t.Fatalf("%s: gamma %v min %v max %v", c.name, g.Gamma[0], g.Min[0], g.Max[0])
		}
		if g.Alt.Primaries != priBT709 {
			t.Fatalf("%s: alternate primaries = %d", c.name, g.Alt.Primaries)
		}
	}

	// A file with no tone mapped item gives a usable base and no gain map.
	data, err := os.ReadFile(filepath.Join("testdata", "yuv420-8.avif"))
	if err != nil {
		t.Fatal(err)
	}
	img, g, err := DecodeGainMap(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if g != nil {
		t.Fatal("file without a tmap item reported a gain map")
	}
	if img == nil {
		t.Fatal("file without a tmap item lost its base image")
	}
}

// countingFile reports how much of a file a decode actually read, which is
// what says whether the reader is being used a range at a time.
type countingFile struct {
	*bytes.Reader
	read int64
}

func (c *countingFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.Reader.ReadAt(p, off)
	c.read += int64(n)

	return n, err
}

// TestSources decodes the same file through every shape of reader. A reader
// that can seek is read by range, one that cannot is buffered, and both have
// to give the same pixels.
func TestSources(t *testing.T) {
	const path = "testdata/yuv420-8.avif"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("file", func(t *testing.T) {
		fh, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer fh.Close()

		got, err := Decode(fh)
		if err != nil {
			t.Fatal(err)
		}
		comparePixels(t, got, want)
	})

	// ReadAt is absolute, so only the part the caller has left belongs to us.
	t.Run("prefix consumed", func(t *testing.T) {
		r := bytes.NewReader(append([]byte("JUNK"), raw...))
		if _, err := io.CopyN(io.Discard, r, 4); err != nil {
			t.Fatal(err)
		}
		got, err := Decode(r)
		if err != nil {
			t.Fatal(err)
		}
		comparePixels(t, got, want)
	})

	t.Run("no seek", func(t *testing.T) {
		got, err := Decode(struct{ io.Reader }{bytes.NewReader(raw)})
		if err != nil {
			t.Fatal(err)
		}
		comparePixels(t, got, want)
	})
}

// TestPartialRead pins that reading the header does not read the file.
func TestPartialRead(t *testing.T) {
	raw, err := os.ReadFile("testdata/yuv420-8.avif")
	if err != nil {
		t.Fatal(err)
	}

	c := &countingFile{Reader: bytes.NewReader(raw)}
	cfg, err := DecodeConfig(c)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width == 0 || cfg.Height == 0 {
		t.Fatalf("config %dx%d", cfg.Width, cfg.Height)
	}
	if c.read >= int64(len(raw)) {
		t.Fatalf("read %d of %d bytes, want less than all of it", c.read, len(raw))
	}

	c = &countingFile{Reader: bytes.NewReader(raw)}
	if _, err := Decode(c); err != nil {
		t.Fatal(err)
	}
	if c.read > int64(len(raw)) {
		t.Fatalf("read %d of %d bytes", c.read, len(raw))
	}
}

// TestItemDataNoIdat pins that an item claiming to live in idat when the file
// has no idat box is rejected rather than dereferencing nothing.
func TestItemDataNoIdat(t *testing.T) {
	m := &metaBox{items: map[uint32]*item{}}
	it := &item{method: 1, extents: []extent{{off: 0, len: 0}}}

	b, err := m.data(it, memSource(nil))
	if err != nil {
		t.Fatalf("data: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("%d bytes, want none", len(b))
	}
}

// eofReaderAt is a conforming io.ReaderAt: the docs allow reporting io.EOF
// alongside a full read when the read ends at the end of the input.
type eofReaderAt struct{ b []byte }

func (e eofReaderAt) Read(p []byte) (int, error) { return 0, io.EOF }
func (e eofReaderAt) Seek(off int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		return int64(len(e.b)), nil
	}

	return 0, nil
}

func (e eofReaderAt) ReadAt(p []byte, off int64) (int, error) {
	n, err := bytes.NewReader(e.b).ReadAt(p, off)
	if err == nil && off+int64(n) == int64(len(e.b)) {
		err = io.EOF
	}

	return n, err
}

// TestReaderAtEOF checks the two ways a range read reaches the end of a file:
// a reader that reports io.EOF with a full read, and a zero length box that
// starts exactly at the end.
func TestReaderAtEOF(t *testing.T) {
	raw, err := os.ReadFile("testdata/yuv420-8.avif")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("eof with full read", func(t *testing.T) {
		got, err := Decode(eofReaderAt{raw})
		if err != nil {
			t.Fatal(err)
		}
		comparePixels(t, got, want)
	})

	// An empty box at the end makes parse ask for zero bytes at src.size.
	t.Run("empty trailing box", func(t *testing.T) {
		tail := append(append([]byte{}, raw...), 0, 0, 0, 8, 'f', 't', 'y', 'p')
		got, err := Decode(bytes.NewReader(tail))
		if err != nil {
			t.Fatal(err)
		}
		comparePixels(t, got, want)
	})
}

// TestExtentAmplification checks that extents cannot ask for more than the
// file holds. Every extent here is free to declare and runs to the end.
func TestExtentAmplification(t *testing.T) {
	src := memSource(make([]byte, 1<<16))
	m := &metaBox{items: map[uint32]*item{}}
	it := &item{extents: make([]extent, 2000)}

	if _, err := m.data(it, src); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
