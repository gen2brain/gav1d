package av1

import "testing"

func TestSseAsm(t *testing.T) {
	if !hasAsmSse() {
		t.Skip("no assembly")
	}

	const stride, sstride = 96, 80
	dst := make([]uint8, stride*40)
	src := make([]uint8, sstride*40)
	g := uint32(5)
	for i := range dst {
		g = g*1664525 + 1013904223
		dst[i] = uint8(g >> 16)
	}
	for i := range src {
		g = g*1664525 + 1013904223
		src[i] = uint8(g >> 16)
	}

	for _, w := range []int{4, 8, 16, 32} {
		for _, h := range []int{4, 8, 16, 32} {
			for _, off := range []int{0, 7, 64} {
				want := sseGo(dst, off, stride, src, sstride, w, h)
				if got := sseKernel(dst, off, stride, src, sstride, w, h); got != want {
					t.Fatalf("%dx%d off=%d: got %d, want %d", w, h, off, got, want)
				}
			}
		}
	}
}

func TestSatd4Asm(t *testing.T) {
	if !hasAsmSatd4() {
		t.Skip("no assembly")
	}

	g := uint32(17)
	next := func(m int32) int32 {
		g = g*1664525 + 1013904223

		return int32(g>>16)%(2*m+1) - m
	}
	for _, stride := range []int{4, 16, 32} {
		for _, mag := range []int32{1, 16, 255, 4095} {
			res := make([]int32, stride*4+8)
			for i := range res {
				res[i] = next(mag)
			}
			for _, off := range []int{0, 4} {
				want := satd4(res, off, stride)
				if got := satd4Kernel(res, off, stride); got != want {
					t.Fatalf("stride=%d mag=%d off=%d: got %d, want %d",
						stride, mag, off, got, want)
				}
			}
		}
	}
}

func TestFwdTxfm4x4Asm(t *testing.T) {
	if !hasAsmResidual() {
		t.Skip("no assembly")
	}

	g := uint32(11)
	next := func() int32 {
		g = g*1664525 + 1013904223

		return int32(g>>16%511) - 255
	}
	tmp := make([]int32, fwdScratch)
	for _, stride := range []int{4, 16, 33} {
		src := make([]int32, stride*4)
		for round := range 64 {
			for i := range src {
				src[i] = next()
				if round == 0 {
					src[i] = 255
				}
			}
			for _, txtp := range []int{dctDct, adstDct, dctAdst, adstAdst} {
				want := make([]int32, 16)
				got := make([]int32, 16)
				fwdTxfmGo(want, src, stride, tx4x4, txtp, tmp)
				if !fwdTxfm4x4(got, src, stride, txtp) {
					t.Skip("no 4x4 transform kernel")
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("stride=%d txtp=%d [%d]: got %d, want %d",
							stride, txtp, i, got[i], want[i])
					}
				}
			}
		}
	}
}

func TestFwdTxfm8x8Asm(t *testing.T) {
	g := uint32(23)
	next := func() int32 {
		g = g*1664525 + 1013904223

		return int32(g>>16%511) - 255
	}
	tmp := make([]int32, fwdScratch)
	for _, stride := range []int{8, 16, 40} {
		src := make([]int32, stride*8)
		for round := range 32 {
			for i := range src {
				src[i] = next()
				if round == 0 {
					src[i] = 255
				}
			}
			for _, txtp := range []int{dctDct, adstDct, dctAdst, adstAdst} {
				want := make([]int32, 64)
				got := make([]int32, 64)
				fwdTxfmGo(want, src, stride, tx8x8, txtp, tmp)
				if !fwdTxfm8x8(got, src, stride, txtp) {
					t.Skip("no 8x8 transform kernel")
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("txtp=%d stride=%d [%d]: got %d, want %d",
							txtp, stride, i, got[i], want[i])
					}
				}
			}
		}
	}
}

func TestSatd8Asm(t *testing.T) {
	if !hasAsmSatd8() {
		t.Skip("no assembly")
	}

	g := uint32(3)
	next := func(m int32) int32 {
		g = g*1664525 + 1013904223

		return int32(g>>16)%(2*m+1) - m
	}
	for _, stride := range []int{8, 16, 32} {
		for _, mag := range []int32{1, 16, 255, 4095} {
			res := make([]int32, stride*8+8)
			for i := range res {
				res[i] = next(mag)
			}
			for _, off := range []int{0, 8} {
				if off+7*stride+8 > len(res) {
					continue
				}
				want := satd8(res, off, stride)
				if got := satd8Kernel(res, off, stride); got != want {
					t.Fatalf("stride=%d mag=%d off=%d: got %d, want %d",
						stride, mag, off, got, want)
				}
			}
		}
	}
}

func TestResidualAsm(t *testing.T) {
	if !hasAsmResidual() {
		t.Skip("no assembly")
	}

	const stride, sstride = 96, 80
	dst := make([]uint8, stride*40)
	src := make([]uint8, sstride*40)
	g := uint32(1)
	next := func() uint8 {
		g = g*1664525 + 1013904223

		return uint8(g >> 16)
	}
	for i := range dst {
		dst[i] = next()
	}
	for i := range src {
		src[i] = next()
	}

	for _, w := range []int{4, 8, 16, 32} {
		for _, h := range []int{4, 8, 16, 32} {
			for _, off := range []int{0, 7, 64} {
				want := make([]int32, w*h)
				got := make([]int32, w*h)
				wantSSD := residualGo(want, dst, off, stride, src, sstride, w, h)
				gotSSD := residualKernel(got, dst, off, stride, src, sstride, w, h)

				if gotSSD != wantSSD {
					t.Fatalf("%dx%d off=%d: ssd %d, want %d", w, h, off, gotSSD, wantSSD)
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("%dx%d off=%d [%d]: got %d, want %d",
							w, h, off, i, got[i], want[i])
					}
				}
			}
		}
	}
}

func TestFwdTxfmRectAsm(t *testing.T) {
	g := uint32(77)
	next := func() int32 {
		g = g*1664525 + 1013904223

		return int32(g>>16%511) - 255
	}
	tmp := make([]int32, fwdScratch)
	for _, c := range []struct {
		tx     int
		w, h   int
		kernel func([]int32, []int32, int, int) bool
	}{
		{rtx4x8, 4, 8, fwdTxfm4x8},
		{rtx8x4, 8, 4, fwdTxfm8x4},
		{rtx8x16, 8, 16, fwdTxfm8x16},
		{rtx16x8, 16, 8, fwdTxfm16x8},
		{tx16x16, 16, 16, fwdTxfm16x16},
		{tx32x32, 32, 32, fwdTxfm32x32},
		{rtx16x32, 16, 32, fwdTxfm16x32},
		{rtx32x16, 32, 16, fwdTxfm32x16},
	} {
		probe := make([]int32, c.w*c.h)
		if !c.kernel(probe, make([]int32, 40*c.h), 40, dctDct) {
			t.Logf("%dx%d: no kernel", c.w, c.h)

			continue
		}
		for _, stride := range []int{c.w, c.w * 2, 40} {
			src := make([]int32, stride*c.h)
			for round := range 64 {
				for i := range src {
					src[i] = next()
					switch round {
					case 0:
						src[i] = 255
					case 1:
						src[i] = -255
					}
				}
				for _, txtp := range []int{dctDct, adstDct, dctAdst, adstAdst} {
					want := make([]int32, c.w*c.h)
					got := make([]int32, c.w*c.h)
					fwdTxfmGo(want, src, stride, c.tx, txtp, tmp)
					if !c.kernel(got, src, stride, txtp) {
						continue
					}
					for i := range want {
						if got[i] != want[i] {
							t.Fatalf("%dx%d txtp=%d stride=%d round=%d [%d]: got %d, want %d",
								c.w, c.h, txtp, stride, round, i, got[i], want[i])
						}
					}
				}
			}
		}
	}
}
