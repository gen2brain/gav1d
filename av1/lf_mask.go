package av1

type av1RestorationUnit struct {
	typ        uint8
	filterH    [3]int8
	filterV    [3]int8
	sgrWeights [2]int8
}

type av1Restoration struct {
	lr [3][4]av1RestorationUnit
}

type av1Filter struct {
	filterY    [2][32][3][2]uint16
	filterUv   [2][32][2][2]uint16
	cdefIdx    [4]int8
	noskipMask [16][2]uint16
}

func calcEIH(lut *filterLUT, sharp int) {
	for level := range 64 {
		limit := level

		if sharp > 0 {
			limit >>= (sharp + 3) >> 2
			limit = min(limit, 9-sharp)
		}
		limit = max(limit, 1)

		lut.i[level] = uint8(limit)
		lut.e[level] = uint8(2*(level+2) + limit)
	}
	lut.sharp[0] = uint64((sharp + 3) >> 2)
	if sharp != 0 {
		lut.sharp[1] = uint64(9 - sharp)
	} else {
		lut.sharp[1] = 0xff
	}
}

func calcLfValue(v *[8][2]uint8, baseLvl, lfDelta, segDelta int,
	mrDelta *loopfilterModeRefDeltas,
) {
	base := clip(clip(baseLvl+lfDelta, 0, 63)+segDelta, 0, 63)

	if mrDelta == nil {
		for r := range 8 {
			v[r][0], v[r][1] = uint8(base), uint8(base)
		}

		return
	}

	sh := b2i(base >= 32)
	x := uint8(clip(base+int(mrDelta.refDelta[0])*(1<<sh), 0, 63))
	v[0][0], v[0][1] = x, x
	for r := 1; r < 8; r++ {
		for m := range 2 {
			delta := int(mrDelta.modeDelta[m]) + int(mrDelta.refDelta[r])
			v[r][m] = uint8(clip(base+delta*(1<<sh), 0, 63))
		}
	}
}

func calcLfValueChroma(v *[8][2]uint8, baseLvl, lfDelta, segDelta int,
	mrDelta *loopfilterModeRefDeltas,
) {
	if baseLvl == 0 {
		clear(v[:])

		return
	}
	calcLfValue(v, baseLvl, lfDelta, segDelta, mrDelta)
}

func calcLfValues(v *[8][4][8][2]uint8, hdr *frameHeader, lfDelta *[4]int8) {
	nSeg := 1
	if hdr.segmentation.enabled != 0 {
		nSeg = 8
	}

	if hdr.loopfilter.levelY[0] == 0 && hdr.loopfilter.levelY[1] == 0 {
		for s := range nSeg {
			clear(v[s][:])
		}

		return
	}

	var mrDeltas *loopfilterModeRefDeltas
	if hdr.loopfilter.modeRefDeltaEnabled != 0 {
		mrDeltas = &hdr.loopfilter.modeRefDeltas
	}

	multi := 0
	if hdr.delta.lf.multi != 0 {
		multi = 1
	}

	for s := range nSeg {
		var segd *segmentationData
		if hdr.segmentation.enabled != 0 {
			segd = &hdr.segmentation.segData.d[s]
		}
		d := func(get func(*segmentationData) int8) int {
			if segd == nil {
				return 0
			}

			return int(get(segd))
		}

		calcLfValue(&v[s][0], int(hdr.loopfilter.levelY[0]), int(lfDelta[0]),
			d(func(s *segmentationData) int8 { return s.deltaLfYV }), mrDeltas)
		calcLfValue(&v[s][1], int(hdr.loopfilter.levelY[1]), int(lfDelta[multi]),
			d(func(s *segmentationData) int8 { return s.deltaLfYH }), mrDeltas)
		calcLfValueChroma(&v[s][2], int(hdr.loopfilter.levelU), int(lfDelta[multi*2]),
			d(func(s *segmentationData) int8 { return s.deltaLfU }), mrDeltas)
		calcLfValueChroma(&v[s][3], int(hdr.loopfilter.levelV), int(lfDelta[multi*3]),
			d(func(s *segmentationData) int8 { return s.deltaLfV }), mrDeltas)
	}
}

func maskEdgesIntra(masks *[2][32][3][2]uint16, by4, bx4, w4, h4, tx int,
	a, l []uint8, aOff, lOff int,
) {
	tDim := &txfmDimensions[tx]
	twl4, thl4 := int(tDim.lw), int(tDim.lh)
	twl4c, thl4c := min(2, twl4), min(2, thl4)

	mask := uint32(1) << by4
	for y := range h4 {
		sidx := b2i(mask >= 0x10000)
		smask := mask >> (sidx << 4)
		masks[0][bx4][min(twl4c, int(l[lOff+y]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	mask = uint32(1) << bx4
	for x := range w4 {
		sidx := b2i(mask >= 0x10000)
		smask := mask >> (sidx << 4)
		masks[1][by4][min(thl4c, int(a[aOff+x]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	hstep := int(tDim.w)
	t := uint64(1) << by4
	inner := uint32((t << h4) - t)
	inner1, inner2 := uint16(inner&0xffff), uint16(inner>>16)
	for x := hstep; x < w4; x += hstep {
		if inner1 != 0 {
			masks[0][bx4+x][twl4c][0] |= inner1
		}
		if inner2 != 0 {
			masks[0][bx4+x][twl4c][1] |= inner2
		}
	}

	vstep := int(tDim.h)
	t = uint64(1) << bx4
	inner = uint32((t << w4) - t)
	inner1, inner2 = uint16(inner&0xffff), uint16(inner>>16)
	for y := vstep; y < h4; y += vstep {
		if inner1 != 0 {
			masks[1][by4+y][thl4c][0] |= inner1
		}
		if inner2 != 0 {
			masks[1][by4+y][thl4c][1] |= inner2
		}
	}

	memset(a[aOff:aOff+w4], uint8(thl4c))
	memset(l[lOff:lOff+h4], uint8(twl4c))
}

type txaBuf [2][2][32][32]uint8

func decompTx(txa *txaBuf, yOff4, xOff4, from, depth, yOff, xOff int,
	txMasks *[2]uint16,
) {
	tDim := &txfmDimensions[from]
	isSplit := 0
	if from != tx4x4 && depth <= 1 {
		isSplit = int(txMasks[depth]>>(yOff*4+xOff)) & 1
	}

	if isSplit != 0 {
		sub := int(tDim.sub)
		htw4, hth4 := int(tDim.w)>>1, int(tDim.h)>>1

		decompTx(txa, yOff4, xOff4, sub, depth+1, yOff*2+0, xOff*2+0, txMasks)
		if tDim.w >= tDim.h {
			decompTx(txa, yOff4, xOff4+htw4, sub, depth+1, yOff*2+0, xOff*2+1, txMasks)
		}
		if tDim.h >= tDim.w {
			decompTx(txa, yOff4+hth4, xOff4, sub, depth+1, yOff*2+1, xOff*2+0, txMasks)
			if tDim.w >= tDim.h {
				decompTx(txa, yOff4+hth4, xOff4+htw4, sub, depth+1,
					yOff*2+1, xOff*2+1, txMasks)
			}
		}

		return
	}

	lw, lh := min(2, int(tDim.lw)), min(2, int(tDim.lh))
	for y := range int(tDim.h) {
		memset(txa[0][0][yOff4+y][xOff4:xOff4+int(tDim.w)], uint8(lw))
		memset(txa[1][0][yOff4+y][xOff4:xOff4+int(tDim.w)], uint8(lh))
		txa[0][1][yOff4+y][xOff4] = tDim.w
	}
	memset(txa[1][1][yOff4][xOff4:xOff4+int(tDim.w)], tDim.h)
}

func maskEdgesInter(masks *[2][32][3][2]uint16, by4, bx4, w4, h4 int, skip bool,
	maxTx int, txMasks *[2]uint16, a, l []uint8, aOff, lOff int, txa *txaBuf,
) {
	tDim := &txfmDimensions[maxTx]

	for yOff, y := 0, 0; y < h4; y, yOff = y+int(tDim.h), yOff+1 {
		for xOff, x := 0, 0; x < w4; x, xOff = x+int(tDim.w), xOff+1 {
			decompTx(txa, y, x, maxTx, 0, yOff, xOff, txMasks)
		}
	}

	mask := uint32(1) << by4
	for y := range h4 {
		sidx := b2i(mask >= 0x10000)
		smask := mask >> (sidx << 4)
		masks[0][bx4][min(int(txa[0][0][y][0]), int(l[lOff+y]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	mask = uint32(1) << bx4
	for x := range w4 {
		sidx := b2i(mask >= 0x10000)
		smask := mask >> (sidx << 4)
		masks[1][by4][min(int(txa[1][0][0][x]), int(a[aOff+x]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	if !skip {
		mask = uint32(1) << by4
		for y := range h4 {
			sidx := b2i(mask >= 0x10000)
			smask := mask >> (sidx << 4)
			ltx := int(txa[0][0][y][0])
			step := int(txa[0][1][y][0])
			for x := step; x < w4; x += step {
				rtx := int(txa[0][0][y][x])
				masks[0][bx4+x][min(rtx, ltx)][sidx] |= uint16(smask)
				ltx = rtx
				step = int(txa[0][1][y][x])
			}
			mask <<= 1
		}

		mask = uint32(1) << bx4
		for x := range w4 {
			sidx := b2i(mask >= 0x10000)
			smask := mask >> (sidx << 4)
			ttx := int(txa[1][0][0][x])
			step := int(txa[1][1][0][x])
			for y := step; y < h4; y += step {
				btx := int(txa[1][0][y][x])
				masks[1][by4+y][min(ttx, btx)][sidx] |= uint16(smask)
				ttx = btx
				step = int(txa[1][1][y][x])
			}
			mask <<= 1
		}
	}

	for y := range h4 {
		l[lOff+y] = txa[0][0][y][w4-1]
	}
	copy(a[aOff:aOff+w4], txa[1][0][h4-1][:w4])
}

func maskEdgesChroma(masks *[2][32][2][2]uint16, cby4, cbx4, cw4, ch4 int,
	skipInter bool, tx int, a, l []uint8, aOff, lOff, ssHor, ssVer int,
) {
	tDim := &txfmDimensions[tx]
	twl4c, thl4c := b2i(tDim.lw != 0), b2i(tDim.lh != 0)
	vbits, hbits := 4-ssVer, 4-ssHor
	vmaskN, hmaskN := 16>>ssVer, 16>>ssHor
	vmax, hmax := uint32(1)<<vmaskN, uint32(1)<<hmaskN

	mask := uint32(1) << cby4
	for y := range ch4 {
		sidx := b2i(mask >= vmax)
		smask := mask >> (sidx << vbits)
		masks[0][cbx4][min(twl4c, int(l[lOff+y]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	mask = uint32(1) << cbx4
	for x := range cw4 {
		sidx := b2i(mask >= hmax)
		smask := mask >> (sidx << hbits)
		masks[1][cby4][min(thl4c, int(a[aOff+x]))][sidx] |= uint16(smask)
		mask <<= 1
	}

	if !skipInter {
		hstep := int(tDim.w)
		t := uint64(1) << cby4
		inner := uint32((t << ch4) - t)
		inner1 := uint16(inner & (1<<vmaskN - 1))
		inner2 := uint16(inner >> vmaskN)
		for x := hstep; x < cw4; x += hstep {
			if inner1 != 0 {
				masks[0][cbx4+x][twl4c][0] |= inner1
			}
			if inner2 != 0 {
				masks[0][cbx4+x][twl4c][1] |= inner2
			}
		}

		vstep := int(tDim.h)
		t = uint64(1) << cbx4
		inner = uint32((t << cw4) - t)
		inner1 = uint16(inner & (1<<hmaskN - 1))
		inner2 = uint16(inner >> hmaskN)
		for y := vstep; y < ch4; y += vstep {
			if inner1 != 0 {
				masks[1][cby4+y][thl4c][0] |= inner1
			}
			if inner2 != 0 {
				masks[1][cby4+y][thl4c][1] |= inner2
			}
		}
	}

	memset(a[aOff:aOff+cw4], uint8(thl4c))
	memset(l[lOff:lOff+ch4], uint8(twl4c))
}

func createLfMaskIntra(lflvl *av1Filter, levelCache []uint8, b4Stride int,
	filterLevel *[4][8][2]uint8, bx, by, iw, ih, bs, ytx, uvtx, layout int,
	ay, ly, auv, luv []uint8, ayOff, lyOff, auvOff, luvOff int, hasChroma bool,
) {
	bDim := &blockDimensions[bs]
	bw4 := min(iw-bx, int(bDim[0]))
	bh4 := min(ih-by, int(bDim[1]))
	bx4, by4 := bx&31, by&31

	if bw4 > 0 && bh4 > 0 {
		off := (by*b4Stride + bx) * 4
		for range bh4 {
			for x := range bw4 {
				levelCache[off+x*4+0] = filterLevel[0][0][0]
				levelCache[off+x*4+1] = filterLevel[1][0][0]
			}
			off += b4Stride * 4
		}

		maskEdgesIntra(&lflvl.filterY, by4, bx4, bw4, bh4, ytx, ay, ly, ayOff, lyOff)
	}

	if !hasChroma {
		return
	}

	ssVer := b2i(layout == pixelLayoutI420)
	ssHor := b2i(layout != pixelLayoutI444)
	cbw4 := min(((iw+ssHor)>>ssHor)-(bx>>ssHor), (int(bDim[0])+ssHor)>>ssHor)
	cbh4 := min(((ih+ssVer)>>ssVer)-(by>>ssVer), (int(bDim[1])+ssVer)>>ssVer)

	if cbw4 <= 0 || cbh4 <= 0 {
		return
	}

	cbx4, cby4 := bx4>>ssHor, by4>>ssVer

	off := ((by>>ssVer)*b4Stride + (bx >> ssHor)) * 4
	for range cbh4 {
		for x := range cbw4 {
			levelCache[off+x*4+2] = filterLevel[2][0][0]
			levelCache[off+x*4+3] = filterLevel[3][0][0]
		}
		off += b4Stride * 4
	}

	maskEdgesChroma(&lflvl.filterUv, cby4, cbx4, cbw4, cbh4, false, uvtx,
		auv, luv, auvOff, luvOff, ssHor, ssVer)
}

func createLfMaskInter(lflvl *av1Filter, levelCache []uint8, b4Stride int,
	filterLevel *[4]uint8, bx, by, iw, ih int, skip bool, bs, maxYtx int,
	txMasks *[2]uint16, uvtx, layout int,
	ay, ly, auv, luv []uint8, ayOff, lyOff, auvOff, luvOff int, hasChroma bool,
	txa *txaBuf,
) {
	bDim := &blockDimensions[bs]
	bw4 := min(iw-bx, int(bDim[0]))
	bh4 := min(ih-by, int(bDim[1]))
	bx4, by4 := bx&31, by&31

	if bw4 > 0 && bh4 > 0 {
		l0, l1 := filterLevel[0], filterLevel[1]
		off := (by*b4Stride + bx) * 4
		n := bw4 * 4
		for range bh4 {
			row := levelCache[off : off+n : off+n]
			for x := 0; x+1 < len(row); x += 4 {
				row[x] = l0
				row[x+1] = l1
			}
			off += b4Stride * 4
		}

		maskEdgesInter(&lflvl.filterY, by4, bx4, bw4, bh4, skip, maxYtx, txMasks,
			ay, ly, ayOff, lyOff, txa)
	}

	if !hasChroma {
		return
	}

	ssVer := b2i(layout == pixelLayoutI420)
	ssHor := b2i(layout != pixelLayoutI444)
	cbw4 := min(((iw+ssHor)>>ssHor)-(bx>>ssHor), (int(bDim[0])+ssHor)>>ssHor)
	cbh4 := min(((ih+ssVer)>>ssVer)-(by>>ssVer), (int(bDim[1])+ssVer)>>ssVer)

	if cbw4 <= 0 || cbh4 <= 0 {
		return
	}

	cbx4, cby4 := bx4>>ssHor, by4>>ssVer

	l2, l3 := filterLevel[2], filterLevel[3]
	off := ((by>>ssVer)*b4Stride + (bx >> ssHor)) * 4
	n := cbw4 * 4
	for range cbh4 {
		row := levelCache[off : off+n : off+n]
		for x := 2; x+1 < len(row); x += 4 {
			row[x] = l2
			row[x+1] = l3
		}
		off += b4Stride * 4
	}

	maskEdgesChroma(&lflvl.filterUv, cby4, cbx4, cbw4, cbh4, skip, uvtx,
		auv, luv, auvOff, luvOff, ssHor, ssVer)
}
