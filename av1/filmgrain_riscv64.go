//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func fgApplyRowRVV(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)

func fgApplyRow8RVV(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	fgApplyRowRVV(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
		n, 15-shift, minValue, maxValue)
}
