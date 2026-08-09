//go:build arm64 && !noasm

package av1

//go:noescape
func fgApplyRowNEON(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)

func fgApplyRow8NEON(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n < 8 {
		fgApplyRow(dst, dstOff, src, srcOff, scaling, grain, n, shift, minValue, maxValue)

		return
	}

	fgApplyRowNEON(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
		n, 15-shift, minValue, maxValue)
}
