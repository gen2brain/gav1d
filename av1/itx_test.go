package av1

import (
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
				invTxfmAdd(dst, 0, stride, coeff, eob, tx, txtp, bitdepthMax)
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
				invTxfmAdd(dst, 0, stride, coeff, eob, tx, txtp, bitdepthMax)
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
