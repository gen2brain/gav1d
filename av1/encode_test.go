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

func encodeRecon(t *testing.T, c *encConfig) ([]byte, []uint8, int) {
	t.Helper()

	var e encoder
	e.init(c)
	seq := writeSeqHdr(c)

	var sh sequenceHeader
	var gb getBits
	gb.init(seq)
	if err := parseSeqHdr(&sh, &gb, false); err != nil {
		t.Fatal(err)
	}

	tile := e.writeTile()
	var fp putBits
	writeFrameHdr(&fp, &sh, c)
	fp.bytealign()

	out := writeObu(nil, obuTd, nil)
	out = writeObu(out, obuSeqHdr, seq)

	return writeObu(out, obuFrame, append(fp.bytes(), tile...)), e.recon, e.stride
}

// TestEncodeResidual holds that the encoder reconstructs exactly what the
// decoder will, which is what stops prediction drifting.
func TestEncodeResidual(t *testing.T) {
	const w, h = 128, 96
	src := make([]uint8, w*h)
	for y := range h {
		for x := range w {
			v := 128 + 100*math.Sin(float64(x)/25)*math.Cos(float64(y)/30)
			src[y*w+x] = uint8(max(0, min(255, int(v))))
		}
	}

	for _, q := range []int{20, 60, 120, 200} {
		c := encConfig{Width: w, Height: h, BitDepth: 8, QIndex: q,
			Monochrome: true, Src: src, SrcStride: w}
		data, recon, stride := encodeRecon(t, &c)

		var d Decoder
		pics, err := d.DecodeOBUs(data)
		if err != nil {
			t.Fatalf("q=%d: %v", q, err)
		}
		if len(pics) != 1 {
			t.Fatalf("q=%d: %d pictures", q, len(pics))
		}
		p := pics[0]
		for y := range h {
			for x := range w {
				if got, want := p.Data[0][y*p.Stride[0]+x], recon[y*stride+x]; got != want {
					t.Fatalf("q=%d [%d,%d]: decoded %d, encoder reconstructed %d",
						q, x, y, got, want)
				}
			}
		}
		p.Release()
	}
}
