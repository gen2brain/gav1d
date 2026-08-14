//go:build amd64 && !noasm

package av1

//go:noescape
func satd8AVX2(res *int32, stride int) int32

//go:noescape
func residualAVX2(res *int32, dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func satd8Kernel(res []int32, off, stride int) int32 {
	if hasAVX2 {
		return satd8AVX2(&res[off], stride)
	}

	return satd8(res, off, stride)
}

func residualKernel(res []int32, dst []uint8, off, stride int, src []uint8,
	sstride, w, h int,
) int64 {
	if hasAVX2 {
		return residualAVX2(&res[0], &dst[off], stride, &src[0], sstride, w, h)
	}

	return residualGo(res, dst, off, stride, src, sstride, w, h)
}
