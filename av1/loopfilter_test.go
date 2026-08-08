package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	lfStride = 128
	lfRows   = 96
	lfPad    = 16
	lfB4S    = 32
	lfWinA   = 64
	lfWinB   = 16
)

func lfWinOff(dir, a, c int) int {
	if dir != 0 {
		return (lfPad+c)*lfStride + lfPad + a
	}

	return (lfPad+a)*lfStride + lfPad + c
}

func runLF[P pixel](t *testing.T, line, dir, plane int, vmask *[3]uint32,
	l []uint8, lut *filterLUT, in, want []string, bitdepthMax int32,
	num func(string) int,
) {
	t.Helper()

	buf := make([]P, lfRows*lfStride)
	base := lfPad*lfStride + lfPad

	idx := 0
	for a := range lfWinA {
		for c := -lfWinB / 2; c < lfWinB/2; c++ {
			buf[lfWinOff(dir, a, c)] = P(num(in[idx]))
			idx++
		}
	}

	lOff := ((lfB4S/2)*lfB4S + lfB4S/2) * 4

	switch {
	case plane == 0 && dir == 0:
		loopFilterHSb128y(buf, base, lfStride, vmask, l, lOff, lfB4S, lut, bitdepthMax)
	case plane == 0:
		loopFilterVSb128y(buf, base, lfStride, vmask, l, lOff, lfB4S, lut, bitdepthMax)
	case dir == 0:
		loopFilterHSb128uv(buf, base, lfStride, (*[2]uint32)(vmask[:2]), l, lOff, lfB4S, lut, bitdepthMax)
	default:
		loopFilterVSb128uv(buf, base, lfStride, (*[2]uint32)(vmask[:2]), l, lOff, lfB4S, lut, bitdepthMax)
	}

	idx = 0
	for a := range lfWinA {
		for c := -lfWinB / 2; c < lfWinB/2; c++ {
			got := int(buf[lfWinOff(dir, a, c)])
			if got != num(want[idx]) {
				t.Fatalf("line %d: dir=%d plane=%d out[a=%d][c=%d] = %d, want %s",
					line, dir, plane, a, c, got, want[idx])
			}
			idx++
		}
	}
}

func TestLoopFilter(t *testing.T) {
	sc, closeFixture := openFixture(t, "lf.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "lf" {
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
		dir, plane := num(fields[2]), num(fields[3])
		sharp := num(fields[4])
		vmask := [3]uint32{
			uint32(num(fields[5])), uint32(num(fields[6])), uint32(num(fields[7])),
		}
		bitdepthMax := int32(1<<bpc - 1)

		p := 8
		l := make([]uint8, lfB4S*lfB4S*4)
		for i := range lfB4S * lfB4S {
			l[i*4] = uint8(num(fields[p+i]))
		}
		p += lfB4S * lfB4S

		n := lfWinA * lfWinB
		in := fields[p : p+n]
		p += n
		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		want := fields[p+1:]

		var lut filterLUT
		for level := range 64 {
			limit := level
			if sharp > 0 {
				limit >>= (sharp + 3) >> 2
				limit = min(limit, 9-sharp)
			}
			limit = max(limit, 1)
			lut.i[level] = uint8(limit)
			lut.e[level] = uint8(2*(level+2) + limit)
		}

		if bpc == 8 {
			runLF[uint8](t, line, dir, plane, &vmask, l, &lut, in, want, bitdepthMax, num)
		} else {
			runLF[uint16](t, line, dir, plane, &vmask, l, &lut, in, want, bitdepthMax, num)
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
