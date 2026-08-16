package av1

import "math"

type encoder struct {
	cfg      EncodeConfig
	seq      sequenceHeader
	cdf      cdfContext
	msac     msacEncoder
	a        []blockContext
	l        blockContext
	bw       int
	bh       int
	layout   int
	lossless bool
	ssHor    int
	ssVer    int
	maxBl    int
	minBl    int
	sp       speedParams
	stride   int
	recon    []uint8
	reconC   [2][]uint8
	cstride  int
	cw       int
	ch       int
	dq       [2]uint16
	qr       [2]quantizer
	lambda   float64
	predSSE  float64
	dsp      *dspContext[uint8]
	itx      []int32
	fwd      []int32
	res      [64 * 64]int32
	coeff    [32 * 32]int32
	lv       [32 * 32]int32
	cf       [32 * 32]int16
	ac       [32 * 32]int16
	cfla     [256][2]int8
	edge     [257]uint8
	levels   [64 * 66]uint8
	ymode    [256]uint8
	uvmode   [256]uint8
	yangle   [256]int8
	uvangle  [256]int8
	bskip    [256]uint8
	part     [nBlLevels][256]uint8
	rect     [4]int
	split    [2]int
	entry    [nBlLevels]sbState
	kept     [nBlLevels]sbState
	dentry   [nBlLevels]decState
	dkept    [nBlLevels]decState
}

type sbState struct {
	luma [64 * 64]uint8
	chr  [2][32 * 32]uint8
	a    blockContext
	l    blockContext
}

type decState struct {
	ymode   [256]uint8
	uvmode  [256]uint8
	yangle  [256]int8
	uvangle [256]int8
	bskip   [256]uint8
	cfla    [256][2]int8
	part    [nBlLevels][256]uint8
}

type speedParams struct {
	minBl  int
	shapes int
	pre    int
	modes  int
}

var speedTable = [11]speedParams{
	{bl8x8, 3, 13, 5},
	{bl8x8, 3, 13, 3},
	{bl8x8, 1, 13, 13},
	{bl8x8, 1, 13, 8},
	{bl8x8, 1, 13, 5},
	{bl8x8, 1, 13, 3},
	{bl8x8, 1, 6, 3},
	{bl16x16, 1, 13, 3},
	{bl16x16, 1, 6, 3},
	{bl32x32, 1, 13, 3},
	{bl32x32, 1, 1, 1},
}

func (e *encoder) init(c *EncodeConfig, seq *sequenceHeader) {
	e.cfg = *c
	e.seq = *seq
	e.bw = ((c.Width + 7) >> 3) << 1
	e.bh = ((c.Height + 7) >> 3) << 1
	e.a = make([]blockContext, (e.bw+31)>>5+1)
	e.layout = pixelLayoutI420
	if c.Monochrome {
		e.layout = pixelLayoutI400
	}
	e.lossless = c.QIndex == 0
	e.ssHor = b2i(e.layout != pixelLayoutI444)
	e.ssVer = b2i(e.layout == pixelLayoutI420)
	e.rect = [4]int{partitionNone, partitionH, partitionV, partitionSplit}
	e.split = [2]int{partitionNone, partitionSplit}
	e.sp = speedTable[min(max(c.Speed, 0), len(speedTable)-1)]
	e.maxBl, e.minBl = bl64x64, bl64x64
	if c.Src != nil {
		e.maxBl, e.minBl = bl32x32, e.sp.minBl
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
	e.qr[0] = newQuantizer(uint32(e.dq[0]))
	e.qr[1] = newQuantizer(uint32(e.dq[1]))
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
		cflAllowed := e.cflAllowed(bs)
		nSym := nUvIntraPredModes - 1 - (1 - cflAllowed)
		cdf := e.cdf.m.uvMode[cflAllowed][mode][:]
		if e.cfg.Src != nil {
			uvMode = int(e.uvmode[sbIdx(bx, by)])
		}
		uvDelta = int(e.uvangle[sbIdx(bx, by)])
		e.msac.symbolAdapt(cdf, nSym, uint32(uvMode))
		if uvMode == cflPred {
			a := e.cfla[sbIdx(bx, by)]
			su, sv := cflSign(int(a[0])), cflSign(int(a[1]))
			e.msac.symbolAdapt(e.cdf.m.cflSign[:], 7, uint32(su*3+sv-1))
			if su != 0 {
				e.msac.symbolAdapt(e.cdf.m.cflAlpha[b2i(su == 2)*3+sv][:], 15,
					uint32(abs(int(a[0]))-1))
			}
			if sv != 0 {
				e.msac.symbolAdapt(e.cdf.m.cflAlpha[b2i(sv == 2)*3+su][:], 15,
					uint32(abs(int(a[1]))-1))
			}
		} else if int(bDim[2])+int(bDim[3]) >= 2 && directional(uvMode) {
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

func (e *encoder) cflAllowed(bs int) int {
	if !e.lossless {
		return b2i(cflAllowedMask&(1<<bs) != 0)
	}
	bDim := &blockDimensions[bs]

	return b2i((int(bDim[0])+e.ssHor)>>e.ssHor == 1 && (int(bDim[1])+e.ssVer)>>e.ssVer == 1)
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
	pc := e.cdf.m.partition[bl][getPartitionCtx(ab, &e.l, bl, by8, bx8)][:]
	bp := int(e.part[bl][sbIdx(bx, by)])
	if bp != partitionSplit && !(haveH && haveV) {
		bp = partitionV
		if haveH {
			bp = partitionH
		}
	}
	n := 1 << ulog2(uint32(hsz))

	switch {
	case haveH && haveV:
		e.msac.symbolAdapt(pc, int(partitionTypeCount[bl]), uint32(bp))
	case haveH:
		e.msac.boolF(b2u(bp == partitionSplit), gatherTopPartitionProb(pc, bl))
	default:
		e.msac.boolF(b2u(bp == partitionSplit), gatherLeftPartitionProb(pc, bl))
	}

	switch bp {
	case partitionSplit:
		switch {
		case haveH && haveV:
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx+hsz, by, bl+1, node.kids[1])
			e.writeSb(bx, by+hsz, bl+1, node.kids[2])
			e.writeSb(bx+hsz, by+hsz, bl+1, node.kids[3])
		case haveH:
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx+hsz, by, bl+1, node.kids[1])
		default:
			e.writeSb(bx, by, bl+1, node.kids[0])
			e.writeSb(bx, by+hsz, bl+1, node.kids[2])
		}
		if bl != bl8x8 {
			return
		}
	case partitionH:
		bs := int(blockSizes[bl][partitionH][0])
		e.writeBlock(bx, by, bs, int(node.h[0]))
		if haveV {
			e.writeBlock(bx, by+hsz, bs, int(node.h[1]))
		}
	case partitionV:
		bs := int(blockSizes[bl][partitionV][0])
		e.writeBlock(bx, by, bs, int(node.v[0]))
		if haveH {
			e.writeBlock(bx+hsz, by, bs, int(node.v[1]))
		}
	case partitionTTopSplit:
		b := &blockSizes[bl][bp]
		e.writeBlock(bx, by, int(b[0]), edgeAllTrAndBl)
		e.writeBlock(bx+hsz, by, int(b[0]), int(node.v[1]))
		e.writeBlock(bx, by+hsz, int(b[1]), int(node.h[1]))
	case partitionTBottomSplit:
		b := &blockSizes[bl][bp]
		e.writeBlock(bx, by, int(b[0]), int(node.h[0]))
		e.writeBlock(bx, by+hsz, int(b[1]), int(node.v[0]))
		e.writeBlock(bx+hsz, by+hsz, int(b[1]), 0)
	case partitionTLeftSplit:
		b := &blockSizes[bl][bp]
		e.writeBlock(bx, by, int(b[0]), edgeAllTrAndBl)
		e.writeBlock(bx, by+hsz, int(b[0]), int(node.h[1]))
		e.writeBlock(bx+hsz, by, int(b[1]), int(node.v[1]))
	case partitionTRightSplit:
		b := &blockSizes[bl][bp]
		e.writeBlock(bx, by, int(b[0]), int(node.v[0]))
		e.writeBlock(bx+hsz, by, int(b[1]), int(node.h[0]))
		e.writeBlock(bx+hsz, by+hsz, int(b[1]), 0)
	case partitionH4:
		bs := int(blockSizes[bl][bp][0])
		e.writeBlock(bx, by, bs, int(node.h[0]))
		e.writeBlock(bx, by+hsz>>1, bs, int(node.h4))
		e.writeBlock(bx, by+hsz, bs, edgeAllLeftHasBottom)
		if by+hsz*3>>1 < e.bh {
			e.writeBlock(bx, by+hsz*3>>1, bs, int(node.h[1]))
		}
	case partitionV4:
		bs := int(blockSizes[bl][bp][0])
		e.writeBlock(bx, by, bs, int(node.v[0]))
		e.writeBlock(bx+hsz>>1, by, bs, int(node.v4))
		e.writeBlock(bx+hsz, by, bs, edgeAllTopHasRight)
		if bx+hsz*3>>1 < e.bw {
			e.writeBlock(bx+hsz*3>>1, by, bs, int(node.v[1]))
		}
	default:
		e.writeBlock(bx, by, int(blockSizes[bl][partitionNone][0]), int(node.o))
	}

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

func encodeConfigValid(c *EncodeConfig) bool {
	if c.Width < 1 || c.Height < 1 || c.BitDepth != 8 {
		return false
	}
	if c.QIndex < 0 || c.QIndex > 255 {
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

// Encode writes c as one all-intra AV1 temporal unit: a temporal delimiter, a
// sequence header and a single frame OBU. It returns nil if the configuration
// is not one the encoder accepts.
func Encode(c EncodeConfig) []byte {
	if !encodeConfigValid(&c) {
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

func (e *encoder) saveDec(d *decState) {
	d.ymode, d.uvmode, d.bskip, d.cfla = e.ymode, e.uvmode, e.bskip, e.cfla
	d.yangle, d.uvangle, d.part = e.yangle, e.uvangle, e.part
}

func (e *encoder) restoreDec(d *decState) {
	e.ymode, e.uvmode, e.bskip, e.cfla = d.ymode, d.uvmode, d.bskip, d.cfla
	e.yangle, e.uvangle, e.part = d.yangle, d.uvangle, d.part
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

func partitionBits(pc []uint16, bl int, haveH, haveV bool, bp int) float64 {
	if haveH && haveV {
		return cdfBits(pc, bp)
	}

	f := float64(gatherLeftPartitionProb(pc, bl))
	if haveH {
		f = float64(gatherTopPartitionProb(pc, bl))
	}
	p := f / (1 << 15)
	if bp != partitionSplit {
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
		cflAllowed := e.cflAllowed(bs)
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
	idx := sbIdx(bx, by)

	run := func(bp int) float64 {
		c := e.lambda * partitionBits(pc, bl, haveH, haveV, bp)
		switch bp {
		case partitionSplit:
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
			if bl != bl8x8 {
				return c
			}
		case partitionH:
			bs := int(blockSizes[bl][partitionH][0])
			c += e.trialBlock(bx, by, bs, int(node.h[0]))
			if haveV {
				c += e.trialBlock(bx, by+hsz, bs, int(node.h[1]))
			}
		case partitionV:
			bs := int(blockSizes[bl][partitionV][0])
			c += e.trialBlock(bx, by, bs, int(node.v[0]))
			if haveH {
				c += e.trialBlock(bx+hsz, by, bs, int(node.v[1]))
			}
		case partitionTTopSplit:
			b := &blockSizes[bl][bp]
			c += e.trialBlock(bx, by, int(b[0]), edgeAllTrAndBl)
			c += e.trialBlock(bx+hsz, by, int(b[0]), int(node.v[1]))
			c += e.trialBlock(bx, by+hsz, int(b[1]), int(node.h[1]))
		case partitionTBottomSplit:
			b := &blockSizes[bl][bp]
			c += e.trialBlock(bx, by, int(b[0]), int(node.h[0]))
			c += e.trialBlock(bx, by+hsz, int(b[1]), int(node.v[0]))
			c += e.trialBlock(bx+hsz, by+hsz, int(b[1]), 0)
		case partitionTLeftSplit:
			b := &blockSizes[bl][bp]
			c += e.trialBlock(bx, by, int(b[0]), edgeAllTrAndBl)
			c += e.trialBlock(bx, by+hsz, int(b[0]), int(node.h[1]))
			c += e.trialBlock(bx+hsz, by, int(b[1]), int(node.v[1]))
		case partitionTRightSplit:
			b := &blockSizes[bl][bp]
			c += e.trialBlock(bx, by, int(b[0]), int(node.v[0]))
			c += e.trialBlock(bx+hsz, by, int(b[1]), int(node.h[0]))
			c += e.trialBlock(bx+hsz, by+hsz, int(b[1]), 0)
		case partitionH4:
			bs := int(blockSizes[bl][bp][0])
			c += e.trialBlock(bx, by, bs, int(node.h[0]))
			c += e.trialBlock(bx, by+hsz>>1, bs, int(node.h4))
			c += e.trialBlock(bx, by+hsz, bs, edgeAllLeftHasBottom)
			if by+hsz*3>>1 < e.bh {
				c += e.trialBlock(bx, by+hsz*3>>1, bs, int(node.h[1]))
			}
		case partitionV4:
			bs := int(blockSizes[bl][bp][0])
			c += e.trialBlock(bx, by, bs, int(node.v[0]))
			c += e.trialBlock(bx+hsz>>1, by, bs, int(node.v4))
			c += e.trialBlock(bx+hsz, by, bs, edgeAllTopHasRight)
			if bx+hsz*3>>1 < e.bw {
				c += e.trialBlock(bx+hsz*3>>1, by, bs, int(node.v[1]))
			}
		default:
			c += e.trialBlock(bx, by, int(blockSizes[bl][partitionNone][0]), int(node.o))
		}
		setCtx(ab.partition[:], bx8, n, alPartCtx[0][bl][bp])
		setCtx(e.l.partition[:], by8, n, alPartCtx[1][bl][bp])

		return c
	}

	var cands []int
	switch {
	case bl < e.maxBl:
		cands = e.rect[3:4]
	case !haveH:
		cands = e.rect[2:3]
	case !haveV:
		cands = e.rect[1:2]
	case bl >= e.minBl:
		cands = e.rect[:e.sp.shapes]
	case e.sp.shapes == 1:
		cands = e.split[:]
	default:
		cands = e.rect[:4]
	}

	if len(cands) == 1 {
		e.part[bl][idx] = uint8(cands[0])

		return run(cands[0])
	}

	e.saveState(bx, by, bl, &e.entry[bl])
	e.saveDec(&e.dentry[bl])
	bestCost, bestPart := math.Inf(1), cands[0]
	for i, bp := range cands {
		if i > 0 {
			e.restoreState(bx, by, bl, &e.entry[bl])
			e.restoreDec(&e.dentry[bl])
		}
		if c := run(bp); c < bestCost {
			bestCost, bestPart = c, bp
			if i < len(cands)-1 {
				e.saveState(bx, by, bl, &e.kept[bl])
				e.saveDec(&e.dkept[bl])
			}
		}
	}
	if bestPart != cands[len(cands)-1] {
		e.restoreState(bx, by, bl, &e.kept[bl])
		e.restoreDec(&e.dkept[bl])
	}
	e.part[bl][idx] = uint8(bestPart)

	return bestCost
}
