//go:build amd64 && !noasm

package av1

//go:noescape
func put8tapHAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh *[8]int8, fv *[8]int16, w, h int)

// mcMidCols is the strip the two-dimensional filter works in, kept narrow so
// the scratch is cheap to clear and stays in L1 between the two passes.
const mcMidCols = 16

//go:noescape
func put8tapH4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapV4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHV4AVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh *[8]int8, fv *[8]int16, w, h int)

//go:noescape
func prep8tapHAVX2(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapVAVX2(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapHVAVX2(dst *int16, src *uint8, srcStride int, mid *int16,
	fh *[8]int8, fv *[8]int16, w, h int)

//go:noescape
func putBilinHAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func putBilinVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func putBilinHVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	mx, my int32, w, h int)

//go:noescape
func prepBilinHAVX2(dst *int16, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func prepBilinVAVX2(dst *int16, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func prepBilinHVAVX2(dst *int16, src *uint8, srcStride int, mid *int16,
	mx, my int32, w, h int)

func putBilin8AVX2(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8,
	srcOff, srcStride, w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		putBilin(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		putBilinHVAVX2(&dst[dstOff], dstStride, &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		putBilinHAVX2(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(mx), w, h)
	default:
		putBilinVAVX2(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prepBilin8AVX2(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		prepBilin(mid, tmp, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		prepBilinHVAVX2(&tmp[0], &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		prepBilinHAVX2(&tmp[0], &src[srcOff], srcStride, int32(mx), w, h)
	default:
		prepBilinVAVX2(&tmp[0], &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prep8tap8AVX2(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 4 && bitdepthMax == 0xff {
		switch {
		case fh != nil && fv != nil:
			var fvw [8]int16
			for i, v := range fv {
				fvw[i] = int16(v)
			}

			prep8tapHVAVX2(&tmp[0], &src[srcOff-3*srcStride-3], srcStride,
				&mid[0], fh, &fvw, w, h)

			return
		case fh != nil:
			prep8tapHAVX2(&tmp[0], &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			prep8tapVAVX2(&tmp[0], &src[srcOff-3*srcStride], srcStride, fv, w, h)

			return
		}
	}

	prep8tap(mid, tmp, src, srcOff, srcStride, w, h, mx, my, filterType, bitdepthMax)
}

func put8tap8AVX2(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 2 {
		hv, hh, vv := put8tapHV4AVX2, put8tapH4AVX2, put8tapV4AVX2
		if w >= 8 {
			hv, hh, vv = put8tapHVAVX2, put8tapHAVX2, put8tapVAVX2
		}

		switch {
		case fh != nil && fv != nil:
			var fvw [8]int16
			for i, v := range fv {
				fvw[i] = int16(v)
			}

			hv(&dst[dstOff], dstStride, &src[srcOff-3*srcStride-3],
				srcStride, &mid[0], fh, &fvw, w, h)

			return
		case fh != nil:
			hh(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			vv(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride,
				fv, w, h)

			return
		}
	}

	put8tap(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my,
		filterType, bitdepthMax)
}
