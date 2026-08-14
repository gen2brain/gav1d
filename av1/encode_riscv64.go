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

//go:noescape
func satd8RVV(res *int32, stride int) int32

//go:noescape
func fwdTxfm8x8RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x8RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm4x8RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm4x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm4x8RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm8x4RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x4RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm8x16RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x16RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm16x8RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm16x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x8RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm16x16RVV(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm16x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x16RVV(&coeff[0], &src[0], srcStride, col, row)

	return true
}

func satd8Kernel(res []int32, off, stride int) int32 {
	return satd8RVV(&res[off], stride)
}

//go:noescape
func fwdTxfm4x4RVV(res *int32, src *int32, srcStride int, colAdst, rowAdst int)

func fwdTxfm4x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	fwdTxfm4x4RVV(&coeff[0], &src[0], srcStride,
		b2i(tx1dTypes[txtp][0] == tx1dAdst), b2i(tx1dTypes[txtp][1] == tx1dAdst))

	return true
}

//go:noescape
func fwdTxfm32x32RVV(res *int32, src *int32, srcStride int)

func fwdTxfm32x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm32x32RVV(&coeff[0], &src[0], srcStride)

	return true
}

//go:noescape
func fwdTxfm16x32RVV(res *int32, src *int32, srcStride int)

func fwdTxfm16x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm16x32RVV(&coeff[0], &src[0], srcStride)

	return true
}

//go:noescape
func fwdTxfm32x16RVV(res *int32, src *int32, srcStride int)

func fwdTxfm32x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm32x16RVV(&coeff[0], &src[0], srcStride)

	return true
}
