package av1

func recMc[P pixel, C coef](t *taskContext[P, C], dst8 []P, dst8Off int,
	dst16 []int16, dstStride, bw4, bh4, bx, by, pl int, m mv,
	refp *refPicture[P], refidx, filter2d int,
) error {
	f := t.f
	ssVer := b2i(pl != 0 && f.layout == pixelLayoutI420)
	ssHor := b2i(pl != 0 && f.layout != pixelLayoutI444)
	hMul, vMul := 4>>ssHor, 4>>ssVer
	mvx, mvy := int(m.x), int(m.y)
	mx := mvx & (15 >> (1 - ssHor))
	my := mvy & (15 >> (1 - ssVer))
	refStride := refp.stride[b2i(pl != 0)]

	var ref []P
	var refOff int

	if refp.w == f.frameHdr.width[0] && refp.h == f.frameHdr.height {
		dx := bx*hMul + mvx>>(3+ssHor)
		dy := by*vMul + mvy>>(3+ssVer)

		var w, h int
		if !samePlane(refp.planes[0], f.cur[0]) {
			w = (refp.w + ssHor) >> ssHor
			h = (refp.h + ssVer) >> ssVer
		} else {
			w = f.bw * 4 >> ssHor
			h = f.bh * 4 >> ssVer
		}

		if err := f.waitRef(refp, (dy+bh4*vMul+b2i(my != 0)*4)<<ssVer); err != nil {
			return err
		}

		if dx < b2i(mx != 0)*3 || dy < b2i(my != 0)*3 ||
			dx+bw4*hMul+b2i(mx != 0)*4 > w ||
			dy+bh4*vMul+b2i(my != 0)*4 > h {
			buf := t.scratch.emuEdge[:]
			emuEdge(bw4*hMul+b2i(mx != 0)*7, bh4*vMul+b2i(my != 0)*7, w, h,
				dx-b2i(mx != 0)*3, dy-b2i(my != 0)*3, buf, 0, 192,
				refp.planes[pl], 0, refStride)
			ref = buf
			refOff = 192*b2i(my != 0)*3 + b2i(mx != 0)*3
			refStride = 192
		} else {
			ref = refp.planes[pl]
			refOff = refStride*dy + dx
		}

		if dst8 != nil {
			mc(f.dsp, t.scratch.mcMid[:], dst8, dst8Off, dstStride, ref, refOff, refStride, bw4*hMul,
				bh4*vMul, mx<<(1-ssHor), my<<(1-ssVer), filter2d, f.bitdepthMax)
		} else {
			mct(f.dsp, t.scratch.mcMid[:], dst16, ref, refOff, refStride, bw4*hMul, bh4*vMul,
				mx<<(1-ssHor), my<<(1-ssVer), filter2d, f.bitdepthMax)
		}

		return nil
	}

	origPosY := (by * vMul << 4) + mvy*(1<<(1-ssVer))
	origPosX := (bx * hMul << 4) + mvx*(1<<(1-ssHor))
	posX := scaleMv(origPosX, f.svc[refidx][0].scale)
	posY := scaleMv(origPosY, f.svc[refidx][1].scale)

	left, top := posX>>10, posY>>10
	right := ((posX + (bw4*hMul-1)*f.svc[refidx][0].step) >> 10) + 1
	bottom := ((posY + (bh4*vMul-1)*f.svc[refidx][1].step) >> 10) + 1

	if err := f.waitRef(refp, (bottom+4)<<ssVer); err != nil {
		return err
	}

	w := (refp.w + ssHor) >> ssHor
	h := (refp.h + ssVer) >> ssVer
	if left < 3 || top < 3 || right+4 > w || bottom+4 > h {
		buf := t.scratch.emuEdge[:]
		emuEdge(right-left+7, bottom-top+7, w, h, left-3, top-3,
			buf, 0, 320, refp.planes[pl], 0, refStride)
		ref = buf
		refOff = 320*3 + 3
		refStride = 320
	} else {
		ref = refp.planes[pl]
		refOff = refStride*top + left
	}

	if dst8 != nil {
		mcScaled(dst8, dst8Off, dstStride, ref, refOff, refStride, bw4*hMul,
			bh4*vMul, posX&0x3ff, posY&0x3ff, f.svc[refidx][0].step,
			f.svc[refidx][1].step, filter2d, f.bitdepthMax)
	} else {
		mctScaled(dst16, ref, refOff, refStride, bw4*hMul, bh4*vMul,
			posX&0x3ff, posY&0x3ff, f.svc[refidx][0].step,
			f.svc[refidx][1].step, filter2d, f.bitdepthMax)
	}

	return nil
}

func samePlane[P pixel](a, b []P) bool {
	return len(a) > 0 && len(b) > 0 && &a[0] == &b[0]
}

func scaleMv(val, scale int) int {
	tmp := int64(val)*int64(scale) + int64(scale-0x4000)*8

	return int(applySign64(int32((abs64(tmp)+128)>>8), tmp)) + 32
}

func recObmc[P pixel, C coef](t *taskContext[P, C], dst []P, dstOff, dstStride int,
	bDim *[4]uint8, pl, bx4, by4, w4, h4 int,
) error {
	f := t.f
	lap := t.scratch.lap[:]
	ssVer := b2i(pl != 0 && f.layout == pixelLayoutI420)
	ssHor := b2i(pl != 0 && f.layout != pixelLayoutI444)
	hMul, vMul := 4>>ssHor, 4>>ssVer

	if t.by > t.ts.tiling.rowStart &&
		(pl == 0 || int(bDim[0])*hMul+int(bDim[1])*vMul >= 16) {
		for i, x := 0, 0; x < w4 && i < min(int(bDim[2]), 4); {
			aR := &t.rt.r[t.rowOff(-1)+t.bx+x+1]
			aBDim := &blockDimensions[aR.bs]
			step4 := clip(int(aBDim[0]), 2, 16)

			if aR.ref[0] > 0 {
				ow4 := min(step4, int(bDim[0]))
				oh4 := min(int(bDim[1]), 16) >> 1
				f2d := int(filter2d[t.a.filter[1][bx4+x+1]][t.a.filter[0][bx4+x+1]])
				if err := recMc(t, lap, 0, nil, ow4*hMul, ow4, (oh4*3+3)>>2,
					t.bx+x, t.by, pl, aR.mv[0], f.refp[aR.ref[0]-1],
					int(aR.ref[0])-1, f2d); err != nil {
					return err
				}
				mcBlendH(dst, dstOff+x*hMul, dstStride, lap, hMul*ow4, vMul*oh4)
				i++
			}
			x += step4
		}
	}

	if t.bx > t.ts.tiling.colStart {
		for i, y := 0, 0; y < h4 && i < min(int(bDim[3]), 4); {
			lR := &t.rt.r[t.rowOff(y+1)+t.bx-1]
			lBDim := &blockDimensions[lR.bs]
			step4 := clip(int(lBDim[1]), 2, 16)

			if lR.ref[0] > 0 {
				ow4 := min(int(bDim[0]), 16) >> 1
				oh4 := min(step4, int(bDim[1]))
				f2d := int(filter2d[t.l.filter[1][by4+y+1]][t.l.filter[0][by4+y+1]])
				if err := recMc(t, lap, 0, nil, hMul*ow4, ow4, oh4,
					t.bx, t.by+y, pl, lR.mv[0], f.refp[lR.ref[0]-1],
					int(lR.ref[0])-1, f2d); err != nil {
					return err
				}
				mcBlendV(dst, dstOff+y*vMul*dstStride, dstStride, lap,
					hMul*ow4, vMul*oh4)
				i++
			}
			y += step4
		}
	}

	return nil
}

func recWarpAffine[P pixel, C coef](t *taskContext[P, C], dst8 []P, dst8Off int,
	dst16 []int16, dstride int, bDim *[4]uint8, pl int, refp *refPicture[P],
	wmp *warpedMotionParams,
) error {
	f := t.f
	ssVer := b2i(pl != 0 && f.layout == pixelLayoutI420)
	ssHor := b2i(pl != 0 && f.layout != pixelLayoutI444)
	hMul, vMul := 4>>ssHor, 4>>ssVer
	mat := &wmp.matrix
	width := (refp.w + ssHor) >> ssHor
	height := (refp.h + ssVer) >> ssVer

	dst16Off := 0
	for y := 0; y < int(bDim[1])*vMul; y += 8 {
		srcY := t.by*4 + ((y + 4) << ssVer)
		mat3y := int64(mat[3])*int64(srcY) + int64(mat[0])
		mat5y := int64(mat[5])*int64(srcY) + int64(mat[1])

		for x := 0; x < int(bDim[0])*hMul; x += 8 {
			srcX := t.bx*4 + ((x + 4) << ssHor)
			mvx := (int64(mat[2])*int64(srcX) + mat3y) >> ssHor
			mvy := (int64(mat[4])*int64(srcX) + mat5y) >> ssVer

			dx := int(int32(mvx>>16)) - 4
			mx := (int(int32(mvx))&0xffff - int(wmp.abcd[0])*4 -
				int(wmp.abcd[1])*7) &^ 0x3f
			dy := int(int32(mvy>>16)) - 4
			my := (int(int32(mvy))&0xffff - int(wmp.abcd[2])*4 -
				int(wmp.abcd[3])*4) &^ 0x3f

			var ref []P
			var refOff int
			refStride := refp.stride[b2i(pl != 0)]

			if err := f.waitRef(refp, (dy+12)<<ssVer); err != nil {
				return err
			}

			if dx < 3 || dx+8+4 > width || dy < 3 || dy+8+4 > height {
				buf := t.scratch.emuEdge[:]
				emuEdge(15, 15, width, height, dx-3, dy-3, buf, 0, 32,
					refp.planes[pl], 0, refStride)
				ref = buf
				refOff = 32*3 + 3
				refStride = 32
			} else {
				ref = refp.planes[pl]
				refOff = refStride*dy + dx
			}
			if dst16 != nil {
				warpAffine8x8t(dst16[dst16Off+x:], dstride, ref, refOff,
					refStride, &wmp.abcd, mx, my, f.bitdepthMax)
			} else {
				warpAffine8x8(dst8, dst8Off+x, dstride, ref, refOff, refStride,
					&wmp.abcd, mx, my, f.bitdepthMax)
			}
		}
		if dst8 != nil {
			dst8Off += 8 * dstride
		} else {
			dst16Off += 8 * dstride
		}
	}

	return nil
}

func readCoefTree[P pixel, C coef](t *taskContext[P, C], bs int, b *av1Block,
	ytx, depth int, txSplit *[2]uint16, xOff, yOff int, dst []P, dstOff int,
) {
	f := t.f
	tDim := &txfmDimensions[ytx]
	txw, txh := int(tDim.w), int(tDim.h)

	if depth < 2 && txSplit[depth] != 0 &&
		txSplit[depth]&(1<<(yOff*4+xOff)) != 0 {
		sub := int(tDim.sub)
		subTDim := &txfmDimensions[sub]
		txsw, txsh := int(subTDim.w), int(subTDim.h)

		readCoefTree(t, bs, b, sub, depth+1, txSplit, xOff*2+0, yOff*2+0,
			dst, dstOff)
		t.bx += txsw
		if txw >= txh && t.bx < f.bw {
			readCoefTree(t, bs, b, sub, depth+1, txSplit, xOff*2+1, yOff*2+0,
				dst, dstOff+4*txsw)
		}
		t.bx -= txsw
		t.by += txsh
		if txh >= txw && t.by < f.bh {
			dstOff += 4 * txsh * f.stride[0]
			readCoefTree(t, bs, b, sub, depth+1, txSplit, xOff*2+0, yOff*2+1,
				dst, dstOff)
			t.bx += txsw
			if txw >= txh && t.bx < f.bw {
				readCoefTree(t, bs, b, sub, depth+1, txSplit, xOff*2+1,
					yOff*2+1, dst, dstOff+4*txsw)
			}
			t.bx -= txsw
		}
		t.by -= txsh

		return
	}

	bx4, by4 := t.bx&31, t.by&31
	cf := t.cf

	eob, txtp, cfCtx := decodeCoefs(t, t.a.lcoef[:], bx4, t.l.lcoef[:], by4,
		ytx, bs, b, 0, 0, cf, 0)
	memset(t.a.lcoef[bx4:bx4+min(txw, f.bw-t.bx)], cfCtx)
	memset(t.l.lcoef[by4:by4+min(txh, f.bh-t.by)], cfCtx)

	mapOff := by4*32 + bx4
	for range txh {
		memset(t.scratch.txtpMap[mapOff:mapOff+txw], uint8(txtp))
		mapOff += 32
	}

	if eob >= 0 {
		itxfmAdd(f.dsp, dst, dstOff, f.stride[0], cf, t.scratch.itx[:], eob, ytx, txtp, f.bitdepthMax)
	}
}

func iiMask(b *av1Block, c, bs int) []uint8 {
	if b.interintraType == interIntraBlend {
		return iiMasks[c][bs-bs32x32][b.interintraMode]
	}

	return wedgeMasks[c][bs-bs32x32][0][b.wedgeIdx]
}

func (t *taskContext[P, C]) interIntraPred(b *av1Block, bs, pl int,
	dst []P, dstOff, stride, w4, h4, cIdx int,
) {
	f := t.f
	ts := t.ts
	ssVer := b2i(pl != 0 && f.layout == pixelLayoutI420)
	ssHor := b2i(pl != 0 && f.layout != pixelLayoutI444)

	m := int(b.interintraMode)
	if m == iiSmoothPred {
		m = smoothPred
	}
	tmp := t.scratch.interintra[:]
	angle := 0

	var topSbEdge []P
	topSbEdgeOff := 0
	if t.by&(f.sbStep-1) == 0 {
		topSbEdge = f.ipredEdge[pl]
		sby := t.by >> f.sbShift
		topSbEdgeOff = f.sb128w * 128 * (sby - 1)
	}

	edge := t.scratch.edge[:]
	m = prepareIntraEdges(t.bx>>ssHor, (t.bx>>ssHor) > (ts.tiling.colStart>>ssHor),
		t.by>>ssVer, (t.by>>ssVer) > (ts.tiling.rowStart>>ssVer),
		ts.tiling.colEnd>>ssHor, ts.tiling.rowEnd>>ssVer,
		0, dst, dstOff, stride, topSbEdge, topSbEdgeOff, m, &angle,
		w4, h4, false, edge, 32, f.bitdepthMax)

	f.dsp.ipred[m](tmp, 0, w4*4, edge, 32, w4*4, h4*4, 0, 0, 0, f.bitdepthMax)
	mcBlend(dst, dstOff, stride, tmp, w4*4, h4*4, iiMask(b, cIdx, bs))
}

func reconBInter[P pixel, C coef](t *taskContext[P, C], bs int, b *av1Block) error {
	f := t.f
	bx4, by4 := t.bx&31, t.by&31
	ssVer := b2i(f.layout == pixelLayoutI420)
	ssHor := b2i(f.layout != pixelLayoutI444)
	cbx4, cby4 := bx4>>ssHor, by4>>ssVer
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	w4, h4 := min(bw4, f.bw-t.bx), min(bh4, f.bh-t.by)
	hasChroma := f.layout != pixelLayoutI400 &&
		(bw4 > ssHor || t.bx&1 != 0) &&
		(bh4 > ssVer || t.by&1 != 0)
	chrLayoutIdx := 0
	if f.layout != pixelLayoutI400 {
		chrLayoutIdx = pixelLayoutI444 - f.layout
	}

	cbh4, cbw4 := (bh4+ssVer)>>ssVer, (bw4+ssHor)>>ssHor
	dstOff := 4 * (t.by*f.stride[0] + t.bx)
	uvdstoff := 4 * ((t.bx >> ssHor) + (t.by>>ssVer)*f.stride[1])

	switch {
	case isKeyOrIntra(f.frameHdr):
		srp := &refPicture[P]{planes: f.srCur, stride: f.srStride,
			w: f.srWidth, h: f.frameHdr.height}
		if err := recMc(t, f.cur[0], dstOff, nil, f.stride[0], bw4, bh4,
			t.bx, t.by, 0, b.mv[0], srp, 0, filter2dBilinear); err != nil {
			return err
		}
		if hasChroma {
			for pl := 1; pl < 3; pl++ {
				if err := recMc(t, f.cur[pl], uvdstoff, nil, f.stride[1],
					bw4<<b2i(bw4 == ssHor), bh4<<b2i(bh4 == ssVer),
					t.bx & ^ssHor, t.by & ^ssVer, pl, b.mv[0], srp, 0,
					filter2dBilinear); err != nil {
					return err
				}
			}
		}

	case b.compType == compInterNone:
		refp := f.refp[b.ref[0]]
		f2d := int(b.filter2d)
		warped := (b.interMode == globalmv && f.gmvWarpAllowed[b.ref[0]]) ||
			(b.motionMode == mmWarp && t.warpmv.typ > wmTypeTranslation)
		wmp := &f.frameHdr.gmv[b.ref[0]]
		if b.motionMode == mmWarp {
			wmp = &t.warpmv
		}

		if min(bw4, bh4) > 1 && warped {
			if err := recWarpAffine(t, f.cur[0], dstOff, nil, f.stride[0],
				bDim, 0, refp, wmp); err != nil {
				return err
			}
		} else {
			if err := recMc(t, f.cur[0], dstOff, nil, f.stride[0], bw4, bh4,
				t.bx, t.by, 0, b.mv[0], refp, int(b.ref[0]), f2d); err != nil {
				return err
			}
			if b.motionMode == mmObmc {
				if err := recObmc(t, f.cur[0], dstOff, f.stride[0], bDim, 0,
					bx4, by4, w4, h4); err != nil {
					return err
				}
			}
		}
		if b.interintraType != 0 {
			t.interIntraPred(b, bs, 0, f.cur[0], dstOff, f.stride[0], bw4, bh4, 0)
		}

		if hasChroma {
			if err := t.reconInterChroma(b, bs, bDim, refp, f2d, warped, wmp,
				uvdstoff, bx4, by4, cbx4, cby4, cbw4, cbh4, w4, h4,
				ssHor, ssVer, chrLayoutIdx); err != nil {
				return err
			}
		}
		t.tl4x4Filter = uint8(f2d)

	default:
		f2d := int(b.filter2d)
		tmp := &t.scratch.compinter
		segMask := t.scratch.segMask[:]
		var mask []uint8
		jntWeight := 0

		for i := range 2 {
			refp := f.refp[b.ref[i]]
			if b.interMode == globalmvGlobalmv && f.gmvWarpAllowed[b.ref[i]] {
				if err := recWarpAffine(t, nil, 0, tmp[i][:], bw4*4, bDim, 0,
					refp, &f.frameHdr.gmv[b.ref[i]]); err != nil {
					return err
				}
			} else {
				if err := recMc(t, nil, 0, tmp[i][:], 0, bw4, bh4, t.bx, t.by,
					0, b.mv[i], refp, int(b.ref[i]), f2d); err != nil {
					return err
				}
			}
		}

		sign := int(b.maskSign)
		switch b.compType {
		case compInterAvg:
			mcAvg(f.cur[0], dstOff, f.stride[0], tmp[0][:], tmp[1][:],
				bw4*4, bh4*4, f.bitdepthMax)
		case compInterWeightedAvg:
			jntWeight = int(f.jntWeights[b.ref[0]][b.ref[1]])
			mcWAvg(f.cur[0], dstOff, f.stride[0], tmp[0][:], tmp[1][:],
				bw4*4, bh4*4, jntWeight, f.bitdepthMax)
		case compInterSeg:
			mcWMask(f.cur[0], dstOff, f.stride[0], tmp[sign][:], tmp[1-sign][:],
				bw4*4, bh4*4, segMask, sign, b2i(chrLayoutIdx > 0),
				b2i(chrLayoutIdx > 1), f.bitdepthMax)
			mask = segMask
		case compInterWedge:
			mask = wedgeMasks[0][bs-bs32x32][0][b.wedgeIdx]
			mcMask(f.cur[0], dstOff, f.stride[0], tmp[sign][:], tmp[1-sign][:],
				bw4*4, bh4*4, mask, f.bitdepthMax)
			if hasChroma {
				mask = wedgeMasks[chrLayoutIdx][bs-bs32x32][sign][b.wedgeIdx]
			}
		}

		if hasChroma {
			for pl := range 2 {
				for i := range 2 {
					refp := f.refp[b.ref[i]]
					if b.interMode == globalmvGlobalmv && min(cbw4, cbh4) > 1 &&
						f.gmvWarpAllowed[b.ref[i]] {
						if err := recWarpAffine(t, nil, 0, tmp[i][:],
							bw4*4>>ssHor, bDim, 1+pl, refp,
							&f.frameHdr.gmv[b.ref[i]]); err != nil {
							return err
						}
					} else {
						if err := recMc(t, nil, 0, tmp[i][:], 0, bw4, bh4,
							t.bx, t.by, 1+pl, b.mv[i], refp, int(b.ref[i]),
							f2d); err != nil {
							return err
						}
					}
				}
				switch b.compType {
				case compInterAvg:
					mcAvg(f.cur[1+pl], uvdstoff, f.stride[1], tmp[0][:],
						tmp[1][:], bw4*4>>ssHor, bh4*4>>ssVer, f.bitdepthMax)
				case compInterWeightedAvg:
					mcWAvg(f.cur[1+pl], uvdstoff, f.stride[1], tmp[0][:],
						tmp[1][:], bw4*4>>ssHor, bh4*4>>ssVer, jntWeight,
						f.bitdepthMax)
				case compInterWedge, compInterSeg:
					mcMask(f.cur[1+pl], uvdstoff, f.stride[1], tmp[sign][:],
						tmp[1-sign][:], bw4*4>>ssHor, bh4*4>>ssVer, mask,
						f.bitdepthMax)
				}
			}
		}
	}

	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer

	if b.skip != 0 {
		memset(t.a.lcoef[bx4:bx4+bw4], 0x40)
		memset(t.l.lcoef[by4:by4+bh4], 0x40)
		if hasChroma {
			for pl := range 2 {
				memset(t.a.ccoef[pl][cbx4:cbx4+cbw4], 0x40)
				memset(t.l.ccoef[pl][cby4:cby4+cbh4], 0x40)
			}
		}

		return nil
	}

	uvtx := &txfmDimensions[b.uvtx]
	ytx := &txfmDimensions[b.maxYtx]
	txSplit := [2]uint16{uint16(b.txSplit0), b.txSplit1}

	for initY := 0; initY < bh4; initY += 16 {
		for initX := 0; initX < bw4; initX += 16 {
			yOff := b2i(initY != 0)
			yDst := dstOff + f.stride[0]*4*initY
			y := initY
			t.by += initY
			for ; y < min(h4, initY+16); y, yOff = y+int(ytx.h), yOff+1 {
				xOff := b2i(initX != 0)
				x := initX
				t.bx += initX
				for ; x < min(w4, initX+16); x, xOff = x+int(ytx.w), xOff+1 {
					readCoefTree(t, bs, b, int(b.maxYtx), 0, &txSplit,
						xOff, yOff, f.cur[0], yDst+x*4)
					t.bx += int(ytx.w)
				}
				yDst += f.stride[0] * 4 * int(ytx.h)
				t.bx -= x
				t.by += int(ytx.h)
			}
			t.by -= y

			if !hasChroma {
				continue
			}

			for pl := range 2 {
				uvDst := uvdstoff + (f.stride[1] * initY * 4 >> ssVer)
				y = initY >> ssVer
				t.by += initY
				for ; y < min(ch4, (initY+16)>>ssVer); y += int(uvtx.h) {
					x := initX >> ssHor
					t.bx += initX
					for ; x < min(cw4, (initX+16)>>ssHor); x += int(uvtx.w) {
						cf := t.cf
						txtp := int(t.scratch.txtpMap[(by4+(y<<ssVer))*32+
							bx4+(x<<ssHor)])
						eob, txtp, cfCtx := decodeCoefs(t, t.a.ccoef[pl][:],
							cbx4+x, t.l.ccoef[pl][:], cby4+y, int(b.uvtx), bs,
							b, 0, 1+pl, cf, txtp)
						ctw := min(int(uvtx.w), (f.bw-t.bx+ssHor)>>ssHor)
						cth := min(int(uvtx.h), (f.bh-t.by+ssVer)>>ssVer)
						memset(t.a.ccoef[pl][cbx4+x:cbx4+x+ctw], cfCtx)
						memset(t.l.ccoef[pl][cby4+y:cby4+y+cth], cfCtx)
						if eob >= 0 {
							itxfmAdd(f.dsp, f.cur[1+pl], uvDst+4*x, f.stride[1], cf, t.scratch.itx[:],
								eob, int(b.uvtx), txtp, f.bitdepthMax)
						}
						t.bx += int(uvtx.w) << ssHor
					}
					uvDst += f.stride[1] * 4 * int(uvtx.h)
					t.bx -= x << ssHor
					t.by += int(uvtx.h) << ssVer
				}
				t.by -= y << ssVer
			}
		}
	}

	return nil
}

func (t *taskContext[P, C]) reconInterChroma(b *av1Block, bs int, bDim *[4]uint8,
	refp *refPicture[P], f2d int, warped bool, wmp *warpedMotionParams,
	uvdstoff, bx4, by4, cbx4, cby4, cbw4, cbh4, w4, h4, ssHor, ssVer,
	chrLayoutIdx int,
) error {
	f := t.f
	bw4, bh4 := int(bDim[0]), int(bDim[1])

	isSub8x8 := bw4 == ssHor || bh4 == ssVer
	if isSub8x8 {
		row := t.rowOff(0)
		if bw4 == 1 {
			isSub8x8 = isSub8x8 && t.rt.r[row+t.bx-1].ref[0] > 0
		}
		if bh4 == ssVer {
			isSub8x8 = isSub8x8 && t.rt.r[t.rowOff(-1)+t.bx].ref[0] > 0
		}
		if bw4 == 1 && bh4 == ssVer {
			isSub8x8 = isSub8x8 && t.rt.r[t.rowOff(-1)+t.bx-1].ref[0] > 0
		}
	}

	if isSub8x8 {
		hOff, vOff := 0, 0
		if bw4 == 1 && bh4 == ssVer {
			r := &t.rt.r[t.rowOff(-1)+t.bx-1]
			for pl := range 2 {
				if err := recMc(t, f.cur[1+pl], uvdstoff, nil, f.stride[1],
					bw4, bh4, t.bx-1, t.by-1, 1+pl, r.mv[0],
					f.refp[r.ref[0]-1], int(r.ref[0])-1,
					int(t.tl4x4Filter)); err != nil {
					return err
				}
			}
			vOff = 2 * f.stride[1]
			hOff = 2
		}
		if bw4 == 1 {
			leftF2d := int(filter2d[t.l.filter[1][by4]][t.l.filter[0][by4]])
			r := &t.rt.r[t.rowOff(0)+t.bx-1]
			for pl := range 2 {
				if err := recMc(t, f.cur[1+pl], uvdstoff+vOff, nil, f.stride[1],
					bw4, bh4, t.bx-1, t.by, 1+pl, r.mv[0],
					f.refp[r.ref[0]-1], int(r.ref[0])-1, leftF2d); err != nil {
					return err
				}
			}
			hOff = 2
		}
		if bh4 == ssVer {
			topF2d := int(filter2d[t.a.filter[1][bx4]][t.a.filter[0][bx4]])
			r := &t.rt.r[t.rowOff(-1)+t.bx]
			for pl := range 2 {
				if err := recMc(t, f.cur[1+pl], uvdstoff+hOff, nil, f.stride[1],
					bw4, bh4, t.bx, t.by-1, 1+pl, r.mv[0],
					f.refp[r.ref[0]-1], int(r.ref[0])-1, topF2d); err != nil {
					return err
				}
			}
			vOff = 2 * f.stride[1]
		}
		for pl := range 2 {
			if err := recMc(t, f.cur[1+pl], uvdstoff+hOff+vOff, nil, f.stride[1],
				bw4, bh4, t.bx, t.by, 1+pl, b.mv[0], refp, int(b.ref[0]),
				f2d); err != nil {
				return err
			}
		}

		return nil
	}

	if min(cbw4, cbh4) > 1 && warped {
		for pl := range 2 {
			if err := recWarpAffine(t, f.cur[1+pl], uvdstoff, nil, f.stride[1],
				bDim, 1+pl, refp, wmp); err != nil {
				return err
			}
		}
	} else {
		for pl := range 2 {
			if err := recMc(t, f.cur[1+pl], uvdstoff, nil, f.stride[1],
				bw4<<b2i(bw4 == ssHor), bh4<<b2i(bh4 == ssVer),
				t.bx & ^ssHor, t.by & ^ssVer, 1+pl, b.mv[0], refp,
				int(b.ref[0]), f2d); err != nil {
				return err
			}
			if b.motionMode == mmObmc {
				if err := recObmc(t, f.cur[1+pl], uvdstoff, f.stride[1], bDim,
					1+pl, bx4, by4, w4, h4); err != nil {
					return err
				}
			}
		}
	}

	if b.interintraType != 0 {
		for pl := range 2 {
			t.interIntraPred(b, bs, 1+pl, f.cur[1+pl], uvdstoff, f.stride[1],
				cbw4, cbh4, chrLayoutIdx)
		}
	}

	return nil
}
