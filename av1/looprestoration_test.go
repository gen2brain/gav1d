package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	lrStride  = 512
	lrRows    = 96
	lrPad     = 16
	lrLpfRows = 16
)

func lrGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func lrBuffers[P pixel](b, iter int, bitdepthMax int32) (dst, lpf, left []P) {
	dst = make([]P, lrRows*lrStride)
	for i := range dst {
		dst[i] = P(lrGen(iter, b*8+0, i) & uint16(bitdepthMax))
	}
	lpf = make([]P, lrLpfRows*lrStride)
	for i := range lpf {
		lpf[i] = P(lrGen(iter, b*8+1, i) & uint16(bitdepthMax))
	}
	left = make([]P, lrRows*4)
	for i := range left {
		left[i] = P(lrGen(iter, b*8+2, i) & uint16(bitdepthMax))
	}

	return dst, lpf, left
}

func runLR[P pixel, C coef](b, iter, kind, w, h, edges int,
	params *looprestorationParams, bitdepthMax int32,
) []int {
	dst, lpf, left := lrBuffers[P](b, iter, bitdepthMax)

	base := lrPad*lrStride + lrPad
	lpfBase := 2*lrStride + lrPad
	d := newLr[P, C]()

	switch kind {
	case 0:
		lrWiener[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	case 1:
		lrSgr5x5[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	case 2:
		lrSgr3x3[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	default:
		lrSgrMix[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	}

	out := make([]int, 0, w*h)
	for y := range h {
		for x := range w {
			out = append(out, int(dst[base+y*lrStride+x]))
		}
	}

	return out
}

func TestLoopRestoration(t *testing.T) {
	sc, closeFixture := openFixture(t, "lr.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "lr" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		bpc := num(fields[1])
		iter := num(fields[2])
		kind := num(fields[3])
		w, h := num(fields[4]), num(fields[5])
		edges := num(fields[6])
		bitdepthMax := int32(1<<bpc - 1)
		b := (bpc - 8) / 2

		var params looprestorationParams
		p := 7
		if kind == 0 {
			for i := range 7 {
				params.filter[0][i] = int16(num(fields[p+i]))
				params.filter[1][i] = int16(num(fields[p+7+i]))
			}
			p += 14
		} else {
			params.sgr.s0 = uint32(num(fields[p]))
			params.sgr.s1 = uint32(num(fields[p+1]))
			params.sgr.w0 = int16(num(fields[p+2]))
			params.sgr.w1 = int16(num(fields[p+3]))
			p += 4
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		want := fields[p+1:]

		var got []int
		if bpc == 8 {
			got = runLR[uint8, int16](b, iter, kind, w, h, edges, &params, bitdepthMax)
		} else {
			got = runLR[uint16, int32](b, iter, kind, w, h, edges, &params, bitdepthMax)
		}

		if len(got) != len(want) {
			t.Fatalf("line %d: %d outputs, want %d", line, len(got), len(want))
		}
		for i, v := range got {
			if v != num(want[i]) {
				t.Fatalf("line %d: bpc=%d kind=%d %dx%d edges=%d out[%d][%d] = %d, want %s",
					line, bpc, kind, w, h, edges, i/w, i%w, v, want[i])
			}
		}

		cases++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d cases", cases)
}

func wienerTaps(iter, k int, bitdepthMax int32) looprestorationParams {
	var params looprestorationParams
	f := &params.filter

	for p := range 2 {
		f[p][0] = int16(lrGen(iter, 9+p, 3*k+0)%16) - 5
		f[p][1] = int16(lrGen(iter, 9+p, 3*k+1)%32) - 23
		f[p][2] = int16(lrGen(iter, 9+p, 3*k+2)%64) - 17
		f[p][6], f[p][5], f[p][4] = f[p][0], f[p][1], f[p][2]
	}

	f[0][3] = -(f[0][0] + f[0][1] + f[0][2]) * 2
	if bitdepthMax != 255 {
		f[0][3] += 128
	}
	f[1][3] = 128 - (f[1][0]+f[1][1]+f[1][2])*2

	return params
}

func runWiener[P pixel, C coef](d *lrContext[P, C], b, iter, w, h, edges int,
	params *looprestorationParams, bitdepthMax int32,
) []P {
	dst, lpf, left := lrBuffers[P](b, iter, bitdepthMax)

	lrWiener[P, C](d, dst, lrPad*lrStride+lrPad, lrStride, left, 0,
		lpf, 2*lrStride+lrPad, w, h, params, edges, bitdepthMax)

	return dst
}

// TestWienerAsm checks the architecture-specific Wiener filter against the pure
// Go one over every edge combination and a spread of widths that straddle the
// vector width. The whole buffer is compared, so a kernel that writes outside
// the unit fails.
func TestWienerAsm(t *testing.T) {
	cases := 0
	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		if bpc == 8 {
			cases += wienerAsmDepth[uint8, int16](t, bitdepthMax)

			continue
		}
		cases += wienerAsmDepth[uint16, int32](t, bitdepthMax)
	}

	t.Logf("%d cases", cases)
}

func wienerAsmDepth[P pixel, C coef](t *testing.T, bitdepthMax int32) int {
	t.Helper()

	ref, got := newLrGo[P, C](), newLr[P, C]()
	b := (bitdepthFromMax(bitdepthMax) - 8) / 2
	widths := []int{1, 2, 5, 6, 7, 8, 15, 16, 17, 31, 32, 33, 47, 64, 253, 256}
	heights := []int{1, 2, 3, 4, 5, 8, 17, 48}
	cases := 0

	for iter := range 2 {
		for k, w := range widths {
			params := wienerTaps(iter, k, bitdepthMax)
			for _, h := range heights {
				for edges := range 16 {
					want := runWiener[P, C](ref, b, iter, w, h, edges, &params, bitdepthMax)
					have := runWiener[P, C](got, b, iter, w, h, edges, &params, bitdepthMax)
					for i := range want {
						if have[i] == want[i] {
							continue
						}
						t.Fatalf("bpc=%d %dx%d edges=%d iter=%d: buf[%d] = %d, want %d",
							bitdepthFromMax(bitdepthMax), w, h, edges, iter,
							i, have[i], want[i])
					}
					cases++
				}
			}
		}
	}

	return cases
}

func runSgr[P pixel, C coef](d *lrContext[P, C], b, iter, kind, w, h, edges int,
	params *looprestorationParams, bitdepthMax int32,
) []P {
	dst, lpf, left := lrBuffers[P](b, iter, bitdepthMax)

	base := lrPad*lrStride + lrPad
	lpfBase := 2*lrStride + lrPad

	switch kind {
	case 0:
		lrSgr5x5[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	case 1:
		lrSgr3x3[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	default:
		lrSgrMix[P, C](d, dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	}

	return dst
}

// TestSgrAsm checks the architecture-specific self-guided restoration against
// the pure Go one over every radius pair the spec allows, each edge
// combination, and widths that straddle the vector width. The whole buffer is
// compared, so a kernel that writes outside the unit fails.
func TestSgrAsm(t *testing.T) {
	cases := 0
	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		if bpc == 8 {
			cases += sgrAsmDepth[uint8, int16](t, bitdepthMax)

			continue
		}
		cases += sgrAsmDepth[uint16, int32](t, bitdepthMax)
	}

	t.Logf("%d cases", cases)
}

func sgrAsmDepth[P pixel, C coef](t *testing.T, bitdepthMax int32) int {
	t.Helper()

	ref, got := newLrGo[P, C](), newLr[P, C]()
	b := (bitdepthFromMax(bitdepthMax) - 8) / 2
	widths := []int{1, 2, 7, 8, 9, 15, 16, 17, 31, 33, 64, 253, 256}
	heights := []int{1, 2, 3, 4, 5, 8, 33}
	cases := 0

	for idx := range sgrParams {
		sp := &sgrParams[idx]
		kind := b2i(sp[0] != 0) + b2i(sp[1] != 0)*2 - 1
		if kind < 0 {
			continue
		}
		for iter := range 2 {
			var params looprestorationParams
			params.sgr.s0 = uint32(sp[0])
			params.sgr.s1 = uint32(sp[1])
			params.sgr.w0 = int16(lrGen(iter, 11, idx) % 97)
			params.sgr.w1 = 128 - params.sgr.w0 - int16(lrGen(iter, 12, idx)%97)

			for k, w := range widths {
				h := heights[k%len(heights)]
				for edges := range 16 {
					want := runSgr[P, C](ref, b, iter, kind, w, h, edges, &params, bitdepthMax)
					have := runSgr[P, C](got, b, iter, kind, w, h, edges, &params, bitdepthMax)
					for i := range want {
						if have[i] == want[i] {
							continue
						}
						t.Fatalf("bpc=%d kind=%d %dx%d edges=%d idx=%d iter=%d: "+
							"buf[%d] = %d, want %d",
							bitdepthFromMax(bitdepthMax), kind, w, h, edges, idx,
							iter, i, have[i], want[i])
					}
					cases++
				}
			}
		}
	}

	return cases
}
