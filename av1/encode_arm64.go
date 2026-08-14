//go:build arm64 && !noasm

package av1

//go:noescape
func residualNEON(res *int32, dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func residualKernel(res []int32, dst []uint8, off, stride int, src []uint8,
	sstride, w, h int,
) int64 {
	return residualNEON(&res[0], &dst[off], stride, &src[0], sstride, w, h)
}

//go:noescape
func sseNEON(dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func sseKernel(dst []uint8, off, stride int, src []uint8, sstride, w, h int) int64 {
	return sseNEON(&dst[off], stride, &src[0], sstride, w, h)
}

//go:noescape
func satd4NEON(res *int32, stride int) int32

func satd4Kernel(res []int32, off, stride int) int32 {
	return satd4NEON(&res[off], stride)
}

//go:noescape
func satd8NEON(res *int32, stride int) int32

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

func satd8Kernel(res []int32, off, stride int) int32 {
	return satd8NEON(&res[off], stride)
}

//go:noescape
func fwdTxfm4x4NEON(res *int32, src *int32, srcStride int, colAdst, rowAdst int)

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	fwdTxfm4x4NEON(&coeff[0], &src[0], srcStride,
		b2i(tx1dTypes[txtp][0] == tx1dAdst), b2i(tx1dTypes[txtp][1] == tx1dAdst))

	return true
}
