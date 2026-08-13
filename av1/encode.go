package av1

type encoder struct {
	cfg     encConfig
	seq     sequenceHeader
	hdr     frameHeader
	cdf     cdfContext
	msac    msacEncoder
	a       []blockContext
	l       blockContext
	bw      int
	bh      int
	layout  int
	splitTo int
	stride  int
	recon   []uint8
	dq      [2]uint16
	dsp     *dspContext[uint8]
	itx     []int32
}

func (e *encoder) init(c *encConfig) {
	e.cfg = *c
	e.bw = (c.Width + 3) >> 2
	e.bh = (c.Height + 3) >> 2
	e.a = make([]blockContext, (e.bw+31)>>5+1)
	e.layout = pixelLayoutI420
	if c.Monochrome {
		e.layout = pixelLayoutI400
	}
	e.splitTo = bl64x64
	if c.Src != nil {
		e.splitTo = bl32x32
	}
	e.stride = e.bw * 4
	e.recon = make([]uint8, e.stride*e.bh*4)
	e.dq[0] = dqTbl[0][c.QIndex][0]
	e.dq[1] = dqTbl[0][c.QIndex][1]
	e.dsp = newDsp[uint8]()
	e.itx = make([]int32, 64*64*2+itxDct64Scratch)
	e.cdf.initStatic(0)
	e.msac.init(false)
}

func (e *encoder) clearCtx() {
	for i := range e.a {
		e.a[i] = blockContext{}
	}
	e.l = blockContext{}
}

func (e *encoder) writeBlock(bx, by, bs int) {
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	bx4, by4 := bx&31, by&31
	ab := &e.a[bx>>5]

	skip := uint32(1)
	if e.cfg.Src != nil {
		skip = 0
	}
	sctx := int(ab.skip[bx4]) + int(e.l.skip[by4])
	e.msac.boolAdapt(e.cdf.m.skip[sctx][:], skip)

	ymodeCdf := e.cdf.kfym[intraModeContext[ab.mode[bx4]]][intraModeContext[e.l.mode[by4]]][:]
	e.msac.symbolAdapt(ymodeCdf, nIntraPredModes-1, dcPred)

	if e.hasChroma(bx, by, bs) {
		cflAllowed := b2i(cflAllowedMask&(1<<bs) != 0)
		cdf := e.cdf.m.uvMode[cflAllowed][dcPred][:]
		e.msac.symbolAdapt(cdf, nUvIntraPredModes-1-(1-cflAllowed), dcPred)
	}

	for i, edge := range [2]*blockContext{ab, &e.l} {
		off, n := bx4, bw4
		if i == 1 {
			off, n = by4, bh4
		}
		setCtx(edge.mode[:], off, n, dcPred)
		setCtx(edge.intra[:], off, n, 1)
		setCtx(edge.skip[:], off, n, uint8(skip))
	}

	if skip != 0 {
		for i, edge := range [2]*blockContext{ab, &e.l} {
			off, n := bx4, bw4
			if i == 1 {
				off, n = by4, bh4
			}
			memset(edge.lcoef[off:off+n], 0x40)
		}

		return
	}

	e.reconBlock(bx, by, bs)
}

func (e *encoder) hasChroma(bx, by, bs int) bool {
	if e.cfg.Monochrome {
		return false
	}
	bDim := &blockDimensions[bs]

	return (int(bDim[0]) > 1 || bx&1 != 0) && (int(bDim[1]) > 1 || by&1 != 0)
}

func (e *encoder) writeSb(bx, by, bl int) {
	hsz := 16 >> bl
	haveH := e.bw > bx+hsz
	haveV := e.bh > by+hsz

	if !haveH && !haveV {
		e.writeSb(bx, by, bl+1)

		return
	}

	bx8, by8 := (bx&31)>>1, (by&31)>>1
	ab := &e.a[bx>>5]
	ctx := getPartitionCtx(ab, &e.l, bl, by8, bx8)
	pc := e.cdf.m.partition[bl][ctx][:]

	if haveH && haveV && bl < e.splitTo {
		e.msac.symbolAdapt(pc, int(partitionTypeCount[bl]), partitionSplit)
		h := hsz
		e.writeSb(bx, by, bl+1)
		e.writeSb(bx+h, by, bl+1)
		e.writeSb(bx, by+h, bl+1)
		e.writeSb(bx+h, by+h, bl+1)

		if bl == bl8x8 {
			n := 1 << ulog2(uint32(hsz))
			setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][partitionSplit])
			setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][partitionSplit])
		}

		return
	}

	bp := partitionNone
	switch {
	case haveH && haveV:
		e.msac.symbolAdapt(pc, int(partitionTypeCount[bl]), partitionNone)
	case haveH:
		bp = partitionH
		e.msac.boolF(0, gatherTopPartitionProb(pc, bl))
	default:
		bp = partitionV
		e.msac.boolF(0, gatherLeftPartitionProb(pc, bl))
	}

	e.writeBlock(bx, by, int(blockSizes[bl][bp][0]))

	n := 1 << ulog2(uint32(hsz))
	setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][bp])
	setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][bp])
}

func (e *encoder) writeTile() []byte {
	e.msac.init(false)
	e.clearCtx()

	sbStep := 16
	for by := 0; by < e.bh; by += sbStep {
		e.l = blockContext{}
		for bx := 0; bx < e.bw; bx += sbStep {
			e.writeSb(bx, by, 1)
		}
	}

	return e.msac.done()
}

func Encode(c encConfig) []byte {
	var e encoder
	e.init(&c)

	seq := writeSeqHdr(&c)
	var sh sequenceHeader
	var gb getBits
	gb.init(seq)
	if err := parseSeqHdr(&sh, &gb, false); err != nil {
		return nil
	}

	var p putBits
	writeFrameHdr(&p, &sh, &c)
	tile := e.writeTile()

	var fp putBits
	writeFrameHdr(&fp, &sh, &c)
	fp.bytealign()
	body := append(fp.bytes(), tile...)

	out := writeObu(nil, obuTd, nil)
	out = writeObu(out, obuSeqHdr, seq)

	return writeObu(out, obuFrame, body)
}
