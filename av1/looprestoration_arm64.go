//go:build arm64 && !noasm

package av1

//go:noescape
func wienerH8NEON(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerH16NEON(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerV8NEON(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

//go:noescape
func wienerV16NEON(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

func wienerH8NEONRow(dst []uint16, dstOff int, src []uint8, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	if n8 := n &^ 7; n8 > 0 {
		wienerH8NEON(&dst[dstOff], &src[srcOff-3], n8, fh, rnd, shift, limit)
		dstOff += n8
		srcOff += n8
		n -= n8
	}
	if n > 0 {
		wienerHRow(dst, dstOff, src, srcOff, n, fh, rnd, shift, limit)
	}
}

func wienerH16NEONRow(dst []uint16, dstOff int, src []uint16, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	if n8 := n &^ 7; n8 > 0 {
		wienerH16NEON(&dst[dstOff], &src[srcOff-3], n8, fh, rnd, shift, limit)
		dstOff += n8
		srcOff += n8
		n -= n8
	}
	if n > 0 {
		wienerHRow(dst, dstOff, src, srcOff, n, fh, rnd, shift, limit)
	}
}

func wienerV8NEONRow(p []uint8, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	rnd, shift := wienerVRnd(bitdepthMax)

	n := w &^ 7
	if n > 0 {
		wienerV8NEON(&p[pOff], &hor[0], rows, fv, n, rnd, shift, bitdepthMax)
	}
	if n == w {
		return
	}

	tail := *rows
	for i := range tail {
		tail[i] += n
	}
	wienerVRow(p, pOff+n, hor, &tail, fv, w-n, bitdepthMax)
}

func wienerV16NEONRow(p []uint16, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	rnd, shift := wienerVRnd(bitdepthMax)

	n := w &^ 7
	if n > 0 {
		wienerV16NEON(&p[pOff], &hor[0], rows, fv, n, rnd, shift, bitdepthMax)
	}
	if n == w {
		return
	}

	tail := *rows
	for i := range tail {
		tail[i] += n
	}
	wienerVRow(p, pOff+n, hor, &tail, fv, w-n, bitdepthMax)
}

func lrInit[P pixel, C coef](l *lrContext[P, C]) {
	if l8, ok := any(l).(*lrContext[uint8, int16]); ok {
		l8.wienerH = wienerH8NEONRow
		l8.wienerV = wienerV8NEONRow
		l8.weight1 = sgrWeight1NEONRow
		l8.weight2 = sgrWeight2NEONRow
		l8.calcAB = sgrCalcABNEONRow
		l8.boxH3 = sgrBoxH3NEONRow
		l8.boxH5 = sgrBoxH5NEONRow
		l8.boxV3 = sgrBoxV3NEONRow
		l8.boxV5 = sgrBoxV5NEONRow
		l8.finish1 = sgrFinish1NEONRow
		l8.finish2 = sgrFinish2NEONRow
	}

	if l16, ok := any(l).(*lrContext[uint16, int32]); ok {
		l16.wienerH = wienerH16NEONRow
		l16.wienerV = wienerV16NEONRow
		l16.weight1 = sgrWeight1_16NEONRow
		l16.weight2 = sgrWeight2_16NEONRow
		l16.calcAB = sgrCalcAB_16NEONRow
		l16.boxH3 = sgrBoxH3_16NEONRow
		l16.boxH5 = sgrBoxH5_16NEONRow
		l16.boxV3 = sgrBoxV3_16NEONRow
		l16.boxV5 = sgrBoxV5_16NEONRow
		l16.finish1 = sgrFinish1_16NEONRow
		l16.finish2 = sgrFinish2_16NEONRow
	}
}
