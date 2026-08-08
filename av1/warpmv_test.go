package av1

import (
	"strconv"
	"strings"
	"testing"
)

func TestWarpMv(t *testing.T) {
	sc, closeFixture := openFixture(t, "warpmv.txt")
	defer closeFixture()

	var shear, affine int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "warpmv" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		switch fields[0] {
		case "S":
			var wm warpedMotionParams
			for i := range 6 {
				wm.matrix[i] = int32(num(fields[1+i]))
			}
			if fields[7] != "=" {
				t.Fatalf("line %d: malformed S record", line)
			}
			got := b2i(getShearParams(&wm))
			if got != num(fields[8]) {
				t.Fatalf("line %d: shear ret = %d, want %s", line, got, fields[8])
			}
			for i := range 4 {
				if int(wm.abcd[i]) != num(fields[9+i]) {
					t.Fatalf("line %d: abcd[%d] = %d, want %s",
						line, i, wm.abcd[i], fields[9+i])
				}
			}
			shear++

		case "A":
			bw4, bh4 := num(fields[1]), num(fields[2])
			bx4, by4 := num(fields[3]), num(fields[4])
			m := mv{y: int16(num(fields[5])), x: int16(num(fields[6]))}
			np := num(fields[7])

			pts := make([][2][2]int32, np)
			p := 8
			for i := range np {
				pts[i][0][0] = int32(num(fields[p]))
				pts[i][0][1] = int32(num(fields[p+1]))
				pts[i][1][0] = int32(num(fields[p+2]))
				pts[i][1][1] = int32(num(fields[p+3]))
				p += 4
			}
			if fields[p] != "=" {
				t.Fatalf("line %d: malformed A record", line)
			}
			p++

			var wm warpedMotionParams
			got := b2i(findAffineInt(pts, np, bw4, bh4, m, &wm, bx4, by4))
			if got != num(fields[p]) {
				t.Fatalf("line %d: affine ret = %d, want %s", line, got, fields[p])
			}
			p++
			for i := range 6 {
				if int(wm.matrix[i]) != num(fields[p+i]) {
					t.Fatalf("line %d: affine mat[%d] = %d, want %s",
						line, i, wm.matrix[i], fields[p+i])
				}
			}
			p += 6

			if fields[p] != "|" {
				t.Fatalf("line %d: malformed A record", line)
			}
			p++

			var wm2 warpedMotionParams
			for i := range 4 {
				wm2.matrix[2+i] = int32(num(fields[p+i]))
			}
			p += 4
			if fields[p] != "=" {
				t.Fatalf("line %d: malformed A record", line)
			}
			p++
			setAffineMv2d(bw4, bh4, m, &wm2, bx4, by4)
			for i := range 2 {
				if int(wm2.matrix[i]) != num(fields[p+i]) {
					t.Fatalf("line %d: mv2d mat[%d] = %d, want %s",
						line, i, wm2.matrix[i], fields[p+i])
				}
			}
			affine++

		default:
			t.Fatalf("line %d: unknown record %q", line, fields[0])
		}
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if shear == 0 || affine == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d shear, %d affine cases", shear, affine)
}
