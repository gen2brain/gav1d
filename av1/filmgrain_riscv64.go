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

//go:noescape
func fgApplyRow16RVV(dst, src *uint16, scaling *uint8, grain *int16,
	n, shift, minValue, maxValue int)

// The vector length setting clamps itself to what is left, so this needs no
// scalar tail.
func fgApplyRow16RVVRow(dst []uint16, dstOff int, src []uint16, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	if n > 0 {
		fgApplyRow16RVV(&dst[dstOff], &src[srcOff], &scaling[0], &grain[0],
			n, shift, minValue, maxValue)
	}
}

//go:noescape
func fguvApplyRow16RVV(dst, src, luma *uint16, scaling *uint8, grain *int16, p *fguv16Params)

func fguvApplyRow16RVVRow(dst []uint16, dstOff int, src []uint16, srcOff int,
	luma []uint16, lumaOff int, scaling []uint8, grain []int16,
	n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool,
) {
	if n <= 0 {
		return
	}

	p := fguv16Params{
		n: n, shift: shift, minValue: minValue, maxValue: maxValue,
		csfl: b2i(csfl), lumaMult: lumaMult, mult: mult, offset: offset,
		pixelMax: pixelMax, sx: sx,
	}
	fguvApplyRow16RVV(&dst[dstOff], &src[srcOff], &luma[lumaOff],
		&scaling[0], &grain[0], &p)
}

func init() {
	grainAR = grainARRVVRow
}

//go:noescape
func grainARRVV(pre *int32, row *int16, rowStride, lag int, coeffs *int8, n int)

func grainARRVVRow(pre []int32, buf *grainLut, y, lag int, coeffs []int8, x0, x1 int) {
	clear(pre[x0:x1])

	if n := x1 - x0; n > 0 {
		grainARRVV(&pre[x0], &buf[y-lag][x0-lag], grainWidth, lag, &coeffs[0], n)
	}
}
