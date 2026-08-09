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

func put8tap16RVV(dst []uint16, dstOff, dstStride int, src []uint16, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w < 1 || (fh == nil && fv == nil) {
		put8tap(dst, dstOff, dstStride, src, srcOff, srcStride,
			w, h, mx, my, filterType, bitdepthMax)

		return
	}

	ib := intermediateBits(bitdepthMax)

	switch {
	case fh != nil && fv != nil:
		var mid [135 * mcMidStride]int16
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
