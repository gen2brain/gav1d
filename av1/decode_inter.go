package av1

import "math/bits"

// readCompRefs parses the reference pair for a compound-predicted block.
func (t *taskContext[P, C]) readCompRefs(b *av1Block, by4, bx4 int,
	haveTop, haveLeft bool,
) {
	ts := t.ts
	a, l := t.a, &t.l
	m := &ts.cdf.mi

	dirCtx := getCompDirCtx(a, l, by4, bx4, haveTop, haveLeft)
	if ts.msac.boolAdapt(m.compDir[dirCtx][:]) != 0 {
		ctx1 := av1GetFwdRefCtx(a, l, by4, bx4, haveTop, haveLeft)
		if ts.msac.boolAdapt(m.compFwdRef[0][ctx1][:]) != 0 {
			ctx2 := av1GetFwdRef2Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[0] = int8(2 + ts.msac.boolAdapt(m.compFwdRef[2][ctx2][:]))
		} else {
			ctx2 := av1GetFwdRef1Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[0] = int8(ts.msac.boolAdapt(m.compFwdRef[1][ctx2][:]))
		}

		ctx3 := av1GetBwdRefCtx(a, l, by4, bx4, haveTop, haveLeft)
		if ts.msac.boolAdapt(m.compBwdRef[0][ctx3][:]) != 0 {
			b.ref[1] = 6
		} else {
			ctx4 := av1GetBwdRef1Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[1] = int8(4 + ts.msac.boolAdapt(m.compBwdRef[1][ctx4][:]))
		}

		return
	}

	uctxP := av1GetRefCtx(a, l, by4, bx4, haveTop, haveLeft)
	if ts.msac.boolAdapt(m.compUniRef[0][uctxP][:]) != 0 {
		b.ref[0], b.ref[1] = 4, 6

		return
	}

	uctxP1 := av1GetUniP1Ctx(a, l, by4, bx4, haveTop, haveLeft)
	b.ref[0] = 0
	b.ref[1] = int8(1 + ts.msac.boolAdapt(m.compUniRef[1][uctxP1][:]))
	if b.ref[1] == 2 {
		uctxP2 := av1GetFwdRef2Ctx(a, l, by4, bx4, haveTop, haveLeft)
		b.ref[1] += int8(ts.msac.boolAdapt(m.compUniRef[2][uctxP2][:]))
	}
}

// readSingleRef parses the reference for a single-predicted block.
func (t *taskContext[P, C]) readSingleRef(b *av1Block, by4, bx4 int,
	haveTop, haveLeft bool,
) {
	ts := t.ts
	a, l := t.a, &t.l
	m := &ts.cdf.mi

	ctx1 := av1GetRefCtx(a, l, by4, bx4, haveTop, haveLeft)
	if ts.msac.boolAdapt(m.ref[0][ctx1][:]) != 0 {
		ctx2 := av1GetBwdRefCtx(a, l, by4, bx4, haveTop, haveLeft)
		if ts.msac.boolAdapt(m.ref[1][ctx2][:]) != 0 {
			b.ref[0] = 6
		} else {
			ctx3 := av1GetBwdRef1Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[0] = int8(4 + ts.msac.boolAdapt(m.ref[5][ctx3][:]))
		}
	} else {
		ctx2 := av1GetFwdRefCtx(a, l, by4, bx4, haveTop, haveLeft)
		if ts.msac.boolAdapt(m.ref[2][ctx2][:]) != 0 {
			ctx3 := av1GetFwdRef2Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[0] = int8(2 + ts.msac.boolAdapt(m.ref[4][ctx3][:]))
		} else {
			ctx3 := av1GetFwdRef1Ctx(a, l, by4, bx4, haveTop, haveLeft)
			b.ref[0] = int8(ts.msac.boolAdapt(m.ref[3][ctx3][:]))
		}
	}

	b.ref[1] = -1
}

func (t *taskContext[P, C]) rowOff(n int) int {
	return t.rt.rows[(t.by&31)+5+n]
}

func findMatchingRef[P pixel, C coef](t *taskContext[P, C], intraEdgeFlags,
	bw4, bh4, w4, h4 int, haveLeft, haveTop bool, ref int, masks *[2]uint64,
) {
	rt := &t.rt
	count := 0
	haveTopleft := haveTop && haveLeft
	haveTopright := max(bw4, bh4) < 32 && haveTop &&
		t.bx+bw4 < t.ts.tiling.colEnd &&
		intraEdgeFlags&edgeI444TopHasRight != 0

	matches := func(off int) bool {
		return int(rt.r[off].ref[0]) == ref+1 && rt.r[off].ref[1] == -1
	}
	dim := func(off, i int) int {
		return int(blockDimensions[rt.r[off].bs][i])
	}

	if haveTop {
		r2 := t.rowOff(-1) + t.bx
		if matches(r2) {
			masks[0] |= 1
			count = 1
		}
		aw4 := dim(r2, 0)
		if aw4 >= bw4 {
			off := t.bx & (aw4 - 1)
			if off != 0 {
				haveTopleft = false
			}
			if aw4-off > bw4 {
				haveTopright = false
			}
		} else {
			mask := uint64(1) << aw4
			for x := aw4; x < w4; x += aw4 {
				r2 += aw4
				if matches(r2) {
					masks[0] |= mask
					count++
					if count >= 8 {
						return
					}
				}
				aw4 = dim(r2, 0)
				mask <<= aw4
			}
		}
	}

	if haveLeft {
		row := 0
		if matches(t.rowOff(row) + t.bx - 1) {
			masks[1] |= 1
			count++
			if count >= 8 {
				return
			}
		}
		lh4 := dim(t.rowOff(row)+t.bx-1, 1)
		if lh4 >= bh4 {
			if t.by&(lh4-1) != 0 {
				haveTopleft = false
			}
		} else {
			mask := uint64(1) << lh4
			for y := lh4; y < h4; y += lh4 {
				row += lh4
				if matches(t.rowOff(row) + t.bx - 1) {
					masks[1] |= mask
					count++
					if count >= 8 {
						return
					}
				}
				lh4 = dim(t.rowOff(row)+t.bx-1, 1)
				mask <<= lh4
			}
		}
	}

	if haveTopleft && matches(t.rowOff(-1)+t.bx-1) {
		masks[1] |= 1 << 32
		count++
		if count >= 8 {
			return
		}
	}
	if haveTopright && matches(t.rowOff(-1)+t.bx+bw4) {
		masks[0] |= 1 << 32
	}
}

func deriveWarpmv[P pixel, C coef](t *taskContext[P, C], bw4, bh4 int,
	masks *[2]uint64, m mv, wmp *warpedMotionParams,
) {
	var pts [8][2][2]int32
	np := 0
	rt := &t.rt

	addSample := func(dx, dy, sx, sy, off int) {
		d := &blockDimensions[rt.r[off].bs]
		pts[np][0][0] = int32(16*(2*dx+sx*int(d[0])) - 8)
		pts[np][0][1] = int32(16*(2*dy+sy*int(d[1])) - 8)
		pts[np][1][0] = pts[np][0][0] + int32(rt.r[off].mv[0].x)
		pts[np][1][1] = pts[np][0][1] + int32(rt.r[off].mv[0].y)
		np++
	}

	if uint32(masks[0]) == 1 && masks[1]>>32 == 0 {
		base := t.rowOff(-1) + t.bx
		off := t.bx & (int(blockDimensions[rt.r[base].bs][0]) - 1)
		addSample(-off, 0, 1, -1, base)
	} else {
		off := 0
		for xmask := uint32(masks[0]); np < 8 && xmask != 0; {
			tz := bits.TrailingZeros32(xmask)
			off += tz
			xmask >>= tz
			addSample(off, 0, 1, -1, t.rowOff(-1)+t.bx+off)
			xmask &^= 1
		}
	}
	if np < 8 && masks[1] == 1 {
		base := t.rowOff(0) + t.bx - 1
		off := t.by & (int(blockDimensions[rt.r[base].bs][1]) - 1)
		addSample(0, -off, -1, 1, t.rowOff(-off)+t.bx-1)
	} else {
		off := 0
		for ymask := uint32(masks[1]); np < 8 && ymask != 0; {
			tz := bits.TrailingZeros32(ymask)
			off += tz
			ymask >>= tz
			addSample(0, off, -1, 1, t.rowOff(off)+t.bx-1)
			ymask &^= 1
		}
	}
	if np < 8 && masks[1]>>32 != 0 {
		addSample(0, 0, -1, -1, t.rowOff(-1)+t.bx-1)
	}
	if np < 8 && masks[0]>>32 != 0 {
		addSample(bw4, 0, 1, -1, t.rowOff(-1)+t.bx+bw4)
	}

	var mvd [8]int32
	ret := 0
	thresh := int32(4 * clip(max(bw4, bh4), 4, 28))
	for i := range np {
		mvd[i] = abs32(pts[i][1][0]-pts[i][0][0]-int32(m.x)) +
			abs32(pts[i][1][1]-pts[i][0][1]-int32(m.y))
		if mvd[i] > thresh {
			mvd[i] = -1
		} else {
			ret++
		}
	}
	if ret == 0 {
		ret = 1
	} else {
		for i, j, k := 0, np-1, 0; k < np-ret; k, i, j = k+1, i+1, j-1 {
			for mvd[i] != -1 {
				i++
			}
			for mvd[j] == -1 {
				j--
			}
			if i > j {
				break
			}
			mvd[i] = mvd[j]
			pts[i] = pts[j]
		}
	}

	if !findAffineInt(pts[:], ret, bw4, bh4, m, wmp, t.bx, t.by) &&
		!getShearParams(wmp) {
		wmp.typ = wmTypeAffine
	} else {
		wmp.typ = wmTypeIdentity
	}
}

func findOddZero(buf []uint8, n int) bool {
	for i := range n {
		if buf[i*2] == 0 {
			return true
		}
	}

	return false
}

func (t *taskContext[P, C]) splatOnerefMv(bs int, b *av1Block, bw4, bh4 int) {
	mode := int(b.interMode)
	ref1 := int8(-1)
	if b.interintraType != 0 {
		ref1 = 0
	}
	tmpl := refmvsBlock{
		ref: [2]int8{b.ref[0] + 1, ref1},
		mv:  [2]mv{b.mv[0]},
		bs:  uint8(bs),
		mf: uint8(b2i(mode == globalmv && min(bw4, bh4) >= 2) |
			b2i(mode == newmv)*2),
	}
	splatMv(t.rt.r, t.rt.rows[(t.by&31)+5:], &tmpl, t.bx, bw4, bh4)
}

func (t *taskContext[P, C]) splatTworefMv(bs int, b *av1Block, bw4, bh4 int) {
	mode := int(b.interMode)
	tmpl := refmvsBlock{
		ref: [2]int8{b.ref[0] + 1, b.ref[1] + 1},
		mv:  [2]mv{b.mv[0], b.mv[1]},
		bs:  uint8(bs),
		mf: uint8(b2i(mode == globalmvGlobalmv) |
			b2i((1<<mode)&0xbc != 0)*2),
	}
	splatMv(t.rt.r, t.rt.rows[(t.by&31)+5:], &tmpl, t.bx, bw4, bh4)
}

func (t *taskContext[P, C]) splatIntrabcMv(bs int, b *av1Block, bw4, bh4 int) {
	tmpl := refmvsBlock{
		ref: [2]int8{0, -1},
		mv:  [2]mv{b.mv[0]},
		bs:  uint8(bs),
	}
	splatMv(t.rt.r, t.rt.rows[(t.by&31)+5:], &tmpl, t.bx, bw4, bh4)
}

func (t *taskContext[P, C]) splatIntraref(bs, bw4, bh4 int) {
	tmpl := refmvsBlock{
		ref: [2]int8{0, -1},
		mv:  [2]mv{{y: -32768, x: -32768}},
		bs:  uint8(bs),
	}
	splatMv(t.rt.r, t.rt.rows[(t.by&31)+5:], &tmpl, t.bx, bw4, bh4)
}

func (t *taskContext[P, C]) decodeBInter(b *av1Block, bs, intraEdgeFlags int,
	seg *segmentationData, segPred int, g *blockGeom,
) error {
	isComp, filter := t.readInterBlock(b, bs, intraEdgeFlags, seg, g)

	if err := t.f.reconInter(t, bs, b); err != nil {
		return err
	}

	t.finishInterBlock(b, bs, segPred, isComp, filter, g)

	return nil
}

func (t *taskContext[P, C]) readInterBlock(b *av1Block, bs, intraEdgeFlags int,
	seg *segmentationData, g *blockGeom,
) (bool, [2]uint8) {
	f := t.f
	ts := t.ts
	bx4, by4 := g.bx4, g.by4
	bw4, bh4 := g.bw4, g.bh4

	var isComp bool
	switch {
	case b.skipMode != 0:
		isComp = true
	case (seg == nil || (seg.ref == -1 && seg.globalmv == 0 && seg.skip == 0)) &&
		f.frameHdr.switchableCompRefs != 0 && min(bw4, bh4) > 1:
		ctx := getCompCtx(t.a, &t.l, by4, bx4, g.haveTop, g.haveLeft)
		isComp = ts.msac.boolAdapt(ts.cdf.mi.comp[ctx][:]) != 0
	}

	var mvstack [8]refmvsCandidate
	var nMvs, ctx int
	var hasSubpelFilter bool

	find := func() {
		ref := [2]int8{b.ref[0] + 1, -1}
		if b.ref[1] >= 0 {
			ref[1] = b.ref[1] + 1
		}
		refmvsFind(&t.rt, mvstack[:], &nMvs, &ctx, ref, bs, intraEdgeFlags,
			t.by, t.bx)
	}

	drlBit := func(i int) uint8 {
		c := getDrlContext(mvstack[:], i)

		return uint8(ts.msac.boolAdapt(ts.cdf.mi.drlBit[c][:]))
	}

	mvPrec := int(f.frameHdr.hp) - int(f.frameHdr.forceIntegerMv)

	switch {
	case b.skipMode != 0:
		b.ref[0] = f.frameHdr.skipModeRefs[0]
		b.ref[1] = f.frameHdr.skipModeRefs[1]
		b.compType = compInterAvg
		b.interMode = nearestmvNearestmv
		b.drlIdx = nearestDrl
		hasSubpelFilter = false

		find()
		b.mv[0] = mvstack[0].mv[0]
		b.mv[1] = mvstack[0].mv[1]
		fixMvPrecision(f.frameHdr, &b.mv[0])
		fixMvPrecision(f.frameHdr, &b.mv[1])

	case isComp:
		t.readCompRefs(b, by4, bx4, g.haveTop, g.haveLeft)
		find()

		b.interMode = uint8(ts.msac.symbolAdapt(ts.cdf.mi.compInterMode[ctx][:],
			nCompInterPredModes-1))
		im := &compInterPredModes[b.interMode]
		b.drlIdx = nearestDrl
		if b.interMode == newmvNewmv {
			if nMvs > 1 {
				b.drlIdx += drlBit(0)
				if b.drlIdx == nearerDrl && nMvs > 2 {
					b.drlIdx += drlBit(1)
				}
			}
		} else if im[0] == nearmv || im[1] == nearmv {
			b.drlIdx = nearerDrl
			if nMvs > 2 {
				b.drlIdx += drlBit(1)
				if b.drlIdx == nearDrl && nMvs > 3 {
					b.drlIdx += drlBit(2)
				}
			}
		}

		hasSubpelFilter = min(bw4, bh4) == 1 || b.interMode != globalmvGlobalmv
		for idx := range 2 {
			switch im[idx] {
			case nearmv, nearestmv:
				b.mv[idx] = mvstack[b.drlIdx].mv[idx]
				fixMvPrecision(f.frameHdr, &b.mv[idx])
			case globalmv:
				gmv := &f.frameHdr.gmv[b.ref[idx]]
				hasSubpelFilter = hasSubpelFilter || gmv.typ == wmTypeTranslation
				b.mv[idx] = getGmv2d(gmv, t.bx, t.by, bw4, bh4, f.frameHdr)
			case newmv:
				b.mv[idx] = mvstack[b.drlIdx].mv[idx]
				readMvResidual(ts, &b.mv[idx], mvPrec)
			}
		}

		t.readCompoundType(b, bs, by4, bx4,
			int(f.refPoc[b.ref[0]]), int(f.refPoc[b.ref[1]]))

	default:
		b.compType = compInterNone

		switch {
		case seg != nil && seg.ref > 0:
			b.ref[0] = seg.ref - 1
		case seg != nil && (seg.globalmv != 0 || seg.skip != 0):
			b.ref[0] = 0
		default:
			t.readSingleRef(b, by4, bx4, g.haveTop, g.haveLeft)
		}
		b.ref[1] = -1

		find()

		segGlobal := seg != nil && (seg.skip != 0 || seg.globalmv != 0)
		if segGlobal || ts.msac.boolAdapt(ts.cdf.mi.newmvMode[ctx&7][:]) != 0 {
			if segGlobal ||
				ts.msac.boolAdapt(ts.cdf.mi.globalmvMode[(ctx>>3)&1][:]) == 0 {
				b.interMode = globalmv
				gmv := &f.frameHdr.gmv[b.ref[0]]
				b.mv[0] = getGmv2d(gmv, t.bx, t.by, bw4, bh4, f.frameHdr)
				hasSubpelFilter = min(bw4, bh4) == 1 || gmv.typ == wmTypeTranslation
			} else {
				hasSubpelFilter = true
				if ts.msac.boolAdapt(ts.cdf.mi.refmvMode[(ctx>>4)&15][:]) != 0 {
					b.interMode = nearmv
					b.drlIdx = nearerDrl
					if nMvs > 2 {
						b.drlIdx += drlBit(1)
						if b.drlIdx == nearDrl && nMvs > 3 {
							b.drlIdx += drlBit(2)
						}
					}
				} else {
					b.interMode = nearestmv
					b.drlIdx = nearestDrl
				}
				b.mv[0] = mvstack[b.drlIdx].mv[0]
				if b.drlIdx < nearDrl {
					fixMvPrecision(f.frameHdr, &b.mv[0])
				}
			}
		} else {
			hasSubpelFilter = true
			b.interMode = newmv
			b.drlIdx = nearestDrl
			if nMvs > 1 {
				b.drlIdx += drlBit(0)
				if b.drlIdx == nearerDrl && nMvs > 2 {
					b.drlIdx += drlBit(1)
				}
			}
			if nMvs > 1 {
				b.mv[0] = mvstack[b.drlIdx].mv[0]
			} else {
				b.mv[0] = mvstack[0].mv[0]
				fixMvPrecision(f.frameHdr, &b.mv[0])
			}
			readMvResidual(ts, &b.mv[0], mvPrec)
		}

		t.readInterIntra(b, bs)

		gmvWarped := f.frameHdr.forceIntegerMv == 0 && b.interMode == globalmv &&
			f.frameHdr.gmv[b.ref[0]].typ > wmTypeTranslation
		overlappable := (g.haveLeft && findOddZero(t.l.intra[by4+1:], g.h4>>1)) ||
			(g.haveTop && findOddZero(t.a.intra[bx4+1:], g.w4>>1))

		if f.frameHdr.switchableMotionMode != 0 &&
			b.interintraType == interIntraNone && min(bw4, bh4) >= 2 &&
			!gmvWarped && overlappable {
			var mask [2]uint64
			findMatchingRef(t, intraEdgeFlags, bw4, bh4, g.w4, g.h4,
				g.haveLeft, g.haveTop, int(b.ref[0]), &mask)
			allowWarp := f.svc[b.ref[0]][0].scale == 0 &&
				f.frameHdr.forceIntegerMv == 0 &&
				f.frameHdr.warpMotion != 0 && mask[0]|mask[1] != 0

			if allowWarp {
				b.motionMode = uint8(ts.msac.symbolAdapt(ts.cdf.mi.motionMode[bs][:], 2))
			} else {
				b.motionMode = uint8(ts.msac.boolAdapt(ts.cdf.mi.obmc[bs][:]))
			}
			if b.motionMode == mmWarp {
				hasSubpelFilter = false
				deriveWarpmv(t, bw4, bh4, &mask, b.mv[0], &t.warpmv)
			}
		} else {
			b.motionMode = mmTranslation
		}
	}

	var filter [2]uint8
	if f.frameHdr.subpelFilterMode == filterSwitchable {
		if hasSubpelFilter {
			comp := b2i(b.compType != compInterNone)
			ctx1 := getFilterCtx(t.a, &t.l, comp, 0, int(b.ref[0]), by4, bx4)
			filter[0] = uint8(ts.msac.symbolAdapt(ts.cdf.mi.filter[0][ctx1][:],
				nSwitchableFilters-1))
			if f.seqHdr.dualFilter != 0 {
				ctx2 := getFilterCtx(t.a, &t.l, comp, 1, int(b.ref[0]), by4, bx4)
				filter[1] = uint8(ts.msac.symbolAdapt(ts.cdf.mi.filter[1][ctx2][:],
					nSwitchableFilters-1))
			} else {
				filter[1] = filter[0]
			}
		} else {
			filter[0], filter[1] = filterRegular8tap, filterRegular8tap
		}
	} else {
		filter[0] = uint8(f.frameHdr.subpelFilterMode)
		filter[1] = filter[0]
	}
	b.filter2d = filter2d[filter[1]][filter[0]]

	t.readVartxTree(b, bs, bx4, by4)

	return isComp, filter
}

func (t *taskContext[P, C]) finishInterBlock(b *av1Block, bs, segPred int,
	isComp bool, filter [2]uint8, g *blockGeom,
) {
	f := t.f
	ts := t.ts
	bDim := &blockDimensions[bs]
	bx4, by4 := g.bx4, g.by4
	bw4, bh4 := g.bw4, g.bh4

	if f.frameHdr.loopfilter.levelY[0] != 0 || f.frameHdr.loopfilter.levelY[1] != 0 {
		wantGlobal := globalmv
		if isComp {
			wantGlobal = globalmvGlobalmv
		}
		isGlobalmv := b2i(int(b.interMode) == wantGlobal)
		var lfLvls [4]uint8
		for i := range lfLvls {
			lfLvls[i] = ts.lflvl[b.segID][i][b.ref[0]+1][1-isGlobalmv]
		}
		txMasks := [2]uint16{uint16(b.txSplit0), b.txSplit1}
		ytx, uvtx := int(b.maxYtx), int(b.uvtx)
		if f.frameHdr.segmentation.lossless[b.segID] != 0 {
			ytx, uvtx = tx4x4, tx4x4
		}
		var auv, luv []uint8
		auvOff, luvOff := 0, 0
		if g.hasChroma {
			auv, luv = t.a.txLpfUv[:], t.l.txLpfUv[:]
			auvOff, luvOff = g.cbx4, g.cby4
		}
		createLfMaskInter(t.lfMask, f.lf.level, f.b4Stride, &lfLvls,
			t.bx, t.by, f.w4, f.h4, b.skip != 0, bs, ytx, &txMasks, uvtx,
			f.layout, t.a.txLpfY[:], t.l.txLpfY[:], auv, luv,
			bx4, by4, auvOff, luvOff, g.hasChroma, &t.scratch.lfTxa)
	}

	if isComp {
		t.splatTworefMv(bs, b, bw4, bh4)
	} else {
		t.splatOnerefMv(bs, b, bw4, bh4)
	}

	for i, edge := range [2]*blockContext{t.a, &t.l} {
		off, n := bx4, bw4
		if i == 1 {
			off, n = by4, bh4
		}
		setCtx(edge.segPred[:], off, n, uint8(segPred))
		setCtx(edge.skipMode[:], off, n, b.skipMode)
		setCtx(edge.intra[:], off, n, 0)
		setCtx(edge.skip[:], off, n, b.skip)
		setCtx(edge.palSz[:], off, n, 0)
		setCtx(t.palSzUv[i][:], off, n, 0)
		setCtxI8(edge.txIntra[:], off, n, int8(bDim[2+i]))
		setCtx(edge.compType[:], off, n, b.compType)
		setCtx(edge.filter[0][:], off, n, filter[0])
		setCtx(edge.filter[1][:], off, n, filter[1])
		setCtx(edge.mode[:], off, n, b.interMode)
		setCtxI8(edge.ref[0][:], off, n, b.ref[0])
		setCtxI8(edge.ref[1][:], off, n, b.ref[1])
	}

	if g.hasChroma {
		setCtx(t.a.uvmode[:], g.cbx4, g.cbw4, dcPred)
		setCtx(t.l.uvmode[:], g.cby4, g.cbh4, dcPred)
	}
}

func (t *taskContext[P, C]) decodeBIntrabc(b *av1Block, bs, intraEdgeFlags int,
	segPred int, g *blockGeom,
) error {
	f := t.f
	ts := t.ts
	bDim := &blockDimensions[bs]
	bx4, by4 := g.bx4, g.by4
	bw4, bh4 := g.bw4, g.bh4
	sb128 := int(f.seqHdr.sb128)

	var mvstack [8]refmvsCandidate
	var nMvs, ctx int
	refmvsFind(&t.rt, mvstack[:], &nMvs, &ctx, [2]int8{0, -1}, bs,
		intraEdgeFlags, t.by, t.bx)

	switch {
	case mvstack[0].mv[0].n() != 0:
		b.mv[0] = mvstack[0].mv[0]
	case mvstack[1].mv[0].n() != 0:
		b.mv[0] = mvstack[1].mv[0]
	case t.by-(16<<sb128) < ts.tiling.rowStart:
		b.mv[0] = mv{y: 0, x: int16(-(512 << sb128) - 2048)}
	default:
		b.mv[0] = mv{y: int16(-(512 << sb128)), x: 0}
	}

	readMvResidual(ts, &b.mv[0], -1)

	borderLeft := ts.tiling.colStart * 4
	borderTop := ts.tiling.rowStart * 4
	if g.hasChroma {
		if bw4 < 2 && f.ssHor != 0 {
			borderLeft += 4
		}
		if bh4 < 2 && f.ssVer != 0 {
			borderTop += 4
		}
	}
	srcLeft := t.bx*4 + int(b.mv[0].x)>>3
	srcTop := t.by*4 + int(b.mv[0].y)>>3
	srcRight := srcLeft + bw4*4
	srcBottom := srcTop + bh4*4
	borderRight := ((ts.tiling.colEnd + (bw4 - 1)) &^ (bw4 - 1)) * 4

	if srcLeft < borderLeft {
		srcRight += borderLeft - srcLeft
		srcLeft += borderLeft - srcLeft
	} else if srcRight > borderRight {
		srcLeft -= srcRight - borderRight
		srcRight -= srcRight - borderRight
	}
	if srcTop < borderTop {
		srcBottom += borderTop - srcTop
		srcTop += borderTop - srcTop
	}

	sbx := (t.bx >> (4 + sb128)) << (6 + sb128)
	sby := (t.by >> (4 + sb128)) << (6 + sb128)
	sbSize := 1 << (6 + sb128)

	if srcBottom > sby && srcRight > sbx {
		if srcTop-borderTop >= srcBottom-sby {
			srcTop -= srcBottom - sby
			srcBottom -= srcBottom - sby
		} else if srcLeft-borderLeft >= srcRight-sbx {
			srcLeft -= srcRight - sbx
			srcRight -= srcRight - sbx
		}
	}
	if srcBottom > sby+sbSize {
		srcTop -= srcBottom - (sby + sbSize)
		srcBottom -= srcBottom - (sby + sbSize)
	}
	if srcBottom > sby && srcRight > sbx {
		return errInvalid
	}

	b.mv[0].x = int16((srcLeft - t.bx*4) * 8)
	b.mv[0].y = int16((srcTop - t.by*4) * 8)

	t.readVartxTree(b, bs, bx4, by4)

	if err := f.reconInter(t, bs, b); err != nil {
		return err
	}

	t.splatIntrabcMv(bs, b, bw4, bh4)

	for i, edge := range [2]*blockContext{t.a, &t.l} {
		off, n := bx4, bw4
		if i == 1 {
			off, n = by4, bh4
		}
		setCtxI8(edge.txIntra[:], off, n, int8(bDim[2+i]))
		setCtx(edge.mode[:], off, n, dcPred)
		setCtx(edge.palSz[:], off, n, 0)
		setCtx(t.palSzUv[i][:], off, n, 0)
		setCtx(edge.segPred[:], off, n, uint8(segPred))
		setCtx(edge.skipMode[:], off, n, 0)
		setCtx(edge.intra[:], off, n, 0)
		setCtx(edge.skip[:], off, n, b.skip)
	}

	if g.hasChroma {
		setCtx(t.a.uvmode[:], g.cbx4, g.cbw4, dcPred)
		setCtx(t.l.uvmode[:], g.cby4, g.cbh4, dcPred)
	}

	return nil
}
