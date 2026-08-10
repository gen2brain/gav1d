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
		loopFilterHSb128y(newDspGo[P](), buf, base, lfStride, vmask, l, lOff,
			lfB4S, lut, bitdepthMax)
	case plane == 0:
		loopFilterVSb128y(newDspGo[P](), buf, base, lfStride, vmask, l, lOff,
			lfB4S, lut, bitdepthMax)
	case dir == 0:
		loopFilterHSb128uv(newDspGo[P](), buf, base, lfStride, (*[2]uint32)(vmask[:2]),
			l, lOff, lfB4S, lut, bitdepthMax)
	default:
		loopFilterVSb128uv(newDspGo[P](), buf, base, lfStride, (*[2]uint32)(vmask[:2]),
			l, lOff, lfB4S, lut, bitdepthMax)
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

// lfLineCounts are the run lengths the drivers produce, all of them multiples
// of four and rotated through the cases so every width meets every count.
var lfLineCounts = []int{4, 8, 12, 16, 24, 48}

func runLFEdge[P pixel](d *dspContext[P], iter, wd, stridea, strideb, lines int,
	bitdepthMax int32, e, i, h int32,
) []P {
	min8 := bitdepthFromMax(bitdepthMax) - 8
	flat := int32(128) << min8
	buf := make([]P, lfRows*lfStride)
	for k := range buf {
		v := int32(lrGen(iter, 5, k))
		switch iter & 3 {
		case 0:
			buf[k] = P(v & bitdepthMax)
		case 1:
			// Flat enough that flat8in and flat8out both hold, which is the
			// only way the six, eight and fourteen tap filters ever run.
			buf[k] = P(flat + (v & 1))
		case 2:
			buf[k] = P(flat + (v & 3))
		default:
			buf[k] = P(flat + (v & 7) - 4 + int32(k%lfStride)/16)
		}
	}

	// Walking a long edge gives a single mutation far more chances to change a
	// rounded output, and the line count decides how much of it each call takes.
	off := lfPad*lfStride + lfPad
	for n := 48; n > 0; n -= lines {
		d.loopFilter(buf, off, e, i, h, stridea, strideb, wd, lines, bitdepthMax)
		off += lines * stridea
	}

	return buf
}

// TestLoopFilterAsm checks the architecture-specific deblocking filter against
// the pure Go one for both edge directions, every tap width and thresholds that
// span from filtering nothing to filtering everything. The whole buffer is
// compared, so a kernel that writes outside its span fails.
func TestLoopFilterAsm(t *testing.T) {
	cases := 0
	for _, bpc := range []int{8, 10, 12} {
		bitdepthMax := int32(1<<bpc - 1)
		if bpc == 8 {
			cases += lfAsmDepth[uint8](t, bitdepthMax)

			continue
		}
		cases += lfAsmDepth[uint16](t, bitdepthMax)
	}

	t.Logf("%d cases", cases)
}

func lfAsmDepth[P pixel](t *testing.T, bitdepthMax int32) int {
	t.Helper()

	ref, got := newDspGo[P](), newDsp[P]()
	cases := 0

	for iter := range 8 {
		for _, wd := range []int{4, 6, 8, 16} {
			for _, dir := range [][2]int{{1, lfStride}, {lfStride, 1}} {
				for _, e := range []int32{0, 3, 20, 100} {
					for _, i := range []int32{0, 1, 6, 40} {
						for _, h := range []int32{0, 2, 30} {
							lines := lfLineCounts[cases%len(lfLineCounts)]
							want := runLFEdge(ref, iter, wd, dir[0], dir[1], lines,
								bitdepthMax, e, i, h)
							have := runLFEdge(got, iter, wd, dir[0], dir[1], lines,
								bitdepthMax, e, i, h)
							for k := range want {
								if have[k] == want[k] {
									continue
								}
								t.Fatalf("bpc=%d wd=%d stridea=%d lines=%d e=%d i=%d h=%d iter=%d: "+
									"buf[%d] = %d, want %d",
									bitdepthFromMax(bitdepthMax), wd, dir[0], lines, e, i, h,
									iter, k, have[k], want[k])
							}
							cases++
						}
					}
				}
			}
		}
	}

	return cases
}
