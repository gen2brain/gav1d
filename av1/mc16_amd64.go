//go:build amd64 && !noasm

package av1

//go:noescape
func mcTap16AVX2(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

//go:noescape
func mcTapMid16AVX2(dst *int16, dstStride int, src *uint16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

//go:noescape
func mcTapFromMid16AVX2(dst *uint16, dstStride int, src *int16, srcStride, tapStride int,
	f *[8]int32, w, h int, rnd int32, shift int, maxv int32)

func put8tap16AVX2(dst []uint16, dstOff, dstStride int, src []uint16, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w < 8 || w&7 != 0 || (fh == nil && fv == nil) {
		put8tap(dst, dstOff, dstStride, src, srcOff, srcStride,
			w, h, mx, my, filterType, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)

	switch {
	case fh != nil && fv != nil:
		var mid [135 * mcMidStride]int16
		ft := mcTaps(fh)
		mcTapMid16AVX2(&mid[0], mcMidStride, &src[srcOff-3*srcStride-3], srcStride, 1,
			&ft, w, h+7, int32(1)<<(5-ib), 6-ib, 0)
		ft = mcTaps(fv)
		mcTapFromMid16AVX2(&dst[dstOff], dstStride, &mid[0], mcMidStride,
			mcMidStride, &ft, w, h, int32(1)<<(5+ib), 6+ib, bitdepthMax)

	case fh != nil:
		ft := mcTaps(fh)
		mcTap16AVX2(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, 1,
			&ft, w, h, int32(32+((1<<(6-ib))>>1)), 6, bitdepthMax)

	default:
		ft := mcTaps(fv)
		mcTap16AVX2(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride, srcStride,
			&ft, w, h, 32, 6, bitdepthMax)
	}
}
