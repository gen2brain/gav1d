//go:build arm64 && !noasm

package av1

//go:noescape
func fgApplyRowNEON(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)

func fgApplyRow8NEON(dst []uint8, dstOff int, src []uint8, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n < 8 {
		fgApplyRow(dst, dstOff, src, srcOff, scaling, grain, n, shift, minValue, maxValue)

		return
	}

	fgApplyRowNEON(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
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
func fguvApplyRowNEON(dst, src, luma, scaling *uint8, grain *int16,
	n, lshift, minValue, maxValue int, p *fguvParams)

func fguvApplyRow8NEON(dst []uint8, dstOff int, src []uint8, srcOff int,
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
	fguvApplyRowNEON(&dst[dstOff], &src[srcOff], &luma[lumaOff], &scaling[0],
		&grain[0], n, 15-shift, minValue, maxValue, &p)
}
