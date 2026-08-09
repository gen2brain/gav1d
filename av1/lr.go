package av1

import "sync"

// sgrXByXWide is the reciprocal table widened so a gather can reach it.
var sgrXByXWide = func() (t [256]uint32) {
	for i, v := range sgrXByX {
		t[i] = uint32(v)
	}

	return t
}()

// lrContext holds the loop restoration kernels. The self-guided half works on
// the sum buffers, which are int16 at 8 bits and int32 above, so unlike
// dspContext this table needs the coefficient type too.
type lrContext[P pixel, C coef] struct {
	wienerH func(dst []uint16, dstOff int, src []P, srcOff, n int,
		fh *[8]int16, rnd int32, shift int, limit int32)
	wienerV func(p []P, pOff int, hor []uint16, rows *[7]int,
		fv *[8]int16, w int, bitdepthMax int32)

	boxH3 func(sumsq []int32, sumsqOff int, sum []C, sumOff int,
		src []P, srcOff, n int)
	boxH5 func(sumsq []int32, sumsqOff int, sum []C, sumOff int,
		src []P, srcOff, n int)

	weight1 func(dst []P, dstOff int, t1 []C, t1Off, w int, w1 int32, bitdepthMax int32)
	weight2 func(dst []P, dstOff int, t1 []C, t1Off int, t2 []C, t2Off, w int,
		w0, w1 int32, bitdepthMax int32)

	calcAB func(aa []int32, aaOff int, bb []C, bbOff, w int, s uint32,
		bitdepthMax int32, n, sgrOneByX int32)

	boxV3 func(sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
		sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int)
	boxV5 func(sumsq []int32, sumsqPtrs []int, sum []C, sumPtrs []int,
		sumsqOut []int32, sumsqOutOff int, sumOut []C, sumOutOff, w int)

	finish1 func(tmp []C, tmpOff int, src []P, srcOff int,
		a []int32, aPtrs []int, b []C, bPtrs []int, w int)
	finish2 func(tmp []C, tmpOff int, src []P, srcOff, srcStride int,
		a []int32, aPtrs []int, b []C, bPtrs []int, w, h int)
}

// newLrGo builds the table with no assembly, which is both the starting point
// for newLr and the reference the assembly is tested against.
func newLrGo[P pixel, C coef]() *lrContext[P, C] {
	return &lrContext[P, C]{
		wienerH: wienerHRow[P],
		wienerV: wienerVRow[P],
		boxH3:   sgrBox3HRow[P, C],
		boxH5:   sgrBox5HRow[P, C],
		weight1: sgrWeightedRow1[P, C],
		weight2: sgrWeightedRow2[P, C],
		calcAB:  sgrCalcRowAB[C],
		boxV3:   sgrBox3RowV[C],
		boxV5:   sgrBox5RowV[C],
		finish1: sgrFinishFilterRow1[P, C],
		finish2: sgrFinishFilter2[P, C],
	}
}

// The table is immutable once built, so every frame of a bit depth shares one.
var (
	lr8 = sync.OnceValue(func() *lrContext[uint8, int16] {
		return buildLr[uint8, int16]()
	})
	lr16 = sync.OnceValue(func() *lrContext[uint16, int32] {
		return buildLr[uint16, int32]()
	})
)

func buildLr[P pixel, C coef]() *lrContext[P, C] {
	l := newLrGo[P, C]()
	lrInit(l)

	return l
}

func newLr[P pixel, C coef]() *lrContext[P, C] {
	var zero P
	switch any(zero).(type) {
	case uint8:
		if l, ok := any(lr8()).(*lrContext[P, C]); ok {
			return l
		}
	case uint16:
		if l, ok := any(lr16()).(*lrContext[P, C]); ok {
			return l
		}
	}

	return buildLr[P, C]()
}
