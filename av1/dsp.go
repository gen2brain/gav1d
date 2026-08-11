package av1

import "sync"

// dspContext holds the kernels that have architecture-specific versions. It is
// built per frame, so the entries are already concrete for the bit depth, and
// assembly never has to reach through a type parameter.
type dspContext[P pixel] struct {
	cdefFilter func(tmp []int16, dst []P, dstOff, dstStride int,
		left []P, leftOff int, top []P, topOff int, bottom []P, bottomOff int,
		priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32)
	cdefDir  func(img []P, imgOff, stride int, bitdepthMax int32) (int, uint32)
	cdefCopy cdefCopyFn[P]

	loopFilter func(dst []P, dstOff int, e, i, h int32,
		stridea, strideb int, wd, lines int, bitdepthMax int32)

	fgApplyRow func(dst []P, dstOff int, src []P, srcOff int,
		scaling []uint8, grain []int16, n, shift, minValue, maxValue int)

	fguvApplyRow func(dst []P, dstOff int, src []P, srcOff int,
		luma []P, lumaOff int, scaling []uint8, grain []int16,
		n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool)

	put8tap func(mid []int16, dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
		w, h, mx, my, filterType int, bitdepthMax int32)

	prep8tap func(mid []int16, tmp []int16, src []P, srcOff, srcStride,
		w, h, mx, my, filterType int, bitdepthMax int32)

	putBilin func(mid []int16, dst []P, dstOff, dstStride int, src []P, srcOff, srcStride,
		w, h, mx, my int, bitdepthMax int32)

	prepBilin func(mid []int16, tmp []int16, src []P, srcOff, srcStride,
		w, h, mx, my int, bitdepthMax int32)

	itxClip func(tmp []int32, rnd int32, shift int, lo, hi int32)
	itxAdd  func(dst []P, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32)
	itxDC   func(dst []P, dstOff, stride int, dc int32, w, h int, bitdepthMax int32)
	itxCol  func(tmp []int32, lanes, stride, n, txtp int, lo, hi int32, scratch []int32) bool

	itxTranspose func(wide, tmp []int32, w, ws, ts, n int)

	// itxLanes is the lane count the one-dimensional kernels step in, which the
	// transform buffers are padded out to.
	itxLanes int

	// Converting a generic function to a func value allocates a closure for
	// the dictionary, so the predictors are resolved once per frame here
	// rather than once per block.
	ipred [nIntraPredModes + 1]func(dst []P, dstOff, stride int, tl []P, tlOff,
		w, h, angle, maxW, maxH int, bitdepthMax int32)
	cflPred [nIntraPredModes + 1]func(dst []P, dstOff, stride int, tl []P, tlOff,
		w, h int, ac []int16, alpha int32, bitdepthMax int32)
}

// newDspGo builds the table with no assembly, which is both the starting point
// for newDsp and the reference the assembly is tested against.
func newDspGo[P pixel]() *dspContext[P] {
	d := &dspContext[P]{
		cdefDir:      cdefFindDir[P],
		cdefCopy:     cdefCopyRows[P],
		loopFilter:   loopFilter[P],
		fgApplyRow:   fgApplyRow[P],
		fguvApplyRow: fguvApplyRow[P],
		put8tap:      put8tap[P],
		prep8tap:     prep8tap[P],
		putBilin:     putBilin[P],
		prepBilin:    prepBilin[P],
		itxClip:      itxClipGo,
		itxAdd:       itxAddGo[P],
		itxDC:        itxDCGo[P],
		itxCol:       itxColNone,
		itxTranspose: itxTransposeGo,
	}
	// Bound once here, since cdefFilterBlock reaches the padding copy
	// through the table.
	d.cdefFilter = func(tmp []int16, dst []P, dstOff, dstStride int,
		left []P, leftOff int, top []P, topOff int, bottom []P, bottomOff int,
		priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
	) {
		cdefFilterBlock(d, tmp, dst, dstOff, dstStride, left, leftOff, top, topOff,
			bottom, bottomOff, priStrength, secStrength, dir, damping, w, h,
			edges, bitdepthMax)
	}

	for m := range d.ipred {
		d.ipred[m] = intraPredFn[P](m)
		d.cflPred[m] = cflPredFn[P](m)
	}

	return d
}

// The table is immutable once built, so every frame of a bit depth shares one.
var (
	dsp8  = sync.OnceValue(func() *dspContext[uint8] { return buildDsp[uint8]() })
	dsp16 = sync.OnceValue(func() *dspContext[uint16] { return buildDsp[uint16]() })
)

func buildDsp[P pixel]() *dspContext[P] {
	d := newDspGo[P]()
	dspInit(d)

	return d
}

func newDsp[P pixel]() *dspContext[P] {
	var zero P
	switch any(zero).(type) {
	case uint8:
		if d, ok := any(dsp8()).(*dspContext[P]); ok {
			return d
		}
	case uint16:
		if d, ok := any(dsp16()).(*dspContext[P]); ok {
			return d
		}
	}

	return buildDsp[P]()
}
