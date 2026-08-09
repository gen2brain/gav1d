package av1

const (
	lrHaveLeft = 1 << iota
	lrHaveRight
	lrHaveTop
	lrHaveBottom
)

const (
	lrRestoreY = 1 << iota
	lrRestoreU
	lrRestoreV
)

const (
	restUnitStride  = 390
	filterOutStride = 384
	lrBufStride     = 384 + 16
	lrScratchRow    = 6 * restUnitStride
)

type looprestorationParams struct {
	filter [2][8]int16
	sgr    struct {
		s0, s1 uint32
		w0, w1 int16
	}
}

// wienerHEdge is the fully conditional form, used for the three pixels at each
// end of the row where the taps reach outside it.
func wienerHEdge[P pixel](dst []uint16, dstOff int, left []P, leftOff int, hasLeft bool,
	src []P, srcOff int, fh *[8]int16, x0, x1, w, edges int,
	rnd int32, shift int, limit int32,
) {
	for x := x0; x < x1; x++ {
		sum := rnd
		for i := range 7 {
			idx := x + i - 3
			switch {
			case idx < 0 && edges&lrHaveLeft == 0:
				sum += int32(src[srcOff]) * int32(fh[i])
			case idx < 0 && hasLeft:
				sum += int32(left[leftOff+4+idx]) * int32(fh[i])
			case idx >= w && edges&lrHaveRight == 0:
				sum += int32(src[srcOff+w-1]) * int32(fh[i])
			default:
				sum += int32(src[srcOff+idx]) * int32(fh[i])
			}
		}

		dst[dstOff+x] = uint16(clip(sum>>shift, 0, limit))
	}
}

func wienerHRow[P pixel](dst []uint16, dstOff int, src []P, srcOff, n int,
	fh *[8]int16, rnd int32, shift int, limit int32,
) {
	f0, f1, f2, f3 := int32(fh[0]), int32(fh[1]), int32(fh[2]), int32(fh[3])
	f4, f5, f6 := int32(fh[4]), int32(fh[5]), int32(fh[6])
	src, dst = src[srcOff-3:], dst[dstOff:]

	for x := range n {
		s := src[x : x+7 : x+7]
		sum := rnd + int32(s[0])*f0 + int32(s[1])*f1 + int32(s[2])*f2 +
			int32(s[3])*f3 + int32(s[4])*f4 + int32(s[5])*f5 + int32(s[6])*f6

		dst[x] = uint16(clip(sum>>shift, 0, limit))
	}
}

// wienerFilterH folds the rounding offset and, at 8 bits, the unscaled centre
// pixel into the taps, so the bulk of the row is an unconditional 7-tap.
func wienerFilterH[P pixel](dsp *dspContext[P], dst []uint16, dstOff int,
	left []P, leftOff int, hasLeft bool, src []P, srcOff int, fh *[8]int16,
	w, edges int, bitdepthMax int32,
) {
	bitdepth := bitdepthFromMax(bitdepthMax)
	shift := 3 + b2i(bitdepth == 12)*2
	rnd := int32(1)<<(bitdepth+6) + int32(1)<<(shift-1)
	limit := int32(1)<<(bitdepth+1+7-shift) - 1

	f := *fh
	if bitdepth == 8 {
		f[3] += 128
	}

	if w < 6 {
		wienerHEdge(dst, dstOff, left, leftOff, hasLeft, src, srcOff, &f,
			0, w, w, edges, rnd, shift, limit)

		return
	}

	start := 0
	if edges&lrHaveLeft == 0 || hasLeft {
		start = 3
		wienerHEdge(dst, dstOff, left, leftOff, hasLeft, src, srcOff, &f,
			0, 3, w, edges, rnd, shift, limit)
	}

	end := w - 3
	if edges&lrHaveRight != 0 {
		end = w
	}

	dsp.wienerH(dst, dstOff+start, src, srcOff+start, end-start, &f, rnd, shift, limit)

	if end < w {
		wienerHEdge(dst, dstOff, left, leftOff, hasLeft, src, srcOff, &f,
			end, w, w, edges, rnd, shift, limit)
	}
}

func wienerVRow[P pixel](p []P, pOff int, hor []uint16, rows *[7]int,
	fv *[8]int16, w int, bitdepthMax int32,
) {
	bitdepth := bitdepthFromMax(bitdepthMax)
	shift := 11 - b2i(bitdepth == 12)*2
	rnd := int32(1)<<(shift-1) - int32(1)<<(bitdepth+shift-1)

	r0, r1, r2, r3 := hor[rows[0]:], hor[rows[1]:], hor[rows[2]:], hor[rows[3]:]
	r4, r5, r6 := hor[rows[4]:], hor[rows[5]:], hor[rows[6]:]
	f0, f1, f2, f3 := int32(fv[0]), int32(fv[1]), int32(fv[2]), int32(fv[3])
	f4, f5, f6 := int32(fv[4]), int32(fv[5]), int32(fv[6])

	for i := range w {
		sum := rnd + int32(r0[i])*f0 + int32(r1[i])*f1 + int32(r2[i])*f2 +
			int32(r3[i])*f3 + int32(r4[i])*f4 + int32(r5[i])*f5 + int32(r6[i])*f6

		p[pOff+i] = P(clip(sum>>shift, 0, bitdepthMax))
	}
}

// wienerFilterV runs at the bottom of a stripe, where the seventh row is a
// repeat of the sixth.
func wienerFilterV[P pixel](dsp *dspContext[P], p []P, pOff int, hor []uint16,
	ptrs *[7]int, fv *[8]int16, w int, bitdepthMax int32,
) {
	rows := [7]int{ptrs[0], ptrs[1], ptrs[2], ptrs[3], ptrs[4], ptrs[5], ptrs[5]}
	dsp.wienerV(p, pOff, hor, &rows, fv, w, bitdepthMax)

	for i := range 5 {
		ptrs[i] = ptrs[i+1]
	}
}

func wienerFilterHV[P pixel](dsp *dspContext[P], p []P, pOff int, hor []uint16,
	ptrs *[7]int, left []P, leftOff int, hasLeft bool, src []P, srcOff int,
	filter *[2][8]int16, w, edges int, bitdepthMax int32,
) {
	wienerFilterH(dsp, hor, lrScratchRow, left, leftOff, hasLeft, src, srcOff,
		&filter[0], w, edges, bitdepthMax)

	rows := [7]int{ptrs[0], ptrs[1], ptrs[2], ptrs[3], ptrs[4], ptrs[5], lrScratchRow}
	dsp.wienerV(p, pOff, hor, &rows, &filter[1], w, bitdepthMax)

	copy(hor[ptrs[6]:ptrs[6]+w], hor[lrScratchRow:lrScratchRow+w])

	for i := range 6 {
		ptrs[i] = ptrs[i+1]
	}
	ptrs[6] = ptrs[0]
}

func lrWiener[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	left []P, leftOff int, lpf []P, lpfOff, w, h int,
	params *looprestorationParams, edges int, bitdepthMax int32,
) {
	var horBuf [7 * restUnitStride]uint16
	hor := horBuf[:]
	var ptrs [7]int
	var rows [6]int
	for i := range 6 {
		rows[i] = i * restUnitStride
	}
	fh, fv := &params.filter[0], &params.filter[1]
	lpfBottom := lpfOff + 6*stride
	src, srcOff := dst, dstOff

	stage := 0

	if edges&lrHaveTop != 0 {
		ptrs = [7]int{rows[0], rows[0], rows[1], rows[2], rows[2], rows[2], 0}

		wienerFilterH(dsp, hor, rows[0], left, 0, false, lpf, lpfOff, fh, w, edges, bitdepthMax)
		lpfOff += stride
		wienerFilterH(dsp, hor, rows[1], left, 0, false, lpf, lpfOff, fh, w, edges, bitdepthMax)

		wienerFilterH(dsp, hor, rows[2], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 1
			goto vert
		}

		ptrs[4], ptrs[5] = rows[3], rows[3]
		wienerFilterH(dsp, hor, rows[3], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 2
			goto vert
		}

		ptrs[5] = rows[4]
		wienerFilterH(dsp, hor, rows[4], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 3
			goto vert
		}
	} else {
		ptrs = [7]int{rows[0], rows[0], rows[0], rows[0], rows[0], rows[0], 0}

		wienerFilterH(dsp, hor, rows[0], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 1
			goto vert
		}

		ptrs[4], ptrs[5] = rows[1], rows[1]
		wienerFilterH(dsp, hor, rows[1], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 2
			goto vert
		}

		ptrs[5] = rows[2]
		wienerFilterH(dsp, hor, rows[2], left, leftOff, true, src, srcOff, fh, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			stage = 3
			goto vert
		}

		ptrs[6] = rows[3]
		wienerFilterHV(dsp, dst, dstOff, hor, &ptrs, left, leftOff, true, src, srcOff,
			&params.filter, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		dstOff += stride

		h--
		if h <= 0 {
			stage = 3
			goto vert
		}

		ptrs[6] = rows[4]
		wienerFilterHV(dsp, dst, dstOff, hor, &ptrs, left, leftOff, true, src, srcOff,
			&params.filter, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		dstOff += stride

		h--
		if h <= 0 {
			stage = 3
			goto vert
		}
	}

	ptrs[6] = ptrs[5] + restUnitStride
	for {
		wienerFilterHV(dsp, dst, dstOff, hor, &ptrs, left, leftOff, true, src, srcOff,
			&params.filter, w, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		dstOff += stride

		h--
		if h <= 0 {
			break
		}
	}

	if edges&lrHaveBottom == 0 {
		stage = 3
		goto vert
	}

	wienerFilterHV(dsp, dst, dstOff, hor, &ptrs, left, 0, false, lpf, lpfBottom,
		&params.filter, w, edges, bitdepthMax)
	lpfBottom += stride
	dstOff += stride

	wienerFilterHV(dsp, dst, dstOff, hor, &ptrs, left, 0, false, lpf, lpfBottom,
		&params.filter, w, edges, bitdepthMax)
	dstOff += stride

	stage = 1

vert:
	if stage == 3 {
		wienerFilterV(dsp, dst, dstOff, hor, &ptrs, fv, w, bitdepthMax)
		dstOff += stride
		stage = 2
	}
	if stage == 2 {
		wienerFilterV(dsp, dst, dstOff, hor, &ptrs, fv, w, bitdepthMax)
		dstOff += stride
	}
	wienerFilterV(dsp, dst, dstOff, hor, &ptrs, fv, w, bitdepthMax)
}

func rotate(sumsqPtrs, sumPtrs []int, n int) {
	tmpsq, tmps := sumsqPtrs[0], sumPtrs[0]
	for i := range n - 1 {
		sumsqPtrs[i] = sumsqPtrs[i+1]
		sumPtrs[i] = sumPtrs[i+1]
	}
	sumsqPtrs[n-1], sumPtrs[n-1] = tmpsq, tmps
}

func rotate5x2(sumsqPtrs, sumPtrs []int) {
	var tmpsq, tmps [2]int
	for i := range 2 {
		tmpsq[i], tmps[i] = sumsqPtrs[i], sumPtrs[i]
	}
	for i := range 3 {
		sumsqPtrs[i] = sumsqPtrs[i+2]
		sumPtrs[i] = sumPtrs[i+2]
	}
	for i := range 2 {
		sumsqPtrs[3+i], sumPtrs[3+i] = tmpsq[i], tmps[i]
	}
}

func sgrBox3RowH[P pixel, C coef](sumsq []int32, sumsqOff int, sum []C, sumOff int,
	left []P, leftOff int, hasLeft bool, src []P, srcOff, w, edges int,
) {
	sumsqOff++
	sumOff++

	var a, b int32
	if edges&lrHaveLeft != 0 {
		if hasLeft {
			a, b = int32(left[leftOff+2]), int32(left[leftOff+3])
		} else {
			a, b = int32(src[srcOff-2]), int32(src[srcOff-1])
		}
	} else {
		a, b = int32(src[srcOff]), int32(src[srcOff])
	}

	for x := -1; x < w+1; x++ {
		var c int32
		if x+1 < w || edges&lrHaveRight != 0 {
			c = int32(src[srcOff+x+1])
		} else {
			c = int32(src[srcOff+w-1])
		}
		sum[sumOff+x] = C(a + b + c)
		sumsq[sumsqOff+x] = a*a + b*b + c*c
		a, b = b, c
	}
}

func sgrBox5RowH[P pixel, C coef](sumsq []int32, sumsqOff int, sum []C, sumOff int,
	left []P, leftOff int, hasLeft bool, src []P, srcOff, w, edges int,
) {
	sumsqOff++
	sumOff++

	var a, b, c int32
	if edges&lrHaveLeft != 0 {
		if hasLeft {
			a, b, c = int32(left[leftOff+1]), int32(left[leftOff+2]), int32(left[leftOff+3])
		} else {
			a, b, c = int32(src[srcOff-3]), int32(src[srcOff-2]), int32(src[srcOff-1])
		}
	} else {
		a, b, c = int32(src[srcOff]), int32(src[srcOff]), int32(src[srcOff])
	}
	d := int32(src[srcOff])

	for x := -1; x < w+1; x++ {
		var e int32
		if x+2 < w || edges&lrHaveRight != 0 {
			e = int32(src[srcOff+x+2])
		} else {
			e = int32(src[srcOff+w-1])
		}
		sum[sumOff+x] = C(a + b + c + d + e)
		sumsq[sumsqOff+x] = a*a + b*b + c*c + d*d + e*e
		a, b, c, d = b, c, d, e
	}
}

func sgrBox35RowH[P pixel, C coef](sumsq3 []int32, sumsq3Off int, sum3 []C, sum3Off int,
	sumsq5 []int32, sumsq5Off int, sum5 []C, sum5Off int,
	left []P, leftOff int, hasLeft bool, src []P, srcOff, w, edges int,
) {
	sgrBox3RowH(sumsq3, sumsq3Off, sum3, sum3Off, left, leftOff, hasLeft, src, srcOff, w, edges)
	sgrBox5RowH(sumsq5, sumsq5Off, sum5, sum5Off, left, leftOff, hasLeft, src, srcOff, w, edges)
}

func sgrBox3RowV[C coef](sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int,
) {
	for x := range w + 2 {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x]
	}
}

func sgrBox5RowV[C coef](sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int,
) {
	for x := range w + 2 {
		sumsqOut[sumsqOutOff+x] = sumsq[sumsqPtrs[0]+x] + sumsq[sumsqPtrs[1]+x] +
			sumsq[sumsqPtrs[2]+x] + sumsq[sumsqPtrs[3]+x] + sumsq[sumsqPtrs[4]+x]
		sumOut[sumOutOff+x] = sum[sumPtrs[0]+x] + sum[sumPtrs[1]+x] + sum[sumPtrs[2]+x] +
			sum[sumPtrs[3]+x] + sum[sumPtrs[4]+x]
	}
}

func sgrCalcRowAB[C coef](aa []int32, aaOff int, bb []C, bbOff, w int, s uint32,
	bitdepthMax int32, n, sgrOneByX int32,
) {
	bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8

	for i := range w + 2 {
		a := (aa[aaOff+i] + ((1 << (2 * bitdepthMin8)) >> 1)) >> (2 * bitdepthMin8)
		b := (int32(bb[bbOff+i]) + ((1 << bitdepthMin8) >> 1)) >> bitdepthMin8

		p := uint32(max(a*n-b*b, 0))
		z := (p*s + (1 << 19)) >> 20
		x := uint32(sgrXByX[min(z, 255)])

		aa[aaOff+i] = int32((x*uint32(bb[bbOff+i])*uint32(sgrOneByX) + (1 << 11)) >> 12)
		bb[bbOff+i] = C(x)
	}
}

func sgrBox3Vert[C coef](sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int, s uint32,
	bitdepthMax int32,
) {
	sgrBox3RowV(sumsq, sumsqPtrs, sum, sumPtrs, sumsqOut, sumsqOutOff, sumOut, sumOutOff, w)
	sgrCalcRowAB(sumsqOut, sumsqOutOff, sumOut, sumOutOff, w, s, bitdepthMax, 9, 455)
	rotate(sumsqPtrs, sumPtrs, 3)
}

func sgrBox5Vert[C coef](sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
	sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int, s uint32,
	bitdepthMax int32,
) {
	sgrBox5RowV(sumsq, sumsqPtrs, sum, sumPtrs, sumsqOut, sumsqOutOff, sumOut, sumOutOff, w)
	sgrCalcRowAB(sumsqOut, sumsqOutOff, sumOut, sumOutOff, w, s, bitdepthMax, 25, 164)
	rotate5x2(sumsqPtrs, sumPtrs)
}

func sgrBox3HV[P pixel, C coef](sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
	aa []int32, aaOff int, bb []C, bbOff int, left []P, leftOff int, hasLeft bool,
	src []P, srcOff, w int, s uint32, edges int, bitdepthMax int32,
) {
	sgrBox3RowH(sumsq, sumsqPtrs[2], sum, sumPtrs[2], left, leftOff, hasLeft,
		src, srcOff, w, edges)
	sgrBox3Vert(sumsq, sumsqPtrs, sum, sumPtrs, aa, aaOff, bb, bbOff, w, s, bitdepthMax)
}

func sgrFinishFilterRow1[P pixel, C coef](tmp []C, tmpOff int, src []P, srcOff int,
	a []int32, aPtrs []int, b []C, bPtrs []int, w int,
) {
	for i := range w {
		j := i + 1
		bs := (int32(b[bPtrs[1]+j])+int32(b[bPtrs[1]+j-1])+int32(b[bPtrs[1]+j+1])+
			int32(b[bPtrs[0]+j])+int32(b[bPtrs[2]+j]))*4 +
			(int32(b[bPtrs[0]+j-1])+int32(b[bPtrs[2]+j-1])+
				int32(b[bPtrs[0]+j+1])+int32(b[bPtrs[2]+j+1]))*3
		as := (a[aPtrs[1]+j]+a[aPtrs[1]+j-1]+a[aPtrs[1]+j+1]+
			a[aPtrs[0]+j]+a[aPtrs[2]+j])*4 +
			(a[aPtrs[0]+j-1]+a[aPtrs[2]+j-1]+
				a[aPtrs[0]+j+1]+a[aPtrs[2]+j+1])*3
		tmp[tmpOff+i] = C((as - bs*int32(src[srcOff+i]) + (1 << 8)) >> 9)
	}
}

func sgrFinishFilter2[P pixel, C coef](tmp []C, tmpOff int, src []P, srcOff, srcStride int,
	a []int32, aPtrs []int, b []C, bPtrs []int, w, h int,
) {
	for i := range w {
		j := i + 1
		bs := (int32(b[bPtrs[0]+j])+int32(b[bPtrs[1]+j]))*6 +
			(int32(b[bPtrs[0]+j-1])+int32(b[bPtrs[1]+j-1])+
				int32(b[bPtrs[0]+j+1])+int32(b[bPtrs[1]+j+1]))*5
		as := (a[aPtrs[0]+j]+a[aPtrs[1]+j])*6 +
			(a[aPtrs[0]+j-1]+a[aPtrs[1]+j-1]+a[aPtrs[0]+j+1]+a[aPtrs[1]+j+1])*5
		tmp[tmpOff+i] = C((as - bs*int32(src[srcOff+i]) + (1 << 8)) >> 9)
	}
	if h <= 1 {
		return
	}

	tmpOff += filterOutStride
	srcOff += srcStride
	aOff, bOff := aPtrs[1]+1, bPtrs[1]+1
	for i := range w {
		bs := int32(b[bOff+i])*6 + (int32(b[bOff+i-1])+int32(b[bOff+i+1]))*5
		as := a[aOff+i]*6 + (a[aOff+i-1]+a[aOff+i+1])*5
		tmp[tmpOff+i] = C((as - bs*int32(src[srcOff+i]) + (1 << 7)) >> 8)
	}
}

func sgrWeightedRow1[P pixel, C coef](dst []P, dstOff int, t1 []C, t1Off, w int,
	w1 int32, bitdepthMax int32,
) {
	for i := range w {
		v := w1 * int32(t1[t1Off+i])
		dst[dstOff+i] = P(clip(int32(dst[dstOff+i])+((v+(1<<10))>>11), 0, bitdepthMax))
	}
}

func sgrWeighted2[P pixel, C coef](dst []P, dstOff, dstStride int, t1 []C, t1Off int,
	t2 []C, t2Off, w, h int, w0, w1 int32, bitdepthMax int32,
) {
	for range h {
		for i := range w {
			v := w0*int32(t1[t1Off+i]) + w1*int32(t2[t2Off+i])
			dst[dstOff+i] = P(clip(int32(dst[dstOff+i])+((v+(1<<10))>>11), 0, bitdepthMax))
		}
		dstOff += dstStride
		t1Off += filterOutStride
		t2Off += filterOutStride
	}
}

func sgrFinish1[P pixel, C coef](dst []P, dstOff *int, stride int,
	a []int32, aPtrs []int, b []C, bPtrs []int, w int, w1 int32, bitdepthMax int32,
) {
	var tmpBuf [384]C
	tmp := tmpBuf[:]

	sgrFinishFilterRow1(tmp, 0, dst, *dstOff, a, aPtrs, b, bPtrs, w)
	sgrWeightedRow1(dst, *dstOff, tmp, 0, w, w1, bitdepthMax)
	*dstOff += stride
	rotate(aPtrs, bPtrs, 3)
}

func sgrFinish2[P pixel, C coef](dst []P, dstOff *int, stride int,
	a []int32, aPtrs []int, b []C, bPtrs []int, w, h int, w1 int32, bitdepthMax int32,
) {
	var tmpBuf [2 * filterOutStride]C
	tmp := tmpBuf[:]

	sgrFinishFilter2(tmp, 0, dst, *dstOff, stride, a, aPtrs, b, bPtrs, w, h)
	sgrWeightedRow1(dst, *dstOff, tmp, 0, w, w1, bitdepthMax)
	*dstOff += stride
	if h > 1 {
		sgrWeightedRow1(dst, *dstOff, tmp, filterOutStride, w, w1, bitdepthMax)
		*dstOff += stride
	}
	rotate(aPtrs, bPtrs, 2)
}

func sgrFinishMix[P pixel, C coef](dst []P, dstOff *int, stride int,
	a5 []int32, a5Ptrs []int, b5 []C, b5Ptrs []int,
	a3 []int32, a3Ptrs []int, b3 []C, b3Ptrs []int,
	w, h int, w0, w1 int32, bitdepthMax int32,
) {
	var tmp5Buf, tmp3Buf [2 * filterOutStride]C
	tmp5, tmp3 := tmp5Buf[:], tmp3Buf[:]

	sgrFinishFilter2(tmp5, 0, dst, *dstOff, stride, a5, a5Ptrs, b5, b5Ptrs, w, h)
	sgrFinishFilterRow1(tmp3, 0, dst, *dstOff, a3, a3Ptrs, b3, b3Ptrs, w)
	if h > 1 {
		sgrFinishFilterRow1(tmp3, filterOutStride, dst, *dstOff+stride,
			a3, a3Ptrs[1:], b3, b3Ptrs[1:], w)
	}
	sgrWeighted2(dst, *dstOff, stride, tmp5, 0, tmp3, 0, w, h, w0, w1, bitdepthMax)
	*dstOff += h * stride
	rotate(a5Ptrs, b5Ptrs, 2)
	rotate(a3Ptrs, b3Ptrs, 4)
}

func lrSgr3x3[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	left []P, leftOff int, lpf []P, lpfOff, w, h int,
	params *looprestorationParams, edges int, bitdepthMax int32,
) {
	var sumsqBuf, aBuf [lrBufStride*3 + 16]int32
	var sumBuf, bBuf [lrBufStride*3 + 16]C
	sumsq, sum := sumsqBuf[:], sumBuf[:]
	a, b := aBuf[:], bBuf[:]

	var sumsqPtrs, sumPtrs, aPtrs, bPtrs, rows [3]int
	for i := range 3 {
		rows[i] = i * lrBufStride
		aPtrs[i] = i * lrBufStride
		bPtrs[i] = i * lrBufStride
	}

	src, srcOff := dst, dstOff
	lpfBottom := lpfOff + 6*stride
	s1 := params.sgr.s1
	w1 := int32(params.sgr.w1)

	if edges&lrHaveTop != 0 {
		sumsqPtrs = [3]int{rows[0], rows[1], rows[2]}
		sumPtrs = sumsqPtrs

		sgrBox3RowH(sumsq, rows[0], sum, rows[0], left, 0, false, lpf, lpfOff, w, edges)
		lpfOff += stride
		sgrBox3RowH(sumsq, rows[1], sum, rows[1], left, 0, false, lpf, lpfOff, w, edges)

		sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
			left, leftOff, true, src, srcOff, w, s1, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		rotate(aPtrs[:], bPtrs[:], 3)

		h--
		if h <= 0 {
			goto vert1
		}

		sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
			left, leftOff, true, src, srcOff, w, s1, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		rotate(aPtrs[:], bPtrs[:], 3)

		h--
		if h <= 0 {
			goto vert2
		}
	} else {
		sumsqPtrs = [3]int{rows[0], rows[0], rows[0]}
		sumPtrs = sumsqPtrs

		sgrBox3RowH(sumsq, rows[0], sum, rows[0], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox3Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
			w, s1, bitdepthMax)
		rotate(aPtrs[:], bPtrs[:], 3)

		h--
		if h <= 0 {
			goto vert1
		}

		sumsqPtrs[2], sumPtrs[2] = rows[1], rows[1]

		sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
			left, leftOff, true, src, srcOff, w, s1, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride
		rotate(aPtrs[:], bPtrs[:], 3)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsqPtrs[2], sumPtrs[2] = rows[2], rows[2]
	}

	for {
		sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
			left, leftOff, true, src, srcOff, w, s1, edges, bitdepthMax)
		leftOff += 4
		srcOff += stride

		sgrFinish1(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, w1, bitdepthMax)

		h--
		if h <= 0 {
			break
		}
	}

	if edges&lrHaveBottom == 0 {
		goto vert2
	}

	sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
		left, 0, false, lpf, lpfBottom, w, s1, edges, bitdepthMax)
	lpfBottom += stride

	sgrFinish1(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, w1, bitdepthMax)

	sgrBox3HV(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
		left, 0, false, lpf, lpfBottom, w, s1, edges, bitdepthMax)

	sgrFinish1(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, w1, bitdepthMax)

	return

vert2:
	sumsqPtrs[2], sumPtrs[2] = sumsqPtrs[1], sumPtrs[1]
	sgrBox3Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
		w, s1, bitdepthMax)

	sgrFinish1(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, w1, bitdepthMax)

	goto output1

vert1:
	sumsqPtrs[2], sumPtrs[2] = sumsqPtrs[1], sumPtrs[1]
	sgrBox3Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
		w, s1, bitdepthMax)
	rotate(aPtrs[:], bPtrs[:], 3)

output1:
	sumsqPtrs[2], sumPtrs[2] = sumsqPtrs[1], sumPtrs[1]
	sgrBox3Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[2], b, bPtrs[2],
		w, s1, bitdepthMax)

	sgrFinish1(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, w1, bitdepthMax)
}

func lrSgr5x5[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	left []P, leftOff int, lpf []P, lpfOff, w, h int,
	params *looprestorationParams, edges int, bitdepthMax int32,
) {
	var sumsqBuf [lrBufStride*5 + 16]int32
	var sumBuf [lrBufStride*5 + 16]C
	var aBuf [lrBufStride*2 + 16]int32
	var bBuf [lrBufStride*2 + 16]C
	sumsq, sum := sumsqBuf[:], sumBuf[:]
	a, b := aBuf[:], bBuf[:]

	var sumsqPtrs, sumPtrs, rows [5]int
	var aPtrs, bPtrs [2]int
	for i := range 5 {
		rows[i] = i * lrBufStride
	}
	for i := range 2 {
		aPtrs[i] = i * lrBufStride
		bPtrs[i] = i * lrBufStride
	}

	src, srcOff := dst, dstOff
	lpfBottom := lpfOff + 6*stride
	s0 := params.sgr.s0
	w0 := int32(params.sgr.w0)

	if edges&lrHaveTop != 0 {
		sumsqPtrs = [5]int{rows[0], rows[0], rows[1], rows[2], rows[3]}
		sumPtrs = sumsqPtrs

		sgrBox5RowH(sumsq, rows[0], sum, rows[0], left, 0, false, lpf, lpfOff, w, edges)
		lpfOff += stride
		sgrBox5RowH(sumsq, rows[1], sum, rows[1], left, 0, false, lpf, lpfOff, w, edges)

		sgrBox5RowH(sumsq, rows[2], sum, rows[2], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			goto vert1
		}

		sgrBox5RowH(sumsq, rows[3], sum, rows[3], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride
		sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
			w, s0, bitdepthMax)
		rotate(aPtrs[:], bPtrs[:], 2)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsqPtrs[3], sumPtrs[3] = rows[4], rows[4]
	} else {
		sumsqPtrs = [5]int{rows[0], rows[0], rows[0], rows[0], rows[0]}
		sumPtrs = sumsqPtrs

		sgrBox5RowH(sumsq, rows[0], sum, rows[0], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			goto vert1
		}

		sumsqPtrs[4], sumPtrs[4] = rows[1], rows[1]

		sgrBox5RowH(sumsq, rows[1], sum, rows[1], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
			w, s0, bitdepthMax)
		rotate(aPtrs[:], bPtrs[:], 2)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsqPtrs[3], sumPtrs[3] = rows[2], rows[2]
		sumsqPtrs[4], sumPtrs[4] = rows[3], rows[3]

		sgrBox5RowH(sumsq, rows[2], sum, rows[2], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			goto odd
		}

		sgrBox5RowH(sumsq, rows[3], sum, rows[3], left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
			w, s0, bitdepthMax)
		sgrFinish2(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, 2, w0, bitdepthMax)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsqPtrs[3], sumPtrs[3] = rows[4], rows[4]
	}

	for {
		sgrBox5RowH(sumsq, sumsqPtrs[3], sum, sumPtrs[3], left, leftOff, true,
			src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		h--
		if h <= 0 {
			goto odd
		}

		sgrBox5RowH(sumsq, sumsqPtrs[4], sum, sumPtrs[4], left, leftOff, true,
			src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
			w, s0, bitdepthMax)
		sgrFinish2(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, 2, w0, bitdepthMax)

		h--
		if h <= 0 {
			break
		}
	}

	if edges&lrHaveBottom == 0 {
		goto vert2
	}

	sgrBox5RowH(sumsq, sumsqPtrs[3], sum, sumPtrs[3], left, 0, false, lpf, lpfBottom, w, edges)
	lpfBottom += stride
	sgrBox5RowH(sumsq, sumsqPtrs[4], sum, sumPtrs[4], left, 0, false, lpf, lpfBottom, w, edges)

output2:
	sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
		w, s0, bitdepthMax)
	sgrFinish2(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, 2, w0, bitdepthMax)

	return

vert2:
	sumsqPtrs[3], sumsqPtrs[4] = sumsqPtrs[2], sumsqPtrs[2]
	sumPtrs[3], sumPtrs[4] = sumPtrs[2], sumPtrs[2]

	goto output2

odd:
	sumsqPtrs[4], sumPtrs[4] = sumsqPtrs[3], sumPtrs[3]

	sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
		w, s0, bitdepthMax)
	sgrFinish2(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, 2, w0, bitdepthMax)

	goto output1

vert1:
	sumsqPtrs[4], sumPtrs[4] = sumsqPtrs[3], sumPtrs[3]

	sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
		w, s0, bitdepthMax)
	rotate(aPtrs[:], bPtrs[:], 2)

output1:
	sumsqPtrs[3], sumsqPtrs[4] = sumsqPtrs[2], sumsqPtrs[2]
	sumPtrs[3], sumPtrs[4] = sumPtrs[2], sumPtrs[2]

	sgrBox5Vert(sumsq, sumsqPtrs[:], sum, sumPtrs[:], a, aPtrs[1], b, bPtrs[1],
		w, s0, bitdepthMax)
	sgrFinish2(dst, &dstOff, stride, a, aPtrs[:], b, bPtrs[:], w, 1, w0, bitdepthMax)
}

func lrSgrMix[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	left []P, leftOff int, lpf []P, lpfOff, w, h int,
	params *looprestorationParams, edges int, bitdepthMax int32,
) {
	var sumsq5Buf [lrBufStride*5 + 16]int32
	var sum5Buf [lrBufStride*5 + 16]C
	var sumsq3Buf [lrBufStride*3 + 16]int32
	var sum3Buf [lrBufStride*3 + 16]C
	var a5Buf [lrBufStride*2 + 16]int32
	var b5Buf [lrBufStride*2 + 16]C
	var a3Buf [lrBufStride*4 + 16]int32
	var b3Buf [lrBufStride*4 + 16]C
	sumsq5, sum5 := sumsq5Buf[:], sum5Buf[:]
	sumsq3, sum3 := sumsq3Buf[:], sum3Buf[:]
	a5, b5 := a5Buf[:], b5Buf[:]
	a3, b3 := a3Buf[:], b3Buf[:]

	var sumsq5Ptrs, sum5Ptrs, rows5 [5]int
	var sumsq3Ptrs, sum3Ptrs, rows3 [3]int
	var a5Ptrs, b5Ptrs [2]int
	var a3Ptrs, b3Ptrs [4]int
	for i := range 5 {
		rows5[i] = i * lrBufStride
	}
	for i := range 3 {
		rows3[i] = i * lrBufStride
	}
	for i := range 2 {
		a5Ptrs[i] = i * lrBufStride
		b5Ptrs[i] = i * lrBufStride
	}
	for i := range 4 {
		a3Ptrs[i] = i * lrBufStride
		b3Ptrs[i] = i * lrBufStride
	}

	src, srcOff := dst, dstOff
	lpfBottom := lpfOff + 6*stride
	s0, s1 := params.sgr.s0, params.sgr.s1
	w0, w1 := int32(params.sgr.w0), int32(params.sgr.w1)

	if edges&lrHaveTop != 0 {
		sumsq5Ptrs = [5]int{rows5[0], rows5[0], rows5[1], rows5[2], rows5[3]}
		sum5Ptrs = sumsq5Ptrs
		sumsq3Ptrs = [3]int{rows3[0], rows3[1], rows3[2]}
		sum3Ptrs = sumsq3Ptrs

		sgrBox35RowH(sumsq3, rows3[0], sum3, rows3[0], sumsq5, rows5[0], sum5, rows5[0],
			left, 0, false, lpf, lpfOff, w, edges)
		lpfOff += stride
		sgrBox35RowH(sumsq3, rows3[1], sum3, rows3[1], sumsq5, rows5[1], sum5, rows5[1],
			left, 0, false, lpf, lpfOff, w, edges)

		sgrBox35RowH(sumsq3, rows3[2], sum3, rows3[2], sumsq5, rows5[2], sum5, rows5[2],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto vert1
		}

		sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2], sumsq5, rows5[3], sum5, rows5[3],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride
		sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
			w, s0, bitdepthMax)
		rotate(a5Ptrs[:], b5Ptrs[:], 2)
		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsq5Ptrs[3], sum5Ptrs[3] = rows5[4], rows5[4]
	} else {
		sumsq5Ptrs = [5]int{rows5[0], rows5[0], rows5[0], rows5[0], rows5[0]}
		sum5Ptrs = sumsq5Ptrs
		sumsq3Ptrs = [3]int{rows3[0], rows3[0], rows3[0]}
		sum3Ptrs = sumsq3Ptrs

		sgrBox35RowH(sumsq3, rows3[0], sum3, rows3[0], sumsq5, rows5[0], sum5, rows5[0],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto vert1
		}

		sumsq5Ptrs[4], sum5Ptrs[4] = rows5[1], rows5[1]
		sumsq3Ptrs[2], sum3Ptrs[2] = rows3[1], rows3[1]

		sgrBox35RowH(sumsq3, rows3[1], sum3, rows3[1], sumsq5, rows5[1], sum5, rows5[1],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
			w, s0, bitdepthMax)
		rotate(a5Ptrs[:], b5Ptrs[:], 2)
		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsq5Ptrs[3], sum5Ptrs[3] = rows5[2], rows5[2]
		sumsq5Ptrs[4], sum5Ptrs[4] = rows5[3], rows5[3]
		sumsq3Ptrs[2], sum3Ptrs[2] = rows3[2], rows3[2]

		sgrBox35RowH(sumsq3, rows3[2], sum3, rows3[2], sumsq5, rows5[2], sum5, rows5[2],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto odd
		}

		sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2], sumsq5, rows5[3], sum5, rows5[3],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
			w, s0, bitdepthMax)
		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		sgrFinishMix(dst, &dstOff, stride, a5, a5Ptrs[:], b5, b5Ptrs[:],
			a3, a3Ptrs[:], b3, b3Ptrs[:], w, 2, w0, w1, bitdepthMax)

		h--
		if h <= 0 {
			goto vert2
		}

		sumsq5Ptrs[3], sum5Ptrs[3] = rows5[4], rows5[4]
	}

	for {
		sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2],
			sumsq5, sumsq5Ptrs[3], sum5, sum5Ptrs[3],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		rotate(a3Ptrs[:], b3Ptrs[:], 4)

		h--
		if h <= 0 {
			goto odd
		}

		sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2],
			sumsq5, sumsq5Ptrs[4], sum5, sum5Ptrs[4],
			left, leftOff, true, src, srcOff, w, edges)
		leftOff += 4
		srcOff += stride

		sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
			w, s0, bitdepthMax)
		sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
			w, s1, bitdepthMax)
		sgrFinishMix(dst, &dstOff, stride, a5, a5Ptrs[:], b5, b5Ptrs[:],
			a3, a3Ptrs[:], b3, b3Ptrs[:], w, 2, w0, w1, bitdepthMax)

		h--
		if h <= 0 {
			break
		}
	}

	if edges&lrHaveBottom == 0 {
		goto vert2
	}

	sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2],
		sumsq5, sumsq5Ptrs[3], sum5, sum5Ptrs[3],
		left, 0, false, lpf, lpfBottom, w, edges)
	lpfBottom += stride
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	rotate(a3Ptrs[:], b3Ptrs[:], 4)

	sgrBox35RowH(sumsq3, sumsq3Ptrs[2], sum3, sum3Ptrs[2],
		sumsq5, sumsq5Ptrs[4], sum5, sum5Ptrs[4],
		left, 0, false, lpf, lpfBottom, w, edges)

output2:
	sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
		w, s0, bitdepthMax)
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	sgrFinishMix(dst, &dstOff, stride, a5, a5Ptrs[:], b5, b5Ptrs[:],
		a3, a3Ptrs[:], b3, b3Ptrs[:], w, 2, w0, w1, bitdepthMax)

	return

vert2:
	sumsq5Ptrs[3], sumsq5Ptrs[4] = sumsq5Ptrs[2], sumsq5Ptrs[2]
	sum5Ptrs[3], sum5Ptrs[4] = sum5Ptrs[2], sum5Ptrs[2]

	sumsq3Ptrs[2], sum3Ptrs[2] = sumsq3Ptrs[1], sum3Ptrs[1]
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	rotate(a3Ptrs[:], b3Ptrs[:], 4)

	sumsq3Ptrs[2], sum3Ptrs[2] = sumsq3Ptrs[1], sum3Ptrs[1]

	goto output2

odd:
	sumsq5Ptrs[4], sum5Ptrs[4] = sumsq5Ptrs[3], sum5Ptrs[3]
	sumsq3Ptrs[2], sum3Ptrs[2] = sumsq3Ptrs[1], sum3Ptrs[1]

	sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
		w, s0, bitdepthMax)
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	sgrFinishMix(dst, &dstOff, stride, a5, a5Ptrs[:], b5, b5Ptrs[:],
		a3, a3Ptrs[:], b3, b3Ptrs[:], w, 2, w0, w1, bitdepthMax)

	goto output1

vert1:
	sumsq5Ptrs[4], sum5Ptrs[4] = sumsq5Ptrs[3], sum5Ptrs[3]
	sumsq3Ptrs[2], sum3Ptrs[2] = sumsq3Ptrs[1], sum3Ptrs[1]

	sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
		w, s0, bitdepthMax)
	rotate(a5Ptrs[:], b5Ptrs[:], 2)
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	rotate(a3Ptrs[:], b3Ptrs[:], 4)

output1:
	sumsq5Ptrs[3], sumsq5Ptrs[4] = sumsq5Ptrs[2], sumsq5Ptrs[2]
	sum5Ptrs[3], sum5Ptrs[4] = sum5Ptrs[2], sum5Ptrs[2]

	sumsq3Ptrs[2], sum3Ptrs[2] = sumsq3Ptrs[1], sum3Ptrs[1]

	sgrBox5Vert(sumsq5, sumsq5Ptrs[:], sum5, sum5Ptrs[:], a5, a5Ptrs[1], b5, b5Ptrs[1],
		w, s0, bitdepthMax)
	sgrBox3Vert(sumsq3, sumsq3Ptrs[:], sum3, sum3Ptrs[:], a3, a3Ptrs[3], b3, b3Ptrs[3],
		w, s1, bitdepthMax)
	rotate(a3Ptrs[:], b3Ptrs[:], 4)

	sgrFinishMix(dst, &dstOff, stride, a5, a5Ptrs[:], b5, b5Ptrs[:],
		a3, a3Ptrs[:], b3, b3Ptrs[:], w, 1, w0, w1, bitdepthMax)
}
