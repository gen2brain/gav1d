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
