//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func fgApplyRowRVV(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)

func fgApplyRow8RVV(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	fgApplyRowRVV(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
		n, 15-shift, minValue, maxValue)
}

type fguvParams struct {
	mults    [2]int16
	pixelMax int16
	_        int16
	offset   int32
	sx       int32
	csfl     int32
}

//go:noescape
func fguvApplyRowRVV(dst, src, luma, scaling *uint8, grain *int16,
	n, lshift, minValue, maxValue int, p *fguvParams)

func fguvApplyRow8RVV(dst []uint8, dstOff int, src []uint8, srcOff int,
	luma []uint8, lumaOff int, scaling []uint8, grain []int16,
	n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool,
) {
	p := fguvParams{
		mults:    [2]int16{int16(lumaMult), int16(mult)},
		pixelMax: int16(pixelMax),
		offset:   int32(offset),
		sx:       int32(sx),
		csfl:     int32(b2i(csfl)),
	}
	fguvApplyRowRVV(&dst[dstOff], &src[srcOff], &luma[lumaOff], &scaling[0],
		&grain[0], n, 15-shift, minValue, maxValue, &p)
}
