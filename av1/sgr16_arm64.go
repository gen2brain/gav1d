//go:build arm64 && !noasm

package av1

//go:noescape
func sgrFinish1_16NEON(tmp *int32, src *uint16, a0, a1, a2, b0, b1, b2 *int32, n int)

//go:noescape
func sgrFinish2Row_16NEON(tmp *int32, src *uint16, a0, a1, b0, b1 *int32, n int)

//go:noescape
func sgrFinish2Mid_16NEON(tmp *int32, src *uint16, a1, b1 *int32, n int)

//go:noescape
func sgrBoxV3_16NEON(sq0, sq1, sq2, s0, s1, s2, sqOut, sOut *int32, n int)

//go:noescape
func sgrBoxV5_16NEON(sq0, sq1, sq2, sq3, sq4, s0, s1, s2, s3, s4, sqOut, sOut *int32, n int)

//go:noescape
func sgrBoxH3_16NEON(sumsq, sum *int32, src *uint16, n int)

//go:noescape
func sgrBoxH5_16NEON(sumsq, sum *int32, src *uint16, n int)

//go:noescape
func sgrCalcAB_16NEON(aa, bb *int32, tab *[256]uint32, n int, s, mul, oneByX uint32,
	rndA, shA, rndB, shB int32)

//go:noescape
func sgrWeight1_16NEON(dst *uint16, t1 *int32, n int, w1, bitdepthMax int32)

//go:noescape
func sgrWeight2_16NEON(dst *uint16, t1, t2 *int32, n int, w0, w1, bitdepthMax int32)

func sgrFinish1_16NEONRow(tmp []int32, tmpOff int, src []uint16, srcOff int,
	a []int32, aPtrs []int, b []int32, bPtrs []int, w int,
) {
	n := w &^ 7
	if n > 0 {
		sgrFinish1_16NEON(&tmp[tmpOff], &src[srcOff],
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

func sgrFinish2_16NEONRow(tmp []int32, tmpOff int, src []uint16, srcOff, srcStride int,
	a []int32, aPtrs []int, b []int32, bPtrs []int, w, h int,
) {
	n := w &^ 7
	if n > 0 {
		sgrFinish2Row_16NEON(&tmp[tmpOff], &src[srcOff], &a[aPtrs[0]], &a[aPtrs[1]],
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
		sgrFinish2Mid_16NEON(&tmp[tmpOff], &src[srcOff], &a[aPtrs[1]], &b[bPtrs[1]], n)
	}
	if n < w {
		sgrFinishFilter2Mid(tmp, tmpOff+n, src, srcOff+n,
			a, aPtrs[1]+1+n, b, bPtrs[1]+1+n, w-n)
	}
}

func sgrBoxV3_16NEONRow(sumsq []int32, sumsqPtrs []int, sum []int32, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int32, sumOutOff, w int,
) {
	cnt := w + 2
	if c8 := cnt &^ 7; c8 > 0 {
		sgrBoxV3_16NEON(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], c8)
		cnt -= c8
		sumsqOutOff, sumOutOff = sumsqOutOff+c8, sumOutOff+c8
		sqp := [3]int{sumsqPtrs[0] + c8, sumsqPtrs[1] + c8, sumsqPtrs[2] + c8}
		sp := [3]int{sumPtrs[0] + c8, sumPtrs[1] + c8, sumPtrs[2] + c8}
		sumsqPtrs, sumPtrs = sqp[:], sp[:]
	}
	for x := range cnt {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x]
	}
}

func sgrBoxV5_16NEONRow(sumsq []int32, sumsqPtrs []int, sum []int32, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []int32, sumOutOff, w int,
) {
	cnt := w + 2
	if c8 := cnt &^ 7; c8 > 0 {
		sgrBoxV5_16NEON(&sumsq[sumsqPtrs[0]], &sumsq[sumsqPtrs[1]], &sumsq[sumsqPtrs[2]],
			&sumsq[sumsqPtrs[3]], &sumsq[sumsqPtrs[4]],
			&sum[sumPtrs[0]], &sum[sumPtrs[1]], &sum[sumPtrs[2]],
			&sum[sumPtrs[3]], &sum[sumPtrs[4]],
			&sumsqOut[sumsqOutOff], &sumOut[sumOutOff], c8)
		cnt -= c8
		sumsqOutOff, sumOutOff = sumsqOutOff+c8, sumOutOff+c8
		sqp := [5]int{}
		sp := [5]int{}
		for i := range 5 {
			sqp[i], sp[i] = sumsqPtrs[i]+c8, sumPtrs[i]+c8
		}
		sumsqPtrs, sumPtrs = sqp[:], sp[:]
	}
	for x := range cnt {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x] + sumsq[sumsqPtrs[3]+x] + sumsq[sumsqPtrs[4]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x] +
			sum[sumPtrs[3]+x] + sum[sumPtrs[4]+x]
	}
}

func sgrBoxH3_16NEONRow(sumsq []int32, sumsqOff int, sum []int32, sumOff int,
	src []uint16, srcOff, n int,
) {
	if n8 := n &^ 7; n8 > 0 {
		sgrBoxH3_16NEON(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n8)
		sumsqOff, sumOff, srcOff, n = sumsqOff+n8, sumOff+n8, srcOff+n8, n-n8
	}
	if n > 0 {
		sgrBox3HRow(sumsq, sumsqOff, sum, sumOff, src, srcOff, n)
	}
}

func sgrBoxH5_16NEONRow(sumsq []int32, sumsqOff int, sum []int32, sumOff int,
	src []uint16, srcOff, n int,
) {
	if n8 := n &^ 7; n8 > 0 {
		sgrBoxH5_16NEON(&sumsq[sumsqOff], &sum[sumOff], &src[srcOff], n8)
		sumsqOff, sumOff, srcOff, n = sumsqOff+n8, sumOff+n8, srcOff+n8, n-n8
	}
	if n > 0 {
		sgrBox5HRow(sumsq, sumsqOff, sum, sumOff, src, srcOff, n)
	}
}

func sgrCalcAB_16NEONRow(aa []int32, aaOff int, bb []int32, bbOff, w int, s uint32,
	bitdepthMax int32, n, sgrOneByX int32,
) {
	bdm8 := int32(bitdepthFromMax(bitdepthMax) - 8)
	cnt := w + 2
	if c8 := cnt &^ 7; c8 > 0 {
		sgrCalcAB_16NEON(&aa[aaOff], &bb[bbOff], &sgrXByXWide, c8,
			s, uint32(n), uint32(sgrOneByX),
			(1<<(2*bdm8))>>1, 2*bdm8, (1<<bdm8)>>1, bdm8)
		aaOff, bbOff, cnt = aaOff+c8, bbOff+c8, cnt-c8
	}
	for i := range cnt {
		a := (aa[aaOff+i] + ((1 << (2 * bdm8)) >> 1)) >> (2 * bdm8)
		b := (bb[bbOff+i] + ((1 << bdm8) >> 1)) >> bdm8

		p := uint32(max(a*n-b*b, 0))
		z := (p*s + (1 << 19)) >> 20
		x := uint32(sgrXByX[min(z, 255)])

		aa[aaOff+i] = int32((x*uint32(bb[bbOff+i])*uint32(sgrOneByX) + (1 << 11)) >> 12)
		bb[bbOff+i] = int32(x)
	}
}

func sgrWeight1_16NEONRow(dst []uint16, dstOff int, t1 []int32, t1Off, w int,
	w1 int32, bitdepthMax int32,
) {
	n := w &^ 7
	if n > 0 {
		sgrWeight1_16NEON(&dst[dstOff], &t1[t1Off], n, w1, bitdepthMax)
	}
	if n < w {
		sgrWeightedRow1(dst, dstOff+n, t1, t1Off+n, w-n, w1, bitdepthMax)
	}
}

func sgrWeight2_16NEONRow(dst []uint16, dstOff int, t1 []int32, t1Off int,
	t2 []int32, t2Off, w int, w0, w1 int32, bitdepthMax int32,
) {
	n := w &^ 7
	if n > 0 {
		sgrWeight2_16NEON(&dst[dstOff], &t1[t1Off], &t2[t2Off], n, w0, w1, bitdepthMax)
	}
	if n < w {
		sgrWeightedRow2(dst, dstOff+n, t1, t1Off+n, t2, t2Off+n, w-n, w0, w1, bitdepthMax)
	}
}
