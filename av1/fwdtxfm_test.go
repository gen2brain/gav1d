package av1

import (
	"math"
	"testing"
)

func fwdRoundtrip(t *testing.T, tx, txtp int, scale float64) float64 {
	t.Helper()

	tDim := &txfmDimensions[tx]
	w, h := 4*int(tDim.w), 4*int(tDim.h)
	sw, sh := min(w, 32), min(h, 32)

	const stride = 128
	src := make([]int32, h*stride)
	pix := make([]uint8, (h+8)*stride)
	want := make([]uint8, (h+8)*stride)

	g := uint32(12345)
	next := func(m uint32) uint32 {
		g = g*1664525 + 1013904223

		return g >> 16 % m
	}
	for y := range h {
		for x := range w {
			v := int32(next(256))
			src[y*stride+x] = v - 128
			pix[y*stride+x] = 128
			want[y*stride+x] = uint8(v)
		}
	}

	coeff := make([]int32, sw*sh)
	c16 := make([]int16, sw*sh)
	fwdTxfmScaled(coeff, src, stride, tx, txtp, scale)
	for i, v := range coeff {
		c16[i] = int16(max(min(v, 32767), -32768))
	}

	d := newDsp[uint8]()
	tmp := make([]int32, 64*64*2+itxDct64Scratch)
	invTxfmAdd(d, pix, 0, stride, c16, tmp, sw*sh-1, tx, txtp, 255)

	var se float64
	for y := range h {
		for x := range w {
			e := float64(int(pix[y*stride+x]) - int(want[y*stride+x]))
			se += e * e
		}
	}

	return math.Sqrt(se / float64(w*h))
}

func TestFwdTxfm(t *testing.T) {
	for tx := range nRectTxSizes {
		if !fwdTxfmSupported(tx) {
			continue
		}
		tDim := &txfmDimensions[tx]
		r := fwdRoundtrip(t, tx, dctDct, fwdScale(tx))
		if r > 0.25 {
			t.Errorf("tx=%d %dx%d rmse %.4f", tx, 4*tDim.w, 4*tDim.h, r)
		}
	}
}

func TestQuantRoundtrip(t *testing.T) {
	const stride = 64
	src := make([]int32, 8*stride)
	g := uint32(999)
	next := func(m uint32) uint32 {
		g = g*1664525 + 1013904223

		return g >> 16 % m
	}
	for y := range 8 {
		for x := range 8 {
			src[y*stride+x] = int32(next(256)) - 128
		}
	}

	coeff := make([]int32, 64)
	levels := make([]int32, 64)
	back := make([]int32, 64)
	fwdTxfm(coeff, src, stride, tx8x8, dctDct)

	for _, qidx := range []int{20, 60, 120, 200, 255} {
		dc := uint32(dqTbl[0][qidx][0])
		ac := uint32(dqTbl[0][qidx][1])
		sh := quantShift(tx8x8)
		quantBlock(levels, coeff, 64, dc, ac, sh)
		dequantBlock(back, levels, 64, dc, ac, sh, 1<<15)

		var worst int32
		for i := range 64 {
			d := back[i] - coeff[i]
			if d < 0 {
				d = -d
			}
			step := int32(ac)
			if i == 0 {
				step = int32(dc)
			}
			if d > worst {
				worst = d
			}
			if d > step {
				t.Fatalf("qidx=%d coef %d: error %d exceeds step %d", qidx, i, d, step)
			}
		}
		t.Logf("qidx=%3d dc=%4d ac=%4d worst coefficient error %d", qidx, dc, ac, worst)
	}
}
