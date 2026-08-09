//go:build amd64 && !noasm

package av1

//go:noescape
func fgApplyRowAVX2(dst, src, scaling *uint8, grain *int16, n, mul, minValue, maxValue int)

func fgApplyRow8AVX2(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n < 16 {
		fgApplyRow(dst, dstOff, src, srcOff, scaling, grain, n, shift, minValue, maxValue)

		return
	}

	fgApplyRowAVX2(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
		n, 1<<(15-shift), minValue, maxValue)
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
func fguvApplyRowAVX2(dst, src, luma, scaling *uint8, grain *int16,
	n, mul, minValue, maxValue int, p *fguvParams)

func fguvApplyRow8AVX2(dst []uint8, dstOff int, src []uint8, srcOff int,
	luma []uint8, lumaOff int, scaling []uint8, grain []int16,
	n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool,
) {
	if n < 8 {
		fguvApplyRow(dst, dstOff, src, srcOff, luma, lumaOff, scaling, grain,
			n, shift, minValue, maxValue, sx, lumaMult, mult, offset, pixelMax, csfl)

		return
	}

	p := fguvParams{
		mults:    [2]int16{int16(lumaMult), int16(mult)},
		pixelMax: int16(pixelMax),
		offset:   int32(offset),
		sx:       int32(sx),
		csfl:     int32(b2i(csfl)),
	}
	fguvApplyRowAVX2(&dst[dstOff], &src[srcOff], &luma[lumaOff], &scaling[0],
		&grain[0], n, 1<<(15-shift), minValue, maxValue, &p)
}
