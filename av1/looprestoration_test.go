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

func runLR[P pixel, C coef](b, iter, kind, w, h, edges int,
	params *looprestorationParams, bitdepthMax int32,
) []int {
	dst := make([]P, lrRows*lrStride)
	for i := range dst {
		dst[i] = P(lrGen(iter, b*8+0, i) & uint16(bitdepthMax))
	}
	lpf := make([]P, lrLpfRows*lrStride)
	for i := range lpf {
		lpf[i] = P(lrGen(iter, b*8+1, i) & uint16(bitdepthMax))
	}
	left := make([]P, lrRows*4)
	for i := range left {
		left[i] = P(lrGen(iter, b*8+2, i) & uint16(bitdepthMax))
	}

	base := lrPad*lrStride + lrPad
	lpfBase := 2*lrStride + lrPad

	switch kind {
	case 0:
		lrWiener[P, C](dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	case 1:
		lrSgr5x5[P, C](dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	case 2:
		lrSgr3x3[P, C](dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
			params, edges, bitdepthMax)
	default:
		lrSgrMix[P, C](dst, base, lrStride, left, 0, lpf, lpfBase, w, h,
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
