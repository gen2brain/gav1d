package av1

func filterPlaneColsY[P pixel, C coef](f *frameContext[P, C], haveLeft bool,
	lvl []uint8, lvlOff, b4Stride int, mask *[32][3][2]uint16,
	dst []P, dstOff, ls, w, starty4, endy4 int,
) {
	for x := range w {
		if !haveLeft && x == 0 {
			continue
		}
		var hmask [3]uint32
		if starty4 == 0 {
			hmask[0] = uint32(mask[x][0][0])
			hmask[1] = uint32(mask[x][1][0])
			hmask[2] = uint32(mask[x][2][0])
			if endy4 > 16 {
				hmask[0] |= uint32(mask[x][0][1]) << 16
				hmask[1] |= uint32(mask[x][1][1]) << 16
				hmask[2] |= uint32(mask[x][2][1]) << 16
			}
		} else {
			hmask[0] = uint32(mask[x][0][1])
			hmask[1] = uint32(mask[x][1][1])
			hmask[2] = uint32(mask[x][2][1])
		}
		loopFilterHSb128y(f.dsp, dst, dstOff+x*4, ls, &hmask, lvl, lvlOff+x*4,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
	}
}

func filterPlaneRowsY[P pixel, C coef](f *frameContext[P, C], haveTop bool,
	lvl []uint8, lvlOff, b4Stride int, mask *[32][3][2]uint16,
	dst []P, dstOff, ls, w, starty4, endy4 int,
) {
	for y := starty4; y < endy4; y, dstOff, lvlOff = y+1, dstOff+4*ls, lvlOff+b4Stride*4 {
		if !haveTop && y == 0 {
			continue
		}
		vmask := [3]uint32{
			uint32(mask[y][0][0]) | uint32(mask[y][0][1])<<16,
			uint32(mask[y][1][0]) | uint32(mask[y][1][1])<<16,
			uint32(mask[y][2][0]) | uint32(mask[y][2][1])<<16,
		}
		loopFilterVSb128y(f.dsp, dst, dstOff, ls, &vmask, lvl, lvlOff+1,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
	}
}

func filterPlaneColsUV[P pixel, C coef](f *frameContext[P, C], haveLeft bool,
	lvl []uint8, lvlOff, b4Stride int, mask *[32][2][2]uint16,
	u, v []P, uvOff, ls, w, starty4, endy4, ssVer int,
) {
	for x := range w {
		if !haveLeft && x == 0 {
			continue
		}
		var hmask [2]uint32
		if starty4 == 0 {
			hmask[0] = uint32(mask[x][0][0])
			hmask[1] = uint32(mask[x][1][0])
			if endy4 > 16>>ssVer {
				hmask[0] |= uint32(mask[x][0][1]) << (16 >> ssVer)
				hmask[1] |= uint32(mask[x][1][1]) << (16 >> ssVer)
			}
		} else {
			hmask[0] = uint32(mask[x][0][1])
			hmask[1] = uint32(mask[x][1][1])
		}
		loopFilterHSb128uv(f.dsp, u, uvOff+x*4, ls, &hmask, lvl, lvlOff+x*4+2,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
		loopFilterHSb128uv(f.dsp, v, uvOff+x*4, ls, &hmask, lvl, lvlOff+x*4+3,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
	}
}

func filterPlaneRowsUV[P pixel, C coef](f *frameContext[P, C], haveTop bool,
	lvl []uint8, lvlOff, b4Stride int, mask *[32][2][2]uint16,
	u, v []P, uvOff, ls, w, starty4, endy4, ssHor int,
) {
	offL := 0
	for y := starty4; y < endy4; y, offL, lvlOff = y+1, offL+4*ls, lvlOff+b4Stride*4 {
		if !haveTop && y == 0 {
			continue
		}
		vmask := [2]uint32{
			uint32(mask[y][0][0]) | uint32(mask[y][0][1])<<(16>>ssHor),
			uint32(mask[y][1][0]) | uint32(mask[y][1][1])<<(16>>ssHor),
		}
		loopFilterVSb128uv(f.dsp, u, uvOff+offL, ls, &vmask, lvl, lvlOff+2,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
		loopFilterVSb128uv(f.dsp, v, uvOff+offL, ls, &vmask, lvl, lvlOff+3,
			b4Stride, &f.lf.limLut, f.bitdepthMax)
	}
}

func loopfilterSbrowCols[P pixel, C coef](f *frameContext[P, C], sby int,
	startOfTileRow int,
) {
	isSb64 := b2i(f.seqHdr.sb128 == 0)
	starty4 := (sby & isSb64) << 4
	sbsz := 32 >> isSb64
	sbl2 := 5 - isSb64
	halign := (f.bh + 31) &^ 31
	ssVer, ssHor := f.ssVer, f.ssHor
	vmax := uint32(1) << (16 >> ssVer)
	hmax := uint32(1) << (16 >> ssHor)
	endy4 := starty4 + min(f.h4-sby*sbsz, sbsz)
	uvEndy4 := (endy4 + ssVer) >> ssVer

	maskOff := (sby >> isSb64) * f.sb128w

	lpfYOff := sby << sbl2
	lpfUvOff := sby << (sbl2 - ssVer)
	for tileCol := 1; ; tileCol++ {
		x := int(f.frameHdr.tiling.colStartSb[tileCol])
		if x<<sbl2 >= f.bw {
			break
		}
		bx4 := 0
		if x&isSb64 != 0 {
			bx4 = 16
		}
		cbx4 := bx4 >> ssHor
		x >>= isSb64

		yh := &f.lflvl[maskOff+x].filterY[0][bx4]
		for y, mask := starty4, uint32(1)<<starty4; y < endy4; y, mask = y+1, mask<<1 {
			sidx := b2i(mask >= 0x10000)
			smask := uint16(mask >> (sidx << 4))
			idx := 2*b2i(yh[2][sidx]&smask != 0) + b2i(yh[1][sidx]&smask != 0)
			yh[2][sidx] &^= smask
			yh[1][sidx] &^= smask
			yh[0][sidx] &^= smask
			yh[min(idx, int(f.lf.txLpfRightEdge[0][lpfYOff+y-starty4]))][sidx] |= smask
		}

		if f.layout != pixelLayoutI400 {
			uvh := &f.lflvl[maskOff+x].filterUv[0][cbx4]
			for y, uvMask := starty4>>ssVer, uint32(1)<<(starty4>>ssVer); y < uvEndy4; y, uvMask = y+1, uvMask<<1 {
				sidx := b2i(uvMask >= vmax)
				smask := uint16(uvMask >> (sidx << (4 - ssVer)))
				idx := b2i(uvh[1][sidx]&smask != 0)
				uvh[1][sidx] &^= smask
				uvh[0][sidx] &^= smask
				uvh[min(idx, int(f.lf.txLpfRightEdge[1][lpfUvOff+y-(starty4>>ssVer)]))][sidx] |= smask
			}
		}
		lpfYOff += halign
		lpfUvOff += halign >> ssVer
	}

	if startOfTileRow != 0 {
		for x := range f.sb128w {
			a := &f.a[f.sb128w*(startOfTileRow-1)+x]
			yv := &f.lflvl[maskOff+x].filterY[1][starty4]
			w := min(32, f.w4-(x<<5))
			for i, mask := 0, uint32(1); i < w; i, mask = i+1, mask<<1 {
				sidx := b2i(mask >= 0x10000)
				smask := uint16(mask >> (sidx << 4))
				idx := 2*b2i(yv[2][sidx]&smask != 0) + b2i(yv[1][sidx]&smask != 0)
				yv[2][sidx] &^= smask
				yv[1][sidx] &^= smask
				yv[0][sidx] &^= smask
				yv[min(idx, int(a.txLpfY[i]))][sidx] |= smask
			}

			if f.layout != pixelLayoutI400 {
				cw := (w + ssHor) >> ssHor
				uvv := &f.lflvl[maskOff+x].filterUv[1][starty4>>ssVer]
				for i, uvMask := 0, uint32(1); i < cw; i, uvMask = i+1, uvMask<<1 {
					sidx := b2i(uvMask >= hmax)
					smask := uint16(uvMask >> (sidx << (4 - ssHor)))
					idx := b2i(uvv[1][sidx]&smask != 0)
					uvv[1][sidx] &^= smask
					uvv[0][sidx] &^= smask
					uvv[min(idx, int(a.txLpfUv[i]))][sidx] |= smask
				}
			}
		}
	}

	lvlOff := f.b4Stride * sby * sbsz * 4
	ptr := sby * f.sbStep * 4 * f.stride[0]
	for x := range f.sb128w {
		filterPlaneColsY(f, x != 0, f.lf.level, lvlOff, f.b4Stride,
			&f.lflvl[maskOff+x].filterY[0], f.cur[0], ptr, f.stride[0],
			min(32, f.w4-x*32), starty4, endy4)
		ptr += 128
		lvlOff += 32 * 4
	}

	if f.frameHdr.loopfilter.levelU == 0 && f.frameHdr.loopfilter.levelV == 0 {
		return
	}

	lvlOff = f.b4Stride * (sby * sbsz >> ssVer) * 4
	uvOff := (sby * f.sbStep * 4 >> ssVer) * f.stride[1]
	for x := range f.sb128w {
		filterPlaneColsUV(f, x != 0, f.lf.level, lvlOff, f.b4Stride,
			&f.lflvl[maskOff+x].filterUv[0], f.cur[1], f.cur[2], uvOff, f.stride[1],
			(min(32, f.w4-x*32)+ssHor)>>ssHor, starty4>>ssVer, uvEndy4, ssVer)
		uvOff += 128 >> ssHor
		lvlOff += (32 >> ssHor) * 4
	}
}

func loopfilterSbrowRows[P pixel, C coef](f *frameContext[P, C], sby int) {
	haveTop := sby > 0
	isSb64 := b2i(f.seqHdr.sb128 == 0)
	maskOff := (sby >> isSb64) * f.sb128w
	starty4 := (sby & isSb64) << 4
	sbsz := 32 >> isSb64
	ssVer, ssHor := f.ssVer, f.ssHor
	endy4 := starty4 + min(f.h4-sby*sbsz, sbsz)
	uvEndy4 := (endy4 + ssVer) >> ssVer

	lvlOff := f.b4Stride * sby * sbsz * 4
	ptr := sby * f.sbStep * 4 * f.stride[0]
	for x := range f.sb128w {
		filterPlaneRowsY(f, haveTop, f.lf.level, lvlOff, f.b4Stride,
			&f.lflvl[maskOff+x].filterY[1], f.cur[0], ptr, f.stride[0],
			min(32, f.w4-x*32), starty4, endy4)
		ptr += 128
		lvlOff += 32 * 4
	}

	if f.frameHdr.loopfilter.levelU == 0 && f.frameHdr.loopfilter.levelV == 0 {
		return
	}

	lvlOff = f.b4Stride * (sby * sbsz >> ssVer) * 4
	uvOff := (sby * f.sbStep * 4 >> ssVer) * f.stride[1]
	for x := range f.sb128w {
		filterPlaneRowsUV(f, haveTop, f.lf.level, lvlOff, f.b4Stride,
			&f.lflvl[maskOff+x].filterUv[1], f.cur[1], f.cur[2], uvOff, f.stride[1],
			(min(32, f.w4-x*32)+ssHor)>>ssHor, starty4>>ssVer, uvEndy4, ssHor)
		uvOff += 128 >> ssHor
		lvlOff += (32 >> ssHor) * 4
	}
}

func backupLpf[P pixel, C coef](f *frameContext[P, C], dst []P, dstOff, dstStride int,
	src []P, srcOff, srcStride, ssVer, ssHor, sb128, row, rowH, srcW, h int,
) {
	dstW := srcW
	if f.frameHdr.superRes.enabled != 0 {
		dstW = (f.srWidth + ssHor) >> ssHor
	}

	stripeH := (64 - 8*b2i(row == 0)) >> ssVer
	srcOff += (stripeH - 2) * srcStride

	if row != 0 {
		top := 4 << sb128
		for i := range 4 {
			copy(dst[dstOff+i*dstStride:][:dstW], dst[dstOff+(top+i)*dstStride:])
		}
	}
	dstOff += 4 * dstStride

	if f.resize {
		for row+stripeH <= rowH {
			nLines := 4 - b2i(row+stripeH+1 == h)
			resize(dst, dstOff, dstStride, src, srcOff, srcStride,
				dstW, nLines, srcW, f.resizeStep[ssHor], f.resizeStart[ssHor],
				f.bitdepthMax)
			row += stripeH
			stripeH = 64 >> ssVer
			srcOff += stripeH * srcStride
			dstOff += nLines * dstStride
			if nLines == 3 {
				copy(dst[dstOff:][:dstW], dst[dstOff-dstStride:])
				dstOff += dstStride
			}
		}

		return
	}

	for row+stripeH <= rowH {
		nLines := 4 - b2i(row+stripeH+1 == h)
		for i := range 4 {
			if i == nLines {
				copy(dst[dstOff:][:srcW], dst[dstOff-dstStride:])
			} else {
				copy(dst[dstOff:][:srcW], src[srcOff:])
			}
			dstOff += dstStride
			srcOff += srcStride
		}
		row += stripeH
		stripeH = 64 >> ssVer
		srcOff += (stripeH - 4) * srcStride
	}
}

func copyLpf[P pixel, C coef](f *frameContext[P, C], src *[3]int, sby int) {
	offset := 8 * b2i(sby != 0)
	sb128 := int(f.seqHdr.sb128)
	restorePlanes := f.lf.restorePlanes

	if f.seqHdr.cdef != 0 || restorePlanes&lrRestoreY != 0 {
		h := f.frameHdr.height
		w := f.bw << 2
		rowH := min((sby+1)<<(6+sb128), h-1)
		yStripe := (sby << (6 + sb128)) - offset
		if restorePlanes&lrRestoreY != 0 || !f.resize {
			backupLpf(f, f.lf.lrLineBuf, f.lf.lrLpfLine[0], f.srStride[0],
				f.cur[0], src[0]-offset*f.stride[0], f.stride[0],
				0, 0, sb128, yStripe, rowH, w, h)
		}
	}

	if (f.seqHdr.cdef == 0 && restorePlanes&(lrRestoreU|lrRestoreV) == 0) ||
		f.layout == pixelLayoutI400 {
		return
	}

	ssVer, ssHor := f.ssVer, f.ssHor
	h := (f.frameHdr.height + ssVer) >> ssVer
	w := f.bw << (2 - ssHor)
	rowH := min((sby+1)<<((6-ssVer)+sb128), h-1)
	offsetUv := offset >> ssVer
	yStripe := (sby << ((6 - ssVer) + sb128)) - offsetUv

	for pl := 1; pl <= 2; pl++ {
		if f.seqHdr.cdef == 0 && restorePlanes&(1<<pl) == 0 {
			continue
		}
		if restorePlanes&(1<<pl) == 0 && f.resize {
			continue
		}
		backupLpf(f, f.lf.lrLineBuf, f.lf.lrLpfLine[pl], f.srStride[1],
			f.cur[pl], src[pl]-offsetUv*f.stride[1], f.stride[1],
			ssVer, ssHor, sb128, yStripe, rowH, w, h)
	}
}
