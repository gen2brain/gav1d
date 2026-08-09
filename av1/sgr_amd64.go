//go:build amd64 && !noasm

package av1

//go:noescape
func sgrFinish1AVX2(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)

//go:noescape
func sgrFinish2RowAVX2(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)

//go:noescape
func sgrFinish2MidAVX2(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)

func sgrFinish1AVX2Row(tmp []int16, tmpOff int, src []uint8, srcOff int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w int,
) {
	n := w &^ 7
	if n > 0 {
		sgrFinish1AVX2(&tmp[tmpOff], &src[srcOff],
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

func sgrFinish2AVX2Row(tmp []int16, tmpOff int, src []uint8, srcOff, srcStride int,
	a []int32, aPtrs []int, b []int16, bPtrs []int, w, h int,
) {
	n := w &^ 7
	if n > 0 {
		sgrFinish2RowAVX2(&tmp[tmpOff], &src[srcOff], &a[aPtrs[0]], &a[aPtrs[1]],
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
		sgrFinish2MidAVX2(&tmp[tmpOff], &src[srcOff], &a[aPtrs[1]], &b[bPtrs[1]], n)
	}
	if n < w {
		sgrFinishFilter2Mid(tmp, tmpOff+n, src, srcOff+n,
			a, aPtrs[1]+1+n, b, bPtrs[1]+1+n, w-n)
	}
}

//go:noescape
func sgrBoxV3AVX2(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)

//go:noescape
func sgrBoxV5AVX2(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16,
	sqOut *int32, sOut *int16, n int)

func sgrBoxV3AVX2Row(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := (w + 2) &^ 7
	if n > 0 {
		sgrBoxV3AVX2(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], n)
	}
	for x := n; x < w+2; x++ {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x]
	}
}

func sgrBoxV5AVX2Row(sumsq []int32, sumsqPtrs []int, sum []int16, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int16, sumOutOff, w int,
) {
	n := (w + 2) &^ 7
	if n > 0 {
		sgrBoxV5AVX2(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
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

//go:noescape
func sgrBoxH3AVX2(sumsq *int32, sum *int16, src *uint8, n int)

//go:noescape
func sgrBoxH5AVX2(sumsq *int32, sum *int16, src *uint8, n int)

func sgrBoxH3AVX2Row(sumsq []int32, sumsqOff int, sum []int16, sumOff int,
	src []uint8, srcOff, n int,
) {
	if n8 := n &^ 7; n8 > 0 {
		sgrBoxH3AVX2(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n8)
		sumsqOff, sumOff, srcOff, n = sumsqOff+n8, sumOff+n8, srcOff+n8, n-n8
	}
	if n > 0 {
		sgrBox3HRow(sumsq, sumsqOff, sum, sumOff, src, srcOff, n)
	}
}

func sgrBoxH5AVX2Row(sumsq []int32, sumsqOff int, sum []int16, sumOff int,
	src []uint8, srcOff, n int,
) {
	if n8 := n &^ 7; n8 > 0 {
		sgrBoxH5AVX2(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n8)
		sumsqOff, sumOff, srcOff, n = sumsqOff+n8, sumOff+n8, srcOff+n8, n-n8
	}
	if n > 0 {
		sgrBox5HRow(sumsq, sumsqOff, sum, sumOff, src, srcOff, n)
	}
}

//go:noescape
func sgrCalcABAVX2(aa *int32, bb *int16, tab *[256]uint32, n int, s, mul, oneByX uint32)

func sgrCalcABAVX2Row(aa []int32, aaOff int, bb []int16, bbOff, w int, s uint32,
	bitdepthMax int32, n, sgrOneByX int32,
) {
	if bitdepthFromMax(bitdepthMax) != 8 {
		sgrCalcRowAB(aa, aaOff, bb, bbOff, w, s, bitdepthMax, n, sgrOneByX)

		return
	}

	cnt := w + 2
	if c8 := cnt &^ 7; c8 > 0 {
		sgrCalcABAVX2(&aa[aaOff], &bb[bbOff], &sgrXByXWide, c8,
			s, uint32(n), uint32(sgrOneByX))
		aaOff, bbOff, cnt = aaOff+c8, bbOff+c8, cnt-c8
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
func sgrWeight1AVX2(dst *uint8, t1 *int16, n int, w1 int32)

//go:noescape
func sgrWeight2AVX2(dst *uint8, t1, t2 *int16, n int, w0, w1 int32)

func sgrWeight1AVX2Row(dst []uint8, dstOff int, t1 []int16, t1Off, w int,
	w1 int32, bitdepthMax int32,
) {
	n := w &^ 7
	if n > 0 {
		sgrWeight1AVX2(&dst[dstOff], &t1[t1Off], n, w1)
	}
	if n < w {
		sgrWeightedRow1(dst, dstOff+n, t1, t1Off+n, w-n, w1, bitdepthMax)
	}
}

func sgrWeight2AVX2Row(dst []uint8, dstOff int, t1 []int16, t1Off int,
	t2 []int16, t2Off, w int, w0, w1 int32, bitdepthMax int32,
) {
	n := w &^ 7
	if n > 0 {
		sgrWeight2AVX2(&dst[dstOff], &t1[t1Off], &t2[t2Off], n, w0, w1)
	}
	if n < w {
		sgrWeightedRow2(dst, dstOff+n, t1, t1Off+n, t2, t2Off+n, w-n, w0, w1, bitdepthMax)
	}
}
