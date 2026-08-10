//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func put8tapHRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

//go:noescape
func prep8tapHRVV(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapVRVV(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapHVRVV(dst *int16, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

//go:noescape
func putBilinHRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func putBilinVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func putBilinHVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)

//go:noescape
func prepBilinHRVV(dst *int16, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func prepBilinVRVV(dst *int16, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func prepBilinHVRVV(dst *int16, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)

func putBilin8RVV(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8,
	srcOff, srcStride, w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		putBilin(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		putBilinHVRVV(&dst[dstOff], dstStride, &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		putBilinHRVV(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(mx), w, h)
	default:
		putBilinVRVV(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prepBilin8RVV(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		prepBilin(mid, tmp, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		prepBilinHVRVV(&tmp[0], &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		prepBilinHRVV(&tmp[0], &src[srcOff], srcStride, int32(mx), w, h)
	default:
		prepBilinVRVV(&tmp[0], &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prep8tap8RVV(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 4 && bitdepthMax == 0xff {
		switch {
		case fh != nil && fv != nil:
			prep8tapHVRVV(&tmp[0], &src[srcOff-3*srcStride-3], srcStride,
				&mid[0], fh, fv, w, h)

			return
		case fh != nil:
			prep8tapHRVV(&tmp[0], &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			prep8tapVRVV(&tmp[0], &src[srcOff-3*srcStride], srcStride, fv, w, h)

			return
		}
	}

	prep8tap(mid, tmp, src, srcOff, srcStride, w, h, mx, my, filterType, bitdepthMax)
}

func put8tap8RVV(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 2 {
		switch {
		case fh != nil && fv != nil:

			put8tapHVRVV(&dst[dstOff], dstStride, &src[srcOff-3*srcStride-3],
				srcStride, &mid[0], fh, fv, w, h)

			return
		case fh != nil:
			put8tapHRVV(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			put8tapVRVV(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride,
				fv, w, h)

			return
		}
	}

	put8tap(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my,
		filterType, bitdepthMax)
}
