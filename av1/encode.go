package av1

import "math"

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
	maxBl   int
	minBl   int
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
	ymode   [256]uint8
	uvmode  [256]uint8
	yangle  [256]int8
	uvangle [256]int8
	bskip   [256]uint8
	split   [nBlLevels][256]bool
	entry   [nBlLevels]sbState
	kept    [nBlLevels]sbState
}

type sbState struct {
	luma [64 * 64]uint8
	chr  [2][32 * 32]uint8
	a    blockContext
	l    blockContext
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
	e.maxBl, e.minBl = bl64x64, bl64x64
	if c.Src != nil {
		e.maxBl, e.minBl = bl32x32, bl8x8
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
		skip = uint32(e.bskip[sbIdx(bx, by)])
	}
	sctx := int(ab.skip[bx4]) + int(e.l.skip[by4])
	e.msac.boolAdapt(e.cdf.m.skip[sctx][:], skip)

	ymodeCdf := e.cdf.kfym[intraModeContext[ab.mode[bx4]]][intraModeContext[e.l.mode[by4]]][:]
	mode := dcPred
	if e.cfg.Src != nil {
		mode = int(e.ymode[sbIdx(bx, by)])
	}
	delta := int(e.yangle[sbIdx(bx, by)])
	e.msac.symbolAdapt(ymodeCdf, nIntraPredModes-1, uint32(mode))
	if int(bDim[2])+int(bDim[3]) >= 2 && directional(mode) {
		e.msac.symbolAdapt(e.cdf.m.angleDelta[mode-vertPred][:], 6, uint32(delta+3))
	}

	uvMode, uvDelta := dcPred, 0
	if hasChroma {
		cflAllowed := b2i(cflAllowedMask&(1<<bs) != 0)
		nSym := nUvIntraPredModes - 1 - (1 - cflAllowed)
		cdf := e.cdf.m.uvMode[cflAllowed][mode][:]
		if e.cfg.Src != nil {
			uvMode = int(e.uvmode[sbIdx(bx, by)])
		}
		uvDelta = int(e.uvangle[sbIdx(bx, by)])
		e.msac.symbolAdapt(cdf, nSym, uint32(uvMode))
		if int(bDim[2])+int(bDim[3]) >= 2 && directional(uvMode) {
			e.msac.symbolAdapt(e.cdf.m.angleDelta[uvMode-vertPred][:], 6, uint32(uvDelta+3))
		}
	}

	if e.cfg.Src != nil {
		e.reconBlock(bx, by, bs, edgeFlags, mode, delta, skip != 0)
		if hasChroma {
			e.reconChroma(bx, by, bs, edgeFlags, uvMode, uvDelta, skip != 0)
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
		if e.cfg.Src == nil {
			memset(edge.lcoef[off:off+n], 0x40)
		}
	}

	if hasChroma {
		cbw4 := (bw4 + e.ssHor) >> e.ssHor
		cbh4 := (bh4 + e.ssVer) >> e.ssVer
		setCtx(ab.uvmode[:], bx4>>e.ssHor, cbw4, uint8(uvMode))
		setCtx(e.l.uvmode[:], by4>>e.ssVer, cbh4, uint8(uvMode))
		if e.cfg.Src == nil {
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
	split := e.split[bl][sbIdx(bx, by)]
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
			if e.cfg.Src != nil {
				e.saveState(bx, by, bl64x64, &e.entry[bl128x128])
				e.trialSb(bx, by, bl64x64, intraEdgeTree[1])
				e.restoreState(bx, by, bl64x64, &e.entry[bl128x128])
			}
			e.writeSb(bx, by, bl64x64, intraEdgeTree[1])
		}
	}

	return e.msac.done()
}

func encConfigValid(c *encConfig) bool {
	if c.Width < 1 || c.Height < 1 || c.BitDepth != 8 {
		return false
	}
	if c.QIndex < 1 || c.QIndex > 255 {
		return false
	}
	if c.Src != nil && c.SrcStride < c.Width {
		return false
	}
	if !c.Monochrome && c.SrcU != nil && c.SrcUVStride < (c.Width+1)>>1 {
		return false
	}

	return c.Monochrome || c.Src == nil || (c.SrcU != nil && c.SrcV != nil)
}

func Encode(c encConfig) []byte {
	if !encConfigValid(&c) {
		return nil
	}

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

func sbIdx(bx, by int) int {
	return (by&15)<<4 | bx&15
}

func (e *encoder) saveState(bx, by, bl int, s *sbState) {
	n := (32 >> bl) * 4
	for y := range n {
		copy(s.luma[y*n:(y+1)*n], e.recon[(by*4+y)*e.stride+bx*4:])
	}
	if !e.cfg.Monochrome {
		cn, cx, cy := n>>e.ssHor, (bx*4)>>e.ssHor, (by*4)>>e.ssVer
		for pl := range 2 {
			for y := range n >> e.ssVer {
				copy(s.chr[pl][y*cn:(y+1)*cn], e.reconC[pl][(cy+y)*e.cstride+cx:])
			}
		}
	}
	s.a = e.a[bx>>5]
	s.l = e.l
}

func (e *encoder) restoreState(bx, by, bl int, s *sbState) {
	n := (32 >> bl) * 4
	for y := range n {
		copy(e.recon[(by*4+y)*e.stride+bx*4:][:n], s.luma[y*n:])
	}
	if !e.cfg.Monochrome {
		cn, cx, cy := n>>e.ssHor, (bx*4)>>e.ssHor, (by*4)>>e.ssVer
		for pl := range 2 {
			for y := range n >> e.ssVer {
				copy(e.reconC[pl][(cy+y)*e.cstride+cx:][:cn], s.chr[pl][y*cn:])
			}
		}
	}
	e.a[bx>>5] = s.a
	e.l = s.l
}

func partitionBits(pc []uint16, bl int, haveH, haveV, split bool) float64 {
	if haveH && haveV {
		if split {
			return cdfBits(pc, partitionSplit)
		}

		return cdfBits(pc, partitionNone)
	}

	f := float64(gatherLeftPartitionProb(pc, bl))
	if haveH {
		f = float64(gatherTopPartitionProb(pc, bl))
	}
	p := f / (1 << 15)
	if !split {
		p = 1 - p
	}
	if p <= 0 {
		return 24
	}

	return -math.Log2(p)
}

func (e *encoder) trialBlock(bx, by, bs, edgeFlags int) float64 {
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	bx4, by4 := bx&31, by&31
	ab := &e.a[bx>>5]

	sctx := int(ab.skip[bx4]) + int(e.l.skip[by4])
	skipCdf := e.cdf.m.skip[sctx][:]
	coded := e.lambda * cdfBits(skipCdf, 0)
	skipped := e.lambda * cdfBits(skipCdf, 1)

	ymodeCdf := e.cdf.kfym[intraModeContext[ab.mode[bx4]]][intraModeContext[e.l.mode[by4]]][:]
	mode, delta, c, sk := e.pickMode(bx, by, bs, edgeFlags, ymodeCdf)
	coded += c
	skipped += sk

	uvMode, uvDelta := dcPred, 0
	hasChroma := e.hasChroma(bx, by, bs)
	if hasChroma {
		cflAllowed := b2i(cflAllowedMask&(1<<bs) != 0)
		cdf := e.cdf.m.uvMode[cflAllowed][mode][:]
		uvMode, uvDelta, c, sk = e.pickUvMode(bx, by, bs, edgeFlags, cdf)
		coded += c
		skipped += sk
		setCtx(ab.uvmode[:], bx4>>e.ssHor, (bw4+e.ssHor)>>e.ssHor, uint8(uvMode))
		setCtx(e.l.uvmode[:], by4>>e.ssVer, (bh4+e.ssVer)>>e.ssVer, uint8(uvMode))
	}

	skip := skipped < coded
	if skip {
		e.reconBlock(bx, by, bs, edgeFlags, mode, delta, true)
		if hasChroma {
			e.reconChroma(bx, by, bs, edgeFlags, uvMode, uvDelta, true)
		}
		coded = skipped
	}

	i := sbIdx(bx, by)
	e.ymode[i] = uint8(mode)
	e.uvmode[i] = uint8(uvMode)
	e.yangle[i] = int8(delta)
	e.uvangle[i] = int8(uvDelta)
	e.bskip[i] = b2u8(skip)

	for i, edge := range [2]*blockContext{ab, &e.l} {
		off, n := bx4, bw4
		if i == 1 {
			off, n = by4, bh4
		}
		setCtx(edge.mode[:], off, n, uint8(mode))
		setCtx(edge.intra[:], off, n, 1)
		setCtx(edge.skip[:], off, n, b2u8(skip))
	}

	return coded
}

func (e *encoder) trialSb(bx, by, bl int, node *edgeNode) float64 {
	hsz := 16 >> bl
	haveH := e.bw > bx+hsz
	haveV := e.bh > by+hsz

	if !haveH && !haveV {
		return e.trialSb(bx, by, bl+1, node.kids[0])
	}

	bx8, by8 := (bx&31)>>1, (by&31)>>1
	ab := &e.a[bx>>5]
	pc := e.cdf.m.partition[bl][getPartitionCtx(ab, &e.l, bl, by8, bx8)][:]
	n := 1 << ulog2(uint32(hsz))

	splitKids := func() float64 {
		c := 0.0
		switch {
		case haveH && haveV:
			c += e.trialSb(bx, by, bl+1, node.kids[0])
			c += e.trialSb(bx+hsz, by, bl+1, node.kids[1])
			c += e.trialSb(bx, by+hsz, bl+1, node.kids[2])
			c += e.trialSb(bx+hsz, by+hsz, bl+1, node.kids[3])
		case haveH:
			c += e.trialSb(bx, by, bl+1, node.kids[0])
			c += e.trialSb(bx+hsz, by, bl+1, node.kids[1])
		default:
			c += e.trialSb(bx, by, bl+1, node.kids[0])
			c += e.trialSb(bx, by+hsz, bl+1, node.kids[2])
		}

		return c
	}

	leaf := func() float64 {
		bp, ef := partitionNone, int(node.o)
		switch {
		case haveH && haveV:
		case haveH:
			bp, ef = partitionH, int(node.h[0])
		default:
			bp, ef = partitionV, int(node.v[0])
		}
		c := e.trialBlock(bx, by, int(blockSizes[bl][bp][0]), ef)
		setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][bp])
		setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][bp])

		return c
	}

	if bl < e.maxBl {
		e.split[bl][sbIdx(bx, by)] = true

		return e.lambda*partitionBits(pc, bl, haveH, haveV, true) + splitKids()
	}
	if bl >= e.minBl {
		e.split[bl][sbIdx(bx, by)] = false

		return e.lambda*partitionBits(pc, bl, haveH, haveV, false) + leaf()
	}

	e.saveState(bx, by, bl, &e.entry[bl])
	splitCost := e.lambda*partitionBits(pc, bl, haveH, haveV, true) + splitKids()
	e.saveState(bx, by, bl, &e.kept[bl])
	e.restoreState(bx, by, bl, &e.entry[bl])

	noneCost := e.lambda*partitionBits(pc, bl, haveH, haveV, false) + leaf()
	if noneCost <= splitCost {
		e.split[bl][sbIdx(bx, by)] = false

		return noneCost
	}

	e.restoreState(bx, by, bl, &e.kept[bl])
	e.split[bl][sbIdx(bx, by)] = true

	return splitCost
}
