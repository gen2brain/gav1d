package av1

import (
	"math"
	"testing"
)

func fwdRoundtrip(t *testing.T, tx, txtp, pattern int) float64 {
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
			var v int32
			switch pattern {
			case 1:
				v = int32(255 * ((x + y) & 1))
			case 2:
				v = 255
			default:
				v = int32(next(256))
			}
			src[y*stride+x] = v - 128
			pix[y*stride+x] = 128
			want[y*stride+x] = uint8(v)
		}
	}

	coeff := make([]int32, sw*sh)
	c16 := make([]int16, sw*sh)
	fwdTxfm(coeff, src, stride, tx, txtp, make([]int32, fwdScratch))
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
		for txtp := range nTxTypes {
			if int(tDim.max)+1 >= tx64x64 && txtp != dctDct && txtp != idtx {
				continue
			}
			if max(tDim.w, tDim.h) > 4 && txtp >= idtx {
				continue
			}
			for pattern := range 3 {
				r := fwdRoundtrip(t, tx, txtp, pattern)
				if r > 0.25 {
					t.Errorf("tx=%d %dx%d txtp=%d pattern=%d rmse %.4f",
						tx, 4*tDim.w, 4*tDim.h, txtp, pattern, r)
				}
			}
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
	fwdTxfm(coeff, src, stride, tx8x8, dctDct, make([]int32, fwdScratch))

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

func quantOneRef(c int32, dq uint32, shift int) int32 {
	neg := c < 0
	if neg {
		c = -c
	}
	v := (uint32(c)<<uint(shift) + dq*uint32(quantRound)/16) / dq
	if v > 0xfffff {
		v = 0xfffff
	}
	if neg {
		return -int32(v)
	}

	return int32(v)
}

func TestQuantReciprocal(t *testing.T) {
	seen := map[uint16]bool{}
	var dqs []uint16
	for qidx := range 256 {
		for _, d := range dqTbl[0][qidx] {
			if !seen[d] {
				seen[d] = true
				dqs = append(dqs, d)
			}
		}
	}

	g := uint64(12345)
	next := func() uint32 {
		g = g*6364136223846793005 + 1442695040888963407

		return uint32(g >> 32)
	}

	for _, d := range dqs {
		q := newQuantizer(uint32(d))
		for _, shift := range []int{0, 1, 2} {
			var cs []int32
			for k := range 300 {
				n := int32(k) * int32(d)
				cs = append(cs, n-1, n, n+1)
			}
			for range 20000 {
				cs = append(cs, int32(next()>>uint(shift)))
			}
			cs = append(cs, 0, 1, -1, 0x7fffffff>>uint(shift), -(0x7fffffff >> uint(shift)))

			for _, c := range cs {
				for _, v := range []int32{c, -c} {
					got, want := quantOne(v, &q, shift), quantOneRef(v, uint32(d), shift)
					if got != want {
						t.Fatalf("dq=%d shift=%d c=%d: got %d want %d", d, shift, v, got, want)
					}
				}
			}
		}
	}
	t.Logf("%d distinct dequant values verified", len(dqs))
}
