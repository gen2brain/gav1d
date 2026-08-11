package av1

const mcMidStride = 128

func intermediateBits(bitdepthMax int32) int {
	if bitdepthFromMax(bitdepthMax) == 8 {
		return 4
	}

	return 14 - bitdepthFromMax(bitdepthMax)
}

func prepBias(bitdepthMax int32) int32 {
	if bitdepthFromMax(bitdepthMax) == 8 {
		return 0
	}

	return 8192
}

func hFilter(mx, w, filterType int) *[8]int8 {
	if mx == 0 {
		return nil
	}
	if w > 4 {
		return &mcSubpelFilters[filterType&3][mx-1]
	}

	return &mcSubpelFilters[3+(filterType&1)][mx-1]
}

func vFilter(my, h, filterType int) *[8]int8 {
	if my == 0 {
		return nil
	}
	if h > 4 {
		return &mcSubpelFilters[filterType>>2][my-1]
	}

	return &mcSubpelFilters[3+((filterType>>2)&1)][my-1]
}

func filter8tapP[P pixel](src []P, off int, f *[8]int8, stride int) int32 {
	sum := int32(0)
	for i := range 8 {
		sum += int32(f[i]) * int32(src[off+(i-3)*stride])
	}

	return sum
}

func filter8tapI(src []int16, off int, f *[8]int8, stride int) int32 {
	sum := int32(0)
	for i := range 8 {
		sum += int32(f[i]) * int32(src[off+(i-3)*stride])
	}

	return sum
}

func mcPut[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride, w, h int) {
	if mcPutFast(dst, dstOff, dstStride, src, srcOff, srcStride, w, h) {
		return
	}

	switch w {
	case 2:
		for range h {
			*(*[2]P)(dst[dstOff : dstOff+2]) = *(*[2]P)(src[srcOff : srcOff+2])
			dstOff += dstStride
			srcOff += srcStride
		}

	case 4:
		for range h {
			*(*[4]P)(dst[dstOff : dstOff+4]) = *(*[4]P)(src[srcOff : srcOff+4])
			dstOff += dstStride
			srcOff += srcStride
		}

	case 8:
		for range h {
			*(*[8]P)(dst[dstOff : dstOff+8]) = *(*[8]P)(src[srcOff : srcOff+8])
			dstOff += dstStride
			srcOff += srcStride
		}

	case 16:
		for range h {
			*(*[16]P)(dst[dstOff : dstOff+16]) = *(*[16]P)(src[srcOff : srcOff+16])
			dstOff += dstStride
			srcOff += srcStride
		}

	case 32:
		for range h {
			*(*[32]P)(dst[dstOff : dstOff+32]) = *(*[32]P)(src[srcOff : srcOff+32])
			dstOff += dstStride
			srcOff += srcStride
		}

	default:
		for range h {
			copy(dst[dstOff:dstOff+w], src[srcOff:srcOff+w])
			dstOff += dstStride
			srcOff += srcStride
		}
	}
}

func mcPrep[P pixel](tmp []int16, src []P, srcOff, srcStride, w, h int, bitdepthMax int32) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)

	tmpOff := 0
	for range h {
		for x := range w {
			tmp[tmpOff+x] = int16(int32(src[srcOff+x])<<ib - bias)
		}
		tmpOff += w
		srcOff += srcStride
	}
}

func put8tap[P pixel](_ []int16, dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	intermediateRnd := int32(32 + ((1 << (6 - ib)) >> 1))

	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	switch {
	case fh != nil && fv != nil:
		var mid [mcMidStride * 135]int16
		midOff := 0
		srcOff -= srcStride * 3
		for range h + 7 {
			for x := range w {
				mid[midOff+x] = int16(rnd8(filter8tapP(src, srcOff+x, fh, 1), 6-ib))
			}
			midOff += mcMidStride
			srcOff += srcStride
		}

		midOff = mcMidStride * 3
		for range h {
			for x := range w {
				v := rnd8(filter8tapI(mid[:], midOff+x, fv, mcMidStride), 6+ib)
				dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			}
			midOff += mcMidStride
			dstOff += dstStride
		}

	case fh != nil:
		for range h {
			for x := range w {
				v := (filter8tapP(src, srcOff+x, fh, 1) + intermediateRnd) >> 6
				dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			}
			dstOff += dstStride
			srcOff += srcStride
		}

	case fv != nil:
		for range h {
			for x := range w {
				v := rnd8(filter8tapP(src, srcOff+x, fv, srcStride), 6)
				dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			}
			dstOff += dstStride
			srcOff += srcStride
		}

	default:
		mcPut(dst, dstOff, dstStride, src, srcOff, srcStride, w, h)
	}
}

func prep8tap[P pixel](_ []int16, tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)

	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	tmpOff := 0

	switch {
	case fh != nil && fv != nil:
		var mid [mcMidStride * 135]int16
		midOff := 0
		srcOff -= srcStride * 3
		for range h + 7 {
			for x := range w {
				mid[midOff+x] = int16(rnd8(filter8tapP(src, srcOff+x, fh, 1), 6-ib))
			}
			midOff += mcMidStride
			srcOff += srcStride
		}

		midOff = mcMidStride * 3
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filter8tapI(mid[:], midOff+x, fv, mcMidStride), 6) - bias)
			}
			midOff += mcMidStride
			tmpOff += w
		}

	case fh != nil:
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filter8tapP(src, srcOff+x, fh, 1), 6-ib) - bias)
			}
			tmpOff += w
			srcOff += srcStride
		}

	case fv != nil:
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filter8tapP(src, srcOff+x, fv, srcStride), 6-ib) - bias)
			}
			tmpOff += w
			srcOff += srcStride
		}

	default:
		mcPrep(tmp, src, srcOff, srcStride, w, h, bitdepthMax)
	}
}

func rnd8(v int32, sh int) int32 {
	return (v + ((1 << sh) >> 1)) >> sh
}

func filterBilinP[P pixel](src []P, off, mxy, stride int) int32 {
	return 16*int32(src[off]) + int32(mxy)*(int32(src[off+stride])-int32(src[off]))
}

func filterBilinI(src []int16, off, mxy, stride int) int32 {
	return 16*int32(src[off]) + int32(mxy)*(int32(src[off+stride])-int32(src[off]))
}

func putBilin[P pixel](_ []int16, dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	intermediateRnd := int32((1 << ib) >> 1)

	switch {
	case mx != 0 && my != 0:
		var mid [mcMidStride * 129]int16
		midOff := 0
		for range h + 1 {
			for x := range w {
				mid[midOff+x] = int16(rnd8(filterBilinP(src, srcOff+x, mx, 1), 4-ib))
			}
			midOff += mcMidStride
			srcOff += srcStride
		}

		midOff = 0
		for range h {
			for x := range w {
				v := rnd8(filterBilinI(mid[:], midOff+x, my, mcMidStride), 4+ib)
				dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			}
			midOff += mcMidStride
			dstOff += dstStride
		}

	case mx != 0:
		for range h {
			for x := range w {
				px := rnd8(filterBilinP(src, srcOff+x, mx, 1), 4-ib)
				dst[dstOff+x] = P(clip((px+intermediateRnd)>>ib, 0, bitdepthMax))
			}
			dstOff += dstStride
			srcOff += srcStride
		}

	case my != 0:
		for range h {
			for x := range w {
				v := rnd8(filterBilinP(src, srcOff+x, my, srcStride), 4)
				dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			}
			dstOff += dstStride
			srcOff += srcStride
		}

	default:
		mcPut(dst, dstOff, dstStride, src, srcOff, srcStride, w, h)
	}
}

func prepBilin[P pixel](_ []int16, tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)

	tmpOff := 0

	switch {
	case mx != 0 && my != 0:
		var mid [mcMidStride * 129]int16
		midOff := 0
		for range h + 1 {
			for x := range w {
				mid[midOff+x] = int16(rnd8(filterBilinP(src, srcOff+x, mx, 1), 4-ib))
			}
			midOff += mcMidStride
			srcOff += srcStride
		}

		midOff = 0
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filterBilinI(mid[:], midOff+x, my, mcMidStride), 4) - bias)
			}
			midOff += mcMidStride
			tmpOff += w
		}

	case mx != 0:
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filterBilinP(src, srcOff+x, mx, 1), 4-ib) - bias)
			}
			tmpOff += w
			srcOff += srcStride
		}

	case my != 0:
		for range h {
			for x := range w {
				tmp[tmpOff+x] = int16(rnd8(filterBilinP(src, srcOff+x, my, srcStride), 4-ib) - bias)
			}
			tmpOff += w
			srcOff += srcStride
		}

	default:
		mcPrep(tmp, src, srcOff, srcStride, w, h, bitdepthMax)
	}
}

var filter2dFilterType = [9]int{0, 4, 8, 2, 6, 10, 1, 5, 9}

func mc[P pixel](dsp *dspContext[P], mid []int16, dst []P, dstOff, dstStride int,
	src []P, srcOff, srcStride, w, h, mx, my, filter2d int, bitdepthMax int32,
) {
	if filter2d == filter2dBilinear {
		dsp.putBilin(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	dsp.put8tap(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my,
		filter2dFilterType[filter2d], bitdepthMax)
}

func mct[P pixel](dsp *dspContext[P], mid []int16, tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my, filter2d int, bitdepthMax int32,
) {
	if filter2d == filter2dBilinear {
		dsp.prepBilin(mid, tmp, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	dsp.prep8tap(mid, tmp, src, srcOff, srcStride, w, h, mx, my,
		filter2dFilterType[filter2d], bitdepthMax)
}

func mcAvg[P pixel](dst []P, dstOff, dstStride int, tmp1, tmp2 []int16, w, h int,
	bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	sh := ib + 1
	rnd := int32(1<<ib) + prepBias(bitdepthMax)*2

	t := 0
	for range h {
		for x := range w {
			v := (int32(tmp1[t+x]) + int32(tmp2[t+x]) + rnd) >> sh
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
		}
		t += w
		dstOff += dstStride
	}
}

func mcWAvg[P pixel](dst []P, dstOff, dstStride int, tmp1, tmp2 []int16, w, h,
	weight int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	sh := ib + 4
	rnd := int32(8<<ib) + prepBias(bitdepthMax)*16

	t := 0
	for range h {
		for x := range w {
			v := (int32(tmp1[t+x])*int32(weight) +
				int32(tmp2[t+x])*int32(16-weight) + rnd) >> sh
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
		}
		t += w
		dstOff += dstStride
	}
}

func mcMask[P pixel](dst []P, dstOff, dstStride int, tmp1, tmp2 []int16, w, h int,
	mask []uint8, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	sh := ib + 6
	rnd := int32(32<<ib) + prepBias(bitdepthMax)*64

	t := 0
	for range h {
		for x := range w {
			m := int32(mask[t+x])
			v := (int32(tmp1[t+x])*m + int32(tmp2[t+x])*(64-m) + rnd) >> sh
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
		}
		t += w
		dstOff += dstStride
	}
}

func mcWMask[P pixel](dst []P, dstOff, dstStride int, tmp1, tmp2 []int16, w, h int,
	mask []uint8, sign, ssHor, ssVer int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bitdepth := bitdepthFromMax(bitdepthMax)
	sh := ib + 6
	rnd := int32(32<<ib) + prepBias(bitdepthMax)*64
	maskSh := bitdepth + ib - 4
	maskRnd := int32(1) << (maskSh - 5)

	t, mOff := 0, 0
	for ; h > 0; h-- {
		for x := 0; x < w; x++ {
			d := int32(tmp1[t+x]) - int32(tmp2[t+x])
			m := min(38+((abs32(d)+maskRnd)>>maskSh), 64)
			dst[dstOff+x] = P(clip((d*m+int32(tmp2[t+x])*64+rnd)>>sh, 0, bitdepthMax))

			if ssHor == 0 {
				mask[mOff+x] = uint8(m)

				continue
			}

			x++
			d = int32(tmp1[t+x]) - int32(tmp2[t+x])
			n := min(38+((abs32(d)+maskRnd)>>maskSh), 64)
			dst[dstOff+x] = P(clip((d*n+int32(tmp2[t+x])*64+rnd)>>sh, 0, bitdepthMax))

			switch {
			case h&ssVer != 0:
				mask[mOff+(x>>1)] = uint8((m + n + int32(mask[mOff+(x>>1)]) + 2 - int32(sign)) >> 2)
			case ssVer != 0:
				mask[mOff+(x>>1)] = uint8(m + n)
			default:
				mask[mOff+(x>>1)] = uint8((m + n + 1 - int32(sign)) >> 1)
			}
		}
		t += w
		dstOff += dstStride
		if ssVer == 0 || h&1 != 0 {
			mOff += w >> ssHor
		}
	}
}

func blendPx(a, b, m int32) int32 {
	return (a*(64-m) + b*m + 32) >> 6
}

func mcBlend[P pixel](dst []P, dstOff, dstStride int, tmp []P, w, h int, mask []uint8) {
	t := 0
	for range h {
		for x := range w {
			dst[dstOff+x] = P(blendPx(int32(dst[dstOff+x]), int32(tmp[t+x]), int32(mask[t+x])))
		}
		dstOff += dstStride
		t += w
	}
}

func mcBlendV[P pixel](dst []P, dstOff, dstStride int, tmp []P, w, h int) {
	mask := obmcMasks[w:]
	n := (w * 3) >> 2

	t := 0
	for range h {
		for x := range n {
			dst[dstOff+x] = P(blendPx(int32(dst[dstOff+x]), int32(tmp[t+x]), int32(mask[x])))
		}
		dstOff += dstStride
		t += w
	}
}

func mcBlendH[P pixel](dst []P, dstOff, dstStride int, tmp []P, w, h int) {
	mask := obmcMasks[h:]
	h = (h * 3) >> 2

	t := 0
	for y := range h {
		m := int32(mask[y])
		for x := range w {
			dst[dstOff+x] = P(blendPx(int32(dst[dstOff+x]), int32(tmp[t+x]), m))
		}
		dstOff += dstStride
		t += w
	}
}

func filterWarpP[P pixel](src []P, off int, f *[8]int8, stride, sh int) int32 {
	sum := int32(0)
	for i := range 8 {
		sum += int32(f[i]) * int32(src[off+(i-3)*stride])
	}

	return (sum + ((1 << sh) >> 1)) >> sh
}

func filterWarpI(src []int16, off int, f *[8]int8, stride, sh int) int32 {
	sum := int32(0)
	for i := range 8 {
		sum += int32(f[i]) * int32(src[off+(i-3)*stride])
	}

	return (sum + ((1 << sh) >> 1)) >> sh
}

func warpAffine8x8[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride int,
	abcd *[4]int16, mx, my int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	var mid [15 * 8]int16

	midOff := 0
	srcOff -= 3 * srcStride
	for range 15 {
		tmx := mx
		for x := range 8 {
			f := &mcWarpFilter[64+((tmx+512)>>10)]
			mid[midOff+x] = int16(filterWarpP(src, srcOff+x, f, 1, 7-ib))
			tmx += int(abcd[0])
		}
		srcOff += srcStride
		midOff += 8
		mx += int(abcd[1])
	}

	midOff = 3 * 8
	for range 8 {
		tmy := my
		for x := range 8 {
			f := &mcWarpFilter[64+((tmy+512)>>10)]
			v := filterWarpI(mid[:], midOff+x, f, 8, 7+ib)
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
			tmy += int(abcd[2])
		}
		midOff += 8
		dstOff += dstStride
		my += int(abcd[3])
	}
}

func emuEdge[P pixel](bw, bh, iw, ih, x, y int, dst []P, dstOff, dstStride int,
	ref []P, refOff, refStride int,
) {
	refOff += clip(y, 0, ih-1)*refStride + clip(x, 0, iw-1)

	leftExt := clip(-x, 0, bw-1)
	rightExt := clip(x+bw-iw, 0, bw-1)
	topExt := clip(-y, 0, bh-1)
	bottomExt := clip(y+bh-ih, 0, bh-1)

	blk := dstOff + topExt*dstStride
	centerW := bw - leftExt - rightExt
	centerH := bh - topExt - bottomExt

	for range centerH {
		copy(dst[blk+leftExt:blk+leftExt+centerW], ref[refOff:refOff+centerW])
		if leftExt != 0 {
			fill(dst[blk:blk+leftExt], dst[blk+leftExt])
		}
		if rightExt != 0 {
			fill(dst[blk+leftExt+centerW:blk+leftExt+centerW+rightExt],
				dst[blk+leftExt+centerW-1])
		}
		refOff += refStride
		blk += dstStride
	}

	blk = dstOff + topExt*dstStride
	for range topExt {
		copy(dst[dstOff:dstOff+bw], dst[blk:blk+bw])
		dstOff += dstStride
	}

	dstOff += centerH * dstStride
	for range bottomExt {
		copy(dst[dstOff:dstOff+bw], dst[dstOff-dstStride:dstOff-dstStride+bw])
		dstOff += dstStride
	}
}

func filter8tapRows(mid []int16, ptrs *[8]int, off int, f *[8]int8) int32 {
	sum := int32(0)
	for i := range 8 {
		sum += int32(f[i]) * int32(mid[ptrs[i]+off])
	}

	return sum
}

func put8tapScaled[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy, filterType int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	intermediateRnd := int32(1<<ib) >> 1
	var mid [mcMidStride * 8]int16
	var midPtrs [8]int
	for i := range 8 {
		midPtrs[i] = mcMidStride * i
	}
	inY := -8
	srcOff -= srcStride * 3

	for range h {
		srcY := my >> 10
		fv := vFilter((my&0x3ff)>>6, h, filterType)

		for inY < srcY {
			imx, ioff := mx, 0
			midPtr := midPtrs[0]
			for i := range 7 {
				midPtrs[i] = midPtrs[i+1]
			}
			midPtrs[7] = midPtr

			for x := range w {
				fh := hFilter(imx>>6, w, filterType)
				if fh != nil {
					mid[midPtr+x] = int16(rnd8(filter8tapP(src, srcOff+ioff, fh, 1), 6-ib))
				} else {
					mid[midPtr+x] = int16(int32(src[srcOff+ioff]) << ib)
				}
				imx += dx
				ioff += imx >> 10
				imx &= 0x3ff
			}

			srcOff += srcStride
			inY++
		}

		for x := range w {
			var v int32
			if fv != nil {
				v = rnd8(filter8tapRows(mid[:], &midPtrs, x, fv), 6+ib)
			} else {
				v = (int32(mid[midPtrs[3]+x]) + intermediateRnd) >> ib
			}
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
		}

		my += dy
		dstOff += dstStride
	}
}

func prep8tapScaled[P pixel](tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy, filterType int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)
	var mid [mcMidStride * 8]int16
	var midPtrs [8]int
	for i := range 8 {
		midPtrs[i] = mcMidStride * i
	}
	inY := -8
	srcOff -= srcStride * 3
	tmpOff := 0

	for range h {
		srcY := my >> 10
		fv := vFilter((my&0x3ff)>>6, h, filterType)

		for inY < srcY {
			imx, ioff := mx, 0
			midPtr := midPtrs[0]
			for i := range 7 {
				midPtrs[i] = midPtrs[i+1]
			}
			midPtrs[7] = midPtr

			for x := range w {
				fh := hFilter(imx>>6, w, filterType)
				if fh != nil {
					mid[midPtr+x] = int16(rnd8(filter8tapP(src, srcOff+ioff, fh, 1), 6-ib))
				} else {
					mid[midPtr+x] = int16(int32(src[srcOff+ioff]) << ib)
				}
				imx += dx
				ioff += imx >> 10
				imx &= 0x3ff
			}

			srcOff += srcStride
			inY++
		}

		for x := range w {
			var v int32
			if fv != nil {
				v = rnd8(filter8tapRows(mid[:], &midPtrs, x, fv), 6)
			} else {
				v = int32(mid[midPtrs[3]+x])
			}
			tmp[tmpOff+x] = int16(v - bias)
		}

		my += dy
		tmpOff += w
	}
}

func filterBilin2(mid []int16, o1, o2, x, mxy int) int32 {
	return 16*int32(mid[o1+x]) + int32(mxy)*(int32(mid[o2+x])-int32(mid[o1+x]))
}

func putBilinScaled[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	var mid [mcMidStride * 2]int16
	inY := -2

	for range h {
		y := my >> 10
		mid1 := (y & 1) * mcMidStride
		mid2 := ((y + 1) & 1) * mcMidStride
		dmy := my & 0x3ff

		for inY < y {
			imx, ioff := mx, 0
			midPtr := (inY & 1) * mcMidStride
			for x := range w {
				mid[midPtr+x] = int16(rnd8(filterBilinP(src, srcOff+ioff, imx>>6, 1), 4-ib))
				imx += dx
				ioff += imx >> 10
				imx &= 0x3ff
			}
			srcOff += srcStride
			inY++
		}

		for x := range w {
			v := rnd8(filterBilin2(mid[:], mid1, mid2, x, dmy>>6), 4+ib)
			dst[dstOff+x] = P(clip(v, 0, bitdepthMax))
		}

		my += dy
		dstOff += dstStride
	}
}

func prepBilinScaled[P pixel](tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)
	var mid [mcMidStride * 2]int16
	inY := -2
	tmpOff := 0

	for range h {
		y := my >> 10
		mid1 := (y & 1) * mcMidStride
		mid2 := ((y + 1) & 1) * mcMidStride
		dmy := my & 0x3ff

		for inY < y {
			imx, ioff := mx, 0
			midPtr := (inY & 1) * mcMidStride
			for x := range w {
				mid[midPtr+x] = int16(rnd8(filterBilinP(src, srcOff+ioff, imx>>6, 1), 4-ib))
				imx += dx
				ioff += imx >> 10
				imx &= 0x3ff
			}
			srcOff += srcStride
			inY++
		}

		for x := range w {
			tmp[tmpOff+x] = int16(rnd8(filterBilin2(mid[:], mid1, mid2, x, dmy>>6), 4) - bias)
		}

		my += dy
		tmpOff += w
	}
}

func warpAffine8x8t[P pixel](tmp []int16, tmpStride int, src []P, srcOff, srcStride int,
	abcd *[4]int16, mx, my int, bitdepthMax int32,
) {
	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)
	var mid [15 * 8]int16

	midOff := 0
	srcOff -= 3 * srcStride
	for range 15 {
		tmx := mx
		for x := range 8 {
			f := &mcWarpFilter[64+((tmx+512)>>10)]
			mid[midOff+x] = int16(filterWarpP(src, srcOff+x, f, 1, 7-ib))
			tmx += int(abcd[0])
		}
		srcOff += srcStride
		midOff += 8
		mx += int(abcd[1])
	}

	midOff = 3 * 8
	tmpOff := 0
	for range 8 {
		tmy := my
		for x := range 8 {
			f := &mcWarpFilter[64+((tmy+512)>>10)]
			tmp[tmpOff+x] = int16(filterWarpI(mid[:], midOff+x, f, 8, 7) - bias)
			tmy += int(abcd[2])
		}
		midOff += 8
		tmpOff += tmpStride
		my += int(abcd[3])
	}
}

func mcScaled[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy, filter2d int, bitdepthMax int32,
) {
	if filter2d == filter2dBilinear {
		putBilinScaled(dst, dstOff, dstStride, src, srcOff, srcStride,
			w, h, mx, my, dx, dy, bitdepthMax)

		return
	}

	put8tapScaled(dst, dstOff, dstStride, src, srcOff, srcStride,
		w, h, mx, my, dx, dy, filter2dFilterType[filter2d], bitdepthMax)
}

func mctScaled[P pixel](tmp []int16, src []P, srcOff, srcStride,
	w, h, mx, my, dx, dy, filter2d int, bitdepthMax int32,
) {
	if filter2d == filter2dBilinear {
		prepBilinScaled(tmp, src, srcOff, srcStride, w, h, mx, my, dx, dy, bitdepthMax)

		return
	}

	prep8tapScaled(tmp, src, srcOff, srcStride, w, h, mx, my, dx, dy,
		filter2dFilterType[filter2d], bitdepthMax)
}
