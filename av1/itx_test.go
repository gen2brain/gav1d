package av1

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestItx(t *testing.T) {
	sc, closeFixture := openFixture(t, "itx.txt")
	defer closeFixture()

	const stride = 80
	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "itx" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		bpc, tx, txtp, eob, w := num(fields[1]), num(fields[2]), num(fields[3]), num(fields[4]), num(fields[5])
		bitdepthMax := int32(1<<bpc - 1)
		h := 4 * int(txfmDimensions[tx].h)
		sw, sh := min(w, 32), min(h, 32)
		nCoef := sw * sh

		bar := 6 + nCoef
		if fields[bar] != "|" {
			t.Fatalf("line %d: malformed record", line)
		}
		eq := bar + 1 + w*h
		if fields[eq] != "=" {
			t.Fatalf("line %d: malformed record at =", line)
		}
		cmark := eq + 1 + w*h
		if fields[cmark] != "c" {
			t.Fatalf("line %d: malformed record at c", line)
		}

		coeffIn := fields[6:bar]
		srcIn := fields[bar+1 : eq]
		wantDst := fields[eq+1 : cmark]
		wantCoef := fields[cmark+1:]

		check := func(gotDst func(y, x int) int, gotCoef func(i int) int) {
			for y := range h {
				for x := range w {
					if gotDst(y, x) != num(wantDst[y*w+x]) {
						t.Fatalf("line %d: bpc=%d tx=%d txtp=%d eob=%d dst[%d][%d] = %d, want %s",
							line, bpc, tx, txtp, eob, y, x, gotDst(y, x), wantDst[y*w+x])
					}
				}
			}
			for i := range nCoef {
				if gotCoef(i) != num(wantCoef[i]) {
					t.Fatalf("line %d: bpc=%d tx=%d txtp=%d eob=%d coeff[%d] = %d, want %s",
						line, bpc, tx, txtp, eob, i, gotCoef(i), wantCoef[i])
				}
			}
		}

		if bpc == 8 {
			dst := make([]uint8, h*stride)
			for y := range h {
				for x := range w {
					dst[y*stride+x] = uint8(num(srcIn[y*w+x]))
				}
			}
			coeff := make([]int16, nCoef)
			for i := range nCoef {
				coeff[i] = int16(num(coeffIn[i]))
			}
			if txtp == whtWht {
				invTxfmAddWht4x4(dst, 0, stride, coeff, eob, bitdepthMax)
			} else {
				invTxfmAdd(newDspGo[uint8](), dst, 0, stride, coeff, make([]int32, itxScratchLen), eob, tx, txtp, bitdepthMax)
			}
			check(func(y, x int) int { return int(dst[y*stride+x]) },
				func(i int) int { return int(coeff[i]) })
		} else {
			dst := make([]uint16, h*stride)
			for y := range h {
				for x := range w {
					dst[y*stride+x] = uint16(num(srcIn[y*w+x]))
				}
			}
			coeff := make([]int32, nCoef)
			for i := range nCoef {
				coeff[i] = int32(num(coeffIn[i]))
			}
			if txtp == whtWht {
				invTxfmAddWht4x4(dst, 0, stride, coeff, eob, bitdepthMax)
			} else {
				invTxfmAdd(newDspGo[uint16](), dst, 0, stride, coeff, make([]int32, itxScratchLen), eob, tx, txtp, bitdepthMax)
			}
			check(func(y, x int) int { return int(dst[y*stride+x]) },
				func(i int) int { return int(coeff[i]) })
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

// TestItxTailAsm checks the architecture-specific transform tails against the
// pure Go ones over every block shape, comparing the whole buffer so a kernel
// that writes outside the block fails.
func TestItxTailAsm(t *testing.T) {
	sizes := []int{4, 8, 16, 32, 64}
	cases := 0

	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		for _, w := range sizes {
			for _, h := range sizes {
				for iter := range 3 {
					if bpc == 8 {
						cases += itxTailCase[uint8](t, newDsp[uint8](), iter, w, h, bitdepthMax)

						continue
					}
					cases += itxTailCase[uint16](t, newDsp[uint16](), iter, w, h, bitdepthMax)
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}

func itxTailCase[P pixel](t *testing.T, d *dspContext[P], iter, w, h int,
	bitdepthMax int32,
) int {
	t.Helper()

	const stride = 128
	gen := func(kind, i int) uint32 {
		v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
		v ^= v >> 15

		return v * 2654435761
	}

	ref := make([]int32, w*h)
	got := make([]int32, w*h)
	for i := range ref {
		ref[i] = int32(gen(0, i)) >> 8
		got[i] = ref[i]
	}
	lo, hi := int32(-(1 << 20)), int32(1<<20-1)
	itxClipGo(ref, 8, 4, lo, hi)
	d.itxClip(got, 8, 4, lo, hi)
	for i := range ref {
		if got[i] != ref[i] {
			t.Fatalf("bpc=%d %dx%d clip[%d] = %d, want %d",
				bitdepthFromMax(bitdepthMax), w, h, i, got[i], ref[i])
		}
	}

	sh := min(h, 32)
	wide := make([]int32, w*sh)
	for i := range wide {
		wide[i] = int32(gen(2, i)) >> 8
	}
	for _, ts := range []int{w, max(w, 8)} {
		for _, n := range []int{sh, min(8, sh), 4} {
			tref := make([]int32, ts*h)
			tgot := make([]int32, ts*h)
			itxTransposeGo(wide, tref, w, sh, ts, n)
			d.itxTranspose(wide, tgot, w, sh, ts, n)
			for i := range tref {
				if tgot[i] != tref[i] {
					t.Fatalf("%dx%d transpose ts=%d n=%d buf[%d] = %d, want %d",
						w, h, ts, n, i, tgot[i], tref[i])
				}
			}
		}
	}

	for _, ats := range []int{w, max(w, 8)} {
		res := make([]int32, ats*h)
		for i := range res {
			res[i] = int32(gen(3, i)) >> 12
		}
		dref := make([]P, stride*(h+2))
		dgot := make([]P, stride*(h+2))
		for i := range dref {
			dref[i] = P(int32(gen(1, i)) & bitdepthMax)
			dgot[i] = dref[i]
		}
		itxAddGo(dref, stride, stride, res, w, h, ats, bitdepthMax)
		d.itxAdd(dgot, stride, stride, res, w, h, ats, bitdepthMax)
		for i := range dref {
			if dgot[i] != dref[i] {
				t.Fatalf("bpc=%d %dx%d ts=%d add buf[%d] = %d, want %d",
					bitdepthFromMax(bitdepthMax), w, h, ats, i, dgot[i], dref[i])
			}
		}
	}

	return 1
}

// TestItxColAsm checks the one-dimensional pass against the pure Go transforms.
func TestItxColAsm(t *testing.T) {
	d := newDsp[uint8]()
	if d.itxCol == nil {
		t.Skip("no column kernel")
	}
	cases := 0

	for iter := range 4 {
		for _, stride := range []int{8, 16, 32, 64} {
			for _, lanes := range []int{4, 8, stride} {
				if lanes > stride {
					continue
				}
				for _, n := range []int{4, 8, 16, 32, 64} {
					for _, txtp := range []int{tx1dDct, tx1dAdst, tx1dIdentity, tx1dFlipadst} {
						scratch := make([]int32, 256)
						lo, hi := int32(math.MinInt16), int32(math.MaxInt16)
						ref := make([]int32, n*stride)
						got := make([]int32, n*stride)
						for i := range ref {
							v := uint32(i)*2654435761 + uint32(iter)*3266489917
							v ^= v >> 15
							ref[i] = int32(v*2654435761)>>18 | 1
							got[i] = ref[i]
						}
						if n == 64 {
							clear(ref[32*stride:])
							clear(got[32*stride:])
						}

						if !d.itxCol(got, lanes, stride, n, txtp, lo, hi, scratch) {
							continue
						}
						fn := tx1dFns[ulog2(uint32(n))-2][txtp]
						if fn == nil {
							t.Fatalf("n=%d txtp=%d: kernel took a transform Go has not",
								n, txtp)
						}
						for i := range lanes {
							fn(ref, i, stride, lo, hi)
						}
						for i := range ref {
							if got[i] != ref[i] {
								t.Fatalf("lanes=%d stride=%d n=%d txtp=%d: [%d,%d] = %d, want %d",
									lanes, stride, n, txtp, i/stride, i%stride, got[i], ref[i])
							}
						}
						cases++
					}
				}
			}
		}
	}

	if cases == 0 {
		t.Fatal("kernel never ran")
	}
	t.Logf("%d cases", cases)
}

func BenchmarkItx(b *testing.B) {
	cases := []struct {
		name     string
		tx, txtp int
	}{
		{"4x4-dct", tx4x4, dctDct},
		{"4x4-adst", tx4x4, adstAdst},
		{"8x4-dct", rtx8x4, dctDct},
		{"4x16-dct", rtx4x16, dctDct},
		{"8x8-dct", tx8x8, dctDct},
		{"16x16-dct", tx16x16, dctDct},
		{"32x32-dct", tx32x32, dctDct},
		{"64x64-dct", tx64x64, dctDct},
		{"16x16-adst", tx16x16, adstAdst},
		{"16x16-idtx", tx16x16, idtx},
		{"32x8-dct", rtx32x8, dctDct},
	}

	d := newDsp[uint8]()
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			tDim := &txfmDimensions[c.tx]
			w, h := 4*int(tDim.w), 4*int(tDim.h)
			sw, sh := min(w, 32), min(h, 32)

			src := make([]int16, sw*sh)
			for i := range src {
				src[i] = int16(i*7&511) - 256
			}
			coeff := make([]int16, sw*sh)
			tmp := make([]int32, itxScratchLen)
			dst := make([]uint8, 128*(h+2))

			for b.Loop() {
				copy(coeff, src)
				invTxfmAdd(d, dst, 0, 128, coeff, tmp, sw*sh-1, c.tx, c.txtp, 255)
			}
			b.ReportMetric(float64(w*h)*float64(b.N)/b.Elapsed().Seconds()/1e6, "Mpixel/s")
		})
	}
}

// TestWidenCoefsAsm checks the widening kernels against the Go loops they
// replace, over the shapes invTxfmAdd asks for: any number of nonzero rows,
// lanes rounded up past it, and strides wider than either.
func TestWidenCoefsAsm(t *testing.T) {
	if widenCoefs16 == nil {
		t.Skip("no widen assembly on this architecture")
	}

	asm, asmRect2 := widenCoefs16, widenCoefs16Rect2
	cases := 0

	for _, cols := range []int{1, 2, 3, 8, 16, 32} {
		for _, n := range []int{1, 2, 5, 7, 8, 9, 15, 16, 17, 31, 32, 64} {
			for _, extra := range []int{0, 1, 7} {
				lanes := (n+7)/8*8 + extra*8
				srcStride, dstStride := n+extra, lanes+extra

				src := make([]int16, cols*srcStride)
				for i := range src {
					v := int32(cdefGen(i*7+cols, n, i)) & 0xffff
					src[i] = int16(v - 0x8000)
				}
				if len(src) > 3 {
					src[0], src[1], src[2], src[3] = 0, -32768, 32767, 1
				}

				for _, rect2 := range []bool{false, true} {
					want := make([]int32, cols*dstStride)
					got := make([]int32, cols*dstStride)
					for i := range want {
						want[i], got[i] = -1, -1
					}

					widenCoefs16, widenCoefs16Rect2 = nil, nil
					if rect2 {
						widenCoefsRect2(want, dstStride, src, srcStride, cols, n, lanes)
					} else {
						widenCoefs(want, dstStride, src, srcStride, cols, n, lanes)
					}
					widenCoefs16, widenCoefs16Rect2 = asm, asmRect2
					if rect2 {
						widenCoefsRect2(got, dstStride, src, srcStride, cols, n, lanes)
					} else {
						widenCoefs(got, dstStride, src, srcStride, cols, n, lanes)
					}

					for i := range want {
						if got[i] != want[i] {
							t.Fatalf("rect2=%v cols=%d n=%d lanes=%d strides=%d/%d: [%d] = %d, want %d",
								rect2, cols, n, lanes, srcStride, dstStride, i, got[i], want[i])
						}
					}
					cases++
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}
