package av1

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
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

	for _, c := range []encConfig{
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 100},
		{Width: 64, Height: 96, BitDepth: 8, QIndex: 60},
		{Width: 96, Height: 64, BitDepth: 8, QIndex: 20},
		{Width: 48, Height: 48, BitDepth: 8, QIndex: 40},
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
