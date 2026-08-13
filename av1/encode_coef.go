package av1

func (e *msacEncoder) golomb(v uint32) {
	x := v + 1
	n := 0
	for x>>uint(n+1) != 0 {
		n++
	}
	for range n {
		e.boolEqui(0)
	}
	e.boolEqui(1)
	for i := n - 1; i >= 0; i-- {
		e.boolEqui(x >> uint(i) & 1)
	}
}

func (e *msacEncoder) hiTok(cdf *[4]uint16, tok uint32) {
	for range 3 {
		if tok < 3 {
			e.symbolAdapt(cdf[:], 3, tok)

			return
		}
		e.symbolAdapt(cdf[:], 3, 3)
		tok -= 3
	}
	e.symbolAdapt(cdf[:], 3, tok)
}

type coefCtx struct {
	a []uint8
	l []uint8
}

func (e *encoder) writeEobPt(sz, chroma, pt int) {
	c := &e.cdf.coef
	switch sz {
	case 0:
		e.msac.symbolAdapt(c.eobBin16[chroma][0][:], 4+0, uint32(pt))
	case 1:
		e.msac.symbolAdapt(c.eobBin32[chroma][0][:], 4+1, uint32(pt))
	case 2:
		e.msac.symbolAdapt(c.eobBin64[chroma][0][:], 4+2, uint32(pt))
	case 3:
		e.msac.symbolAdapt(c.eobBin128[chroma][0][:], 4+3, uint32(pt))
	case 4:
		e.msac.symbolAdapt(c.eobBin256[chroma][0][:], 4+4, uint32(pt))
	case 5:
		e.msac.symbolAdapt(c.eobBin512[chroma][:], 4+5, uint32(pt))
	case 6:
		e.msac.symbolAdapt(c.eobBin1024[chroma][:], 4+6, uint32(pt))
	}
}

// writeCoefs writes one transform block of quantised levels, in the order the
// decoder reads them: end of block, then levels back down the scan, then the
// signs forward.
func (e *encoder) writeCoefs(cc *coefCtx, aOff, lOff, tx, bs, plane int,
	yMode uint8, lv []int32, levels []uint8,
) uint8 {
	tDim := &txfmDimensions[tx]
	chroma := b2i(plane != 0)
	slw, slh := min(int(tDim.lw), tx32x32), min(int(tDim.lh), tx32x32)
	sz := slw + slh
	sw, sh := 4<<slw, 4<<slh
	scan := scans[tx]

	eob := -1
	for i := sw*sh - 1; i >= 0; i-- {
		if lv[scan[i]] != 0 {
			eob = i

			break
		}
	}

	sctx := getSkipCtx(tDim, bs, cc.a, aOff, cc.l, lOff, chroma, e.layout)
	if eob < 0 {
		e.msac.boolAdapt(e.cdf.coef.skip[tDim.ctx][sctx][:], 1)

		return 0x40
	}
	e.msac.boolAdapt(e.cdf.coef.skip[tDim.ctx][sctx][:], 0)

	if chroma == 0 && int(tDim.max)+1 < tx64x64 && e.cfg.QIndex != 0 {
		if tDim.min == tx16x16 {
			e.msac.symbolAdapt(e.cdf.m.txtpIntra2[tDim.min][yMode][:], 4, 1)
		} else {
			e.msac.symbolAdapt(e.cdf.m.txtpIntra1[tDim.min][yMode][:], 6, 1)
		}
	}

	pt := 0
	if eob > 0 {
		pt = int(ulog2(uint32(eob))) + 1
	}
	e.writeEobPt(sz, chroma, pt)
	if pt > 1 {
		n := pt - 2
		e.msac.boolAdapt(e.cdf.coef.eobHiBit[tDim.ctx][chroma][n][:],
			uint32(eob>>uint(n))&1)
		e.msac.bools(uint32(eob)&(1<<uint(n)-1), n)
	}

	loCdf := &e.cdf.coef.baseTok[tDim.ctx][chroma]
	eobCdf := &e.cdf.coef.eobBaseTok[tDim.ctx][chroma]
	hiCdf := &e.cdf.coef.brTok[min(int(tDim.ctx), 3)][chroma]
	loCtxOff := loCtxOffFromScan[tx]
	stride := 4 << slh

	clear(levels[:stride*(sw+2)])

	abs := func(v int32) uint32 {
		if v < 0 {
			return uint32(-v)
		}

		return uint32(v)
	}

	rc := uint32(scan[eob])
	tok := min(abs(lv[rc]), 15)
	if eob == 0 {
		e.msac.symbolAdapt(eobCdf[0][:], 2, min(tok, 3)-1)
		if tok >= 3 {
			e.msac.hiTok(&hiCdf[0], tok-3)
		}
		sign := b2u(lv[0] < 0)
		dcSignCtx := getDcSignCtx(tx, cc.a, aOff, cc.l, lOff)
		e.msac.boolAdapt(e.cdf.coef.dcSign[chroma][dcSignCtx][:], sign)
		full := abs(lv[0])
		if tok == 15 {
			e.msac.golomb(full - 15)
		}

		return uint8(min(full, 63)) | uint8((sign-1)&(2<<6))
	}

	ctx := uint32(1 + b2i(eob > 2<<sz) + b2i(eob > 4<<sz))
	e.msac.symbolAdapt(eobCdf[ctx][:], 2, min(tok, 3)-1)
	if tok >= 3 {
		x, y := rc>>uint(slh+2), rc&uint32(4<<slh-1)
		c := uint32(7)
		if x|y > 1 {
			c = 14
		}
		e.msac.hiTok(&hiCdf[c], tok-3)
		levels[rc] = uint8(tok + 3<<6)
	} else {
		levels[rc] = uint8(tok * 0x41)
	}

	for i := eob - 1; i > 0; i-- {
		rcI := uint32(scan[i])
		ctx, mag := getLoCtx2D(levels, int(rcI), uint32(loCtxOff[i]), stride)
		t := min(abs(lv[rcI]), 15)
		e.msac.symbolAdapt(loCdf[ctx][:], 3, min(t, 3))
		if t >= 3 {
			mag &= 63
			base := uint32(7)
			if (rcI>>uint(slh+2))|(rcI&uint32(4<<slh-1)) > 1 {
				base = 14
			}
			if mag > 12 {
				ctx = base + 6
			} else {
				ctx = base + (mag+1)>>1
			}
			e.msac.hiTok(&hiCdf[ctx], t-3)
			levels[rcI] = uint8(t + 3<<6)
		} else {
			levels[rcI] = uint8(t * 0x41)
		}
	}

	dcTok := min(abs(lv[0]), 15)
	e.msac.symbolAdapt(loCdf[0][:], 3, min(dcTok, 3))
	if dcTok >= 3 {
		mag := (uint32(levels[1]) + uint32(levels[stride]) + uint32(levels[stride+1])) & 63
		dctx := uint32(6)
		if mag <= 12 {
			dctx = (mag + 1) >> 1
		}
		e.msac.hiTok(&hiCdf[dctx], dcTok-3)
	}

	culLevel := uint32(0)
	dcSignLevel := uint32(0)
	if dcTok != 0 {
		sign := b2u(lv[0] < 0)
		dcSignCtx := getDcSignCtx(tx, cc.a, aOff, cc.l, lOff)
		e.msac.boolAdapt(e.cdf.coef.dcSign[chroma][dcSignCtx][:], sign)
		dcSignLevel = (sign - 1) & (2 << 6)
		full := abs(lv[0])
		if dcTok == 15 {
			e.msac.golomb(full - 15)
		}
		culLevel = min(full, 0xfffff)
	} else {
		dcSignLevel = 1 << 6
	}

	for i := 1; i <= eob; i++ {
		rcI := uint32(scan[i])
		if lv[rcI] == 0 {
			continue
		}
		e.msac.boolEqui(b2u(lv[rcI] < 0))
		full := abs(lv[rcI])
		if full >= 15 {
			e.msac.golomb(full - 15)
		}
		culLevel += full
	}

	return uint8(min(culLevel, 63)) | uint8(dcSignLevel)
}
