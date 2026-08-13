package av1

type encoder struct {
	cfg     encConfig
	seq     sequenceHeader
	cdf     cdfContext
	msac    msacEncoder
	a       []blockContext
	l       blockContext
	bw      int
	bh      int
	layout  int
	ssHor   int
	ssVer   int
	splitTo int
	stride  int
	recon   []uint8
	reconC  [2][]uint8
	cstride int
	cw      int
	ch      int
	dq      [2]uint16
	lambda  float64
	dsp     *dspContext[uint8]
	itx     []int32
	fwd     []int32
	res     [64 * 64]int32
	coeff   [32 * 32]int32
	lv      [32 * 32]int32
	cf      [32 * 32]int16
	edge    [257]uint8
	levels  [64 * 66]uint8
}

func (e *encoder) init(c *encConfig, seq *sequenceHeader) {
	e.cfg = *c
	e.seq = *seq
	e.bw = ((c.Width + 7) >> 3) << 1
	e.bh = ((c.Height + 7) >> 3) << 1
	e.a = make([]blockContext, (e.bw+31)>>5+1)
	e.layout = pixelLayoutI420
	if c.Monochrome {
		e.layout = pixelLayoutI400
	}
	e.ssHor = b2i(e.layout != pixelLayoutI444)
	e.ssVer = b2i(e.layout == pixelLayoutI420)
	e.splitTo = bl64x64
	if c.Src != nil {
		e.splitTo = bl32x32
	}
	e.stride = (e.bw*4 + 63) &^ 63
	e.recon = make([]uint8, e.stride*((e.bh*4+63)&^63))
	if !c.Monochrome {
		e.cw = (c.Width + 1) >> 1
		e.ch = (c.Height + 1) >> 1
		e.cstride = (((e.bw+1)>>1)*4 + 63) &^ 63
		rows := (((e.bh+1)>>1)*4 + 63) &^ 63
		for pl := range 2 {
			e.reconC[pl] = make([]uint8, e.cstride*rows)
		}
	}
	e.dq[0] = dqTbl[0][c.QIndex][0]
	e.dq[1] = dqTbl[0][c.QIndex][1]
	e.lambda = float64(e.dq[1]) * float64(e.dq[1]) / 1024
	e.dsp = newDsp[uint8]()
	e.itx = make([]int32, 64*64*2+itxDct64Scratch)
	e.fwd = make([]int32, fwdScratch)
	e.cdf.initStatic(uint8(c.QIndex))
	e.msac.precarry = make([]uint16, 0, e.bw*e.bh*4)
	e.msac.init(false)
}

func (e *encoder) clearCtx() {
	for i := range e.a {
		resetContext(&e.a[i], true)
	}
	resetContext(&e.l, true)
}

func (e *encoder) writeBlock(bx, by, bs, edgeFlags int) {
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	bx4, by4 := bx&31, by&31
	ab := &e.a[bx>>5]
	hasChroma := e.hasChroma(bx, by, bs)

	skip := uint32(1)
	if e.cfg.Src != nil {
		skip = 0
	}
	sctx := int(ab.skip[bx4]) + int(e.l.skip[by4])
	e.msac.boolAdapt(e.cdf.m.skip[sctx][:], skip)

	ymodeCdf := e.cdf.kfym[intraModeContext[ab.mode[bx4]]][intraModeContext[e.l.mode[by4]]][:]
	mode := dcPred
	if skip == 0 {
		mode = e.pickMode(bx, by, bs, edgeFlags, ymodeCdf)
	}
	e.msac.symbolAdapt(ymodeCdf, nIntraPredModes-1, uint32(mode))
	if int(bDim[2])+int(bDim[3]) >= 2 && mode >= vertPred && mode <= vertLeftPred {
		e.msac.symbolAdapt(e.cdf.m.angleDelta[mode-vertPred][:], 6, 3)
	}

	uvMode := dcPred
	if hasChroma {
		cflAllowed := b2i(cflAllowedMask&(1<<bs) != 0)
		nSym := nUvIntraPredModes - 1 - (1 - cflAllowed)
		cdf := e.cdf.m.uvMode[cflAllowed][mode][:]
		if skip == 0 {
			uvMode = e.pickUvMode(bx, by, bs, edgeFlags, cdf)
		}
		e.msac.symbolAdapt(cdf, nSym, uint32(uvMode))
		if int(bDim[2])+int(bDim[3]) >= 2 && uvMode >= vertPred && uvMode <= vertLeftPred {
			e.msac.symbolAdapt(e.cdf.m.angleDelta[uvMode-vertPred][:], 6, 3)
		}
	}

	if skip == 0 {
		e.reconBlock(bx, by, bs, edgeFlags, mode)
		if hasChroma {
			e.reconChroma(bx, by, bs, edgeFlags, uvMode)
		}
	}

	for i, edge := range [2]*blockContext{ab, &e.l} {
		off, n := bx4, bw4
		if i == 1 {
			off, n = by4, bh4
		}
		setCtx(edge.mode[:], off, n, uint8(mode))
		setCtx(edge.intra[:], off, n, 1)
		setCtx(edge.skip[:], off, n, uint8(skip))
		if skip != 0 {
			memset(edge.lcoef[off:off+n], 0x40)
		}
	}

	if hasChroma {
		cbw4 := (bw4 + e.ssHor) >> e.ssHor
		cbh4 := (bh4 + e.ssVer) >> e.ssVer
		setCtx(ab.uvmode[:], bx4>>e.ssHor, cbw4, uint8(uvMode))
		setCtx(e.l.uvmode[:], by4>>e.ssVer, cbh4, uint8(uvMode))
		if skip != 0 {
			for pl := range 2 {
				memset(ab.ccoef[pl][bx4>>e.ssHor:][:cbw4], 0x40)
				memset(e.l.ccoef[pl][by4>>e.ssVer:][:cbh4], 0x40)
			}
		}
	}
}

func (e *encoder) hasChroma(bx, by, bs int) bool {
	if e.cfg.Monochrome {
		return false
	}
	bDim := &blockDimensions[bs]

	return (int(bDim[0]) > e.ssHor || bx&1 != 0) && (int(bDim[1]) > e.ssVer || by&1 != 0)
}

func (e *encoder) writeSb(bx, by, bl int, node *edgeNode) {
	hsz := 16 >> bl
	haveH := e.bw > bx+hsz
	haveV := e.bh > by+hsz

	if !haveH && !haveV {
		e.writeSb(bx, by, bl+1, node.kids[0])

		return
	}

	bx8, by8 := (bx&31)>>1, (by&31)>>1
	ab := &e.a[bx>>5]
	ctx := getPartitionCtx(ab, &e.l, bl, by8, bx8)
	pc := e.cdf.m.partition[bl][ctx][:]
	split := bl < e.splitTo
	n := 1 << ulog2(uint32(hsz))

	if split {
		switch {
		case haveH && haveV:
			e.msac.symbolAdapt(pc, int(partitionTypeCount[bl]), partitionSplit)
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx+hsz, by, bl+1, node.kids[1])
			e.writeSb(bx, by+hsz, bl+1, node.kids[2])
			e.writeSb(bx+hsz, by+hsz, bl+1, node.kids[3])
		case haveH:
			e.msac.boolF(1, gatherTopPartitionProb(pc, bl))
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx+hsz, by, bl+1, node.kids[1])
		default:
			e.msac.boolF(1, gatherLeftPartitionProb(pc, bl))
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx, by+hsz, bl+1, node.kids[2])
		}

		if bl == bl8x8 {
			setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][partitionSplit])
			setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][partitionSplit])
		}

		return
	}

	bp := partitionNone
	edgeFlags := int(node.o)
	switch {
	case haveH && haveV:
		e.msac.symbolAdapt(pc, int(partitionTypeCount[bl]), partitionNone)
	case haveH:
		bp = partitionH
		edgeFlags = int(node.h[0])
		e.msac.boolF(0, gatherTopPartitionProb(pc, bl))
	default:
		bp = partitionV
		edgeFlags = int(node.v[0])
		e.msac.boolF(0, gatherLeftPartitionProb(pc, bl))
	}

	e.writeBlock(bx, by, int(blockSizes[bl][bp][0]), edgeFlags)

	setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][bp])
	setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][bp])
}

func (e *encoder) writeTile() []byte {
	e.msac.init(false)
	e.clearCtx()

	sbStep := 16
	for by := 0; by < e.bh; by += sbStep {
		resetContext(&e.l, true)
		for bx := 0; bx < e.bw; bx += sbStep {
			e.writeSb(bx, by, bl64x64, intraEdgeTree[1])
		}
	}

	return e.msac.done()
}

func Encode(c encConfig) []byte {
	seq := writeSeqHdr(&c)
	var sh sequenceHeader
	var gb getBits
	gb.init(seq)
	if err := parseSeqHdr(&sh, &gb, false); err != nil {
		return nil
	}

	var e encoder
	e.init(&c, &sh)
	tile := e.writeTile()

	var fp putBits
	writeFrameHdr(&fp, &sh, &c)
	fp.bytealign()
	body := append(fp.bytes(), tile...)

	out := writeObu(nil, obuTd, nil)
	out = writeObu(out, obuSeqHdr, seq)

	return writeObu(out, obuFrame, body)
}
