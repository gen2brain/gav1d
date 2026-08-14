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

//go:noescape
func fwdTxfm8x8NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x8NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm4x8NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm4x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm4x8NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm8x4NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x4(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x4NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm8x16NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm8x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm8x16NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm16x8NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm16x8(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x8NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
}

//go:noescape
func fwdTxfm16x16NEON(res *int32, src *int32, srcStride, colAdst, rowAdst int)

func fwdTxfm16x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp > adstAdst {
		return false
	}
	col := b2i(tx1dTypes[txtp][0] == tx1dAdst)
	row := b2i(tx1dTypes[txtp][1] == tx1dAdst)
	fwdTxfm16x16NEON(&coeff[0], &src[0], srcStride, col, row)

	return true
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

//go:noescape
func fwdTxfm32x32NEON(res *int32, src *int32, srcStride int)

func fwdTxfm32x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm32x32NEON(&coeff[0], &src[0], srcStride)

	return true
}

//go:noescape
func fwdTxfm16x32NEON(res *int32, src *int32, srcStride int)

func fwdTxfm16x32(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm16x32NEON(&coeff[0], &src[0], srcStride)

	return true
}

//go:noescape
func fwdTxfm32x16NEON(res *int32, src *int32, srcStride int)

func fwdTxfm32x16(coeff []int32, src []int32, srcStride, txtp int) bool {
	if txtp != dctDct {
		return false
	}
	fwdTxfm32x16NEON(&coeff[0], &src[0], srcStride)

	return true
}
