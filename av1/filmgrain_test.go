package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	fgTestW      = 84
	fgTestH      = 70
	fgTestStride = 128
)

func fgGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func fgFillPoints(points [][2]uint8, maxN, iter, kind int) int {
	n := int(fgGen(iter, kind, 0)) % (maxN + 1)
	x := int(fgGen(iter, kind, 1)) % 16
	for i := range n {
		if x > 255 {
			return i
		}
		points[i][0] = uint8(x)
		points[i][1] = uint8(fgGen(iter, kind, 2+i*2))
		x += 1 + int(fgGen(iter, kind, 3+i*2))%24
	}

	return n
}

func fgFillData(iter, monochrome, ssHor, ssVer int) filmGrainData {
	var d filmGrainData

	d.seed = uint32(fgGen(iter, 1, 0))
	d.numYPoints = fgFillPoints(d.yPoints[:], 14, iter, 2)

	if monochrome == 0 {
		d.chromaScalingFromLuma = b2i(fgGen(iter, 3, 0)%4 == 0)
	}

	if monochrome != 0 || d.chromaScalingFromLuma != 0 ||
		(ssVer == 1 && ssHor == 1 && d.numYPoints == 0) {
		d.numUvPoints[0], d.numUvPoints[1] = 0, 0
	} else {
		for pl := range 2 {
			d.numUvPoints[pl] = fgFillPoints(d.uvPoints[pl][:], 10, iter, 4+pl)
		}
		if ssHor == 1 && ssVer == 1 &&
			(d.numUvPoints[0] != 0) != (d.numUvPoints[1] != 0) {
			d.numUvPoints[0], d.numUvPoints[1] = 0, 0
		}
	}

	d.scalingShift = 8 + int(fgGen(iter, 6, 0))%4
	d.arCoeffLag = int(fgGen(iter, 7, 0)) % 4

	numYPos := 2 * d.arCoeffLag * (d.arCoeffLag + 1)
	if d.numYPoints != 0 {
		for i := range numYPos {
			d.arCoeffsY[i] = int8(int(fgGen(iter, 8, i)&0xff) - 128)
		}
	}

	for pl := range 2 {
		if d.numUvPoints[pl] == 0 && d.chromaScalingFromLuma == 0 {
			continue
		}
		numUvPos := numYPos + b2i(d.numYPoints != 0)
		for i := range numUvPos {
			d.arCoeffsUv[pl][i] = int8(int(fgGen(iter, 9+pl, i)&0xff) - 128)
		}
		if d.numYPoints == 0 {
			d.arCoeffsUv[pl][numUvPos] = 0
		}
	}

	d.arCoeffShift = uint64(6 + fgGen(iter, 11, 0)%4)
	d.grainScaleShift = int(fgGen(iter, 12, 0)) % 4

	for pl := range 2 {
		if d.numUvPoints[pl] == 0 {
			continue
		}
		d.uvMult[pl] = int(fgGen(iter, 13, pl)&0xff) - 128
		d.uvLumaMult[pl] = int(fgGen(iter, 14, pl)&0xff) - 128
		d.uvOffset[pl] = int(fgGen(iter, 15, pl)&0x1ff) - 256
	}

	d.overlapFlag = b2i(fgGen(iter, 16, 0)%4 != 0)
	d.clipToRestrictedRange = b2i(fgGen(iter, 17, 0)%4 == 0)

	return d
}

type fgResult struct {
	scaling [3][]uint8
	lut     [3]grainLut
	planes  [3][]int
}

func runFilmGrain[P pixel](iter, bpc, layout, w, h int, isID bool) fgResult {
	ssVer := b2i(layout == pixelLayoutI420)
	ssHor := b2i(layout != pixelLayoutI444)
	monochrome := b2i(layout == pixelLayoutI400)
	nplanes := 3
	if monochrome != 0 {
		nplanes = 1
	}
	cw, ch := (w+ssHor)>>ssHor, (h+ssVer)>>ssVer

	data := fgFillData(iter, monochrome, ssHor, ssVer)

	var src, dst [3][]P
	stride := [3]int{fgTestStride, fgTestStride, fgTestStride}
	bitdepthMax := 1<<bpc - 1
	for pl := range nplanes {
		src[pl] = make([]P, fgTestStride*fgTestH)
		dst[pl] = make([]P, fgTestStride*fgTestH)
		ph := h
		if pl != 0 {
			ph = ch
		}
		for y := range ph {
			for x := range fgTestStride {
				i := y*fgTestStride + x
				src[pl][i] = P(int(fgGen(iter, 20+pl, i)) & bitdepthMax)
			}
		}
	}

	var fg filmGrainCtx
	prepGrain(&fg, &data, layout, bpc, w, h, &dst, &src, &stride)
	for row := range (h + fgBlockSize - 1) / fgBlockSize {
		applyGrainRow(&fg, &data, layout, bpc, w, h, isID, &dst, &src, &stride, row)
	}

	res := fgResult{scaling: fg.scaling, lut: fg.lut}
	for pl := range nplanes {
		pw, ph := w, h
		if pl != 0 {
			pw, ph = cw, ch
		}
		out := make([]int, 0, pw*ph)
		for y := range ph {
			for x := range pw {
				out = append(out, int(dst[pl][y*fgTestStride+x]))
			}
		}
		res.planes[pl] = out
	}

	return res
}

func TestFilmGrain(t *testing.T) {
	sc, closeFixture := openFixture(t, "filmgrain.txt")
	defer closeFixture()

	var cases int
	var res fgResult
	var bpc, layout int
	var pl int

	num := func(line int, s string) int {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}

		return v
	}

	cmp := func(line int, what string, got []int, want []string) {
		if len(got) != len(want) {
			t.Fatalf("line %d: %s has %d values, want %d", line, what, len(got), len(want))
		}
		for i, v := range got {
			if v != num(line, want[i]) {
				t.Fatalf("line %d: bpc=%d layout=%d %s[%d] = %d, want %s",
					line, bpc, layout, what, i, v, want[i])
			}
		}
	}

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "filmgrain" {
			continue
		}

		switch fields[0] {
		case "C":
			iter := num(line, fields[1])
			bpc = num(line, fields[2])
			layout = num(line, fields[3])
			w, h := num(line, fields[4]), num(line, fields[5])
			isID := num(line, fields[6]) != 0
			if bpc == 8 {
				res = runFilmGrain[uint8](iter, bpc, layout, w, h, isID)
			} else {
				res = runFilmGrain[uint16](iter, bpc, layout, w, h, isID)
			}
			pl = 0
			cases++

		case "S":
			got := make([]int, len(fields)-1)
			for i := range got {
				got[i] = int(res.scaling[pl][i])
			}
			cmp(line, "scaling", got, fields[1:])
			pl++
			if pl == 3 {
				pl = 0
			}

		case "G":
			got := make([]int, 0, grainHeight*grainWidth)
			for y := range grainHeight {
				for x := range grainWidth {
					got = append(got, int(res.lut[pl][y][x]))
				}
			}
			cmp(line, "grain_lut", got, fields[1:])
			pl++
			if pl == 3 {
				pl = 0
			}

		case "P":
			cmp(line, "plane", res.planes[pl], fields[1:])
			pl++

		default:
			t.Fatalf("line %d: unknown record %q", line, fields[0])
		}
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	t.Logf("%d cases", cases)
}
