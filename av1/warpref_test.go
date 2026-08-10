package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	warprefStride = 128
	warprefRows   = 37
)

func warprefGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func warprefFill(r []refmvsBlock, iter, ref int, m mv) {
	for i := range r {
		b := &r[i]
		b.mv[0].y = int16(warprefGen(iter, 0, i*4+0)&0x1fff) - 0x1000
		b.mv[0].x = int16(warprefGen(iter, 0, i*4+1)&0x1fff) - 0x1000
		b.mv[1].y = int16(warprefGen(iter, 0, i*4+2)&0x1fff) - 0x1000
		b.mv[1].x = int16(warprefGen(iter, 0, i*4+3)&0x1fff) - 0x1000
		if warprefGen(iter, 6, i)&1 != 0 {
			b.ref[0] = int8(ref + 1)
			b.ref[1] = -1
			b.mv[0].y = m.y + int16(warprefGen(iter, 7, i*2+0)%161) - 80
			b.mv[0].x = m.x + int16(warprefGen(iter, 7, i*2+1)%161) - 80
		} else {
			b.ref[0] = int8(warprefGen(iter, 1, i*2+0)%9) - 1
			b.ref[1] = int8(warprefGen(iter, 1, i*2+1)%9) - 1
		}
		b.bs = uint8(warprefGen(iter, 2, i) % nBsSizes)
		b.mf = uint8(warprefGen(iter, 3, i) & 3)
	}
}

func checkSplat[P pixel, C coef](t *testing.T, line int, tc *taskContext[P, C],
	bs int, a []string, num func(string) int,
) {
	t.Helper()

	var b av1Block
	isComp := num(a[0]) != 0
	b.ref[0] = int8(num(a[1]))
	b.ref[1] = int8(num(a[2]))
	b.mv[0] = mv{y: int16(num(a[3])), x: int16(num(a[4]))}
	b.mv[1] = mv{y: int16(num(a[5])), x: int16(num(a[6]))}
	b.interintraType = uint8(num(a[7]))
	b.interMode = uint8(num(a[8]))
	bw4, bh4 := num(a[9]), num(a[10])

	if a[11] != "=" {
		t.Fatalf("line %d: malformed splat record", line)
	}
	want := a[12:]

	if isComp {
		tc.splatTworefMv(bs, &b, bw4, bh4)
	} else {
		tc.splatOnerefMv(bs, &b, bw4, bh4)
	}
	sb := &tc.rt.r[tc.rt.rows[(tc.by&31)+5]+tc.bx]
	got := []int{
		int(sb.mv[0].y), int(sb.mv[0].x), int(sb.mv[1].y), int(sb.mv[1].x),
		int(sb.ref[0]), int(sb.ref[1]), int(sb.bs), int(sb.mf),
	}

	tc.splatIntraref(bs, bw4, bh4)
	got = append(got, int(sb.mv[0].y), int(sb.mv[0].x),
		int(sb.ref[0]), int(sb.ref[1]), int(sb.mf))

	tc.splatIntrabcMv(bs, &b, bw4, bh4)
	got = append(got, int(sb.mv[0].y), int(sb.mv[0].x),
		int(sb.ref[0]), int(sb.ref[1]), int(sb.mf))

	for i, v := range got {
		if v != num(want[i]) {
			t.Fatalf("line %d: bs=%d comp=%v splat[%d] = %d, want %s",
				line, bs, isComp, i, v, want[i])
		}
	}
}

func TestWarpRef(t *testing.T) {
	sc, closeFixture := openFixture(t, "warpref.txt")
	defer closeFixture()

	var cases, affine int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "warpref" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}
		unum := func(s string) uint64 {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		p := 1
		iter, bs, ref := num(fields[p]), num(fields[p+1]), num(fields[p+2])
		edgeFlags := num(fields[p+3])
		bx, by := num(fields[p+4]), num(fields[p+5])
		haveLeft, haveTop := num(fields[p+6]) != 0, num(fields[p+7]) != 0
		colEnd := num(fields[p+8])
		p += 9

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++
		wantMasks := [2]uint64{unum(fields[p]), unum(fields[p+1])}
		p += 2
		if fields[p] != "|" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++
		m := mv{y: int16(num(fields[p])), x: int16(num(fields[p+1]))}
		p += 2
		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		bDim := &blockDimensions[bs]
		bw4, bh4 := int(bDim[0]), int(bDim[1])
		bx4, by4 := bx&31, by&31
		w4, h4 := min(bw4, 32-bx4), min(bh4, 32-by4)

		ts := new(tileState)
		ts.tiling.colEnd = colEnd
		tc := &taskContext[uint8, int16]{scratch: &scratch[uint8]{}, ts: ts, bx: bx, by: by}
		tc.rt.r = make([]refmvsBlock, warprefRows*warprefStride)
		for i := range warprefRows {
			tc.rt.rows[i] = i * warprefStride
		}
		warprefFill(tc.rt.r, iter, ref, m)

		var masks [2]uint64
		findMatchingRef(tc, edgeFlags, bw4, bh4, w4, h4, haveLeft, haveTop,
			ref, &masks)

		if masks != wantMasks {
			t.Fatalf("line %d: bs=%d masks = %d/%d, want %d/%d", line, bs,
				masks[0], masks[1], wantMasks[0], wantMasks[1])
		}

		if masks[0]|masks[1] == 0 {
			if num(fields[p]) != -1 {
				t.Fatalf("line %d: expected no warpmv record", line)
			}
			if fields[p+1] != "S" {
				t.Fatalf("line %d: malformed splat record", line)
			}
			checkSplat(t, line, tc, bs, fields[p+2:], num)
			cases++

			continue
		}

		var wm warpedMotionParams
		deriveWarpmv(tc, bw4, bh4, &masks, m, &wm)

		if wm.typ != num(fields[p]) {
			t.Fatalf("line %d: bs=%d wm.typ = %d, want %s", line, bs, wm.typ, fields[p])
		}
		p++
		for i := range 6 {
			if int(wm.matrix[i]) != num(fields[p+i]) {
				t.Fatalf("line %d: bs=%d wm.matrix[%d] = %d, want %s",
					line, bs, i, wm.matrix[i], fields[p+i])
			}
		}
		p += 6
		for i := range 4 {
			if int(wm.abcd[i]) != num(fields[p+i]) {
				t.Fatalf("line %d: bs=%d wm.abcd[%d] = %d, want %s",
					line, bs, i, wm.abcd[i], fields[p+i])
			}
		}

		if wm.typ == wmTypeAffine {
			affine++
		}
		p += 4

		splatOff := p
		if fields[splatOff] != "S" {
			t.Fatalf("line %d: malformed splat record", line)
		}
		checkSplat(t, line, tc, bs, fields[splatOff+1:], num)

		cases++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 || affine == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d cases, %d affine", cases, affine)
}
