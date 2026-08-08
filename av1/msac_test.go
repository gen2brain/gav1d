package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestMsac(t *testing.T) {
	sc, closeFixture := openFixture(t, "msac.txt")
	defer closeFixture()

	var s msacContext
	var cdf []uint16

	var ops int

	num := func(s string) int64 {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}

		return v
	}

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "msac" {
			continue
		}

		switch fields[0] {
		case "B":
			data, err := hex.DecodeString(fields[1])
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			s.init(data, fields[2] == "0")

			continue
		case "C":
			nSym := int(num(fields[1]))
			cdf = make([]uint16, 16)
			for i := range nSym + 1 {
				cdf[i] = uint16(num(fields[2+i]))
			}

			continue
		}

		eq := 0
		for eq < len(fields) && fields[eq] != "=" {
			eq++
		}
		if eq == len(fields) {
			t.Fatalf("line %d: malformed record", line)
		}

		args := make([]int64, eq-1)
		for i := range args {
			args[i] = num(fields[1+i])
		}

		op := fields[0]

		var got int64
		switch op {
		case "S":
			got = int64(s.symbolAdapt(cdf, int(args[0])))
		case "A":
			got = int64(s.boolAdapt(cdf))
		case "H":
			got = int64(s.hiTok(cdf))
		case "E":
			got = int64(s.boolEqui())
		case "F":
			got = int64(s.boolF(uint32(args[0])))
		case "N":
			got = int64(s.bools(int(args[0])))
		case "U":
			got = int64(s.uniform(uint32(args[0])))
		case "X":
			got = int64(s.subexp(int32(args[0]), int32(args[1]), uint32(args[2])))
		default:
			t.Fatalf("line %d: unknown op %q", line, op)
		}

		res := fields[eq+1:]
		want := num(res[0])
		wantRng, wantCnt, wantPos := num(res[1]), num(res[2]), num(res[3])
		wantDif, err := strconv.ParseUint(res[4], 10, 64)
		if err != nil {
			t.Fatal(err)
		}

		if got != want {
			t.Fatalf("line %d: op %s%v = %d, want %d", line, op, args, got, want)
		}
		if int64(s.rng) != wantRng || int64(s.cnt) != wantCnt ||
			int64(s.pos) != wantPos || s.dif != wantDif {
			t.Fatalf("line %d: op %s%v state:\n got rng=%d cnt=%d pos=%d dif=%d\nwant rng=%d cnt=%d pos=%d dif=%d",
				line, op, args, s.rng, s.cnt, s.pos, s.dif, wantRng, wantCnt, wantPos, wantDif)
		}

		for i := range len(res) - 5 {
			if int64(cdf[i]) != num(res[5+i]) {
				t.Fatalf("line %d: op %s%v cdf[%d] = %d, want %s",
					line, op, args, i, cdf[i], res[5+i])
			}
		}

		ops++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if ops == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d ops", ops)
}
