package av1

// dspContext holds the kernels that have architecture-specific versions. It is
// built per frame, so the entries are already concrete for the bit depth, and
// assembly never has to reach through a type parameter.
type dspContext[P pixel] struct {
	cdefFilter func(dst []P, dstOff, dstStride int,
		left []P, leftOff int, top []P, topOff int, bottom []P, bottomOff int,
		priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32)
	cdefDir func(img []P, imgOff, stride int, bitdepthMax int32) (int, uint32)

	fgApplyRow func(dst []P, dstOff int, src []P, srcOff int,
		scaling []uint8, grain []int16, n, shift, minValue, maxValue int)

	fguvApplyRow func(dst []P, dstOff int, src []P, srcOff int,
		luma []P, lumaOff int, scaling []uint8, grain []int16,
		n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool)
}

// newDspGo builds the table with no assembly, which is both the starting point
// for newDsp and the reference the assembly is tested against.
func newDspGo[P pixel]() *dspContext[P] {
	return &dspContext[P]{
		cdefFilter:   cdefFilterBlock[P],
		cdefDir:      cdefFindDir[P],
		fgApplyRow:   fgApplyRow[P],
		fguvApplyRow: fguvApplyRow[P],
	}
}

func newDsp[P pixel]() *dspContext[P] {
	d := newDspGo[P]()
	dspInit(d)

	return d
}
