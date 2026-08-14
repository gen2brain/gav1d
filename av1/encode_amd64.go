//go:build amd64 && !noasm

package av1

//go:noescape
func fwdTxfm8x8AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm16x32AVX512(res *int32, src *int32, srcStride int)

//go:noescape
func fwdTxfm32x16AVX512(res *int32, src *int32, srcStride int)

//go:noescape
func fwdTxfm32x32AVX512(res *int32, src *int32, srcStride int)

//go:noescape
func fwdTxfm16x16AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm16x8AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm8x16AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm4x8AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm8x4AVX512(res *int32, src *int32, srcStride, colAdst, rowAdst int)

//go:noescape
func fwdTxfm4x4AVX2(res *int32, src *int32, srcStride int, colAdst, rowAdst int)

//go:noescape
func sseAVX2(dst *uint8, stride int, src *uint8, sstride, w, h int) int64

//go:noescape
func satd4AVX2(res *int32, stride int) int32

//go:noescape
func satd8AVX2(res *int32, stride int) int32

//go:noescape
func residualAVX2(res *int32, dst *uint8, stride int, src *uint8, sstride, w, h int) int64

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX2 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm4x4AVX2(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func sseKernel(dst []uint8, off, stride int, src []uint8, sstride, w, h int) int64 {
	if hasAVX2 {
		return sseAVX2(&dst[off], stride, &src[0], sstride, w, h)
	}

	return sseGo(dst, off, stride, src, sstride, w, h)
}

func satd4Kernel(res []int32, off, stride int) int32 {
	if hasAVX2 {
		return satd4AVX2(&res[off], stride)
	}

	return satd4(res, off, stride)
}

func fwdTxfm8x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x8AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm4x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm4x8AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm8x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x4AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm8x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x16AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm16x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x8AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm16x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x16AVX512(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func fwdTxfm32x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp != dctDct {
		return false
	}
	fwdTxfm32x32AVX512(&coeff[0], &src[0], srcStride)

	return true
}

func fwdTxfm16x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp != dctDct {
		return false
	}
	fwdTxfm16x32AVX512(&coeff[0], &src[0], srcStride)

	return true
}

func fwdTxfm32x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if !hasAVX512 || txtp != dctDct {
		return false
	}
	fwdTxfm32x16AVX512(&coeff[0], &src[0], srcStride)

	return true
}

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
