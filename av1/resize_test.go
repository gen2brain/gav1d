package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	resizeStride = 1024
	resizeRows   = 72
)

func resizeGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func runResize[P pixel](b, iter, dstW, srcW, h, dx, mx0 int, bitdepthMax int32) []int {
	src := make([]P, resizeRows*resizeStride)
	for i := range src {
		src[i] = P(resizeGen(iter, b*4+0, i) & uint16(bitdepthMax))
	}
	dst := make([]P, resizeRows*resizeStride)

	resize(dst, 0, resizeStride, src, 0, resizeStride, dstW, h, srcW, dx, mx0, bitdepthMax)

	out := make([]int, 0, dstW*h)
	for y := range h {
		for x := range dstW {
			out = append(out, int(dst[y*resizeStride+x]))
		}
	}

	return out
}

func TestResize(t *testing.T) {
	sc, closeFixture := openFixture(t, "resize.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "resize" {
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
		dstW, srcW := num(fields[3]), num(fields[4])
		h := num(fields[5])
		dx, mx0 := num(fields[6]), num(fields[7])
		bitdepthMax := int32(1<<bpc - 1)
		b := (bpc - 8) / 2

		if got := scaleFac(srcW, dstW); got != dx {
			t.Fatalf("line %d: scaleFac = %d, want %d", line, got, dx)
		}
		if got := getUpscaleX0(srcW, dstW, dx); got != mx0 {
			t.Fatalf("line %d: getUpscaleX0 = %d, want %d", line, got, mx0)
		}

		if fields[8] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		want := fields[9:]

		var got []int
		if bpc == 8 {
			got = runResize[uint8](b, iter, dstW, srcW, h, dx, mx0, bitdepthMax)
		} else {
			got = runResize[uint16](b, iter, dstW, srcW, h, dx, mx0, bitdepthMax)
		}

		if len(got) != len(want) {
			t.Fatalf("line %d: %d outputs, want %d", line, len(got), len(want))
		}
		for i, v := range got {
			if v != num(want[i]) {
				t.Fatalf("line %d: bpc=%d %d->%d h=%d out[%d][%d] = %d, want %s",
					line, bpc, srcW, dstW, h, i/dstW, i%dstW, v, want[i])
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
