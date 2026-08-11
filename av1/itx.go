package av1

import (
	"math"
	"unsafe"
)

type pixel interface{ ~uint8 | ~uint16 }

type coef interface{ ~int16 | ~int32 }

const (
	itxDct64Scratch = 64*64 + 64*32
	itxScratchLen   = itxDct64Scratch + 256
)

var txfmDimensions = [nRectTxSizes]txfmInfo{
	tx4x4:    {w: 1, h: 1, lw: 0, lh: 0, min: 0, max: 0, sub: tx4x4, ctx: 0},
	tx8x8:    {w: 2, h: 2, lw: 1, lh: 1, min: 1, max: 1, sub: tx4x4, ctx: 1},
	tx16x16:  {w: 4, h: 4, lw: 2, lh: 2, min: 2, max: 2, sub: tx8x8, ctx: 2},
	tx32x32:  {w: 8, h: 8, lw: 3, lh: 3, min: 3, max: 3, sub: tx16x16, ctx: 3},
	tx64x64:  {w: 16, h: 16, lw: 4, lh: 4, min: 4, max: 4, sub: tx32x32, ctx: 4},
	rtx4x8:   {w: 1, h: 2, lw: 0, lh: 1, min: 0, max: 1, sub: tx4x4, ctx: 1},
	rtx8x4:   {w: 2, h: 1, lw: 1, lh: 0, min: 0, max: 1, sub: tx4x4, ctx: 1},
	rtx8x16:  {w: 2, h: 4, lw: 1, lh: 2, min: 1, max: 2, sub: tx8x8, ctx: 2},
	rtx16x8:  {w: 4, h: 2, lw: 2, lh: 1, min: 1, max: 2, sub: tx8x8, ctx: 2},
	rtx16x32: {w: 4, h: 8, lw: 2, lh: 3, min: 2, max: 3, sub: tx16x16, ctx: 3},
	rtx32x16: {w: 8, h: 4, lw: 3, lh: 2, min: 2, max: 3, sub: tx16x16, ctx: 3},
	rtx32x64: {w: 8, h: 16, lw: 3, lh: 4, min: 3, max: 4, sub: tx32x32, ctx: 4},
	rtx64x32: {w: 16, h: 8, lw: 4, lh: 3, min: 3, max: 4, sub: tx32x32, ctx: 4},
	rtx4x16:  {w: 1, h: 4, lw: 0, lh: 2, min: 0, max: 2, sub: rtx4x8, ctx: 1},
	rtx16x4:  {w: 4, h: 1, lw: 2, lh: 0, min: 0, max: 2, sub: rtx8x4, ctx: 1},
	rtx8x32:  {w: 2, h: 8, lw: 1, lh: 3, min: 1, max: 3, sub: rtx8x16, ctx: 2},
	rtx32x8:  {w: 8, h: 2, lw: 3, lh: 1, min: 1, max: 3, sub: rtx16x8, ctx: 2},
	rtx16x64: {w: 4, h: 16, lw: 2, lh: 4, min: 2, max: 4, sub: rtx16x32, ctx: 3},
	rtx64x16: {w: 16, h: 4, lw: 4, lh: 2, min: 2, max: 4, sub: rtx32x16, ctx: 3},
}

var itxShift = [nRectTxSizes]int{
	tx4x4:    0,
	tx8x8:    1,
	tx16x16:  2,
	tx32x32:  2,
	tx64x64:  2,
	rtx4x8:   0,
	rtx8x4:   0,
	rtx8x16:  1,
	rtx16x8:  1,
	rtx16x32: 1,
	rtx32x16: 1,
	rtx32x64: 1,
	rtx64x32: 1,
	rtx4x16:  1,
	rtx16x4:  1,
	rtx8x32:  2,
	rtx32x8:  2,
	rtx16x64: 2,
	rtx64x16: 2,
}

// itxTxtpSwap reorders a TxfmType from bitstream vertical_horizontal to pass order.
var itxTxtpSwap = [nTxTypes]uint8{
	dctDct:           dctDct,
	adstDct:          dctAdst,
	dctAdst:          adstDct,
	adstAdst:         adstAdst,
	flipadstDct:      dctFlipadst,
	dctFlipadst:      flipadstDct,
	flipadstFlipadst: flipadstFlipadst,
	adstFlipadst:     flipadstAdst,
	flipadstAdst:     adstFlipadst,
	idtx:             idtx,
	vDct:             hDct,
	hDct:             vDct,
	vAdst:            hAdst,
	hAdst:            vAdst,
	vFlipadst:        hFlipadst,
	hFlipadst:        vFlipadst,
}

// widenCoefs widens one column of coefficients per step into the row buffer
// and zeroes the lanes past the last nonzero one.
// widenCoefs16 and widenCoefs16Rect2 are the assembly forms, set once from
// what the CPU reports. Twelve bit coefficients are already int32 and take the
// Go path, where the plain widening is a copy.
var (
	widenCoefs16      func(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
	widenCoefs16Rect2 func(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
)

func widenCoefs[C coef](dst []int32, dstStride int, src []C, srcStride, cols, n, lanes int) {
	var z C
	if unsafe.Sizeof(z) == 2 && widenCoefs16 != nil && cols > 0 {
		widenCoefs16(&dst[0], dstStride,
			(*int16)(unsafe.Pointer(unsafe.SliceData(src))), srcStride, cols, n, lanes)

		return
	}
	if unsafe.Sizeof(z) == 4 {
		for x := range cols {
			d := dst[x*dstStride:]
			s := unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(src))), len(src))
			copy(d[:n], s[x*srcStride:][:n])
			clear(d[n:lanes])
		}

		return
	}
	for x := range cols {
		d := dst[x*dstStride:]
		for y, v := range src[x*srcStride:][:n] {
			d[y] = int32(v)
		}
		clear(d[n:lanes])
	}
}

// widenCoefsRect2 is widenCoefs for the transforms whose sides differ by two,
// which carry an extra 1/sqrt(2).
func widenCoefsRect2[C coef](dst []int32, dstStride int, src []C, srcStride, cols, n, lanes int) {
	var z C
	if unsafe.Sizeof(z) == 2 && widenCoefs16Rect2 != nil && cols > 0 {
		widenCoefs16Rect2(&dst[0], dstStride,
			(*int16)(unsafe.Pointer(unsafe.SliceData(src))), srcStride, cols, n, lanes)

		return
	}
	for x := range cols {
		d := dst[x*dstStride:]
		for y, v := range src[x*srcStride:][:n] {
			d[y] = (int32(v)*181 + 128) >> 8
		}
		clear(d[n:lanes])
	}
}

func invTxfmAdd[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	coeff []C, tmp []int32, eob, tx, txtp int, bitdepthMax int32,
) {
	txtp = int(itxTxtpSwap[txtp])
	tDim := &txfmDimensions[tx]
	w, h := 4*int(tDim.w), 4*int(tDim.h)
	hasDconly := txtp == dctDct
	shift := itxShift[tx]

	isRect2 := w*2 == h || h*2 == w
	rnd := int32((1 << shift) >> 1)

	if (hasDconly && eob == 0) || (!hasDconly && eob < 0) {
		dc := int32(coeff[0])
		coeff[0] = 0
		if isRect2 {
			dc = (dc*181 + 128) >> 8
		}
		dc = (dc*181 + 128) >> 8
		dc = (dc + rnd) >> shift
		dc = (dc*181 + 128 + 2048) >> 12
		dsp.itxDC(dst, dstOff, stride, dc, w, h, bitdepthMax)

		return
	}

	txtps := &tx1dTypes[txtp]
	sh, sw := min(h, 32), min(w, 32)

	var rowClipMin, colClipMin int32
	if bitdepthMax == 0xff {
		rowClipMin, colClipMin = math.MinInt16, math.MinInt16
	} else {
		rowClipMin = int32(^uint32(bitdepthMax) << 7)
		colClipMin = int32(^uint32(bitdepthMax) << 5)
	}
	rowClipMax, colClipMax := ^rowClipMin, ^colClipMin

	var lastNonzeroCol int
	switch {
	case txtps[1] == tx1dIdentity && txtps[0] != tx1dIdentity:
		lastNonzeroCol = min(sh-1, eob)
	case txtps[0] == tx1dIdentity && txtps[1] != tx1dIdentity:
		lastNonzeroCol = eob >> (tDim.lw + 2)
	default:
		lastNonzeroCol = int(lastNonzeroColFromEob[tx][eob])
	}

	n := lastNonzeroCol + 1
	lanes, ws, ts, tw, th := n, sh, w, w, h
	if g := dsp.itxLanes; g > 0 {
		ws, ts, tw, th = max(sh, g), max(w, g), max(w, g), max(h, g)
		lanes = min((n+g-1)/g*g, ws)
	}
	wide := tmp[th*ts : th*ts+tw*ws]

	if isRect2 {
		widenCoefsRect2(wide, ws, coeff, sh, sw, n, lanes)
	} else {
		widenCoefs(wide, ws, coeff, sh, sw, n, lanes)
	}
	for x := sw; x < tw; x++ {
		clear(wide[x*ws:][:lanes])
	}

	clear(coeff[:sw*sh])

	if !dsp.itxCol(wide, lanes, ws, w, int(txtps[0]), rowClipMin, rowClipMax, tmp[itxDct64Scratch:]) {
		first1d := tx1dFns[tDim.lw][txtps[0]]
		for y := range n {
			first1d(wide, y, ws, rowClipMin, rowClipMax)
		}
		lanes = n
	}

	dsp.itxTranspose(wide, tmp, tw, ws, ts, lanes)
	dsp.itxClip(tmp[:lanes*ts], rnd, shift, colClipMin, colClipMax)
	clear(tmp[lanes*ts : th*ts])

	if !dsp.itxCol(tmp, ts, ts, h, int(txtps[1]), colClipMin, colClipMax, tmp[itxDct64Scratch:]) {
		second1d := tx1dFns[tDim.lh][txtps[1]]
		for x := range w {
			second1d(tmp, x, ts, colClipMin, colClipMax)
		}
	}

	dsp.itxAdd(dst, dstOff, stride, tmp, w, h, ts, bitdepthMax)
}

func invTxfmAddWht4x4[P pixel, C coef](dst []P, dstOff, stride int, coeff []C,
	eob int, bitdepthMax int32,
) {
	var tmp [4 * 4]int32

	for y := range 4 {
		c := y * 4
		for x := range 4 {
			tmp[c+x] = int32(coeff[y+x*4]) >> 2
		}
		invWht41d(tmp[:], c, 1)
	}
	for i := range 4 * 4 {
		coeff[i] = 0
	}

	for x := range 4 {
		invWht41d(tmp[:], x, 4)
	}

	for y := range 4 {
		row := dstOff + y*stride
		for x := range 4 {
			dst[row+x] = P(clip(int32(dst[row+x])+tmp[y*4+x], 0, bitdepthMax))
		}
	}
}

func itxfmAdd[P pixel, C coef](dsp *dspContext[P], dst []P, dstOff, stride int,
	coeff []C, tmp []int32, eob, tx, txtp int, bitdepthMax int32,
) {
	if txtp == whtWht {
		invTxfmAddWht4x4(dst, dstOff, stride, coeff, eob, bitdepthMax)

		return
	}
	invTxfmAdd(dsp, dst, dstOff, stride, coeff, tmp, eob, tx, txtp, bitdepthMax)
}

func itxTransposeGo(wide, tmp []int32, w, ws, ts, n int) {
	for y := range n {
		d := tmp[y*ts:][:w]
		for x := range d {
			d[x] = wide[x*ws+y]
		}
	}
}

func itxClipGo(tmp []int32, rnd int32, shift int, lo, hi int32) {
	for i, v := range tmp {
		tmp[i] = clip((v+rnd)>>shift, lo, hi)
	}
}

func itxColNone([]int32, int, int, int, int, int32, int32, []int32) bool {
	return false
}

func itxDCGo[P pixel](dst []P, dstOff, stride int, dc int32, w, h int,
	bitdepthMax int32,
) {
	for y := range h {
		row := dst[dstOff+y*stride:][:w]
		for x := range row {
			row[x] = P(clip(int32(row[x])+dc, 0, bitdepthMax))
		}
	}
}

func itxAddGo[P pixel](dst []P, dstOff, stride int, tmp []int32, w, h, ts int,
	bitdepthMax int32,
) {
	for y := range h {
		row := dst[dstOff+y*stride:][:w]
		src := tmp[y*ts:][:w]
		for x, v := range src {
			row[x] = P(clip(int32(row[x])+((v+8)>>4), 0, bitdepthMax))
		}
	}
}
