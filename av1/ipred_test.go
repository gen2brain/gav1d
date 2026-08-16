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

func ipredFillTl[P pixel](tl []P, iter int, bitdepthMax int32) {
	s := uint32(iter*7919 + 1)
	for i := range tl {
		s = s*1103515245 + 12345
		switch iter {
		case 0:
			tl[i] = P(int32(s>>16) & bitdepthMax)
		case 1:
			tl[i] = P(bitdepthMax * int32(i&1))
		default:
			tl[i] = P(bitdepthMax)
		}
	}
}

var z1Angles = []int{36, 39, 42, 45, 48, 51, 54, 58, 61, 64, 67, 70, 73, 76, 81, 84, 87}

func ipredAngles(mode int) []int {
	switch mode {
	case z1Pred:
		var a []int
		for _, ang := range z1Angles {
			a = append(a, ang, ang|angleSmoothEdgeFlag|angleUseEdgeFilterFlag,
				ang|angleUseEdgeFilterFlag)
		}

		return a
	case filterPred:
		return []int{0, 1, 2, 3, 4}
	}

	return []int{0}
}

func runIpredAsm[P pixel](t *testing.T, modes []int, bitdepthMax int32) int {
	d, ref := newDsp[P](), newDspGo[P]()

	sizes := []int{4, 8, 16, 32, 64}
	cases := 0

	tl := make([]P, 2*ipredTL+1)
	got := make([]P, 64*ipredStride)
	want := make([]P, 64*ipredStride)

	for iter := range 3 {
		ipredFillTl(tl, iter, bitdepthMax)

		for _, m := range modes {
			for _, w := range sizes {
				for _, h := range sizes {
					for _, angle := range ipredAngles(m) {
						for i := range got {
							got[i] = P(0xA5)
							want[i] = P(0xA5)
						}

						d.ipred[m](got, 0, ipredStride, tl, ipredTL, w, h, angle, 0, 0, bitdepthMax)
						ref.ipred[m](want, 0, ipredStride, tl, ipredTL, w, h, angle, 0, 0, bitdepthMax)

						for i := range want {
							if got[i] == want[i] {
								continue
							}
							t.Fatalf("bpc=%d iter=%d mode=%d %dx%d angle=%d: [%d,%d] = %d, want %d",
								bitdepthMax, iter, m, w, h, angle, i/ipredStride,
								i%ipredStride, got[i], want[i])
						}
						cases++
					}
				}
			}
		}
	}

	return cases
}

func TestIpredAsm(t *testing.T) {
	m8 := []int{smoothPred, smoothVPred, smoothHPred, paethPred, vertPred,
		horPred, dcPred, topDcPred, leftDcPred, dc128Pred, filterPred, z1Pred}
	m16 := []int{smoothPred, smoothVPred, smoothHPred, paethPred, vertPred,
		horPred, dcPred, topDcPred, leftDcPred, dc128Pred}
	if ipredHas16Deep {
		m16 = append(m16, filterPred, z1Pred)
	}

	cases := runIpredAsm[uint8](t, m8, 255)
	cases += runIpredAsm[uint16](t, m16, 1023)
	cases += runIpredAsm[uint16](t, m16, 4095)

	t.Logf("%d cases", cases)
}

func benchIpred[P pixel](b *testing.B, tag string, bitdepthMax int32) {
	d := newDsp[P]()

	modes := []struct {
		mode  int
		name  string
		angle int
	}{
		{smoothPred, "smooth", 0}, {smoothVPred, "smoothV", 0},
		{smoothHPred, "smoothH", 0}, {paethPred, "paeth", 0},
		{vertPred, "v", 0}, {horPred, "h", 0}, {dcPred, "dc", 0},
		{filterPred, "filter", 0},
		{z1Pred, "z1", 45 | angleUseEdgeFilterFlag},
		{z2Pred, "z2", 135 | angleUseEdgeFilterFlag},
		{z3Pred, "z3", 225 | angleUseEdgeFilterFlag},
	}
	sizes := []int{4, 8, 16, 32}

	tl := make([]P, 2*ipredTL+1)
	ipredFillTl(tl, 0, bitdepthMax)
	dst := make([]P, 64*ipredStride)

	for _, m := range modes {
		for _, w := range sizes {
			fn := d.ipred[m.mode]
			b.Run(tag+"/"+m.name+"/"+strconv.Itoa(w), func(b *testing.B) {
				for b.Loop() {
					fn(dst, 0, ipredStride, tl, ipredTL, w, w, m.angle, 0, 0, bitdepthMax)
				}
			})
		}
	}
}

func BenchmarkIpred(b *testing.B) {
	benchIpred[uint8](b, "8", 255)
	benchIpred[uint16](b, "16", 1023)
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
