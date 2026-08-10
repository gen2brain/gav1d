//go:build arm64 && !noasm

package av1

//go:noescape
func put8tapHNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

//go:noescape
func put8tapH4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapV4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHV4NEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

//go:noescape
func prep8tapHNEON(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapVNEON(dst *int16, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func prep8tapHVNEON(dst *int16, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

//go:noescape
func putBilinHNEON(dst *uint8, dstStride int, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func putBilinVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func putBilinHVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)

//go:noescape
func prepBilinHNEON(dst *int16, src *uint8, srcStride int, mx int32, w, h int)

//go:noescape
func prepBilinVNEON(dst *int16, src *uint8, srcStride int, my int32, w, h int)

//go:noescape
func prepBilinHVNEON(dst *int16, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)

func putBilin8NEON(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8,
	srcOff, srcStride, w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		putBilin(mid, dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		putBilinHVNEON(&dst[dstOff], dstStride, &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		putBilinHNEON(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(mx), w, h)
	default:
		putBilinVNEON(&dst[dstOff], dstStride, &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prepBilin8NEON(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my int, bitdepthMax int32,
) {
	if w < 4 || bitdepthMax != 0xff || (mx == 0 && my == 0) {
		prepBilin(mid, tmp, src, srcOff, srcStride, w, h, mx, my, bitdepthMax)

		return
	}

	switch {
	case mx != 0 && my != 0:
		prepBilinHVNEON(&tmp[0], &src[srcOff], srcStride, &mid[0],
			int32(mx), int32(my), w, h)
	case mx != 0:
		prepBilinHNEON(&tmp[0], &src[srcOff], srcStride, int32(mx), w, h)
	default:
		prepBilinVNEON(&tmp[0], &src[srcOff], srcStride, int32(my), w, h)
	}
}

func prep8tap8NEON(mid []int16, tmp []int16, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 4 && bitdepthMax == 0xff {
		switch {
		case fh != nil && fv != nil:
			prep8tapHVNEON(&tmp[0], &src[srcOff-3*srcStride-3], srcStride,
				&mid[0], fh, fv, w, h)

			return
		case fh != nil:
			prep8tapHNEON(&tmp[0], &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			prep8tapVNEON(&tmp[0], &src[srcOff-3*srcStride], srcStride, fv, w, h)

			return
		}
	}

	prep8tap(mid, tmp, src, srcOff, srcStride, w, h, mx, my, filterType, bitdepthMax)
}

func put8tap8NEON(mid []int16, dst []uint8, dstOff, dstStride int, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 2 {
		hv, hh, vv := put8tapHV4NEON, put8tapH4NEON, put8tapV4NEON
		if w >= 8 {
			hv, hh, vv = put8tapHVNEON, put8tapHNEON, put8tapVNEON
		}

		switch {
		case fh != nil && fv != nil:

			hv(&dst[dstOff], dstStride, &src[srcOff-3*srcStride-3],
				srcStride, &mid[0], fh, fv, w, h)

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
