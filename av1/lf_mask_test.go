package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	lfmaskB4Stride = 64
	lfmaskIH       = 64
)

func TestCreateLfMaskInter(t *testing.T) {
	sc, closeFixture := openFixture(t, "lfmask.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "lfmask" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		p := 1
		bs, layout, skip := num(fields[p]), num(fields[p+1]), num(fields[p+2]) != 0
		bx, by := num(fields[p+3]), num(fields[p+4])
		iw, ih := num(fields[p+5]), num(fields[p+6])
		maxYtx, uvtx := num(fields[p+7]), num(fields[p+8])
		txMasks := [2]uint16{uint16(num(fields[p+9])), uint16(num(fields[p+10]))}
		hasChroma := num(fields[p+11]) != 0
		lvlRef, lvlGmv := num(fields[p+12]), num(fields[p+13])
		p += 14

		var filterLevel [4][8][2]uint8
		for i := range 4 {
			for j := range 8 {
				for k := range 2 {
					filterLevel[i][j][k] = uint8(num(fields[p]))
					p++
				}
			}
		}

		var ay, ly, auv, luv [32]uint8
		for i := range 32 {
			ay[i] = uint8(num(fields[p]))
			ly[i] = uint8(num(fields[p+1]))
			auv[i] = uint8(num(fields[p+2]))
			luv[i] = uint8(num(fields[p+3]))
			p += 4
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		var lflvl av1Filter
		levelCache := make([]uint8, lfmaskIH*lfmaskB4Stride*4)

		var lvls [4]uint8
		for i := range lvls {
			lvls[i] = filterLevel[i][lvlRef][lvlGmv]
		}

		createLfMaskInter(&lflvl, levelCache, lfmaskB4Stride, &lvls,
			bx, by, iw, ih, skip, bs, maxYtx, &txMasks, uvtx, layout,
			ay[:], ly[:], auv[:], luv[:], 0, 0, 0, 0, hasChroma)

		for i := range 2 {
			for j := range 32 {
				for k := range 3 {
					for s := range 2 {
						if int(lflvl.filterY[i][j][k][s]) != num(fields[p]) {
							t.Fatalf("line %d: bs=%d filterY[%d][%d][%d][%d] = %d, want %s",
								line, bs, i, j, k, s, lflvl.filterY[i][j][k][s], fields[p])
						}
						p++
					}
				}
			}
		}
		for i := range 2 {
			for j := range 32 {
				for k := range 2 {
					for s := range 2 {
						if int(lflvl.filterUv[i][j][k][s]) != num(fields[p]) {
							t.Fatalf("line %d: bs=%d filterUv[%d][%d][%d][%d] = %d, want %s",
								line, bs, i, j, k, s, lflvl.filterUv[i][j][k][s], fields[p])
						}
						p++
					}
				}
			}
		}
		for i := range 32 {
			for j, v := range [4]uint8{ay[i], ly[i], auv[i], luv[i]} {
				if int(v) != num(fields[p]) {
					t.Fatalf("line %d: bs=%d edge[%d][%d] = %d, want %s",
						line, bs, j, i, v, fields[p])
				}
				p++
			}
		}
		for y := range lfmaskIH {
			for x := range 32 {
				for k := range 4 {
					got := levelCache[(y*lfmaskB4Stride+x)*4+k]
					if int(got) != num(fields[p]) {
						t.Fatalf("line %d: bs=%d level[%d][%d][%d] = %d, want %s",
							line, bs, y, x, k, got, fields[p])
					}
					p++
				}
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
