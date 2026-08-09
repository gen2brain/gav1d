//go:build arm64 && !noasm

package av1

//go:noescape
func put8tapHNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)

//go:noescape
func put8tapHVNEON(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16,
	fh, fv *[8]int8, w, h int)

// mcMidCols is the strip the two-dimensional filter works in, kept narrow so
// the scratch is cheap to clear and stays in L1 between the two passes.
const mcMidCols = 8

func put8tap8NEON(dst []uint8, dstOff, dstStride int, src []uint8, srcOff, srcStride,
	w, h, mx, my, filterType int, bitdepthMax int32,
) {
	fh := hFilter(mx, w, filterType)
	fv := vFilter(my, h, filterType)

	if w >= 8 {
		switch {
		case fh != nil && fv != nil:
			var mid [135 * mcMidCols]int16

			put8tapHVNEON(&dst[dstOff], dstStride, &src[srcOff-3*srcStride-3],
				srcStride, &mid[0], fh, fv, w, h)

			return
		case fh != nil:
			put8tapHNEON(&dst[dstOff], dstStride, &src[srcOff-3], srcStride, fh, w, h)

			return
		case fv != nil:
			put8tapVNEON(&dst[dstOff], dstStride, &src[srcOff-3*srcStride], srcStride,
				fv, w, h)

			return
		}
	}

	put8tap(dst, dstOff, dstStride, src, srcOff, srcStride, w, h, mx, my,
		filterType, bitdepthMax)
}
