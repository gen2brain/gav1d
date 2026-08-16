package av1

import "math"

const encEdgeOff = 128

var intraModes = [nIntraPredModes]int{
	dcPred, smoothPred, paethPred, vertPred, horPred,
	smoothVPred, smoothHPred, diagDownLeftPred, diagDownRightPred,
	vertRightPred, horDownPred, horUpPred, vertLeftPred,
}

var log2Tab [1<<15 + 1]float32

func init() {
	for i := 1; i < len(log2Tab); i++ {
		log2Tab[i] = float32(15 - math.Log2(float64(i)))
	}
	log2Tab[0] = 24
}

func boolFBits(f, bit uint32) float64 {
	if bit == 0 {
		f = 1<<15 - f
	}
	if f > 1<<15 {
		return 24
	}

	return float64(log2Tab[f])
}

func cdfBits(cdf []uint16, val int) float64 {
	hi := uint32(1 << 15)
	if val > 0 {
		hi = uint32(cdf[val-1])
	}
	lo := uint32(cdf[val])
	if lo >= hi {
		return 24
	}

	return float64(log2Tab[hi-lo])
}

var levelBits [16]float64

func init() {
	for i := range levelBits {
		levelBits[i] = 3 + 2*math.Log2(float64(i)+1)
	}
}

func sseGo(dst []uint8, off, stride int, src []uint8, sstride, w, h int) int64 {
	var ssd int64
	for j := range h {
		s := src[j*sstride:][:w]
		d := dst[off+j*stride:][:w]
		for i := range d {
			v := int64(s[i]) - int64(d[i])
			ssd += v * v
		}
	}

	return ssd
}

func sse(dst []uint8, off, stride int, src []uint8, sstride, sw, sh, px, py, w, h int) float64 {
	if px+w <= sw && py+h <= sh {
		return float64(sseKernel(dst, off, stride, src[py*sstride+px:], sstride, w, h))
	}

	var s int64
	for j := range h {
		sy := min(py+j, sh-1)
		for i := range w {
			d := int64(src[sy*sstride+min(px+i, sw-1)]) - int64(dst[off+j*stride+i])
			s += d * d
		}
	}

	return float64(s)
}

func had8(v *[8]int32) {
	a0, a1, a2, a3 := v[0]+v[4], v[1]+v[5], v[2]+v[6], v[3]+v[7]
	a4, a5, a6, a7 := v[0]-v[4], v[1]-v[5], v[2]-v[6], v[3]-v[7]
	b0, b1, b2, b3 := a0+a2, a1+a3, a0-a2, a1-a3
	b4, b5, b6, b7 := a4+a6, a5+a7, a4-a6, a5-a7
	v[0], v[1] = b0+b1, b0-b1
	v[2], v[3] = b2+b3, b2-b3
	v[4], v[5] = b4+b5, b4-b5
	v[6], v[7] = b6+b7, b6-b7
}

func had4(v *[4]int32) {
	a0, a1, a2, a3 := v[0]+v[2], v[1]+v[3], v[0]-v[2], v[1]-v[3]
	v[0], v[1] = a0+a1, a0-a1
	v[2], v[3] = a2+a3, a2-a3
}

func satd8(res []int32, off, stride int) int32 {
	var b [8][8]int32
	for j := range 8 {
		var r [8]int32
		copy(r[:], res[off+j*stride:])
		had8(&r)
		for i := range 8 {
			b[i][j] = r[i]
		}
	}

	var sum int32
	for i := range 8 {
		c := b[i]
		had8(&c)
		for _, v := range c {
			sum += abs32(v)
		}
	}

	return (sum + 4) >> 3
}

func satd4(res []int32, off, stride int) int32 {
	var b [4][4]int32
	for j := range 4 {
		var r [4]int32
		copy(r[:], res[off+j*stride:])
		had4(&r)
		for i := range 4 {
			b[i][j] = r[i]
		}
	}

	var sum int32
	for i := range 4 {
		c := b[i]
		had4(&c)
		for _, v := range c {
			sum += abs32(v)
		}
	}

	return (sum + 2) >> 2
}

func satd(res []int32, w, h int) int32 {
	var sum int32
	if w >= 8 && h >= 8 {
		for y := 0; y < h; y += 8 {
			for x := 0; x < w; x += 8 {
				sum += satd8Kernel(res, y*w+x, w)
			}
		}

		return sum
	}
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			sum += satd4Kernel(res, y*w+x, w)
		}
	}

	return sum
}

func residualGo(res []int32, dst []uint8, off, stride int, src []uint8, sstride,
	w, h int,
) int64 {
	var ssd int64
	for j := range h {
		s := src[j*sstride:][:w]
		d := dst[off+j*stride:][:w]
		r := res[j*w:][:w]
		for i := range r {
			v := int32(s[i]) - int32(d[i])
			r[i] = v
			ssd += int64(v) * int64(v)
		}
	}

	return ssd
}

func residualEdge(res []int32, dst []uint8, off, stride int, src []uint8, sstride,
	sw, sh, px, py, w, h int,
) int64 {
	var ssd int64
	for j := range h {
		sy := min(py+j, sh-1)
		row := j * w
		for i := range w {
			d := int32(src[sy*sstride+min(px+i, sw-1)]) - int32(dst[off+j*stride+i])
			res[row+i] = d
			ssd += int64(d) * int64(d)
		}
	}

	return ssd
}

func residual(res []int32, dst []uint8, off, stride int, src []uint8, sstride,
	sw, sh, px, py, w, h int,
) float64 {
	if px+w <= sw && py+h <= sh {
		return float64(residualKernel(res, dst, off, stride, src[py*sstride+px:], sstride, w, h))
	}

	return float64(residualEdge(res, dst, off, stride, src, sstride, sw, sh, px, py, w, h))
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

func (e *encoder) txOf(bs, plane int) int {
	if e.lossless {
		return tx4x4
	}
	if plane == 0 {
		return int(maxTxfmSizeForBs[bs][0])
	}

	return int(maxTxfmSizeForBs[bs][e.layout])
}

func (e *encoder) txtpFor(tx, plane, mode int) int {
	if e.lossless {
		return whtWht
	}
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

func shortlist(cost *[nIntraPredModes]float64, out []int, n int) []int {
	out = out[:0]
	for range n {
		best, bestCost := -1, math.Inf(1)
		for i, c := range cost {
			if c < bestCost {
				best, bestCost = i, c
			}
		}
		if best < 0 || math.IsInf(bestCost, 1) {
			break
		}
		cost[best] = math.Inf(1)
		out = append(out, intraModes[best])
	}

	return out
}

func directional(mode int) bool {
	return mode >= vertPred && mode <= vertLeftPred
}

func (e *encoder) modeBits(mode, delta int, cdf []uint16) float64 {
	bits := cdfBits(cdf, mode)
	if directional(mode) {
		bits += cdfBits(e.cdf.m.angleDelta[mode-vertPred][:], delta+3)
	}

	return bits
}

func (e *encoder) pickMode(bx, by, bs, edgeFlags int, cdf []uint16) (int, int, float64, float64) {
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	tx := e.txOf(bs, 0)
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	bx4, by4 := bx&31, by&31
	ab := &e.a[bx>>5]
	smFl := smFlag(ab, bx4) | smFlag(&e.l, by4) | int(e.seq.intraEdgeFilter)<<10
	ef := lumaTxEdge(edgeFlags, w4, h4, 0, 0, 0, 0, int(tDim.w), int(tDim.h))
	off := 4 * (by*e.stride + bx)

	pred := func(m, delta int) {
		e.predIntra(e.recon, off, e.stride, bx, by, e.bw, e.bh,
			int(tDim.w), int(tDim.h), m, delta, ef, smFl,
			4*e.bw-4*bx, 4*e.bh-4*by)
	}

	cheap := func(m, delta int) float64 {
		pred(m, delta)
		residual(e.res[:], e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
			e.cfg.Width, e.cfg.Height, 4*bx, 4*by, tw, th)

		return float64(satd(e.res[:], tw, th)) + e.lambda*e.modeBits(m, delta, cdf)
	}

	cc := coefCtx{a: ab.lcoef[:], l: e.l.lcoef[:]}
	trial := func(m, delta int) (float64, float64) {
		pred(m, delta)
		bits := e.lambda * e.modeBits(m, delta, cdf)

		eob := e.quantTx(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
			e.cfg.Width, e.cfg.Height, 4*bx, 4*by, tx, e.txtpFor(tx, 0, m))
		skip := e.predSSE + bits
		rate := e.coefBits(&cc, bx4, by4, tx, bs, 0, m)
		e.reconTx(e.recon, off, e.stride, tx, e.txtpFor(tx, 0, m), eob)

		return sse(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
			e.cfg.Width, e.cfg.Height, 4*bx, 4*by, tw, th) +
			bits + e.lambda*rate, skip
	}

	var pre [nIntraPredModes]float64
	for i := range pre {
		pre[i] = math.Inf(1)
	}
	for i, m := range intraModes[:e.sp.pre] {
		pre[i] = cheap(m, 0)
	}

	var buf [nIntraPredModes]int
	cands := shortlist(&pre, buf[:], e.sp.modes)

	best, bestDelta := dcPred, 0
	bestCost, bestSkip := math.Inf(1), math.Inf(1)
	for _, m := range cands {
		if c, sk := trial(m, 0); c < bestCost {
			best, bestCost, bestSkip = m, c, sk
		}
	}

	if directional(best) && int(bDim[2])+int(bDim[3]) >= 2 {
		d, dCost := 0, math.Inf(1)
		for delta := -3; delta <= 3; delta++ {
			if delta == 0 {
				continue
			}
			if c := cheap(best, delta); c < dCost {
				d, dCost = delta, c
			}
		}
		if c, sk := trial(best, d); c < bestCost {
			bestDelta, bestCost, bestSkip = d, c, sk
		}
	}

	trial(best, bestDelta)

	return best, bestDelta, bestCost, bestSkip
}

func cflSignBits(cdf *cdfModeContext, u, v int) float64 {
	su, sv := cflSign(u), cflSign(v)
	bits := cdfBits(cdf.cflSign[:], su*3+sv-1)
	if su != 0 {
		bits += cdfBits(cdf.cflAlpha[b2i(su == 2)*3+sv][:], abs(u)-1)
	}
	if sv != 0 {
		bits += cdfBits(cdf.cflAlpha[b2i(sv == 2)*3+su][:], abs(v)-1)
	}

	return bits
}

func cflSign(a int) int {
	switch {
	case a < 0:
		return 1
	case a > 0:
		return 2
	}

	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func (e *encoder) cflAcFor(bx, by, bs int) {
	ssHor, ssVer := e.ssHor, e.ssVer
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	cbw4 := (int(bDim[0]) + ssHor) >> ssHor
	cbh4 := (int(bDim[1]) + ssVer) >> ssVer
	tDim := &txfmDimensions[e.txOf(bs, 1)]

	furthestR := ((cw4 << ssHor) + int(tDim.w) - 1) &^ (int(tDim.w) - 1)
	furthestB := ((ch4 << ssVer) + int(tDim.h) - 1) &^ (int(tDim.h) - 1)
	off := 4*(bx&^ssHor) + 4*(by&^ssVer)*e.stride

	cflAc(e.ac[:], e.recon, off, e.stride,
		cbw4-(furthestR>>ssHor), cbh4-(furthestB>>ssVer),
		cbw4*4, cbh4*4, ssHor, ssVer)
}

func (e *encoder) predCfl(pl, off, cbx, cby, tw4, th4, alpha, maxW, maxH int) {
	angle := 0
	m := prepareIntraEdges(cbx, cbx > 0, cby, cby > 0,
		e.bw>>e.ssHor, e.bh>>e.ssVer, 0, e.reconC[pl], off, e.cstride, nil, 0,
		dcPred, &angle, tw4, th4, false, e.edge[:], encEdgeOff, 255)
	e.dsp.cflPred[m](e.reconC[pl], off, e.cstride, e.edge[:], encEdgeOff,
		tw4*4, th4*4, e.ac[:], int32(alpha), 255)
}

func (e *encoder) pickAlpha(pl, off, cbx, cby, tw4, th4 int, src []uint8,
	px, py, w, h, maxW, maxH int,
) int {
	e.predCfl(pl, off, cbx, cby, tw4, th4, 0, maxW, maxH)

	var num, den float64
	for j := range h {
		sy := min(py+j, e.ch-1)
		for i := range w {
			a := float64(e.ac[j*w+i])
			d := float64(src[sy*e.cfg.SrcUVStride+min(px+i, e.cw-1)]) -
				float64(e.reconC[pl][off+j*e.cstride+i])
			num += a * d
			den += a * a
		}
	}
	if den == 0 {
		return 0
	}

	want := int(math.Round(64 * num / den))
	best, bestSSE := 0, math.Inf(1)
	for _, a := range [3]int{want - 1, want, want + 1} {
		if a < -16 || a > 16 {
			continue
		}
		e.predCfl(pl, off, cbx, cby, tw4, th4, a, maxW, maxH)
		if d := sse(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
			e.cw, e.ch, px, py, w, h); d < bestSSE {
			best, bestSSE = a, d
		}
	}

	return best
}

func (e *encoder) applyCfl(bs, off, cbx, cby, tw4, th4 int, alpha [2]int8, maxW, maxH int) {
	for pl := range 2 {
		if alpha[pl] == 0 {
			e.predIntra(e.reconC[pl], off, e.cstride, cbx, cby,
				e.bw>>e.ssHor, e.bh>>e.ssVer, tw4, th4, dcPred, 0, 0,
				angleUseEdgeFilterFlag, maxW, maxH)

			continue
		}
		e.predCfl(pl, off, cbx, cby, tw4, th4, int(alpha[pl]), maxW, maxH)
	}
}

func (e *encoder) reconCfl(bs, off, cbx, cby, cbx4, cby4, tx int, alpha [2]int8) {
	for pl := range 2 {
		src := e.cfg.SrcU
		if pl == 1 {
			src = e.cfg.SrcV
		}
		txtp := e.txtpFor(tx, 1, cflPred)
		eob := e.quantTx(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
			e.cw, e.ch, 4*cbx, 4*cby, tx, txtp)
		e.reconTx(e.reconC[pl], off, e.cstride, tx, txtp, eob)
	}
}

func (e *encoder) trialCfl(bs, off, cbx, cby, tw4, th4, cbx4, cby4, tx int,
	alpha [2]int8, maxW, maxH int, cdf []uint16,
) (float64, float64) {
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	ab := &e.a[cbx<<e.ssHor>>5]
	c := e.lambda * (cdfBits(cdf, cflPred) +
		cflSignBits(&e.cdf.m, int(alpha[0]), int(alpha[1])))
	skip := c

	e.applyCfl(bs, off, cbx, cby, tw4, th4, alpha, maxW, maxH)
	for pl := range 2 {
		src := e.cfg.SrcU
		if pl == 1 {
			src = e.cfg.SrcV
		}
		cc := coefCtx{a: ab.ccoef[pl][:], l: e.l.ccoef[pl][:]}
		txtp := e.txtpFor(tx, 1, cflPred)
		eob := e.quantTx(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
			e.cw, e.ch, 4*cbx, 4*cby, tx, txtp)
		skip += e.predSSE
		rate := e.coefBits(&cc, cbx4, cby4, tx, bs, 1+pl, cflPred)
		e.reconTx(e.reconC[pl], off, e.cstride, tx, txtp, eob)
		c += sse(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
			e.cw, e.ch, 4*cbx, 4*cby, tw, th) + e.lambda*rate
	}

	return c, skip
}

func (e *encoder) pickUvMode(bx, by, bs, edgeFlags int, cdf []uint16) (int, int, float64, float64) {
	ssHor, ssVer := e.ssHor, e.ssVer
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	cbx4, cby4 := (bx&31)>>ssHor, (by&31)>>ssVer
	cbx, cby := bx>>ssHor, by>>ssVer
	tx := e.txOf(bs, 1)
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	ab := &e.a[bx>>5]
	smFl := smUvFlag(ab, cbx4) | smUvFlag(&e.l, cby4) | angleUseEdgeFilterFlag
	ef := e.chromaTxEdge(edgeFlags, cw4, ch4, 0, 0, 0, 0, int(tDim.w), int(tDim.h))
	off := 4 * (cby*e.cstride + cbx)

	plane := func(pl int) []uint8 {
		if pl == 1 {
			return e.cfg.SrcV
		}

		return e.cfg.SrcU
	}

	pred := func(pl, m, delta int) {
		e.predIntra(e.reconC[pl], off, e.cstride, cbx, cby,
			e.bw>>ssHor, e.bh>>ssVer, int(tDim.w), int(tDim.h), m, delta, ef, smFl,
			(4*e.bw+ssHor-4*bx)>>ssHor, (4*e.bh+ssVer-4*by)>>ssVer)
	}

	cheap := func(m, delta int) float64 {
		c := e.lambda * e.modeBits(m, delta, cdf)
		for pl := range 2 {
			pred(pl, m, delta)
			residual(e.res[:], e.reconC[pl], off, e.cstride, plane(pl), e.cfg.SrcUVStride,
				e.cw, e.ch, 4*cbx, 4*cby, tw, th)
			c += float64(satd(e.res[:], tw, th))
		}

		return c
	}

	trial := func(m, delta int) (float64, float64) {
		txtp := e.txtpFor(tx, 1, m)
		c := e.lambda * e.modeBits(m, delta, cdf)
		skip := c
		for pl := range 2 {
			src := plane(pl)
			cc := coefCtx{a: ab.ccoef[pl][:], l: e.l.ccoef[pl][:]}
			pred(pl, m, delta)
			eob := e.quantTx(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
				e.cw, e.ch, 4*cbx, 4*cby, tx, txtp)
			skip += e.predSSE
			rate := e.coefBits(&cc, cbx4, cby4, tx, bs, 1+pl, m)
			e.reconTx(e.reconC[pl], off, e.cstride, tx, txtp, eob)
			c += sse(e.reconC[pl], off, e.cstride, src, e.cfg.SrcUVStride,
				e.cw, e.ch, 4*cbx, 4*cby, tw, th) +
				e.lambda*rate
		}

		return c, skip
	}

	var pre [nIntraPredModes]float64
	for i := range pre {
		pre[i] = math.Inf(1)
	}
	for i, m := range intraModes[:e.sp.pre] {
		pre[i] = cheap(m, 0)
	}

	var buf [nIntraPredModes]int
	cands := shortlist(&pre, buf[:], e.sp.modes)

	best, bestDelta := dcPred, 0
	bestCost, bestSkip := math.Inf(1), math.Inf(1)
	for _, m := range cands {
		if c, sk := trial(m, 0); c < bestCost {
			best, bestCost, bestSkip = m, c, sk
		}
	}

	if directional(best) && int(bDim[2])+int(bDim[3]) >= 2 {
		d, dCost := 0, math.Inf(1)
		for delta := -3; delta <= 3; delta++ {
			if delta == 0 {
				continue
			}
			if c := cheap(best, delta); c < dCost {
				d, dCost = delta, c
			}
		}
		if c, sk := trial(best, d); c < bestCost {
			bestDelta, bestCost, bestSkip = d, c, sk
		}
	}

	maxW := (4*e.bw + ssHor - 4*bx) >> ssHor
	maxH := (4*e.bh + ssVer - 4*by) >> ssVer
	alpha := [2]int8{}
	if e.cflAllowed(bs) != 0 {
		e.cflAcFor(bx, by, bs)
		for pl := range 2 {
			alpha[pl] = int8(e.pickAlpha(pl, off, cbx, cby, int(tDim.w), int(tDim.h),
				plane(pl), 4*cbx, 4*cby, tw, th, maxW, maxH))
		}
		if alpha[0] != 0 || alpha[1] != 0 {
			if c, sk := e.trialCfl(bs, off, cbx, cby, int(tDim.w), int(tDim.h),
				cbx4, cby4, tx, alpha, maxW, maxH, cdf); c < bestCost {
				best, bestDelta, bestCost, bestSkip = cflPred, 0, c, sk
			}
		}
	}
	e.cfla[sbIdx(bx, by)] = alpha

	if best == cflPred {
		e.applyCfl(bs, off, cbx, cby, int(tDim.w), int(tDim.h), alpha, maxW, maxH)
		e.reconCfl(bs, off, cbx, cby, cbx4, cby4, tx, alpha)

		return best, 0, bestCost, bestSkip
	}

	trial(best, bestDelta)

	return best, bestDelta, bestCost, bestSkip
}

func (e *encoder) coefBits(cc *coefCtx, aOff, lOff, tx, bs, plane, mode int) float64 {
	e.msac.count = true
	e.msac.bits = 0
	e.writeCoefs(cc, aOff, lOff, tx, bs, plane, uint8(mode), e.lv[:], e.levels[:])
	e.msac.count = false

	return e.msac.bits
}

func (e *encoder) quantTx(dst []uint8, off, stride int, src []uint8, sstride, sw, sh,
	px, py, tx, txtp int,
) int {
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	shift := quantShift(tx)

	e.predSSE = residual(e.res[:], dst, off, stride, src, sstride, sw, sh, px, py, tw, th)
	fwdTxfm(e.coeff[:], e.res[:], tw, tx, txtp, e.fwd)

	cw, ch := min(tw, 32), min(th, 32)
	n := cw * ch
	e.lv[0] = quantOne(e.coeff[0], &e.qr[0], shift)
	for i := 1; i < n; i++ {
		e.lv[i] = quantOne(e.coeff[i], &e.qr[1], shift)
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
	itxfmAdd(e.dsp, dst, off, stride, e.cf[:], e.itx[:], eob, tx, txtp, 255)
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

func (e *encoder) reconBlock(bx, by, bs, edgeFlags, mode, delta int, skip bool) {
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	bx4, by4 := bx&31, by&31
	tx := e.txOf(bs, 0)
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
						int(tDim.w), int(tDim.h), mode, delta, ef, smFl,
						4*e.bw-4*(bx+x), 4*e.bh-4*(by+y))
					if skip {
						memset(cc.a[bx4+x:bx4+x+int(tDim.w)], 0x40)
						memset(cc.l[by4+y:by4+y+int(tDim.h)], 0x40)

						continue
					}
					e.codeTx(e.recon, off, e.stride, e.cfg.Src, e.cfg.SrcStride,
						e.cfg.Width, e.cfg.Height, 4*(bx+x), 4*(by+y),
						tx, bs, 0, mode, &cc, bx4+x, by4+y,
						min(int(tDim.w), e.bw-bx-x), min(int(tDim.h), e.bh-by-y))
				}
			}
		}
	}
}

func (e *encoder) reconChroma(bx, by, bs, edgeFlags, mode, delta int, skip bool) {
	ssHor, ssVer := e.ssHor, e.ssVer
	alpha := e.cfla[sbIdx(bx, by)]
	if mode == cflPred {
		e.cflAcFor(bx, by, bs)
	}
	bDim := &blockDimensions[bs]
	w4, h4 := min(int(bDim[0]), e.bw-bx), min(int(bDim[1]), e.bh-by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	cbx4, cby4 := (bx&31)>>ssHor, (by&31)>>ssVer
	cbx, cby := bx>>ssHor, by>>ssVer
	tx := e.txOf(bs, 1)
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

						maxW := (4*e.bw + ssHor - 4*(lx&^ssHor)) >> ssHor
						maxH := (4*e.bh + ssVer - 4*(ly&^ssVer)) >> ssVer
						if mode == cflPred && alpha[pl] != 0 {
							e.predCfl(pl, off, cbx+x, cby+y, int(tDim.w), int(tDim.h),
								int(alpha[pl]), maxW, maxH)
						} else {
							um := mode
							if um == cflPred {
								um = dcPred
							}
							e.predIntra(dst, off, e.cstride, cbx+x, cby+y,
								e.bw>>ssHor, e.bh>>ssVer, int(tDim.w), int(tDim.h),
								um, delta, ef, smFl, maxW, maxH)
						}
						if skip {
							memset(cc.a[cbx4+x:cbx4+x+int(tDim.w)], 0x40)
							memset(cc.l[cby4+y:cby4+y+int(tDim.h)], 0x40)

							continue
						}
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
