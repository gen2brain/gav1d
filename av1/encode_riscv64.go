//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func residualRVV(res *int32, dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func residualKernel(res []int32, dst []uint8, off, stride int, src []uint8,
	sstride, w, h int,
) int64 {
	return residualRVV(&res[0], &dst[off], stride, &src[0], sstride, w, h)
}

//go:noescape
func sseRVV(dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func sseKernel(dst []uint8, off, stride int, src []uint8, sstride, w, h int) int64 {
	return sseRVV(&dst[off], stride, &src[0], sstride, w, h)
}

//go:noescape
func satd4RVV(res *int32, stride int) int32

func satd4Kernel(res []int32, off, stride int) int32 {
	return satd4RVV(&res[off], stride)
}

func satd8Kernel(res []int32, off, stride int) int32 { return satd8(res, off, stride) }

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool { return false }
