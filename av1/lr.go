package av1

import "sync"

// lrContext holds the loop restoration kernels. The self-guided half works on
// the sum buffers, which are int16 at 8 bits and int32 above, so unlike
// dspContext this table needs the coefficient type too.
type lrContext[P pixel, C coef] struct {
	wienerH func(dst []uint16, dstOff int, src []P, srcOff, n int,
		fh *[8]int16, rnd int32, shift int, limit int32)
	wienerV func(p []P, pOff int, hor []uint16, rows *[7]int,
		fv *[8]int16, w int, bitdepthMax int32)

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
