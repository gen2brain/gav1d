//go:build !amd64 || noasm

package av1

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool {
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
