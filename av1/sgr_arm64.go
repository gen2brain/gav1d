//go:build arm64 && !noasm

package av1

//go:noescape
func sgrFinish1NEON(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)

//go:noescape
func sgrFinish2RowNEON(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)

//go:noescape
func sgrFinish2MidNEON(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)

func sgrFinish1NEONRow(tmp []int16, tmpOff int, src []uint8, srcOff int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w int,
) {
	n := w &^ 3
	if n > 0 {
		sgrFinish1NEON(&tmp[tmpOff], &src[srcOff],
			&a[aPtrs[0]], &a[aPtrs[1]], &a[aPtrs[2]],
			&b[bPtrs[0]], &b[bPtrs[1]], &b[bPtrs[2]], n)
	}
	if n == w {
		return
	}

	ap, bp := [3]int{}, [3]int{}
	for i := range 3 {
		ap[i], bp[i] = aPtrs[i]+n, bPtrs[i]+n
	}
	sgrFinishFilterRow1(tmp, tmpOff+n, src, srcOff+n, a, ap[:], b, bp[:], w-n)
}

func sgrFinish2NEONRow(tmp []int16, tmpOff int, src []uint8, srcOff, srcStride int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w, h int,
) {
	n := w &^ 3
	if n > 0 {
		sgrFinish2RowNEON(&tmp[tmpOff], &src[srcOff], &a[aPtrs[0]], &a[aPtrs[1]],
			&b[bPtrs[0]], &b[bPtrs[1]], n)
	}
	if n < w {
		ap, bp := [2]int{}, [2]int{}
		for i := range 2 {
			ap[i], bp[i] = aPtrs[i]+n, bPtrs[i]+n
		}
		sgrFinishFilter2Row(tmp, tmpOff+n, src, srcOff+n, a, ap[:], b, bp[:], w-n)
	}
	if h <= 1 {
		return
	}

	tmpOff += filterOutStride
	srcOff += srcStride
	if n > 0 {
		sgrFinish2MidNEON(&tmp[tmpOff], &src[srcOff], &a[aPtrs[1]], &b[bPtrs[1]], n)
	}
	if n < w {
		sgrFinishFilter2Mid(tmp, tmpOff+n, src, srcOff+n,
			a, aPtrs[1]+1+n, b, bPtrs[1]+1+n, w-n)
	}
}

//go:noescape
func sgrBoxV3NEON(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)

//go:noescape
func sgrBoxV5NEON(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16,
	sqOut *int32, sOut *int16, n int)

func sgrBoxV3NEONRow(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := (w + 2) &^ 7
	if n > 0 {
		sgrBoxV3NEON(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], n)
	}
	for x := n; x < w+2; x++ {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x]
	}
}

func sgrBoxV5NEONRow(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := (w + 2) &^ 7
	if n > 0 {
		sgrBoxV5NEON(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sumsq[sumsqPtrs[3]], &sumsq[sumsqPtrs[4]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sum[sumPtrs[3]], &sum[sumPtrs[4]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], n)
	}
	for x := n; x < w+2; x++ {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x] + sumsq[sumsqPtrs[3]+x] + sumsq[sumsqPtrs[4]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x] +
			sum[sumPtrs[3]+x] + sum[sumPtrs[4]+x]
	}
}
