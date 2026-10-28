package av1

import "testing"

func TestSeqHdrRoundtrip(t *testing.T) {
	for _, c := range []EncodeConfig{
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
	for _, tc := range []struct {
		c          EncodeConfig
		cols, rows int
	}{
		{EncodeConfig{Width: 64, Height: 64, BitDepth: 8, QIndex: 100}, 1, 1},
		{EncodeConfig{Width: 320, Height: 240, BitDepth: 8, QIndex: 40}, 1, 1},
		{EncodeConfig{Width: 1920, Height: 1080, BitDepth: 10, QIndex: 180}, 1, 1},
		{EncodeConfig{Width: 48, Height: 48, BitDepth: 8, QIndex: 60, Monochrome: true}, 1, 1},
		{EncodeConfig{Width: 4096, Height: 2304, BitDepth: 8, QIndex: 60}, 1, 1},
		{EncodeConfig{Width: 4200, Height: 2200, BitDepth: 8, QIndex: 60}, 2, 1},
		{EncodeConfig{Width: 3648, Height: 2736, BitDepth: 8, QIndex: 60}, 1, 2},
		{EncodeConfig{Width: 8192, Height: 64, BitDepth: 8, QIndex: 0}, 2, 1},
		{EncodeConfig{Width: 8192, Height: 8192, BitDepth: 8, QIndex: 60}, 2, 4},
	} {
		c := tc.c
		seq := &sequenceHeader{}
		var gb getBits
		gb.init(writeSeqHdr(&c))
		if err := parseSeqHdr(seq, &gb, false); err != nil {
			t.Fatalf("seq: %v", err)
		}

		tl := newEncTiling(seq, c.Width, c.Height)
		var p putBits
		writeFrameHdr(&p, &c, tl, 3)

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
		if int(h.tiling.cols) != tc.cols || int(h.tiling.rows) != tc.rows {
			t.Errorf("%dx%d: tiling %dx%d, want %dx%d", c.Width, c.Height,
				h.tiling.cols, h.tiling.rows, tc.cols, tc.rows)
		}
		if tl.cols() != tc.cols || tl.rows() != tc.rows {
			t.Errorf("%dx%d: encoder tiling %dx%d, want %dx%d", c.Width, c.Height,
				tl.cols(), tl.rows(), tc.cols, tc.rows)
		}
		for i := range tc.cols {
			if int(h.tiling.colStartSb[i]) != tl.colStartSb[i] {
				t.Errorf("%dx%d: column %d starts at %d, want %d", c.Width, c.Height,
					i, tl.colStartSb[i], h.tiling.colStartSb[i])
			}
		}
		for i := range tc.rows {
			if int(h.tiling.rowStartSb[i]) != tl.rowStartSb[i] {
				t.Errorf("%dx%d: row %d starts at %d, want %d", c.Width, c.Height,
					i, tl.rowStartSb[i], h.tiling.rowStartSb[i])
			}
		}
		if tc.cols*tc.rows > 1 && (h.tiling.nBytes != 3 || h.tiling.update != 0) {
			t.Errorf("%dx%d: tile size bytes %d, update %d", c.Width, c.Height,
				h.tiling.nBytes, h.tiling.update)
		}
		if h.segmentation.enabled != 0 || h.quant.qm != 0 || h.allowIntrabc != 0 {
			t.Errorf("seg %d qm %d intrabc %d", h.segmentation.enabled, h.quant.qm, h.allowIntrabc)
		}
		if fb.remaining() < 0 {
			t.Errorf("overran the header by %d bits", -fb.remaining())
		}
	}
}
