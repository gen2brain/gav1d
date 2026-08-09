//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func sgrFinish1RVV(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)

//go:noescape
func sgrFinish2RowRVV(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)

//go:noescape
func sgrFinish2MidRVV(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)

// The vector length setting clamps itself to what is left, so these need no
// scalar tail.
func sgrFinish1RVVRow(tmp []int16, tmpOff int, src []uint8, srcOff int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w int,
) {
	if w <= 0 {
		return
	}
	sgrFinish1RVV(&tmp[tmpOff], &src[srcOff],
		&a[aPtrs[0]], &a[aPtrs[1]], &a[aPtrs[2]],
		&b[bPtrs[0]], &b[bPtrs[1]], &b[bPtrs[2]], w)
}

func sgrFinish2RVVRow(tmp []int16, tmpOff int, src []uint8, srcOff, srcStride int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w, h int,
) {
	if w <= 0 {
		return
	}
	sgrFinish2RowRVV(&tmp[tmpOff], &src[srcOff], &a[aPtrs[0]], &a[aPtrs[1]],
		&b[bPtrs[0]], &b[bPtrs[1]], w)
	if h <= 1 {
		return
	}
	sgrFinish2MidRVV(&tmp[tmpOff+filterOutStride], &src[srcOff+srcStride],
		&a[aPtrs[1]], &b[bPtrs[1]], w)
}

//go:noescape
func sgrBoxV3RVV(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)

//go:noescape
func sgrBoxV5RVV(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16,
	sqOut *int32, sOut *int16, n int)

func sgrBoxV3RVVRow(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := w + 2
	if n > 0 {
		sgrBoxV3RVV(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], n)
	}
}

func sgrBoxV5RVVRow(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := w + 2
	if n > 0 {
		sgrBoxV5RVV(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sumsq[sumsqPtrs[3]], &sumsq[sumsqPtrs[4]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sum[sumPtrs[3]], &sum[sumPtrs[4]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], n)
	}
}

//go:noescape
func sgrBoxH3RVV(sumsq *int32, sum *int16, src *uint8, n int)

//go:noescape
func sgrBoxH5RVV(sumsq *int32, sum *int16, src *uint8, n int)

func sgrBoxH3RVVRow(sumsq []int32, sumsqOff int, sum []int16, sumOff int,
	src []uint8, srcOff, n int,
) {
	if n > 0 {
		sgrBoxH3RVV(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n)
	}
}

func sgrBoxH5RVVRow(sumsq []int32, sumsqOff int, sum []int16, sumOff int,
	src []uint8, srcOff, n int,
) {
	if n > 0 {
		sgrBoxH5RVV(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n)
	}
}

//go:noescape
func sgrCalcABRVV(aa *int32, bb *int16, tab *[256]uint32, n int, s, mul, oneByX uint32)

func sgrCalcABRVVRow(aa []int32, aaOff int, bb []int16, bbOff, w int, s uint32,
	bitdepthMax int32, n, sgrOneByX int32,
) {
	if bitdepthFromMax(bitdepthMax) != 8 {
		sgrCalcRowAB(aa, aaOff, bb, bbOff, w, s, bitdepthMax, n, sgrOneByX)

		return
	}

	cnt := w + 2
	if cnt > 0 {
		sgrCalcABRVV(&aa[aaOff], &bb[bbOff], &sgrXByXWide, cnt,
			s, uint32(n), uint32(sgrOneByX))
		cnt = 0
	}
	for i := range cnt {
		a := aa[aaOff+i]
		b := int32(bb[bbOff+i])
		p := uint32(max(a*n-b*b, 0))
		z := (p*s + (1 << 19)) >> 20
		x := uint32(sgrXByX[min(z, 255)])
		aa[aaOff+i] = int32((x*uint32(b)*uint32(sgrOneByX) + (1 << 11)) >> 12)
		bb[bbOff+i] = int16(x)
	}
}

//go:noescape
func sgrWeight1RVV(dst *uint8, t1 *int16, n int, w1 int32)

//go:noescape
func sgrWeight2RVV(dst *uint8, t1, t2 *int16, n int, w0, w1 int32)

func sgrWeight1RVVRow(dst []uint8, dstOff int, t1 []int16, t1Off, w int,
	w1 int32, bitdepthMax int32,
) {
	n := w
	if n > 0 {
		sgrWeight1RVV(&dst[dstOff], &t1[t1Off], n, w1)
	}
	if n < w {
		sgrWeightedRow1(dst, dstOff+n, t1, t1Off+n, w-n, w1, bitdepthMax)
	}
}

func sgrWeight2RVVRow(dst []uint8, dstOff int, t1 []int16, t1Off int,
	t2 []int16, t2Off, w int, w0, w1 int32, bitdepthMax int32,
) {
	n := w
	if n > 0 {
		sgrWeight2RVV(&dst[dstOff], &t1[t1Off], &t2[t2Off], n, w0, w1)
	}
	if n < w {
		sgrWeightedRow2(dst, dstOff+n, t1, t1Off+n, t2, t2Off+n, w-n, w0, w1, bitdepthMax)
	}
}
