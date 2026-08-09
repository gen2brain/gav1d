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

func wienerVNEONRnd(bitdepthMax int32) (rnd int32, shift int) {
	bitdepth := bitdepthFromMax(bitdepthMax)
	shift = 11 - b2i(bitdepth == 12)*2
	rnd = int32(1)<<(shift-1) - int32(1)<<(bitdepth+shift-1)

	return rnd, shift
}

func wienerV8NEONRow(p []uint8, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	rnd, shift := wienerVNEONRnd(bitdepthMax)

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
	rnd, shift := wienerVNEONRnd(bitdepthMax)

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
		l8.finish1 = sgrFinish1NEONRow
		l8.finish2 = sgrFinish2NEONRow
	}

	if l16, ok := any(l).(*lrContext[uint16, int32]); ok {
		l16.wienerH = wienerH16NEONRow
		l16.wienerV = wienerV16NEONRow
	}
}
