package av1

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	for _, c := range []EncodeConfig{
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 100},
		{Width: 64, Height: 96, BitDepth: 8, QIndex: 60},
	} {
		data := Encode(c)
		if data == nil {
			t.Fatalf("%dx%d: encode returned nil", c.Width, c.Height)
		}

		var d Decoder
		pics, err := d.DecodeOBUs(data)
		if err != nil {
			t.Fatalf("%dx%d: decode: %v (%d bytes)", c.Width, c.Height, err, len(data))
		}
		if len(pics) != 1 {
			t.Fatalf("%dx%d: %d pictures", c.Width, c.Height, len(pics))
		}
		p := pics[0]
		if p.Width != c.Width || p.Height != c.Height {
			t.Errorf("got %dx%d, want %dx%d", p.Width, p.Height, c.Width, c.Height)
		}
		t.Logf("%dx%d q=%d: %d bytes, decoded %dx%d", c.Width, c.Height, c.QIndex,
			len(data), p.Width, p.Height)
		p.Release()
	}
}

func TestEncodeConfig(t *testing.T) {
	src := make([]uint8, 64*64)
	ok := EncodeConfig{Width: 64, Height: 64, BitDepth: 8, QIndex: 100,
		Monochrome: true, Src: src, SrcStride: 64}
	if Encode(ok) == nil {
		t.Fatal("valid config rejected")
	}

	for name, c := range map[string]EncodeConfig{
		"qindex past the dequant table": {Width: 64, Height: 64, BitDepth: 8,
			QIndex: 256, Monochrome: true, Src: src, SrcStride: 64},
		"zero width": {Width: 0, Height: 64, BitDepth: 8, QIndex: 100},
		"ten bit":    {Width: 64, Height: 64, BitDepth: 10, QIndex: 100},
		"stride below width": {Width: 64, Height: 64, BitDepth: 8, QIndex: 100,
			Monochrome: true, Src: src, SrcStride: 32},
		"colour without chroma planes": {Width: 64, Height: 64, BitDepth: 8,
			QIndex: 100, Src: src, SrcStride: 64},
	} {
		if Encode(c) != nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestEncodeDav1d(t *testing.T) {
	if dav1dBin == "" {
		t.Skip("dav1d not found (set DAV1D_BIN)")
	}

	res := make([]uint8, 256*256)
	for i := range res {
		res[i] = uint8((i*7 + i/256*3) & 0xff)
	}

	for _, c := range []EncodeConfig{
		{Width: 128, Height: 96, BitDepth: 8, QIndex: 10, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 128, Height: 96, BitDepth: 8, QIndex: 40, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 192, Height: 128, BitDepth: 8, QIndex: 100, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 256, Height: 256, BitDepth: 8, QIndex: 180, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 255, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 128, Height: 128, BitDepth: 8, QIndex: 30,
			Src: res, SrcStride: 256, SrcU: res[1024:], SrcV: res[2048:], SrcUVStride: 128},
		{Width: 96, Height: 64, BitDepth: 8, QIndex: 150,
			Src: res, SrcStride: 256, SrcU: res[512:], SrcV: res[1536:], SrcUVStride: 128},
		{Width: 90, Height: 54, BitDepth: 8, QIndex: 60, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 200, Height: 150, BitDepth: 8, QIndex: 120, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 48, Height: 80, BitDepth: 8, QIndex: 30, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 130, Height: 70, BitDepth: 8, QIndex: 80,
			Src: res, SrcStride: 256, SrcU: res[1024:], SrcV: res[2048:], SrcUVStride: 128},
		{Width: 122, Height: 66, BitDepth: 8, QIndex: 40,
			Src: res, SrcStride: 256, SrcU: res[768:], SrcV: res[2560:], SrcUVStride: 128},
		{Width: 97, Height: 33, BitDepth: 8, QIndex: 80, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 0, Monochrome: true,
			Src: res, SrcStride: 256},
		{Width: 96, Height: 64, BitDepth: 8, QIndex: 0,
			Src: res, SrcStride: 256, SrcU: res[512:], SrcV: res[1536:], SrcUVStride: 128},
		{Width: 33, Height: 33, BitDepth: 8, QIndex: 80,
			Src: res, SrcStride: 256, SrcU: res[1024:], SrcV: res[2048:], SrcUVStride: 128},
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 100},
		{Width: 64, Height: 96, BitDepth: 8, QIndex: 60},
		{Width: 96, Height: 64, BitDepth: 8, QIndex: 20},
		{Width: 48, Height: 48, BitDepth: 8, QIndex: 40},
		{Width: 128, Height: 64, BitDepth: 8, QIndex: 60},
		{Width: 64, Height: 128, BitDepth: 8, QIndex: 60},
		{Width: 128, Height: 128, BitDepth: 8, QIndex: 30},
		{Width: 320, Height: 240, BitDepth: 8, QIndex: 100},
		{Width: 640, Height: 480, BitDepth: 8, QIndex: 150},
		{Width: 128, Height: 96, BitDepth: 8, QIndex: 40, FullRange: true,
			Monochrome: true, Src: res, SrcStride: 256},
		{Width: 128, Height: 128, BitDepth: 8, QIndex: 30, FullRange: true,
			Src: res, SrcStride: 256, SrcU: res[1024:], SrcV: res[2048:], SrcUVStride: 128},
	} {
		for speed := range len(speedTable) {
			c.Speed = speed
			encodeDav1d(t, c)
		}
	}
}

func encodeDav1d(t *testing.T, c EncodeConfig) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.obu")
	data := Encode(c)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ref, err := exec.Command(dav1dBin, "-i", path, "-o", "-", "--muxer", "md5").Output()
	if err != nil {
		t.Fatalf("%dx%d s=%d: dav1d: %v", c.Width, c.Height, c.Speed, err)
	}

	var d Decoder
	pics, err := d.DecodeOBUs(data)
	if err != nil {
		t.Fatalf("%dx%d s=%d: gav1d: %v", c.Width, c.Height, c.Speed, err)
	}
	var buf bytes.Buffer
	for _, p := range pics {
		planeRows(p, &buf)
		p.Release()
	}
	sum := md5.Sum(buf.Bytes())
	got := hex.EncodeToString(sum[:])

	if !strings.Contains(string(ref), got) {
		t.Fatalf("%dx%d q=%d s=%d: md5 %s, dav1d says %s", c.Width, c.Height,
			c.QIndex, c.Speed, got, strings.TrimSpace(string(ref)))
	}
}

// TestEncodeLossless holds that qindex 0 reproduces the source exactly.
func TestEncodeLossless(t *testing.T) {
	const stride = 96
	src := make([]uint8, stride*stride)
	for i := range src {
		src[i] = uint8((i*7 + i/stride*3) & 0xff)
	}

	for _, sz := range [][2]int{{64, 64}, {96, 48}, {33, 17}} {
		w, h := sz[0], sz[1]
		for _, mono := range []bool{true, false} {
			c := EncodeConfig{Width: w, Height: h, BitDepth: 8, QIndex: 0,
				Monochrome: mono, Src: src, SrcStride: stride}
			if !mono {
				c.SrcU, c.SrcV, c.SrcUVStride = src[stride:], src[3*stride:], stride
			}
			data := Encode(c)
			if data == nil {
				t.Fatalf("%dx%d mono=%v: encode returned nil", w, h, mono)
			}

			var d Decoder
			pics, err := d.DecodeOBUs(data)
			if err != nil {
				t.Fatalf("%dx%d mono=%v: %v", w, h, mono, err)
			}
			p := pics[0]
			for y := range h {
				for x := range w {
					if got, want := p.Data[0][y*p.Stride[0]+x], src[y*stride+x]; got != want {
						t.Fatalf("%dx%d mono=%v [%d,%d]: got %d, want %d",
							w, h, mono, x, y, got, want)
					}
				}
			}
			if !mono {
				for pl := range 2 {
					s := c.SrcU
					if pl == 1 {
						s = c.SrcV
					}
					for y := range (h + 1) / 2 {
						for x := range (w + 1) / 2 {
							got := p.Data[1+pl][y*p.Stride[1]+x]
							if want := s[y*stride+x]; got != want {
								t.Fatalf("%dx%d plane %d [%d,%d]: got %d, want %d",
									w, h, 1+pl, x, y, got, want)
							}
						}
					}
				}
			}
			t.Logf("%dx%d mono=%v: %d bytes, exact", w, h, mono, len(data))
			p.Release()
		}
	}
}

func encodeRecon(t *testing.T, c *EncodeConfig) ([]byte, *encoder) {
	t.Helper()

	seq := writeSeqHdr(c)

	var sh sequenceHeader
	var gb getBits
	gb.init(seq)
	if err := parseSeqHdr(&sh, &gb, false); err != nil {
		t.Fatal(err)
	}

	var e encoder
	e.init(c, &sh)
	tile := e.writeTile()
	var fp putBits
	writeFrameHdr(&fp, &sh, c)
	fp.bytealign()

	out := writeObu(nil, obuTd, nil)
	out = writeObu(out, obuSeqHdr, seq)

	return writeObu(out, obuFrame, append(fp.bytes(), tile...)), &e
}

// TestEncodeChroma checks that a 4:2:0 frame carries all three planes.
func TestEncodeChroma(t *testing.T) {
	const w, h = 128, 128
	cw, ch := w/2, h/2
	y := make([]uint8, w*h)
	u := make([]uint8, cw*ch)
	v := make([]uint8, cw*ch)
	for j := range h {
		for i := range w {
			y[j*w+i] = uint8(max(0, min(255, int(128+90*math.Sin(float64(i)/18)*math.Cos(float64(j)/23)))))
		}
	}
	for j := range ch {
		for i := range cw {
			u[j*cw+i] = uint8(max(0, min(255, 128+int(60*math.Sin(float64(i)/9)))))
			v[j*cw+i] = uint8(max(0, min(255, 128+int(60*math.Cos(float64(j)/11)))))
		}
	}

	var prev int
	for _, q := range []int{20, 100, 200} {
		c := EncodeConfig{Width: w, Height: h, BitDepth: 8, QIndex: q,
			Src: y, SrcStride: w, SrcU: u, SrcV: v, SrcUVStride: cw}
		data, _ := encodeRecon(t, &c)

		var d Decoder
		pics, err := d.DecodeOBUs(data)
		if err != nil {
			t.Fatalf("q=%d: %v", q, err)
		}
		p := pics[0]
		if p.Layout != LayoutI420 {
			t.Fatalf("layout %v, want 4:2:0", p.Layout)
		}
		var su int
		for j := range ch {
			for i := range cw {
				d := int(p.Data[1][j*p.Stride[1]+i]) - int(u[j*cw+i])
				if d < 0 {
					d = -d
				}
				su += d
			}
		}
		if su/(cw*ch) > 8 {
			t.Errorf("q=%d: chroma mean error %d too high", q, su/(cw*ch))
		}
		if prev != 0 && len(data) >= prev {
			t.Errorf("q=%d: %d bytes, not smaller than %d", q, len(data), prev)
		}
		prev = len(data)
		p.Release()
	}
}

// TestEncodeResidual holds that the encoder reconstructs exactly what the
// decoder will, which is what stops prediction drifting.
func TestEncodeResidual(t *testing.T) {
	const stride = 256
	src := make([]uint8, stride*stride)
	for y := range stride {
		for x := range stride {
			v := 128 + 100*math.Sin(float64(x)/25)*math.Cos(float64(y)/30)
			src[y*stride+x] = uint8(max(0, min(255, int(v))))
		}
	}

	for _, sz := range [][2]int{{128, 96}, {97, 33}, {122, 66}, {200, 150}} {
		w, h := sz[0], sz[1]
		for _, mono := range []bool{true, false} {
			for _, q := range []int{20, 60, 120, 200} {
				c := EncodeConfig{Width: w, Height: h, BitDepth: 8, QIndex: q,
					Monochrome: mono, Src: src, SrcStride: stride}
				if !mono {
					c.SrcU, c.SrcV, c.SrcUVStride = src[stride:], src[3*stride:], stride
				}
				data, e := encodeRecon(t, &c)

				var d Decoder
				pics, err := d.DecodeOBUs(data)
				if err != nil {
					t.Fatalf("%dx%d q=%d: %v", w, h, q, err)
				}
				if len(pics) != 1 {
					t.Fatalf("%dx%d q=%d: %d pictures", w, h, q, len(pics))
				}
				p := pics[0]
				for y := range h {
					for x := range w {
						if got, want := p.Data[0][y*p.Stride[0]+x], e.recon[y*e.stride+x]; got != want {
							t.Fatalf("%dx%d q=%d luma [%d,%d]: decoded %d, encoder reconstructed %d",
								w, h, q, x, y, got, want)
						}
					}
				}
				for pl := range 2 {
					if mono {
						break
					}
					for y := range e.ch {
						for x := range e.cw {
							got := p.Data[1+pl][y*p.Stride[1]+x]
							want := e.reconC[pl][y*e.cstride+x]
							if got != want {
								t.Fatalf("%dx%d q=%d plane %d [%d,%d]: decoded %d, encoder reconstructed %d",
									w, h, q, 1+pl, x, y, got, want)
							}
						}
					}
				}
				p.Release()
			}
		}
	}
}

func TestEncodeSpeed(t *testing.T) {
	const w, h = 192, 160
	cw, ch := w/2, h/2
	y := make([]uint8, w*h)
	u := make([]uint8, cw*ch)
	v := make([]uint8, cw*ch)
	for j := range h {
		for i := range w {
			y[j*w+i] = uint8(max(0, min(255, int(128+
				90*math.Sin(float64(i)/13)*math.Cos(float64(j)/17)+
				float64((i/4^j/4)&15)))))
		}
	}
	for j := range ch {
		for i := range cw {
			u[j*cw+i] = uint8(max(0, min(255, 128+int(50*math.Sin(float64(i)/7)))))
			v[j*cw+i] = uint8(max(0, min(255, 128+int(50*math.Cos(float64(j)/9)))))
		}
	}

	cfg := func(speed int) EncodeConfig {
		return EncodeConfig{Width: w, Height: h, BitDepth: 8, QIndex: 120, Speed: speed,
			Src: y, SrcStride: w, SrcU: u, SrcV: v, SrcUVStride: cw}
	}

	var slowest, fastest []byte
	for speed := range len(speedTable) {
		c := cfg(speed)
		data, e := encodeRecon(t, &c)

		var d Decoder
		pics, err := d.DecodeOBUs(data)
		if err != nil {
			t.Fatalf("speed %d: %v", speed, err)
		}
		p := pics[0]
		if p.Width != w || p.Height != h {
			t.Fatalf("speed %d: got %dx%d", speed, p.Width, p.Height)
		}

		for j := range h {
			for i := range w {
				if got, want := p.Data[0][j*p.Stride[0]+i], e.recon[j*e.stride+i]; got != want {
					t.Fatalf("speed %d luma [%d,%d]: decoded %d, encoder reconstructed %d",
						speed, i, j, got, want)
				}
			}
		}
		for pl := range 2 {
			for j := range ch {
				for i := range cw {
					got := p.Data[1+pl][j*p.Stride[1]+i]
					if want := e.reconC[pl][j*e.cstride+i]; got != want {
						t.Fatalf("speed %d plane %d [%d,%d]: decoded %d, encoder reconstructed %d",
							speed, 1+pl, i, j, got, want)
					}
				}
			}
		}
		p.Release()

		switch speed {
		case 0:
			slowest = data
		case len(speedTable) - 1:
			fastest = data
		}
		t.Logf("speed %2d: %d bytes", speed, len(data))
	}

	if len(fastest) <= len(slowest) {
		t.Errorf("speed %d coded %d bytes, not more than speed 0's %d",
			len(speedTable)-1, len(fastest), len(slowest))
	}

	for _, c := range []struct {
		speed int
		want  []byte
	}{
		{-1, slowest}, {-1000, slowest},
		{len(speedTable), fastest}, {1000, fastest},
	} {
		cc := cfg(c.speed)
		if got := Encode(cc); !bytes.Equal(got, c.want) {
			t.Errorf("speed %d: %d bytes, want the clamped %d", c.speed, len(got), len(c.want))
		}
	}
}
