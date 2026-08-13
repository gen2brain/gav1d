package av1

const encEdgeOff = 128

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

func (e *encoder) codeTx(dst []uint8, off, stride int, src []uint8, sstride, sw, sh,
	px, py, tx, bs, plane, mode int, cc *coefCtx, aOff, lOff, aw, lh int,
) {
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	shift := quantShift(tx)

	for j := range th {
		sy := min(py+j, sh-1)
		row := j * tw
		for i := range tw {
			sx := min(px+i, sw-1)
			e.res[row+i] = int32(src[sy*sstride+sx]) - int32(dst[off+j*stride+i])
		}
	}

	fwdTxfm(e.coeff[:], e.res[:], tw, tx, dctDct)
	cw, ch := min(tw, 32), min(th, 32)
	for i := range cw * ch {
		d := e.dq[1]
		if i == 0 {
			d = e.dq[0]
		}
		e.lv[i] = quantOne(e.coeff[i], uint32(d), shift)
	}

	ctx := e.writeCoefs(cc, aOff, lOff, tx, bs, plane, uint8(mode), e.lv[:], e.levels[:])
	memset(cc.a[aOff:aOff+aw], ctx)
	memset(cc.l[lOff:lOff+lh], ctx)

	eob := -1
	scan := scans[tx]
	for i := cw*ch - 1; i >= 0; i-- {
		if e.lv[scan[i]] != 0 {
			eob = i

			break
		}
	}
	if eob < 0 {
		return
	}

	clear(e.cf[:cw*ch])
	for i := range cw * ch {
		if e.lv[i] == 0 {
			continue
		}
		dqv := uint32(e.dq[1])
		if i == 0 {
			dqv = uint32(e.dq[0])
		}
		d := min(int32(uint32(abs32(e.lv[i]))*dqv>>uint(shift)), 1<<15-1)
		if e.lv[i] < 0 {
			d = -d
		}
		e.cf[i] = int16(d)
	}
	invTxfmAdd(e.dsp, dst, off, stride, e.cf[:], e.itx[:], eob, tx, dctDct, 255)
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
		subH4 := min(h4, 16+initY)
		for initX := 0; initX < w4; initX += 16 {
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
			subW4 := min(w4, initX+16)

			for y := initY; y < subH4; y += int(tDim.h) {
				for x := initX; x < subW4; x += int(tDim.w) {
					ef := 0
					if !((y > initY || sbHasTr == 0) && x+int(tDim.w) >= subW4) {
						ef |= edgeI444TopHasRight
					}
					if !(x > initX || (sbHasBl == 0 && y+int(tDim.h) >= subH4)) {
						ef |= edgeI444LeftHasBottom
					}

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
		subCh4 := min(ch4, (initY+16)>>ssVer)
		for initX := 0; initX < w4; initX += 16 {
			uvSbHasTr := 0
			switch {
			case (initX+16)>>ssHor < cw4:
				uvSbHasTr = 1
			case initY != 0:
			default:
				uvSbHasTr = edgeFlags & (edgeI420TopHasRight >> (e.layout - 1))
			}
			uvSbHasBl := 0
			switch {
			case initX != 0:
			case (initY+16)>>ssVer < ch4:
				uvSbHasBl = 1
			default:
				uvSbHasBl = edgeFlags & (edgeI420LeftHasBottom >> (e.layout - 1))
			}
			subCw4 := min(cw4, (initX+16)>>ssHor)

			for pl := range 2 {
				src := e.cfg.SrcU
				if pl == 1 {
					src = e.cfg.SrcV
				}
				dst := e.reconC[pl]
				cc := coefCtx{a: ab.ccoef[pl][:], l: e.l.ccoef[pl][:]}

				for y := initY >> ssVer; y < subCh4; y += int(tDim.h) {
					for x := initX >> ssHor; x < subCw4; x += int(tDim.w) {
						ef := 0
						if !((y > initY>>ssVer || uvSbHasTr == 0) && x+int(tDim.w) >= subCw4) {
							ef |= edgeI444TopHasRight
						}
						if !(x > initX>>ssHor || (uvSbHasBl == 0 && y+int(tDim.h) >= subCh4)) {
							ef |= edgeI444LeftHasBottom
						}

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
