//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func mcTap16RVV(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

//go:noescape
func mcTapMid16RVV(dst *int16, dstStride int, src *uint16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

//go:noescape
func mcTapFromMid16RVV(dst *uint16, dstStride int, src *int16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

func put8tap16RVV(mid []int16, dst []uint16, dstOff, dstStride int, src []uint16, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w < 1 || (fh == nil && fv == nil) {
		put8tap(mid, dst, dstOff, dstStride, src, srcOff, srcStride,
			w, h, mx, my, filterType, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)

	switch {
	case fh != nil && fv != nil:
		ft := mcTaps(fh)
		mcTapMid16RVV(&mid[0], mcMidStride, &src[srcOff-3*srcStride-3], srcStride, 1,
			&ft, w, h+7, int32(1)<<(5-ib), 6-ib, 0)
		ft = mcTaps(fv)
		mcTapFromMid16RVV(&dst[dstOff], dstStride, &mid[0], mcMidStride,
			mcMidStride, &ft, w, h, int32(1)<<(5+ib), 6+ib, bitdepthMax)

	case fh != nil:
		ft := mcTaps(fh)
		mcTap16RVV(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, 1,
			&ft, w, h, int32(32+((1<<(6-ib))>>1)), 6, bitdepthMax)

	default:
		ft := mcTaps(fv)
		mcTap16RVV(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride, srcStride,
			&ft, w, h, 32, 6, bitdepthMax)
	}
}

//go:noescape
func mcTapPrepMid16RVV(dst *int16, dstStride int, src *int16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

func prep8tap16RVV(mid []int16, tmp []int16, src []uint16, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w < 8 || w&7 != 0 || (fh == nil && fv == nil) {
		prep8tap(mid, tmp, src, srcOff, srcStride, w, h, mx, my, filterType, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)
	rnd := int32(1)<<(5-ib) - prepBias(bitdepthMax)<<(6-ib)

	switch {
	case fh != nil && fv != nil:
		ft := mcTaps(fh)
		mcTapMid16RVV(&mid[0], mcMidStride, &src[srcOff-3*srcStride-3], srcStride, 1,
			&ft, w, h+7, int32(1)<<(5-ib), 6-ib, 0)
		ft = mcTaps(fv)
		mcTapPrepMid16RVV(&tmp[0], w, &mid[0], mcMidStride, mcMidStride,
			&ft, w, h, 32-prepBias(bitdepthMax)<<6, 6, 0)

	case fh != nil:
		ft := mcTaps(fh)
		mcTapMid16RVV(&tmp[0], w, &src[srcOff-3], srcStride, 1, &ft, w, h, rnd, 6-ib, 0)

	default:
		ft := mcTaps(fv)
		mcTapMid16RVV(&tmp[0], w, &src[srcOff-3*srcStride], srcStride, srcStride,
			&ft, w, h, rnd, 6-ib, 0)
	}
}

//go:noescape
func mcBilin16RVV(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int,
	mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)

//go:noescape
func mcBilinMid16RVV(dst *int16, dstStride int, src *uint16, srcStride, tapStride int,
	mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)

//go:noescape
func mcBilinFromMid16RVV(dst *uint16, dstStride int, src *int16, srcStride, tapStride int,
	mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)

//go:noescape
func mcBilinPrepMid16RVV(dst *int16, dstStride int, src *int16, srcStride, tapStride int,
	mxy int32, w, h int, rnd int32, shift int, rnd2 int32, shift2 int, maxv int32)

func putBilin16RVV(mid []int16, dst []uint16, dstOff, dstStride int, src []uint16,
	srcOff, srcStride, w, h, mx, my int, bitdepthMax int32,
) {
	if w < 8 || w&7 != 0 || (mx == 0 && my == 0) {
		putBilin(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)
	hrnd := int32(1<<(4-ib)) >> 1

	switch {
	case mx != 0 && my != 0:
		mcBilinMid16RVV(&mid[0], mcMidStride, &src[srcOff], srcStride, 1,
			int32(mx), w, h+1, hrnd, 4-ib, 0, 0, 0)
		mcBilinFromMid16RVV(&dst[dstOff], dstStride, &mid[0], mcMidStride, mcMidStride,
			int32(my), w, h, int32(1<<(4+ib))>>1, 4+ib, 0, 0, bitdepthMax)

	case mx != 0:
		mcBilin16RVV(&dst[dstOff], dstStride, &src[srcOff], srcStride, 1,
			int32(mx), w, h, hrnd, 4-ib, int32(1<<ib)>>1, ib, bitdepthMax)

	default:
		mcBilin16RVV(&dst[dstOff], dstStride, &src[srcOff], srcStride, srcStride,
			int32(my), w, h, 8, 4, 0, 0, bitdepthMax)
	}
}

func prepBilin16RVV(mid []int16, tmp []int16, src []uint16, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	if w < 8 || w&7 != 0 || (mx == 0 && my == 0) {
		prepBilin(mid, tmp, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)
	bias := prepBias(bitdepthMax)
	hrnd := int32(1<<(4-ib)) >> 1
	rnd := hrnd - bias<<(4-ib)

	switch {
	case mx != 0 && my != 0:
		mcBilinMid16RVV(&mid[0], mcMidStride, &src[srcOff], srcStride, 1,
			int32(mx), w, h+1, hrnd, 4-ib, 0, 0, 0)
		mcBilinPrepMid16RVV(&tmp[0], w, &mid[0], mcMidStride, mcMidStride,
			int32(my), w, h, 8-bias<<4, 4, 0, 0, 0)

	case mx != 0:
		mcBilinMid16RVV(&tmp[0], w, &src[srcOff], srcStride, 1,
			int32(mx), w, h, rnd, 4-ib, 0, 0, 0)

	default:
		mcBilinMid16RVV(&tmp[0], w, &src[srcOff], srcStride, srcStride,
			int32(my), w, h, rnd, 4-ib, 0, 0, 0)
	}
}
