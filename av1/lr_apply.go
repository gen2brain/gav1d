package av1

func lrStripe[P pixel, C coef](t *taskContext[P, C], p []P, pOff int,
	left []P, leftOff, x, y, plane, unitW, rowH int, lr *av1RestorationUnit, edges int,
) {
	f := t.f
	chroma := b2i(plane != 0)
	ssVer := chroma & b2i(f.layout == pixelLayoutI420)
	stride := f.srStride[chroma]
	lpfOff := f.lf.lrLpfLine[plane] + x

	sby := (y + (8<<ssVer)*b2i(y != 0)) >> (6 - ssVer + int(f.seqHdr.sb128))
	stripeH := min((64-8*b2i(y == 0))>>ssVer, rowH-y)

	params := &t.scratch.lrParams
	*params = looprestorationParams{}
	var fn func(lr *lrContext[P, C], dst []P, dstOff, stride int,
		left []P, leftOff int, lpf []P, lpfOff, w, h int,
		params *looprestorationParams, edges int, bitdepthMax int32)

	if lr.typ == restorationWiener {
		flt := &params.filter
		flt[0][0], flt[0][6] = int16(lr.filterH[0]), int16(lr.filterH[0])
		flt[0][1], flt[0][5] = int16(lr.filterH[1]), int16(lr.filterH[1])
		flt[0][2], flt[0][4] = int16(lr.filterH[2]), int16(lr.filterH[2])
		flt[0][3] = -(flt[0][0] + flt[0][1] + flt[0][2]) * 2
		if f.bpc != 8 {
			flt[0][3] += 128
		}

		flt[1][0], flt[1][6] = int16(lr.filterV[0]), int16(lr.filterV[0])
		flt[1][1], flt[1][5] = int16(lr.filterV[1]), int16(lr.filterV[1])
		flt[1][2], flt[1][4] = int16(lr.filterV[2]), int16(lr.filterV[2])
		flt[1][3] = 128 - (flt[1][0]+flt[1][1]+flt[1][2])*2

		fn = func(lr *lrContext[P, C], dst []P, dstOff, stride int,
			left []P, leftOff int, lpf []P, lpfOff, w, h int,
			params *looprestorationParams, edges int, bitdepthMax int32,
		) {
			lrWiener(lr, &t.scratch.lrw, dst, dstOff, stride, left, leftOff, lpf, lpfOff,
				w, h, params, edges, bitdepthMax)
		}
	} else {
		sgrIdx := int(lr.typ) - restorationSgrproj
		sp := &sgrParams[sgrIdx]
		params.sgr.s0 = uint32(sp[0])
		params.sgr.s1 = uint32(sp[1])
		params.sgr.w0 = int16(lr.sgrWeights[0])
		params.sgr.w1 = 128 - int16(lr.sgrWeights[0]+lr.sgrWeights[1])

		switch b2i(sp[0] != 0) + b2i(sp[1] != 0)*2 - 1 {
		case 0:
			fn = lrSgr5x5[P, C]
		case 1:
			fn = lrSgr3x3[P, C]
		default:
			fn = lrSgrMix[P, C]
		}
	}

	for y+stripeH <= rowH {
		if sby+1 != f.sbh || y+stripeH != rowH {
			edges |= lrHaveBottom
		} else {
			edges &^= lrHaveBottom
		}

		fn(f.lr, p, pOff, stride, left, leftOff, f.lf.lrLineBuf, lpfOff,
			unitW, stripeH, params, edges, f.bitdepthMax)

		leftOff += stripeH * 4
		y += stripeH
		pOff += stripeH * stride
		edges |= lrHaveTop
		stripeH = min(64>>ssVer, rowH-y)
		if stripeH == 0 {
			break
		}
		lpfOff += 4 * stride
	}
}

func backup4xU[P pixel](dst []P, dstOff int, src []P, srcOff, srcStride, u int) {
	for range u {
		copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		dstOff += 4
		srcOff += srcStride
	}
}

func lrSbrowPlane[P pixel, C coef](t *taskContext[P, C], pOff, y, w, h, rowH, plane int) {
	f := t.f
	chroma := b2i(plane != 0)
	ssVer := chroma & b2i(f.layout == pixelLayoutI420)
	ssHor := chroma & b2i(f.layout != pixelLayoutI444)
	pStride := f.srStride[chroma]

	unitSizeLog2 := int(f.frameHdr.restoration.unitSize[b2i(plane != 0)])
	unitSize := 1 << unitSizeLog2
	halfUnitSize := unitSize >> 1
	maxUnitSize := unitSize + halfUnitSize

	rowY := y + (8>>ssVer)*b2i(y != 0)
	shiftHor := 7 - ssHor

	preLrBorder := &t.scratch.lrBorder
	*preLrBorder = [2][(128 + 8) * 4]P{}
	var lr [2]*av1RestorationUnit

	edges := lrHaveRight
	if y > 0 {
		edges |= lrHaveTop
	}

	alignedUnitPos := rowY &^ (unitSize - 1)
	if alignedUnitPos != 0 && alignedUnitPos+halfUnitSize > h {
		alignedUnitPos -= unitSize
	}
	alignedUnitPos <<= ssVer
	sbIdx := (alignedUnitPos >> 7) * f.srSb128w
	unitIdx := ((alignedUnitPos >> 6) & 1) << 1
	lr[0] = &f.lrMask[sbIdx].lr[plane][unitIdx]
	restore := lr[0].typ != restorationNone
	x, bit := 0, 0
	backupH := rowH - y

	for x+maxUnitSize <= w {
		nextX := x + unitSize
		nextUIdx := unitIdx + ((nextX >> (shiftHor - 1)) & 1)
		lr[1-bit] = &f.lrMask[sbIdx+(nextX>>shiftHor)].lr[plane][nextUIdx]
		restoreNext := lr[1-bit].typ != restorationNone
		if restoreNext {
			backup4xU(preLrBorder[bit][:], 0, f.srCur[plane], pOff+unitSize-4, pStride, backupH)
		}
		if restore {
			lrStripe(t, f.srCur[plane], pOff, preLrBorder[1-bit][:], 0, x, y, plane,
				unitSize, rowH, lr[bit], edges)
		}
		x = nextX
		restore = restoreNext

		pOff += unitSize
		edges |= lrHaveLeft
		bit ^= 1
	}

	if restore {
		edges &^= lrHaveRight
		lrStripe(t, f.srCur[plane], pOff, preLrBorder[1-bit][:], 0, x, y, plane,
			w-x, rowH, lr[bit], edges)
	}
}

func lrSbrow[P pixel, C coef](t *taskContext[P, C], sby int) {
	f := t.f
	offsetY := 8 * b2i(sby != 0)
	restorePlanes := f.lf.restorePlanes
	sb128 := int(f.seqHdr.sb128)
	notLast := b2i(sby+1 < f.sbh)
	y := sby * f.sbStep * 4

	if restorePlanes&lrRestoreY != 0 {
		h := f.frameHdr.height
		w := f.srWidth
		rowH := min((sby+1)<<(6+sb128)-8*notLast, h)
		yStripe := (sby << (6 + sb128)) - offsetY
		lrSbrowPlane(t, (y-offsetY)*f.srStride[0], yStripe, w, h, rowH, 0)
	}

	if restorePlanes&(lrRestoreU|lrRestoreV) == 0 {
		return
	}

	ssVer, ssHor := f.ssVer, f.ssHor
	h := (f.frameHdr.height + ssVer) >> ssVer
	w := (f.srWidth + ssHor) >> ssHor
	rowH := min((sby+1)<<((6-ssVer)+sb128)-(8>>ssVer)*notLast, h)
	offsetUv := offsetY >> ssVer
	yStripe := (sby << ((6 - ssVer) + sb128)) - offsetUv
	uvOff := (y * f.srStride[1] >> ssVer) - offsetUv*f.srStride[1]

	for pl := 1; pl <= 2; pl++ {
		if restorePlanes&(1<<pl) == 0 {
			continue
		}
		lrSbrowPlane(t, uvOff, yStripe, w, h, rowH, pl)
	}
}
