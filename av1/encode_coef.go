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
	for {
		if tok < 3 {
			e.symbolAdapt(cdf[:], 3, tok)

			return
		}
		e.symbolAdapt(cdf[:], 3, 3)
		tok -= 3
	}
}

type coefCtx struct {
	a []uint8
	l []uint8
}

// writeDCOnly writes one transform block carrying a single DC level.
func (e *encoder) writeDCOnly(cc *coefCtx, aOff, lOff, tx, bs, plane int,
	yMode uint8, level int32,
) uint8 {
	tDim := &txfmDimensions[tx]
	chroma := b2i(plane != 0)

	sctx := getSkipCtx(tDim, bs, cc.a, aOff, cc.l, lOff, chroma, e.layout)
	if level == 0 {
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

	slw, slh := min(int(tDim.lw), tx32x32), min(int(tDim.lh), tx32x32)
	sz := slw + slh
	e.writeEobPt(sz, chroma, 0)

	sign := uint32(0)
	if level < 0 {
		sign = 1
		level = -level
	}
	tok := uint32(level)

	eobCdf := &e.cdf.coef.eobBaseTok[tDim.ctx][chroma]
	hiCdf := &e.cdf.coef.brTok[min(int(tDim.ctx), 3)][chroma]

	if tok < 3 {
		e.msac.symbolAdapt(eobCdf[0][:], 2, tok-1)
	} else {
		e.msac.symbolAdapt(eobCdf[0][:], 2, 2)
		e.msac.hiTok(&hiCdf[0], min(tok, 15)-3)
	}

	dcSignCtx := getDcSignCtx(tx, cc.a, aOff, cc.l, lOff)
	e.msac.boolAdapt(e.cdf.coef.dcSign[chroma][dcSignCtx][:], sign)

	if tok >= 15 {
		e.msac.golomb(tok - 15)
	}

	return uint8(min(tok, 63)) | uint8((sign-1)&(2<<6))
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
