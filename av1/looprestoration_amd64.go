//go:build amd64 && !noasm

package av1

//go:noescape
func wienerH8AVX2(dst *uint16, src *uint8, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerH16AVX2(dst *uint16, src *uint16, n int, c *[8]int16, rnd int32, shift int, limit int32)

//go:noescape
func wienerV8AVX2(p *uint8, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

//go:noescape
func wienerV16AVX2(p *uint16, hor *uint16, rows *[7]int, c *[8]int16, n int, rnd int32,
	shift int, limit int32)

// wienerPairs reorders the taps into the four multiply-add pairs the kernels
// consume, with the lone seventh tap in the high half so no load reaches
// further right than the scalar loop does.
func wienerPairs(f *[8]int16) [8]int16 {
	return [8]int16{f[0], f[1], f[2], f[3], f[4], f[5], 0, f[6]}
}

func wienerH8AVX2Row(dst []uint16, dstOff int, src []uint8, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	c := wienerPairs(fh)
	if n16 := n &^ 15; n16 > 0 {
		wienerH8AVX2(&dst[dstOff], &src[srcOff-3], n16, &c, rnd, shift, limit)
		dstOff += n16
		srcOff += n16
		n -= n16
	}
	if n > 0 {
		wienerHRow(dst, dstOff, src, srcOff, n, fh, rnd, shift, limit)
	}
}

func wienerH16AVX2Row(dst []uint16, dstOff int, src []uint16, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	c := wienerPairs(fh)
	if n16 := n &^ 15; n16 > 0 {
		wienerH16AVX2(&dst[dstOff], &src[srcOff-3], n16, &c, rnd, shift, limit)
		dstOff += n16
		srcOff += n16
		n -= n16
	}
	if n > 0 {
		wienerHRow(dst, dstOff, src, srcOff, n, fh, rnd, shift, limit)
	}
}

func wienerVAVX2Rnd(bitdepthMax int32) (rnd int32, shift int) {
	bitdepth := bitdepthFromMax(bitdepthMax)
	shift = 11 - b2i(bitdepth == 12)*2
	rnd = int32(1)<<(shift-1) - int32(1)<<(bitdepth+shift-1)

	return rnd, shift
}

func wienerV8AVX2Row(p []uint8, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	rnd, shift := wienerVAVX2Rnd(bitdepthMax)
	c := wienerPairs(fv)

	n := w &^ 15
	if n > 0 {
		wienerV8AVX2(&p[pOff], &hor[0], rows, &c, n, rnd, shift, bitdepthMax)
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

func wienerV16AVX2Row(p []uint16, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	rnd, shift := wienerVAVX2Rnd(bitdepthMax)
	c := wienerPairs(fv)

	n := w &^ 15
	if n > 0 {
		wienerV16AVX2(&p[pOff], &hor[0], rows, &c, n, rnd, shift, bitdepthMax)
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
	if !hasAVX2 {
		return
	}

	if l8, ok := any(l).(*lrContext[uint8, int16]); ok {
		l8.wienerH = wienerH8AVX2Row
		l8.wienerV = wienerV8AVX2Row
		l8.boxV3 = sgrBoxV3AVX2Row
		l8.boxV5 = sgrBoxV5AVX2Row
		l8.finish1 = sgrFinish1AVX2Row
		l8.finish2 = sgrFinish2AVX2Row
	}

	if l16, ok := any(l).(*lrContext[uint16, int32]); ok {
		l16.wienerH = wienerH16AVX2Row
		l16.wienerV = wienerV16AVX2Row
	}
}
