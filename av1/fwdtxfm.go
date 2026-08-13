package av1

import "math"

const fwdMaxDim = 64

var fwdDct [5][]float64

var fwdAdst [5][]float64

func fwdBasisIdx(n int) int {
	return ulog2(uint32(n)) - 2
}

func init() {
	for _, n := range []int{4, 8, 16, 32, 64} {
		i := fwdBasisIdx(n)
		d := make([]float64, n*n)
		for k := range n {
			s := math.Sqrt2
			if k == 0 {
				s = 1
			}
			for x := range n {
				d[k*n+x] = s * math.Cos(math.Pi*float64(2*x+1)*float64(k)/float64(2*n)) / math.Sqrt(float64(n))
			}
		}
		fwdDct[i] = d

		if n > 16 {
			continue
		}
		a := make([]float64, n*n)
		for k := range n {
			for x := range n {
				a[k*n+x] = 2 * math.Sin(math.Pi*float64(2*x+1)*float64(2*k+1)/float64(4*n)) / math.Sqrt(float64(2*n))
			}
		}
		fwdAdst[i] = a
	}
}

func fwdBasis(t1d, n int) []float64 {
	switch t1d {
	case tx1dDct:
		return fwdDct[fwdBasisIdx(n)]
	case tx1dAdst, tx1dFlipadst:
		return fwdAdst[fwdBasisIdx(n)]
	}

	return nil
}

func fwd1d(dst, src []float64, t1d, n int, flip bool) {
	if flip {
		for i := range n / 2 {
			src[i], src[n-1-i] = src[n-1-i], src[i]
		}
	}

	b := fwdBasis(t1d, n)
	if b == nil {
		for i := range n {
			dst[i] = src[i] * math.Sqrt2
		}

		return
	}

	for k := range n {
		var s float64
		row := b[k*n:]
		for x := range n {
			s += row[x] * src[x]
		}
		dst[k] = s
	}
}

func txtpFlip(txtp int) (bool, bool) {
	var ud, lr bool
	switch txtp {
	case flipadstDct, flipadstAdst, vFlipadst:
		ud = true
	case dctFlipadst, adstFlipadst, hFlipadst:
		lr = true
	case flipadstFlipadst:
		ud, lr = true, true
	}

	return ud, lr
}

func fwdTxfm(coeff []int32, src []int32, srcStride, tx, txtp int) {
	fwdTxfmScaled(coeff, src, srcStride, tx, txtp, fwdScale(tx))
}

func fwdTxfmScaled(coeff []int32, src []int32, srcStride, tx, txtp int, scale float64) {
	tDim := &txfmDimensions[tx]
	w, h := 4*int(tDim.w), 4*int(tDim.h)
	sw, sh := min(w, 32), min(h, 32)
	txtps := &tx1dTypes[int(itxTxtpSwap[txtp])]
	ud, lr := txtpFlip(txtp)

	var line [fwdMaxDim]float64
	var out [fwdMaxDim]float64
	tmp := make([]float64, w*h)

	for x := range w {
		for y := range h {
			line[y] = float64(src[y*srcStride+x])
		}
		fwd1d(out[:], line[:h], int(txtps[1]), h, ud)
		for y := range h {
			tmp[y*w+x] = out[y]
		}
	}

	clear(coeff[:sw*sh])
	for y := range sh {
		copy(line[:w], tmp[y*w:])
		fwd1d(out[:], line[:w], int(txtps[0]), w, lr)
		for x := range sw {
			coeff[x*sh+y] = int32(math.Round(out[x] * scale))
		}
	}
}

var fwdScaleTbl = [nRectTxSizes]float64{
	tx4x4: 8, tx8x8: 8, tx16x16: 8, tx32x32: 4,
	rtx4x8: 8, rtx8x4: 8, rtx8x16: 8, rtx16x8: 8,
	rtx16x32: 4, rtx32x16: 4, rtx4x16: 8, rtx16x4: 8,
	rtx8x32: 8, rtx32x8: 8,
}

func fwdScale(tx int) float64 {
	return fwdScaleTbl[tx]
}

func fwdTxfmSupported(tx int) bool {
	return fwdScaleTbl[tx] != 0
}
