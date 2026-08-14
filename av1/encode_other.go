//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package av1

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func sseKernel(dst []uint8, off, stride int, src []uint8, sstride, w, h int) int64 {
	return sseGo(dst, off, stride, src, sstride, w, h)
}

func satd4Kernel(res []int32, off, stride int) int32 {
	return satd4(res, off, stride)
}

func fwdTxfm8x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm4x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm8x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm8x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm16x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm16x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm32x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func satd8Kernel(res []int32, off, stride int) int32 {
	return satd8(res, off, stride)
}

func residualKernel(res []int32, dst []uint8, off, stride int, src []uint8,
	sstride, w, h int,
) int64 {
	return residualGo(res, dst, off, stride, src, sstride, w, h)
}

func fwdTxfm16x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}

func fwdTxfm32x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	return false
}
