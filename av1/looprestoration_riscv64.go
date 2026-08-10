//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func wienerH8RVV(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerH16RVV(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerV8RVV(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

//go:noescape
func wienerV16RVV(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

// The vector length setting clamps itself to what is left, so these need no
// scalar tail.
func wienerH8RVVRow(dst []uint16, dstOff int, src []uint8, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	if n > 0 {
		wienerH8RVV(&dst[dstOff], &src[srcOff-3], n, fh, rnd, shift, limit)
	}
}

func wienerH16RVVRow(dst []uint16, dstOff int, src []uint16, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	if n > 0 {
		wienerH16RVV(&dst[dstOff], &src[srcOff-3], n, fh, rnd, shift, limit)
	}
}

func wienerV8RVVRow(p []uint8, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	if w <= 0 {
		return
	}

	rnd, shift := wienerVRnd(bitdepthMax)
	wienerV8RVV(&p[pOff], &hor[0], rows, fv, w, rnd, shift, bitdepthMax)
}

func wienerV16RVVRow(p []uint16, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	if w <= 0 {
		return
	}

	rnd, shift := wienerVRnd(bitdepthMax)
	wienerV16RVV(&p[pOff], &hor[0], rows, fv, w, rnd, shift, bitdepthMax)
}

func lrInit[P pixel, C coef](l *lrContext[P, C]) {
	if l8, ok := any(l).(*lrContext[uint8, int16]); ok {
		l8.wienerH = wienerH8RVVRow
		l8.wienerV = wienerV8RVVRow
		l8.weight1 = sgrWeight1RVVRow
		l8.weight2 = sgrWeight2RVVRow
		l8.calcAB = sgrCalcABRVVRow
		l8.boxH3 = sgrBoxH3RVVRow
		l8.boxH5 = sgrBoxH5RVVRow
		l8.boxV3 = sgrBoxV3RVVRow
		l8.boxV5 = sgrBoxV5RVVRow
		l8.finish1 = sgrFinish1RVVRow
		l8.finish2 = sgrFinish2RVVRow
	}

	if l16, ok := any(l).(*lrContext[uint16, int32]); ok {
		l16.wienerH = wienerH16RVVRow
		l16.wienerV = wienerV16RVVRow
		l16.weight1 = sgrWeight1_16RVVRow
		l16.weight2 = sgrWeight2_16RVVRow
		l16.calcAB = sgrCalcAB_16RVVRow
		l16.boxH3 = sgrBoxH3_16RVVRow
		l16.boxH5 = sgrBoxH5_16RVVRow
		l16.boxV3 = sgrBoxV3_16RVVRow
		l16.boxV5 = sgrBoxV5_16RVVRow
		l16.finish1 = sgrFinish1_16RVVRow
		l16.finish2 = sgrFinish2_16RVVRow
	}
}
