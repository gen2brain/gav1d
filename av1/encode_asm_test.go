package av1

import "testing"

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
