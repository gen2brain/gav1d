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

func wienerVRVVRnd(bitdepthMax int32) (rnd int32, shift int) {
	bitdepth := bitdepthFromMax(bitdepthMax)
	shift = 11 - b2i(bitdepth == 12)*2
	rnd = int32(1)<<(shift-1) - int32(1)<<(bitdepth+shift-1)

	return rnd, shift
}

func wienerV8RVVRow(p []uint8, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	if w <= 0 {
		return
	}

	rnd, shift := wienerVRVVRnd(bitdepthMax)
	wienerV8RVV(&p[pOff], &hor[0], rows, fv, w, rnd, shift, bitdepthMax)
}

func wienerV16RVVRow(p []uint16, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	if w <= 0 {
		return
	}

	rnd, shift := wienerVRVVRnd(bitdepthMax)
	wienerV16RVV(&p[pOff], &hor[0], rows, fv, w, rnd, shift, bitdepthMax)
}
