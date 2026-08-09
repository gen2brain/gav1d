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
