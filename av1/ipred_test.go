package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	ipredStride = 80
	ipredTL     = 128
)

func ipredDispatch[P pixel](mode int) func(dst []P, dstOff, stride int, tl []P, tlOff,
	w, h, angle, maxW, maxH int, bitdepthMax int32,
) {
	switch mode {
	case dcPred:
		return ipredDc[P]
	case vertPred:
		return ipredV[P]
	case horPred:
		return ipredH[P]
	case leftDcPred:
		return ipredDcLeft[P]
	case topDcPred:
		return ipredDcTop[P]
	case dc128Pred:
		return ipredDc128[P]
	case z1Pred:
		return ipredZ1[P]
	case z2Pred:
		return ipredZ2[P]
	case z3Pred:
		return ipredZ3[P]
	case smoothPred:
		return ipredSmooth[P]
	case smoothVPred:
		return ipredSmoothV[P]
	case smoothHPred:
		return ipredSmoothH[P]
	case paethPred:
		return ipredPaeth[P]
	case filterPred:
		return ipredFilter[P]
	}

	return nil
}

func runIpred[P pixel](t *testing.T, line int, mode, w, h, angle, maxW, maxH int,
	tlIn []string, want []string, bitdepthMax int32, num func(string) int,
) {
	fn := ipredDispatch[P](mode)
	if fn == nil {
		t.Fatalf("line %d: no Go fn for mode %d", line, mode)
	}

	tl := make([]P, 2*ipredTL+1)
	for i := range tl {
		tl[i] = P(num(tlIn[i]))
	}
	dst := make([]P, 64*ipredStride)

	fn(dst, 0, ipredStride, tl, ipredTL, w, h, angle, maxW, maxH, bitdepthMax)

	for y := range h {
		for x := range w {
			if int(dst[y*ipredStride+x]) != num(want[y*w+x]) {
				t.Fatalf("line %d: mode=%d %dx%d angle=%d dst[%d][%d] = %d, want %s",
					line, mode, w, h, angle, y, x, dst[y*ipredStride+x], want[y*w+x])
			}
		}
	}
}

func runPalPred[P pixel](t *testing.T, line int, w, h int, palIn, idxIn, want []string,
	num func(string) int,
) {
	pal := make([]P, 8)
	for i := range 8 {
		pal[i] = P(num(palIn[i]))
	}
	idx := make([]uint8, len(idxIn))
	for i := range idxIn {
		idx[i] = uint8(num(idxIn[i]))
	}
	dst := make([]P, 64*ipredStride)

	palPred(dst, 0, ipredStride, pal, idx, w, h)

	for y := range h {
		for x := range w {
			if int(dst[y*ipredStride+x]) != num(want[y*w+x]) {
				t.Fatalf("line %d: palPred %dx%d dst[%d][%d] = %d, want %s",
					line, w, h, y, x, dst[y*ipredStride+x], want[y*w+x])
			}
		}
	}
}

func runCflAc[P pixel](t *testing.T, line int, cw, ch, wPad, hPad, ssHor, ssVer int,
	yIn, want []string, num func(string) int,
) {
	yw, yh := cw<<ssHor, ch<<ssVer
	ypx := make([]P, 64*ipredStride)
	for y := range yh {
		for x := range yw {
			ypx[y*ipredStride+x] = P(num(yIn[y*yw+x]))
		}
	}
	ac := make([]int16, 32*32)

	cflAc(ac, ypx, 0, ipredStride, wPad, hPad, cw, ch, ssHor, ssVer)

	for i := range cw * ch {
		if int(ac[i]) != num(want[i]) {
			t.Fatalf("line %d: cflAc %dx%d wPad=%d hPad=%d ac[%d] = %d, want %s",
				line, cw, ch, wPad, hPad, i, ac[i], want[i])
		}
	}
}

func runCflPred[P pixel](t *testing.T, line int, mode, w, h int, alpha int32,
	tlIn, acIn, want []string, bitdepthMax int32, num func(string) int,
) {
	var fn func(dst []P, dstOff, stride int, tl []P, tlOff, w, h int,
		ac []int16, alpha int32, bitdepthMax int32)
	switch mode {
	case dcPred:
		fn = ipredCfl[P]
	case leftDcPred:
		fn = ipredCflLeft[P]
	case topDcPred:
		fn = ipredCflTop[P]
	case dc128Pred:
		fn = ipredCfl128[P]
	default:
		t.Fatalf("line %d: no Go cfl fn for mode %d", line, mode)
	}

	tl := make([]P, 2*ipredTL+1)
	for i := range tl {
		tl[i] = P(num(tlIn[i]))
	}
	ac := make([]int16, 32*32)
	for i := range w * h {
		ac[i] = int16(num(acIn[i]))
	}
	dst := make([]P, 64*ipredStride)

	fn(dst, 0, ipredStride, tl, ipredTL, w, h, ac, alpha, bitdepthMax)

	for y := range h {
		for x := range w {
			if int(dst[y*ipredStride+x]) != num(want[y*w+x]) {
				t.Fatalf("line %d: cflPred mode=%d %dx%d alpha=%d dst[%d][%d] = %d, want %s",
					line, mode, w, h, alpha, y, x, dst[y*ipredStride+x], want[y*w+x])
			}
		}
	}
}

func TestIpred(t *testing.T) {
	sc, closeFixture := openFixture(t, "ipred.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "ipred" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		eq := 0
		for eq < len(fields) && fields[eq] != "=" {
			eq++
		}
		if eq == len(fields) {
			t.Fatalf("line %d: malformed record", line)
		}
		want := fields[eq+1:]

		switch fields[0] {
		case "P":
			bpc, mode := num(fields[1]), num(fields[2])
			w, h := num(fields[3]), num(fields[4])
			angle, maxW, maxH := num(fields[5]), num(fields[6]), num(fields[7])
			tlIn := fields[8:eq]
			bitdepthMax := int32(1<<bpc - 1)

			if bpc == 8 {
				runIpred[uint8](t, line, mode, w, h, angle, maxW, maxH, tlIn, want, bitdepthMax, num)
			} else {
				runIpred[uint16](t, line, mode, w, h, angle, maxW, maxH, tlIn, want, bitdepthMax, num)
			}

		case "C":
			bpc, mode := num(fields[1]), num(fields[2])
			w, h := num(fields[3]), num(fields[4])
			alpha := int32(num(fields[5]))
			amark := 6 + 2*ipredTL + 1
			if fields[amark] != "a" {
				t.Fatalf("line %d: malformed cfl record", line)
			}
			tlIn := fields[6:amark]
			acIn := fields[amark+1 : eq]
			bitdepthMax := int32(1<<bpc - 1)

			if bpc == 8 {
				runCflPred[uint8](t, line, mode, w, h, alpha, tlIn, acIn, want, bitdepthMax, num)
			} else {
				runCflPred[uint16](t, line, mode, w, h, alpha, tlIn, acIn, want, bitdepthMax, num)
			}

		case "L":
			bpc := num(fields[1])
			w, h := num(fields[2]), num(fields[3])
			palIn := fields[4:12]
			idxIn := fields[12:eq]

			if bpc == 8 {
				runPalPred[uint8](t, line, w, h, palIn, idxIn, want, num)
			} else {
				runPalPred[uint16](t, line, w, h, palIn, idxIn, want, num)
			}

		case "A":
			bpc, fmtIdx := num(fields[1]), num(fields[2])
			cw, ch := num(fields[3]), num(fields[4])
			wPad, hPad := num(fields[5]), num(fields[6])
			ssHor, ssVer := 1, 0
			if fmtIdx == 2 {
				ssHor = 0
			}
			if fmtIdx == 0 {
				ssVer = 1
			}
			yIn := fields[8:eq]

			if bpc == 8 {
				runCflAc[uint8](t, line, cw, ch, wPad, hPad, ssHor, ssVer, yIn, want, num)
			} else {
				runCflAc[uint16](t, line, cw, ch, wPad, hPad, ssHor, ssVer, yIn, want, num)
			}

		default:
			t.Fatalf("line %d: unknown op %q", line, fields[0])
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
