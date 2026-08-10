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

//go:noescape
func fgApplyRow16AVX2(dst, src *uint16, scaling *uint8, grain *int16,
	n, shift, minValue, maxValue int)

func fgApplyRow16AVX2Row(dst []uint16, dstOff int, src []uint16, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n8 := n &^ 7; n8 > 0 {
		fgApplyRow16AVX2(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
			n8, shift, minValue, maxValue)
		dstOff, srcOff, grain, n = dstOff+n8, srcOff+n8, grain[n8:], n-n8
	}
	if n > 0 {
		fgApplyRow(dst, dstOff, src, srcOff, scaling, grain, n, shift, minValue, maxValue)
	}
}

//go:noescape
func fguvApplyRow16AVX2(dst, src, luma *uint16, scaling *uint8, grain *int16, p *fguv16Params)

func fguvApplyRow16AVX2Row(dst []uint16, dstOff int, src []uint16, srcOff int,
	luma []uint16, lumaOff int, scaling []uint8, grain []int16,
	n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool,
) {
	if n8 := n &^ 7; n8 > 0 {
		p := fguv16Params{
			n: n8, shift: shift, minValue: minValue, maxValue: maxValue,
			csfl: b2i(csfl), lumaMult: lumaMult, mult: mult, offset: offset,
			pixelMax: pixelMax, sx: sx,
		}
		fguvApplyRow16AVX2(&dst[dstOff], &src[srcOff], &luma[lumaOff],
			&scaling[0], &grain[0], &p)
		dstOff, srcOff, lumaOff = dstOff+n8, srcOff+n8, lumaOff+(n8<<sx)
		grain, n = grain[n8:], n-n8
	}
	if n > 0 {
		fguvApplyRow(dst, dstOff, src, srcOff, luma, lumaOff, scaling, grain,
			n, shift, minValue, maxValue, sx, lumaMult, mult, offset, pixelMax, csfl)
	}
}

func init() {
	if hasAVX2 {
		grainAR = grainARAVX2Row
	}
}

//go:noescape
func grainARAVX2(pre *int32, row *int16, rowStride, lag int, coeffs *int8, n int)

func grainARAVX2Row(pre []int32, buf *grainLut, y, lag int, coeffs []int8, x0, x1 int) {
	clear(pre[x0:x1])

	n := x1 - x0
	if n8 := n &^ 7; n8 > 0 {
		grainARAVX2(&pre[x0], &buf[y-lag][x0-lag], grainWidth, lag, &coeffs[0], n8)
	}
	if n&7 != 0 {
		grainARAcc(pre, buf, y, lag, coeffs, x0+(n&^7), x1)
	}
}
