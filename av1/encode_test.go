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
	for _, c := range []encConfig{
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

func TestEncodeDav1d(t *testing.T) {
	if dav1dBin == "" {
		t.Skip("dav1d not found (set DAV1D_BIN)")
	}

	res := make([]uint8, 256*256)
	for i := range res {
		res[i] = uint8((i*7 + i/256*3) & 0xff)
	}

	for _, c := range []encConfig{
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
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.obu")
		data := Encode(c)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}

		ref, err := exec.Command(dav1dBin, "-i", path, "-o", "-", "--muxer", "md5").Output()
		if err != nil {
			t.Fatalf("%dx%d: dav1d: %v", c.Width, c.Height, err)
		}

		var d Decoder
		pics, err := d.DecodeOBUs(data)
		if err != nil {
			t.Fatalf("%dx%d: gav1d: %v", c.Width, c.Height, err)
		}
		var buf bytes.Buffer
		for _, p := range pics {
			planeRows(p, &buf)
			p.Release()
		}
		sum := md5.Sum(buf.Bytes())
		got := hex.EncodeToString(sum[:])

		if !strings.Contains(string(ref), got) {
			t.Fatalf("%dx%d: md5 %s, dav1d says %s", c.Width, c.Height, got, strings.TrimSpace(string(ref)))
		}
		t.Logf("%dx%d q=%d: %d bytes, gav1d and dav1d agree (%s)",
			c.Width, c.Height, c.QIndex, len(data), got[:16])
	}
}

func encodeRecon(t *testing.T, c *encConfig) ([]byte, *encoder) {
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
		c := encConfig{Width: w, Height: h, BitDepth: 8, QIndex: q,
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
				c := encConfig{Width: w, Height: h, BitDepth: 8, QIndex: q,
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
