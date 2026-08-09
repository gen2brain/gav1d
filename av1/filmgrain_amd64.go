//go:build amd64 && !noasm

package av1

//go:noescape
func fgApplyRowAVX2(dst, src, scaling *uint8, grain *int16, n, mul, minValue, maxValue int)

func fgApplyRow8AVX2(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n < 16 {
		fgApplyRow(dst, dstOff, src, srcOff, scaling, grain, n, shift, minValue, maxValue)

		return
	}

	fgApplyRowAVX2(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
		n, 1<<(15-shift), minValue, maxValue)
}
