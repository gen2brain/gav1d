package av1

// EncodeConfig describes one all-intra key frame for [Encode]. The source
// planes are 8-bit, with chroma at half resolution in both directions unless
// Monochrome is set. A nil Src encodes a mid-gray frame.
type EncodeConfig struct {
	// Width and Height are the frame size in luma samples, at least 1.
	Width  int
	Height int
	// BitDepth must be 8.
	BitDepth int
	// QIndex is the quantizer, 0 to 255, where 0 is lossless.
	QIndex int
	// Speed is the search effort, 0 to 10, where 0 searches the most and 10
	// the least. Values outside the range are clamped.
	Speed int
	// Threads bounds the goroutines a frame's tiles are coded over. Zero
	// means GOMAXPROCS, one runs serially. Output is identical either way.
	Threads int
	// Monochrome encodes luma only, leaving SrcU and SrcV unused.
	Monochrome bool
	// FullRange signals that the samples span 0 to 255 rather than the
	// studio swing of 16 to 235 for luma and 16 to 240 for chroma. It only
	// sets color_range in the sequence header; the samples are coded as given.
	FullRange bool
	// Src is the luma plane, SrcStride its row stride, which is at least Width.
	Src       []uint8
	SrcStride int
	// SrcU and SrcV are the chroma planes, each (Width+1)/2 by (Height+1)/2
	// samples, sharing the row stride SrcUVStride.
	SrcU        []uint8
	SrcV        []uint8
	SrcUVStride int
}

func writeObu(out []byte, typ int, payload []byte) []byte {
	var h putBits
	h.bit(0)
	h.bits32(uint32(typ), 4)
	h.bit(0)
	h.bit(1)
	h.bit(0)
	out = append(out, h.bytes()...)

	var sz putBits
	sz.uleb128(uint32(len(payload)))
	out = append(out, sz.bytes()...)

	return append(out, payload...)
}

func trailingBits(p *putBits) {
	p.bit(1)
	p.bytealign()
}

func writeSeqHdr(c *EncodeConfig) []byte {
	var p putBits

	p.bits32(0, 3)
	p.bit(1)
	p.bit(1)
	p.bits32(0, 3)
	p.bits32(0, 2)

	wn, hn := bitsFor(c.Width-1), bitsFor(c.Height-1)
	p.bits32(uint32(wn-1), 4)
	p.bits32(uint32(hn-1), 4)
	p.bits32(uint32(c.Width-1), wn)
	p.bits32(uint32(c.Height-1), hn)

	p.bit(0)
	p.bit(0)
	p.bit(1)

	p.bit(0)
	p.bit(0)
	p.bit(0)

	p.bit(b2u(c.BitDepth > 8))
	p.bit(b2u(c.Monochrome))
	p.bit(0)
	p.bit(b2u(c.FullRange))
	if !c.Monochrome {
		p.bits32(0, 2)
		p.bit(0)
	}
	p.bit(0)

	trailingBits(&p)

	return p.bytes()
}

func bitsFor(v int) int {
	n := 1
	for v>>uint(n) != 0 {
		n++
	}

	return n
}

type encTiling struct {
	sbw, sbh   int
	log2Cols   int
	log2Rows   int
	moreCols   bool
	moreRows   bool
	colStartSb []int
	rowStartSb []int
}

func newEncTiling(seq *sequenceHeader, w, h int) *encTiling {
	sbszMin1 := (64 << seq.sb128) - 1
	sbszLog2 := 6 + int(seq.sb128)
	t := &encTiling{
		sbw: (w + sbszMin1) >> sbszLog2,
		sbh: (h + sbszMin1) >> sbszLog2,
	}
	maxCols := tileLog2(1, min(t.sbw, maxTileCols))
	maxRows := tileLog2(1, min(t.sbh, maxTileRows))
	t.log2Cols = tileLog2(4096>>sbszLog2, t.sbw)
	minTiles := max(tileLog2(4096*2304>>(2*sbszLog2), t.sbw*t.sbh), t.log2Cols)
	t.log2Rows = max(minTiles-t.log2Cols, 0)
	t.moreCols = t.log2Cols < maxCols
	t.moreRows = t.log2Rows < maxRows

	tileW := 1 + (t.sbw-1)>>t.log2Cols
	for sbx := 0; sbx < t.sbw; sbx += tileW {
		t.colStartSb = append(t.colStartSb, sbx)
	}
	t.colStartSb = append(t.colStartSb, t.sbw)
	tileH := 1 + (t.sbh-1)>>t.log2Rows
	for sby := 0; sby < t.sbh; sby += tileH {
		t.rowStartSb = append(t.rowStartSb, sby)
	}
	t.rowStartSb = append(t.rowStartSb, t.sbh)

	return t
}

func (t *encTiling) cols() int {
	return len(t.colStartSb) - 1
}

func (t *encTiling) rows() int {
	return len(t.rowStartSb) - 1
}

func (t *encTiling) tile(i, bw, bh int) tileTiling {
	row, col := i/t.cols(), i%t.cols()

	return tileTiling{
		colStart: t.colStartSb[col] << 4,
		colEnd:   min(t.colStartSb[col+1]<<4, bw),
		rowStart: t.rowStartSb[row] << 4,
		rowEnd:   min(t.rowStartSb[row+1]<<4, bh),
		col:      col,
		row:      row,
	}
}

func writeFrameHdr(p *putBits, c *EncodeConfig, t *encTiling, tileSizeBytes int) {
	p.bit(0)
	p.bit(0)

	p.bit(0)

	p.bit(1)
	if t.moreCols {
		p.bit(0)
	}
	if t.moreRows {
		p.bit(0)
	}
	if t.log2Cols != 0 || t.log2Rows != 0 {
		p.bits32(0, t.log2Cols+t.log2Rows)
		p.bits32(uint32(tileSizeBytes-1), 2)
	}

	p.bits32(uint32(c.QIndex), 8)
	p.bit(0)
	if !c.Monochrome {
		p.bit(0)
		p.bit(0)
	}
	p.bit(0)

	p.bit(0)

	if c.QIndex != 0 {
		p.bit(0)

		p.bits32(0, 6)
		p.bits32(0, 6)
		p.bits32(0, 3)
		p.bit(0)

		p.bit(0)
	}

	p.bit(0)
}
