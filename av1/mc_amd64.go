//go:build amd64 && !noasm

package av1

//go:noescape
func put8tapHAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapVAVX2(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

func put8tap8AVX2(dst []uint8, dstOff, dstStride int, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 8 {
		switch {
		case fh != nil && fv != nil:
		case fh != nil:
			put8tapHAVX2(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			put8tapVAVX2(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride,
				fv, w, h)

			return
		}
	}

	put8tap(dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my,
		filterType, bitdepthMax)
}
