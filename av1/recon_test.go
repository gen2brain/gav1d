package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestReconCtx(t *testing.T) {
	sc, closeFixture := openFixture(t, "recon.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "recon" {
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
		want := fields[eq+1:]

		switch fields[0] {
		case "S":
			tx, bs := num(fields[1]), num(fields[2])
			chroma, layout := num(fields[3]), num(fields[4])
			a := make([]uint8, 32)
			l := make([]uint8, 32)
			for i := range 32 {
				a[i] = uint8(num(fields[5+i]))
				l[i] = uint8(num(fields[37+i]))
			}

			got := getSkipCtx(&txfmDimensions[tx], bs, a, 0, l, 0, chroma, layout)
			if int(got) != num(want[0]) {
				t.Fatalf("line %d: getSkipCtx tx=%d bs=%d chroma=%d layout=%d = %d, want %s",
					line, tx, bs, chroma, layout, got, want[0])
			}

		case "D":
			tx := num(fields[1])
			a := make([]uint8, 32)
			l := make([]uint8, 32)
			for i := range 32 {
				a[i] = uint8(num(fields[2+i]))
				l[i] = uint8(num(fields[34+i]))
			}

			got := getDcSignCtx(tx, a, 0, l, 0)
			if int(got) != num(want[0]) {
				t.Fatalf("line %d: getDcSignCtx tx=%d = %d, want %s", line, tx, got, want[0])
			}

		case "L":
			txClass, ctxIdx := num(fields[1]), num(fields[2])
			x, y := uint32(num(fields[3])), uint32(num(fields[4]))
			stride := num(fields[5])
			levels := make([]uint8, 16*16)
			for i := range levels {
				levels[i] = uint8(num(fields[6+i]))
			}

			var got, hiMag uint32
			if txClass == txClass2d {
				offset := uint32(loCtxOffsets[ctxIdx][min(y, 4)][min(x, 4)])
				got, hiMag = getLoCtx2D(levels, 0, offset, stride)
			} else {
				got, hiMag = getLoCtx1D(levels, 0, y, stride)
			}
			if int(got) != num(want[0]) || int(hiMag) != num(want[1]) {
				t.Fatalf("line %d: getLoCtx class=%d x=%d y=%d = (%d, %d), want (%s, %s)",
					line, txClass, x, y, got, hiMag, want[0], want[1])
			}

		case "G":
			data, err := hex.DecodeString(fields[1])
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			var s msacContext
			s.init(data, true)
			for i := range want {
				got := readGolomb(&s)
				if int(got) != num(want[i]) {
					t.Fatalf("line %d: readGolomb[%d] = %d, want %s", line, i, got, want[i])
				}
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
