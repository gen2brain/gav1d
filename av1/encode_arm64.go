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

func satd8Kernel(res []int32, off, stride int) int32 {
	return satd8NEON(&res[off], stride)
}

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool { return false }
