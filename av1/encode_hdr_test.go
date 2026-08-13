package av1

import "testing"

func TestSeqHdrRoundtrip(t *testing.T) {
	for _, c := range []encConfig{
		{Width: 64, Height: 64, BitDepth: 8},
		{Width: 1920, Height: 1080, BitDepth: 10},
		{Width: 33, Height: 17, BitDepth: 8, Monochrome: true},
	} {
		h := &sequenceHeader{}

		var gb getBits
		gb.init(writeSeqHdr(&c))
		if err := parseSeqHdr(h, &gb, false); err != nil {
			t.Fatalf("%dx%d bd=%d: %v", c.Width, c.Height, c.BitDepth, err)
		}
		if h.maxWidth != c.Width || h.maxHeight != c.Height {
			t.Errorf("size %dx%d, want %dx%d", h.maxWidth, h.maxHeight, c.Width, c.Height)
		}
		wantHbd := uint8(0)
		if c.BitDepth > 8 {
			wantHbd = 1
		}
		if h.hbd != wantHbd {
			t.Errorf("hbd %d, want %d", h.hbd, wantHbd)
		}
		if h.monochrome != b2u8(c.Monochrome) {
			t.Errorf("monochrome %d", h.monochrome)
		}
		if h.reducedStillPictureHeader != 1 || h.stillPicture != 1 {
			t.Errorf("still %d reduced %d", h.stillPicture, h.reducedStillPictureHeader)
		}
	}
}

func TestFrameHdrRoundtrip(t *testing.T) {
	for _, c := range []encConfig{
		{Width: 64, Height: 64, BitDepth: 8, QIndex: 100},
		{Width: 320, Height: 240, BitDepth: 8, QIndex: 40},
		{Width: 1920, Height: 1080, BitDepth: 10, QIndex: 180},
		{Width: 48, Height: 48, BitDepth: 8, QIndex: 60, Monochrome: true},
	} {
		seq := &sequenceHeader{}
		var gb getBits
		gb.init(writeSeqHdr(&c))
		if err := parseSeqHdr(seq, &gb, false); err != nil {
			t.Fatalf("seq: %v", err)
		}

		var p putBits
		writeFrameHdr(&p, seq, &c)

		var d Decoder
		d.seqHdr = seq
		d.frameHdr = &frameHeader{}
		var fb getBits
		fb.init(p.bytes())
		if err := d.parseFrameHdr(&fb); err != nil {
			t.Fatalf("%dx%d q=%d: %v", c.Width, c.Height, c.QIndex, err)
		}

		h := d.frameHdr
		if h.frameType != frameTypeKey || h.showFrame != 1 {
			t.Errorf("type %d show %d", h.frameType, h.showFrame)
		}
		if h.width[1] != c.Width || h.height != c.Height {
			t.Errorf("size %dx%d, want %dx%d", h.width[1], h.height, c.Width, c.Height)
		}
		if int(h.quant.yac) != c.QIndex {
			t.Errorf("yac %d, want %d", h.quant.yac, c.QIndex)
		}
		if h.tiling.cols != 1 || h.tiling.rows != 1 {
			t.Errorf("tiling %dx%d, want 1x1", h.tiling.cols, h.tiling.rows)
		}
		if h.segmentation.enabled != 0 || h.quant.qm != 0 || h.allowIntrabc != 0 {
			t.Errorf("seg %d qm %d intrabc %d", h.segmentation.enabled, h.quant.qm, h.allowIntrabc)
		}
		if fb.remaining() < 0 {
			t.Errorf("overran the header by %d bits", -fb.remaining())
		}
	}
}
