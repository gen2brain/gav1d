package avif

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/color"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func synthNRGBA(w, h int, alpha bool) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			c := color.NRGBA{
				R: uint8(x * 255 / max(w-1, 1)),
				G: uint8(y * 255 / max(h-1, 1)),
				B: uint8((x/8 ^ y/8) * 8 & 0xff),
				A: 0xff,
			}
			if alpha {
				c.A = uint8(min(x*255/max(w-1, 1)+16, 255))
			}
			m.SetNRGBA(x, y, c)
		}
	}

	return m
}

func rgbPSNR(t *testing.T, a, b image.Image) float64 {
	t.Helper()

	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		t.Fatalf("size %v against %v", ab, bb)
	}

	var sum float64
	var n int

	for y := range ab.Dy() {
		for x := range ab.Dx() {
			r0, g0, b0, _ := a.At(ab.Min.X+x, ab.Min.Y+y).RGBA()
			r1, g1, b1, _ := b.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
			for _, d := range []float64{
				float64(r0>>8) - float64(r1>>8),
				float64(g0>>8) - float64(g1>>8),
				float64(b0>>8) - float64(b1>>8),
			} {
				sum += d * d
				n++
			}
		}
	}

	if sum == 0 {
		return math.Inf(1)
	}

	return 10 * math.Log10(255*255/(sum/float64(n)))
}

// TestEncodeRoundTrip holds that our own decoder reads back what we wrote.
func TestEncodeRoundTrip(t *testing.T) {
	sizes := []image.Point{{1, 1}, {2, 3}, {7, 5}, {16, 16}, {33, 17}, {64, 64}, {129, 71}}

	for _, sz := range sizes {
		src := synthNRGBA(sz.X, sz.Y, false)

		for _, q := range []int{30, 60, 90} {
			var buf bytes.Buffer
			if err := Encode(&buf, src, EncodeOptions{Quality: q}); err != nil {
				t.Fatalf("%dx%d q%d: %v", sz.X, sz.Y, q, err)
			}

			got, err := Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("%dx%d q%d: decode: %v", sz.X, sz.Y, q, err)
			}

			if got.Bounds().Dx() != sz.X || got.Bounds().Dy() != sz.Y {
				t.Fatalf("%dx%d q%d: bounds %v", sz.X, sz.Y, q, got.Bounds())
			}

			if sz.X < 16 || sz.Y < 16 || q < 90 {
				continue
			}

			if p := rgbPSNR(t, src, got); p < 30 {
				t.Errorf("%dx%d q%d: psnr %.1f dB", sz.X, sz.Y, q, p)
			}
		}
	}
}

// TestEncodeGray holds that a grayscale source stays monochrome.
func TestEncodeGray(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 64, 48))
	for y := range 48 {
		for x := range 64 {
			src.SetGray(x, y, color.Gray{Y: uint8((x*4 + y*2) & 0xff)})
		}
	}

	var buf bytes.Buffer
	if err := Encode(&buf, src, EncodeOptions{Lossless: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode(bytes.NewReader(buf.Bytes()), Options{ToYCbCr: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	g, ok := got.(*image.Gray)
	if !ok {
		t.Fatalf("decoded %T, want *image.Gray", got)
	}

	for y := range 48 {
		for x := range 64 {
			if g.GrayAt(x, y) != src.GrayAt(x, y) {
				t.Fatalf("pixel %d,%d: got %v, want %v", x, y,
					g.GrayAt(x, y), src.GrayAt(x, y))
			}
		}
	}
}

// TestEncodeYCbCrLossless holds that planar input survives untouched, which is
// the one path with no color conversion between the source and the bitstream.
func TestEncodeYCbCrLossless(t *testing.T) {
	src := image.NewYCbCr(image.Rect(0, 0, 96, 64), image.YCbCrSubsampleRatio420)
	for i := range src.Y {
		src.Y[i] = uint8(i * 7 % 256)
	}
	for i := range src.Cb {
		src.Cb[i] = uint8(i * 11 % 256)
		src.Cr[i] = uint8(i * 13 % 256)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, src, EncodeOptions{Lossless: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode(bytes.NewReader(buf.Bytes()), Options{ToYCbCr: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	out, ok := got.(*image.YCbCr)
	if !ok {
		t.Fatalf("decoded %T, want *image.YCbCr", got)
	}

	// The decoded planes carry the decoder's own aligned stride, so the
	// comparison has to go through the offsets rather than the raw slices.
	for y := range 64 {
		for x := range 96 {
			if out.Y[out.YOffset(x, y)] != src.Y[src.YOffset(x, y)] {
				t.Fatalf("luma at %d,%d", x, y)
			}
		}
	}

	for y := range 32 {
		for x := range 48 {
			o, i := out.COffset(x*2, y*2), src.COffset(x*2, y*2)
			if out.Cb[o] != src.Cb[i] || out.Cr[o] != src.Cr[i] {
				t.Fatalf("chroma at %d,%d", x, y)
			}
		}
	}
}

// TestEncodeAlpha holds that the alpha item survives the round trip.
func TestEncodeAlpha(t *testing.T) {
	src := synthNRGBA(64, 48, true)

	var buf bytes.Buffer
	if err := Encode(&buf, src, EncodeOptions{Quality: 90, QualityAlpha: 100}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	out, ok := got.(*image.NRGBA)
	if !ok {
		t.Fatalf("decoded %T, want *image.NRGBA", got)
	}

	for y := range 48 {
		for x := range 64 {
			want := src.NRGBAAt(x, y).A
			if a := out.NRGBAAt(x, y).A; a != want {
				t.Fatalf("alpha at %d,%d: got %d, want %d", x, y, a, want)
			}
		}
	}
}

// TestEncodeAvifdec checks the container against libavif, which is the only
// reader here that did not also write the file.
func TestEncodeAvifdec(t *testing.T) {
	bin, err := exec.LookPath("avifdec")
	if err != nil {
		t.Skip("avifdec not found")
	}

	cases := []struct {
		name  string
		img   image.Image
		alpha bool
	}{
		{"rgb", synthNRGBA(129, 71, false), false},
		{"rgba", synthNRGBA(64, 48, true), true},
		{"gray", func() image.Image {
			g := image.NewGray(image.Rect(0, 0, 48, 32))
			for i := range g.Pix {
				g.Pix[i] = uint8(i * 5 % 256)
			}

			return g
		}(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Encode(&buf, tc.img, EncodeOptions{Quality: 90}); err != nil {
				t.Fatalf("encode: %v", err)
			}

			dir := t.TempDir()
			in := filepath.Join(dir, "out.avif")
			out := filepath.Join(dir, "out.png")
			if err := os.WriteFile(in, buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(bin, in, out)
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("avifdec: %v\n%s", err, b)
			}

			ref, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}

			libavif, _, err := image.Decode(bytes.NewReader(ref))
			if err != nil {
				t.Fatalf("png: %v", err)
			}

			ours, err := Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			p := rgbPSNR(t, ours, libavif)
			if p < 45 {
				t.Errorf("libavif disagrees, psnr %.1f dB", p)
			}
			t.Logf("%d bytes, libavif agrees at %.1f dB", buf.Len(), p)
		})
	}
}

func TestEncodeSpeed(t *testing.T) {
	src := synthNRGBA(96, 80, true)

	enc := func(o EncodeOptions) []byte {
		var buf bytes.Buffer
		if err := Encode(&buf, src, o); err != nil {
			t.Fatalf("%+v: %v", o, err)
		}
		if _, err := Decode(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatalf("%+v: decode: %v", o, err)
		}

		return buf.Bytes()
	}

	slowest := enc(EncodeOptions{Quality: 60, Speed: 0})
	fastest := enc(EncodeOptions{Quality: 60, Speed: 10})
	if bytes.Equal(slowest, fastest) {
		t.Error("speed 0 and speed 10 coded the same bytes")
	}

	if got := enc(EncodeOptions{Quality: 60, Speed: -1}); !bytes.Equal(got,
		enc(EncodeOptions{Quality: 60, Speed: DefaultSpeed})) {
		t.Error("a negative speed did not select DefaultSpeed")
	}

	if got := enc(EncodeOptions{Quality: 60}); !bytes.Equal(got, slowest) {
		t.Error("the zero value did not select speed 0")
	}

	var buf bytes.Buffer
	if err := Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), enc(EncodeOptions{Speed: DefaultSpeed})) {
		t.Error("omitted options did not select DefaultSpeed")
	}
}

func TestEncodeTiles(t *testing.T) {
	for _, sz := range [][2]int{{4200, 2200}, {3648, 2736}} {
		src := synthNRGBA(sz[0], sz[1], true)

		var one, all bytes.Buffer
		if err := Encode(&one, src, EncodeOptions{Quality: 60, Speed: 10, Threads: 1}); err != nil {
			t.Fatalf("%v: %v", sz, err)
		}
		if err := Encode(&all, src, EncodeOptions{Quality: 60, Speed: 10}); err != nil {
			t.Fatalf("%v: %v", sz, err)
		}
		if !bytes.Equal(one.Bytes(), all.Bytes()) {
			t.Fatalf("%v: threads changed the output", sz)
		}

		img, err := Decode(bytes.NewReader(one.Bytes()))
		if err != nil {
			t.Fatalf("%v: %v", sz, err)
		}
		if b := img.Bounds(); b.Dx() != sz[0] || b.Dy() != sz[1] {
			t.Fatalf("%v: decoded %v", sz, b)
		}
		if p := rgbPSNR(t, src, img); p < 30 {
			t.Errorf("%v: psnr %.1f dB", sz, p)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	src := synthNRGBA(512, 512, false)

	b.ReportAllocs()
	b.SetBytes(512 * 512)

	for b.Loop() {
		var buf bytes.Buffer
		if err := Encode(&buf, src, EncodeOptions{Quality: 60}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFromImageGolden(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 4096, 4096))
	for c := range 1 << 24 {
		copy(m.Pix[4*c:], []uint8{uint8(c >> 16), uint8(c >> 8), uint8(c), 0xff})
	}

	p, err := fromImage(m)
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.New()
	sum.Write(p.y)
	sum.Write(p.u)
	sum.Write(p.v)

	if got, want := hex.EncodeToString(sum.Sum(nil)), "4e5a55c3077209f5cc1f9ee04930c296"; got != want {
		t.Fatalf("converted planes hash to %s, want %s", got, want)
	}
}
