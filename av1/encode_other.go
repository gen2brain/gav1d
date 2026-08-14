//go:build !amd64 || noasm

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

func satd8Kernel(res []int32, off, stride int) int32 {
	return satd8(res, off, stride)
}

func residualKernel(res []int32, dst []uint8, off, stride int, src []uint8,
	sstride, w, h int,
) int64 {
	return residualGo(res, dst, off, stride, src, sstride, w, h)
}
