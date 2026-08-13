package av1

import "math"

const encEdgeOff = 128

var intraModes = [nIntraPredModes]int{
	dcPred, vertPred, horPred, diagDownLeftPred, diagDownRightPred,
	vertRightPred, horDownPred, horUpPred, vertLeftPred,
	smoothPred, smoothVPred, smoothHPred, paethPred,
}

func cdfBits(cdf []uint16, val int) float64 {
	hi := float64(1 << 15)
	if val > 0 {
		hi = float64(cdf[val-1])
	}
	p := (hi - float64(cdf[val])) / (1 << 15)
	if p <= 0 {
		return 24
	}

	return -math.Log2(p)
}

var levelBits [16]float64

func init() {
	for i := range levelBits {
		levelBits[i] = 3 + 2*math.Log2(float64(i)+1)
	}
}

func rateBits(lv []int32, scan []uint16, eob int) float64 {
	if eob < 0 {
		return 1
	}
	bits := 4 + 0.3*float64(eob)
	for i := 0; i <= eob; i++ {
		l := abs32(lv[scan[i]])
		if l == 0 {
			continue
		}
		if int(l) < len(levelBits) {
			bits += levelBits[l]

			continue
		}
		bits += 3 + 2*math.Log2(float64(l)+1)
	}

	return bits
}

func sse(dst []uint8, off, stride int, src []uint8, sstride, sw, sh, px, py, w, h int) float64 {
	var s float64
	for j := range h {
		sy := min(py+j, sh-1)
		for i := range w {
			d := float64(src[sy*sstride+min(px+i, sw-1)]) - float64(dst[off+j*stride+i])
			s += d * d
		}
	}

	return s
}

func residual(res []int32, dst []uint8, off, stride int, src []uint8, sstride,
	sw, sh, px, py, w, h int,
) {
	for j := range h {
		sy := min(py+j, sh-1)
		row := j * w
		for i := range w {
			res[row+i] = int32(src[sy*sstride+min(px+i, sw-1)]) - int32(dst[off+j*stride+i])
		}
	}
}

func txEdge(sbHasTr, sbHasBl, initX, initY, x, y, tw4, th4, subW4, subH4 int) int {
	ef := 0
	if !((y > initY || sbHasTr == 0) && x+tw4 >= subW4) {
		ef |= edgeI444TopHasRight
	}
	if !(x > initX || (sbHasBl == 0 && y+th4 >= subH4)) {
		ef |= edgeI444LeftHasBottom
	}

	return ef
}

func lumaTxEdge(edgeFlags, w4, h4, initX, initY, x, y, tw4, th4 int) int {
	sbHasTr := 0
	switch {
	case initX+16 < w4:
		sbHasTr = 1
	case initY != 0:
	default:
		sbHasTr = edgeFlags & edgeI444TopHasRight
	}
	sbHasBl := 0
	switch {
	case initX != 0:
	case initY+16 < h4:
		sbHasBl = 1
	default:
		sbHasBl = edgeFlags & edgeI444LeftHasBottom
	}

	return txEdge(sbHasTr, sbHasBl, initX, initY, x, y, tw4, th4,
		min(w4, initX+16), min(h4, initY+16))
}

func (e *encoder) chromaTxEdge(edgeFlags, cw4, ch4, initX, initY, x, y, tw4, th4 int) int {
	ssHor, ssVer := e.ssHor, e.ssVer

	sbHasTr := 0
	switch {
	case (initX+16)>>ssHor < cw4:
		sbHasTr = 1
	case initY != 0:
	default:
		sbHasTr = edgeFlags & (edgeI420TopHasRight >> (e.layout - 1))
	}
	sbHasBl := 0
	switch {
	case initX != 0:
	case (initY+16)>>ssVer < ch4:
		sbHasBl = 1
	default:
		sbHasBl = edgeFlags & (edgeI420LeftHasBottom >> (e.layout - 1))
	}

	return txEdge(sbHasTr, sbHasBl, initX>>ssHor, initY>>ssVer, x, y, tw4, th4,
		min(cw4, (initX+16)>>ssHor), min(ch4, (initY+16)>>ssVer))
}

func (e *encoder) txtpFor(tx, plane, mode int) int {
	tDim := &txfmDimensions[tx]
	if plane == 0 || int(tDim.max)+1 >= tx64x64 {
		return dctDct
	}

	return int(txtpFromUvmode[mode])
}

func (e *encoder) predIntra(dst []uint8, off, stride, x, y, w, h, tw, th,
	mode, angle, edgeFlags, smFl, maxW, maxH int,
) {
	a := angle
	m := prepareIntraEdges(x, x > 0, y, y > 0, w, h, edgeFlags,
		dst, off, stride, nil, 0, mode, &a, tw, th,
		e.seq.intraEdgeFilter != 0, e.edge[:], encEdgeOff, 255)
	e.dsp.ipred[m](dst, off, stride, e.edge[:], encEdgeOff, tw*4, th*4,
		a|smFl, maxW, maxH, 255)
}

func (e *encoder) modeBits(mode int, cdf []uint16) float64 {
	bits := cdfBits(cdf, mode)
	if mode >= vertPred && mode <= vertLeftPred {
		bits += cdfBits(e.cdf.m.angleDelta[mode-vertPred][:], 3)
	}

	return bits
}

func (e *encoder) pickMode(bx, by, bs, edgeFlags int, cdf []uint16) (int, float64) {
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	tx := int(maxTxfmSizeForBs[bs][0])
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	bx4, by4 := bx&31, by&31
	ab := &e.a[bx>>5]
	smFl := smFlag(ab, bx4) | smFlag(&e.l, by4) | int(e.seq.intraEdgeFilter)<<10
	ef := lumaTxEdge(edgeFlags, w4, h4, 0, 0, 0, 0, int(tDim.w), int(tDim.h))
	off := 4 * (by*e.stride + bx)

	trial := func(m int) float64 {
		e.predIntra(e.recon, off, e.stride, bx, by, e.bw, e.bh,
			int(tDim.w), int(tDim.h), m, 0, ef, smFl,
			4*e.bw-4*bx, 4*e.bh-4*by)
		eob := e.quantTx(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
			e.cfg.Width, e.cfg.Height, 4*bx, 4*by, tx, dctDct)
		e.reconTx(e.recon, off, e.stride, tx, dctDct, eob)

		return sse(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
			e.cfg.Width, e.cfg.Height, 4*bx, 4*by, tw, th) +
			e.lambda*(rateBits(e.lv[:], scans[tx], eob)+e.modeBits(m, cdf))
	}

	best, bestCost := dcPred, math.Inf(1)
	for _, m := range intraModes {
		if c := trial(m); c < bestCost {
			best, bestCost = m, c
		}
	}
	if best != intraModes[len(intraModes)-1] {
		trial(best)
	}

	return best, bestCost
}

func (e *encoder) pickUvMode(bx, by, bs, edgeFlags int, cdf []uint16) (int, float64) {
	ssHor, ssVer := e.ssHor, e.ssVer
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	cbx4, cby4 := (bx&31)>>ssHor, (by&31)>>ssVer
	cbx, cby := bx>>ssHor, by>>ssVer
	tx := int(maxTxfmSizeForBs[bs][e.layout])
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	ab := &e.a[bx>>5]
	smFl := smUvFlag(ab, cbx4) | smUvFlag(&e.l, cby4) | angleUseEdgeFilterFlag
	ef := e.chromaTxEdge(edgeFlags, cw4, ch4, 0, 0, 0, 0, int(tDim.w), int(tDim.h))
	off := 4 * (cby*e.cstride + cbx)

	trial := func(m int) float64 {
		txtp := e.txtpFor(tx, 1, m)
		c := e.lambda * e.modeBits(m, cdf)
		for pl := range 2 {
			src := e.cfg.SrcU
			if pl == 1 {
				src = e.cfg.SrcV
			}
			e.predIntra(e.reconC[pl], off, e.cstride, cbx, cby,
				e.bw>>ssHor, e.bh>>ssVer, int(tDim.w), int(tDim.h), m, 0, ef, smFl,
				(4*e.bw+ssHor-4*bx)>>ssHor, (4*e.bh+ssVer-4*by)>>ssVer)
			eob := e.quantTx(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
				e.cw, e.ch, 4*cbx, 4*cby, tx, txtp)
			e.reconTx(e.reconC[pl], off, e.cstride, tx, txtp, eob)
			c += sse(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
				e.cw, e.ch, 4*cbx, 4*cby, tw, th) +
				e.lambda*rateBits(e.lv[:], scans[tx], eob)
		}

		return c
	}

	best, bestCost := dcPred, math.Inf(1)
	for _, m := range intraModes {
		if c := trial(m); c < bestCost {
			best, bestCost = m, c
		}
	}
	if best != intraModes[len(intraModes)-1] {
		trial(best)
	}

	return best, bestCost
}

func (e *encoder) quantTx(dst []uint8, off, stride int, src []uint8, sstride, sw, sh,
	px, py, tx, txtp int,
) int {
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	shift := quantShift(tx)

	residual(e.res[:], dst, off, stride, src, sstride, sw, sh, px, py, tw, th)
	fwdTxfm(e.coeff[:], e.res[:], tw, tx, txtp, e.fwd)

	cw, ch := min(tw, 32), min(th, 32)
	for i := range cw * ch {
		d := e.dq[1]
		if i == 0 {
			d = e.dq[0]
		}
		e.lv[i] = quantOne(e.coeff[i], uint32(d), shift)
	}

	scan := scans[tx]
	for i := cw*ch - 1; i >= 0; i-- {
		if e.lv[scan[i]] != 0 {
			return i
		}
	}

	return -1
}

func (e *encoder) reconTx(dst []uint8, off, stride, tx, txtp, eob int) {
	if eob < 0 {
		return
	}
	tDim := &txfmDimensions[tx]
	shift := quantShift(tx)
	n := min(4*int(tDim.w), 32) * min(4*int(tDim.h), 32)

	clear(e.cf[:n])
	scan := scans[tx]
	for i := 0; i <= eob; i++ {
		rc := scan[i]
		if e.lv[rc] == 0 {
			continue
		}
		dqv := uint32(e.dq[1])
		if rc == 0 {
			dqv = uint32(e.dq[0])
		}
		d := min(int32(uint32(abs32(e.lv[rc]))*dqv>>uint(shift)), 1<<15-1)
		if e.lv[rc] < 0 {
			d = -d
		}
		e.cf[rc] = int16(d)
	}
	invTxfmAdd(e.dsp, dst, off, stride, e.cf[:], e.itx[:], eob, tx, txtp, 255)
}

func (e *encoder) codeTx(dst []uint8, off, stride int, src []uint8, sstride, sw, sh,
	px, py, tx, bs, plane, mode int, cc *coefCtx, aOff, lOff, aw, lh int,
) {
	txtp := e.txtpFor(tx, plane, mode)
	eob := e.quantTx(dst, off, stride, src, sstride, sw, sh, px, py, tx, txtp)

	ctx := e.writeCoefs(cc, aOff, lOff, tx, bs, plane, uint8(mode), e.lv[:], e.levels[:])
	memset(cc.a[aOff:aOff+aw], ctx)
	memset(cc.l[lOff:lOff+lh], ctx)

	e.reconTx(dst, off, stride, tx, txtp, eob)
}

func (e *encoder) reconBlock(bx, by, bs, edgeFlags, mode int) {
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	bx4, by4 := bx&31, by&31
	tx := int(maxTxfmSizeForBs[bs][0])
	tDim := &txfmDimensions[tx]
	ab := &e.a[bx>>5]
	cc := coefCtx{a: ab.lcoef[:], l: e.l.lcoef[:]}
	smFl := smFlag(ab, bx4) | smFlag(&e.l, by4) | int(e.seq.intraEdgeFilter)<<10

	for initY := 0; initY < h4; initY += 16 {
		for initX := 0; initX < w4; initX += 16 {
			for y := initY; y < min(h4, initY+16); y += int(tDim.h) {
				for x := initX; x < min(w4, initX+16); x += int(tDim.w) {
					ef := lumaTxEdge(edgeFlags, w4, h4, initX, initY, x, y,
						int(tDim.w), int(tDim.h))
					off := 4 * ((by+y)*e.stride + bx + x)

					e.predIntra(e.recon, off, e.stride, bx+x, by+y, e.bw, e.bh,
						int(tDim.w), int(tDim.h), mode, 0, ef, smFl,
						4*e.bw-4*(bx+x), 4*e.bh-4*(by+y))
					e.codeTx(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
						e.cfg.Width, e.cfg.Height, 4*(bx+x), 4*(by+y),
						tx, bs, 0, mode, &cc, bx4+x, by4+y,
						min(int(tDim.w), e.bw-bx-x), min(int(tDim.h), e.bh-by-y))
				}
			}
		}
	}
}

func (e *encoder) reconChroma(bx, by, bs, edgeFlags, mode int) {
	ssHor, ssVer := e.ssHor, e.ssVer
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	cbx4, cby4 := (bx&31)>>ssHor, (by&31)>>ssVer
	cbx, cby := bx>>ssHor, by>>ssVer
	tx := int(maxTxfmSizeForBs[bs][e.layout])
	tDim := &txfmDimensions[tx]
	ab := &e.a[bx>>5]
	smFl := smUvFlag(ab, cbx4) | smUvFlag(&e.l, cby4) | angleUseEdgeFilterFlag

	for initY := 0; initY < h4; initY += 16 {
		for initX := 0; initX < w4; initX += 16 {
			for pl := range 2 {
				src := e.cfg.SrcU
				if pl == 1 {
					src = e.cfg.SrcV
				}
				dst := e.reconC[pl]
				cc := coefCtx{a: ab.ccoef[pl][:], l: e.l.ccoef[pl][:]}

				for y := initY >> ssVer; y < min(ch4, (initY+16)>>ssVer); y += int(tDim.h) {
					for x := initX >> ssHor; x < min(cw4, (initX+16)>>ssHor); x += int(tDim.w) {
						ef := e.chromaTxEdge(edgeFlags, cw4, ch4, initX, initY,
							x, y, int(tDim.w), int(tDim.h))
						lx := bx + initX + ((x - initX>>ssHor) << ssHor)
						ly := by + initY + ((y - initY>>ssVer) << ssVer)
						off := 4 * ((cby+y)*e.cstride + cbx + x)

						e.predIntra(dst, off, e.cstride, cbx+x, cby+y,
							e.bw>>ssHor, e.bh>>ssVer, int(tDim.w), int(tDim.h),
							mode, 0, ef, smFl,
							(4*e.bw+ssHor-4*(lx&^ssHor))>>ssHor,
							(4*e.bh+ssVer-4*(ly&^ssVer))>>ssVer)
						e.codeTx(dst, off, e.cstride, src, e.cfg.SrcUVStride,
							e.cw, e.ch, 4*(cbx+x), 4*(cby+y),
							tx, bs, 1+pl, mode, &cc, cbx4+x, cby4+y,
							min(int(tDim.w), (e.bw-lx+ssHor)>>ssHor),
							min(int(tDim.h), (e.bh-ly+ssVer)>>ssVer))
					}
				}
			}
		}
	}
}
