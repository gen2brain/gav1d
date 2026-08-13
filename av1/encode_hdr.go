package av1

type encConfig struct {
	Width       int
	Height      int
	BitDepth    int
	QIndex      int
	Monochrome  bool
	Src         []uint8
	SrcStride   int
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

func writeSeqHdr(c *encConfig) []byte {
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
	p.bit(0)
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

func tileLog2Bits(seq *sequenceHeader, w, h int) (int, int) {
	sbszMin1 := (64 << seq.sb128) - 1
	sbszLog2 := 6 + int(seq.sb128)
	sbw := (w + sbszMin1) >> sbszLog2
	sbh := (h + sbszMin1) >> sbszLog2
	minCols := tileLog2(4096>>sbszLog2, sbw)
	maxCols := tileLog2(1, min(sbw, maxTileCols))
	maxRows := tileLog2(1, min(sbh, maxTileRows))
	minTiles := max(tileLog2(4096*2304>>(2*sbszLog2), sbw*sbh), minCols)
	minRows := max(minTiles-minCols, 0)

	nCols := 0
	if minCols < maxCols {
		nCols = 1
	}
	nRows := 0
	if minRows < maxRows {
		nRows = 1
	}

	return nCols, nRows
}

func writeFrameHdr(p *putBits, seq *sequenceHeader, c *encConfig) {
	p.bit(0)
	p.bit(0)

	p.bit(0)

	nCols, nRows := tileLog2Bits(seq, c.Width, c.Height)
	p.bit(1)
	for range nCols {
		p.bit(0)
	}
	for range nRows {
		p.bit(0)
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
	}

	p.bits32(0, 6)
	p.bits32(0, 6)
	p.bits32(0, 3)
	p.bit(0)

	p.bit(0)

	p.bit(0)
}
