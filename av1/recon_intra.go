package av1

func smFlag(b *blockContext, idx int) int {
	if b.intra[idx] == 0 {
		return 0
	}
	m := b.mode[idx]
	if m == smoothPred || m == smoothHPred || m == smoothVPred {
		return angleSmoothEdgeFlag
	}

	return 0
}

func smUvFlag(b *blockContext, idx int) int {
	m := b.uvmode[idx]
	if m == smoothPred || m == smoothHPred || m == smoothVPred {
		return angleSmoothEdgeFlag
	}

	return 0
}

func memset(dst []uint8, v uint8) {
	for i := range dst {
		dst[i] = v
	}
}

func reconBIntra[P pixel, C coef](t *taskContext[P, C], bs int,
	intraEdgeFlags int, b *av1Block,
) {
	ts := t.ts
	f := t.f
	bx4, by4 := t.bx&31, t.by&31
	ssVer := b2i(f.layout == pixelLayoutI420)
	ssHor := b2i(f.layout != pixelLayoutI444)
	cbx4, cby4 := bx4>>ssHor, by4>>ssVer
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	w4, h4 := min(bw4, f.bw-t.bx), min(bh4, f.bh-t.by)
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	hasChroma := f.layout != pixelLayoutI400 &&
		(bw4 > ssHor || t.bx&1 != 0) &&
		(bh4 > ssVer || t.by&1 != 0)
	tDim := &txfmDimensions[b.tx]
	uvTDim := &txfmDimensions[b.uvtx]

	edge := t.scratch.edge[:]
	const edgeOff = 128
	cbw4, cbh4 := (bw4+ssHor)>>ssHor, (bh4+ssVer)>>ssVer

	intraEdgeFilterFlag := int(f.seqHdr.intraEdgeFilter) << 10

	for initY := 0; initY < h4; initY += 16 {
		subH4 := min(h4, 16+initY)
		subCh4 := min(ch4, (initY+16)>>ssVer)

		for initX := 0; initX < w4; initX += 16 {
			if b.palSz[0] != 0 {
				dstOff := 4 * (t.by*f.stride[0] + t.bx)
				palPred(f.cur[0], dstOff, f.stride[0], t.scratch.pal[0][:],
					t.scratch.palIdxY[:], bw4*4, bh4*4)
			}

			intraFlags := smFlag(t.a, bx4) | smFlag(&t.l, by4) | intraEdgeFilterFlag

			sbHasTr := 0
			switch {
			case initX+16 < w4:
				sbHasTr = 1
			case initY != 0:
				sbHasTr = 0
			default:
				sbHasTr = intraEdgeFlags & edgeI444TopHasRight
			}
			sbHasBl := 0
			switch {
			case initX != 0:
				sbHasBl = 0
			case initY+16 < h4:
				sbHasBl = 1
			default:
				sbHasBl = intraEdgeFlags & edgeI444LeftHasBottom
			}

			var y, x int
			subW4 := min(w4, initX+16)

			for y, t.by = initY, t.by+initY; y < subH4; y, t.by = y+int(tDim.h), t.by+int(tDim.h) {
				dstOff := 4 * (t.by*f.stride[0] + t.bx + initX)

				for x, t.bx = initX, t.bx+initX; x < subW4; x, t.bx = x+int(tDim.w), t.bx+int(tDim.w) {
					if b.palSz[0] == 0 {
						angle := int(b.yAngle)
						edgeFlags := 0
						if !((y > initY || sbHasTr == 0) && x+int(tDim.w) >= subW4) {
							edgeFlags |= edgeI444TopHasRight
						}
						if !(x > initX || (sbHasBl == 0 && y+int(tDim.h) >= subH4)) {
							edgeFlags |= edgeI444LeftHasBottom
						}

						var topSbEdge []P
						topSbEdgeOff := 0
						if t.by&(f.sbStep-1) == 0 {
							topSbEdge = f.ipredEdge[0]
							sby := t.by >> f.sbShift
							topSbEdgeOff = f.sb128w * 128 * (sby - 1)
						}

						m := prepareIntraEdges(t.bx, t.bx > ts.tiling.colStart,
							t.by, t.by > ts.tiling.rowStart,
							ts.tiling.colEnd, ts.tiling.rowEnd,
							edgeFlags, f.cur[0], dstOff, f.stride[0],
							topSbEdge, topSbEdgeOff, int(b.yMode), &angle,
							int(tDim.w), int(tDim.h), f.seqHdr.intraEdgeFilter != 0,
							edge, edgeOff, f.bitdepthMax)

						f.dsp.ipred[m](f.cur[0], dstOff, f.stride[0], edge, edgeOff,
							int(tDim.w)*4, int(tDim.h)*4, angle|intraFlags,
							4*f.bw-4*t.bx, 4*f.bh-4*t.by, f.bitdepthMax)
					}

					if b.skip == 0 {
						cf := t.cf
						eob, txtp, cfCtx := decodeCoefs(t, t.a.lcoef[:], bx4+x,
							t.l.lcoef[:], by4+y, int(b.tx), bs, b, 1, 0, cf, 0)
						memset(t.a.lcoef[bx4+x:bx4+x+min(int(tDim.w), f.bw-t.bx)], cfCtx)
						memset(t.l.lcoef[by4+y:by4+y+min(int(tDim.h), f.bh-t.by)], cfCtx)
						if eob >= 0 {
							itxfmAdd(f.dsp, f.cur[0], dstOff, f.stride[0], cf, t.scratch.itx[:], eob,
								int(b.tx), txtp, f.bitdepthMax)
						}
					} else {
						memset(t.a.lcoef[bx4+x:bx4+x+int(tDim.w)], 0x40)
						memset(t.l.lcoef[by4+y:by4+y+int(tDim.h)], 0x40)
					}
					dstOff += 4 * int(tDim.w)
				}
				t.bx -= x
			}
			t.by -= y

			if !hasChroma {
				continue
			}

			stride := f.stride[1]

			if b.uvMode == cflPred {
				ac := t.scratch.ac[:]
				ySrcOff := 4*(t.bx & ^ssHor) + 4*(t.by & ^ssVer)*f.stride[0]
				uvOff := 4 * ((t.bx >> ssHor) + (t.by>>ssVer)*stride)

				furthestR := ((cw4 << ssHor) + int(tDim.w) - 1) & ^(int(tDim.w) - 1)
				furthestB := ((ch4 << ssVer) + int(tDim.h) - 1) & ^(int(tDim.h) - 1)

				cflAc(ac, f.cur[0], ySrcOff, f.stride[0],
					cbw4-(furthestR>>ssHor), cbh4-(furthestB>>ssVer),
					cbw4*4, cbh4*4, ssHor, ssVer)

				for pl := range 2 {
					if b.cflAlpha[pl] == 0 {
						continue
					}
					angle := 0
					var topSbEdge []P
					topSbEdgeOff := 0
					if (t.by & ^ssVer)&(f.sbStep-1) == 0 {
						topSbEdge = f.ipredEdge[pl+1]
						sby := t.by >> f.sbShift
						topSbEdgeOff = f.sb128w * 128 * (sby - 1)
					}
					xpos, ypos := t.bx>>ssHor, t.by>>ssVer
					xstart := ts.tiling.colStart >> ssHor
					ystart := ts.tiling.rowStart >> ssVer

					m := prepareIntraEdges(xpos, xpos > xstart, ypos, ypos > ystart,
						ts.tiling.colEnd>>ssHor, ts.tiling.rowEnd>>ssVer,
						0, f.cur[1+pl], uvOff, stride, topSbEdge, topSbEdgeOff,
						dcPred, &angle, int(uvTDim.w), int(uvTDim.h), false,
						edge, edgeOff, f.bitdepthMax)

					f.dsp.cflPred[m](f.cur[1+pl], uvOff, stride, edge, edgeOff,
						int(uvTDim.w)*4, int(uvTDim.h)*4, ac, int32(b.cflAlpha[pl]),
						f.bitdepthMax)
				}
			} else if b.palSz[1] != 0 {
				uvDstOff := 4 * ((t.bx >> ssHor) + (t.by>>ssVer)*f.stride[1])
				palPred(f.cur[1], uvDstOff, f.stride[1], t.scratch.pal[1][:],
					t.scratch.palIdxUv[:], cbw4*4, cbh4*4)
				palPred(f.cur[2], uvDstOff, f.stride[1], t.scratch.pal[2][:],
					t.scratch.palIdxUv[:], cbw4*4, cbh4*4)
			}

			smUvFl := smUvFlag(t.a, cbx4) | smUvFlag(&t.l, cby4)

			uvSbHasTr := 0
			switch {
			case (initX+16)>>ssHor < cw4:
				uvSbHasTr = 1
			case initY != 0:
				uvSbHasTr = 0
			default:
				uvSbHasTr = intraEdgeFlags & (edgeI420TopHasRight >> (f.layout - 1))
			}
			uvSbHasBl := 0
			switch {
			case initX != 0:
				uvSbHasBl = 0
			case (initY+16)>>ssVer < ch4:
				uvSbHasBl = 1
			default:
				uvSbHasBl = intraEdgeFlags & (edgeI420LeftHasBottom >> (f.layout - 1))
			}

			subCw4 := min(cw4, (initX+16)>>ssHor)

			for pl := range 2 {
				for y, t.by = initY>>ssVer, t.by+initY; y < subCh4; y, t.by = y+int(uvTDim.h), t.by+int(uvTDim.h)<<ssVer {
					dstOff := 4 * ((t.by>>ssVer)*stride + ((t.bx + initX) >> ssHor))

					for x, t.bx = initX>>ssHor, t.bx+initX; x < subCw4; x, t.bx = x+int(uvTDim.w), t.bx+int(uvTDim.w)<<ssHor {
						skipPred := (b.uvMode == cflPred && b.cflAlpha[pl] != 0) || b.palSz[1] != 0

						if !skipPred {
							angle := int(b.uvAngle)
							edgeFlags := 0
							if !((y > initY>>ssVer || uvSbHasTr == 0) && x+int(uvTDim.w) >= subCw4) {
								edgeFlags |= edgeI444TopHasRight
							}
							if !(x > initX>>ssHor ||
								(uvSbHasBl == 0 && y+int(uvTDim.h) >= subCh4)) {
								edgeFlags |= edgeI444LeftHasBottom
							}

							var topSbEdge []P
							topSbEdgeOff := 0
							if (t.by & ^ssVer)&(f.sbStep-1) == 0 {
								topSbEdge = f.ipredEdge[1+pl]
								sby := t.by >> f.sbShift
								topSbEdgeOff = f.sb128w * 128 * (sby - 1)
							}

							uvMode := int(b.uvMode)
							if uvMode == cflPred {
								uvMode = dcPred
							}
							xpos, ypos := t.bx>>ssHor, t.by>>ssVer
							xstart := ts.tiling.colStart >> ssHor
							ystart := ts.tiling.rowStart >> ssVer

							m := prepareIntraEdges(xpos, xpos > xstart, ypos, ypos > ystart,
								ts.tiling.colEnd>>ssHor, ts.tiling.rowEnd>>ssVer,
								edgeFlags, f.cur[1+pl], dstOff, stride,
								topSbEdge, topSbEdgeOff, uvMode, &angle,
								int(uvTDim.w), int(uvTDim.h),
								f.seqHdr.intraEdgeFilter != 0, edge, edgeOff, f.bitdepthMax)

							angle |= intraEdgeFilterFlag
							f.dsp.ipred[m](f.cur[1+pl], dstOff, stride, edge, edgeOff,
								int(uvTDim.w)*4, int(uvTDim.h)*4, angle|smUvFl,
								(4*f.bw+ssHor-4*(t.bx & ^ssHor))>>ssHor,
								(4*f.bh+ssVer-4*(t.by & ^ssVer))>>ssVer, f.bitdepthMax)
						}

						if b.skip == 0 {
							cf := t.cf
							eob, txtp, cfCtx := decodeCoefs(t, t.a.ccoef[pl][:], cbx4+x,
								t.l.ccoef[pl][:], cby4+y, int(b.uvtx), bs, b, 1, 1+pl, cf, 0)
							ctw := min(int(uvTDim.w), (f.bw-t.bx+ssHor)>>ssHor)
							cth := min(int(uvTDim.h), (f.bh-t.by+ssVer)>>ssVer)
							memset(t.a.ccoef[pl][cbx4+x:cbx4+x+ctw], cfCtx)
							memset(t.l.ccoef[pl][cby4+y:cby4+y+cth], cfCtx)
							if eob >= 0 {
								itxfmAdd(f.dsp, f.cur[1+pl], dstOff, stride, cf, t.scratch.itx[:], eob,
									int(b.uvtx), txtp, f.bitdepthMax)
							}
						} else {
							memset(t.a.ccoef[pl][cbx4+x:cbx4+x+int(uvTDim.w)], 0x40)
							memset(t.l.ccoef[pl][cby4+y:cby4+y+int(uvTDim.h)], 0x40)
						}
						dstOff += int(uvTDim.w) * 4
					}
					t.bx -= x << ssHor
				}
				t.by -= y << ssVer
			}
		}
	}
}

func readPalPlane[P pixel, C coef](t *taskContext[P, C], b *av1Block,
	pl, szCtx, bx4, by4 int,
) {
	ts := t.ts
	f := t.f

	palSz := int(ts.msac.symbolAdapt(ts.cdf.m.palSz[pl][szCtx][:], 6)) + 2
	b.palSz[pl] = uint8(palSz)

	var cache [16]P
	var usedCache [8]P

	lCache := int(t.l.palSz[by4])
	if pl != 0 {
		lCache = int(t.palSzUv[1][by4])
	}
	aCache := 0
	if by4&15 != 0 {
		if pl != 0 {
			aCache = int(t.palSzUv[0][bx4])
		} else {
			aCache = int(t.a.palSz[bx4])
		}
	}

	l := &t.alPal[1][by4][pl]
	a := &t.alPal[0][bx4][pl]
	li, ai, nCache := 0, 0, 0

	for lCache != 0 && aCache != 0 {
		if l[li] < a[ai] {
			if nCache == 0 || cache[nCache-1] != l[li] {
				cache[nCache] = l[li]
				nCache++
			}
			li++
			lCache--
		} else {
			if a[ai] == l[li] {
				li++
				lCache--
			}
			if nCache == 0 || cache[nCache-1] != a[ai] {
				cache[nCache] = a[ai]
				nCache++
			}
			ai++
			aCache--
		}
	}
	for ; lCache > 0; lCache-- {
		if nCache == 0 || cache[nCache-1] != l[li] {
			cache[nCache] = l[li]
			nCache++
		}
		li++
	}
	for ; aCache > 0; aCache-- {
		if nCache == 0 || cache[nCache-1] != a[ai] {
			cache[nCache] = a[ai]
			nCache++
		}
		ai++
	}

	i := 0
	for n := 0; n < nCache && i < palSz; n++ {
		if ts.msac.boolEqui() != 0 {
			usedCache[i] = cache[n]
			i++
		}
	}
	nUsedCache := i

	pal := t.scratch.pal[pl][:]
	if i >= palSz {
		copy(pal[:nUsedCache], usedCache[:nUsedCache])

		return
	}

	bpc := f.bpc
	maxv := 1<<bpc - 1
	notUv := b2i(pl == 0)

	prev := int(ts.msac.bools(bpc))
	pal[i] = P(prev)
	i++

	if i < palSz {
		bits := bpc - 3 + int(ts.msac.bools(2))
		for {
			delta := int(ts.msac.bools(bits))
			prev = min(prev+delta+notUv, maxv)
			pal[i] = P(prev)
			i++
			if prev+notUv >= maxv {
				for ; i < palSz; i++ {
					pal[i] = P(maxv)
				}

				break
			}
			bits = min(bits, 1+ulog2(uint32(maxv-prev-notUv)))
			if i >= palSz {
				break
			}
		}
	}

	n, m := 0, nUsedCache
	for i = range palSz {
		if n < nUsedCache && (m >= palSz || usedCache[n] <= pal[m]) {
			pal[i] = usedCache[n]
			n++
		} else {
			pal[i] = pal[m]
			m++
		}
	}
}

func readPalUv[P pixel, C coef](t *taskContext[P, C], b *av1Block, szCtx, bx4, by4 int) {
	readPalPlane(t, b, 1, szCtx, bx4, by4)

	ts := t.ts
	pal := t.scratch.pal[2][:]
	bpc := t.f.bpc
	palSz := int(b.palSz[1])

	if ts.msac.boolEqui() != 0 {
		bits := bpc - 4 + int(ts.msac.bools(2))
		maxv := 1<<bpc - 1
		prev := int(ts.msac.bools(bpc))
		pal[0] = P(prev)
		for i := 1; i < palSz; i++ {
			delta := int(ts.msac.bools(bits))
			if delta != 0 && ts.msac.boolEqui() != 0 {
				delta = -delta
			}
			prev = (prev + delta) & maxv
			pal[i] = P(prev)
		}

		return
	}

	for i := range palSz {
		pal[i] = P(ts.msac.bools(bpc))
	}
}

func copyPalBlockY[P pixel, C coef](t *taskContext[P, C], bx4, by4, bw4, bh4 int) {
	pal := &t.scratch.pal[0]
	for x := range bw4 {
		t.alPal[0][bx4+x][0] = *pal
	}
	for y := range bh4 {
		t.alPal[1][by4+y][0] = *pal
	}
}

func copyPalBlockUv[P pixel, C coef](t *taskContext[P, C], bx4, by4, bw4, bh4 int) {
	for pl := 1; pl <= 2; pl++ {
		pal := &t.scratch.pal[pl]
		for x := range bw4 {
			t.alPal[0][bx4+x][pl] = *pal
		}
		for y := range bh4 {
			t.alPal[1][by4+y][pl] = *pal
		}
	}
}
