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

func TestSymbolAdaptAsm(t *testing.T) {
	rnd := func(seed *uint32) uint32 {
		*seed ^= *seed << 13
		*seed ^= *seed >> 17
		*seed ^= *seed << 5

		return *seed
	}

	data := make([]byte, 4096)
	seed := uint32(12345)
	for i := range data {
		data[i] = byte(rnd(&seed))
	}

	cases := 0
	for _, n := range []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 15} {
		for _, width := range []int{4, 8, 16} {
			if width < n+1 {
				continue
			}
			for _, upd := range []bool{false, true} {
				var a, b msacContext
				a.init(data, !upd)
				b.init(data, !upd)

				ca := make([]uint16, width)
				cb := make([]uint16, width)
				for i := range ca {
					ca[i] = uint16(32768 - (i+1)*32768/(n+1))
				}
				ca[n] = 0
				copy(cb, ca)

				for iter := range 3000 {
					got := a.symbolAdapt(ca, n)
					want := b.symbolAdaptGo(cb, n)

					if got != want || a.dif != b.dif || a.rng != b.rng ||
						a.cnt != b.cnt || a.pos != b.pos {
						t.Fatalf("n=%d w=%d upd=%v iter=%d: sym %d/%d dif %#x/%#x rng %#x/%#x cnt %d/%d pos %d/%d",
							n, width, upd, iter, got, want, a.dif, b.dif,
							a.rng, b.rng, a.cnt, b.cnt, a.pos, b.pos)
					}
					for i := 0; i <= n; i++ {
						if ca[i] != cb[i] {
							t.Fatalf("n=%d w=%d upd=%v iter=%d: cdf[%d] = %d, want %d",
								n, width, upd, iter, i, ca[i], cb[i])
						}
					}
					cases++
				}
			}
		}
	}

	t.Logf("%d cases", cases)
}

func TestBoolAsm(t *testing.T) {
	rnd := func(seed *uint32) uint32 {
		*seed ^= *seed << 13
		*seed ^= *seed >> 17
		*seed ^= *seed << 5

		return *seed
	}

	data := make([]byte, 4096)
	seed := uint32(987)
	for i := range data {
		data[i] = byte(rnd(&seed))
	}

	cases := 0
	for _, upd := range []bool{false, true} {
		for kind := range 3 {
			var a, b msacContext
			a.init(data, !upd)
			b.init(data, !upd)

			ca := []uint16{20000, 0, 0, 0}
			cb := []uint16{20000, 0, 0, 0}

			fseed := uint32(4242)
			for iter := range 5000 {
				var got, want uint32
				switch kind {
				case 0:
					got, want = a.boolEqui(), b.boolEquiGo()
				case 1:
					f := rnd(&fseed)%32000 + 200
					got, want = a.boolF(f), b.boolFGo(f)
				default:
					got, want = a.boolAdapt(ca), b.boolAdaptGo(cb)
				}

				if got != want || a.dif != b.dif || a.rng != b.rng ||
					a.cnt != b.cnt || a.pos != b.pos {
					t.Fatalf("kind=%d upd=%v iter=%d: %d/%d dif %#x/%#x rng %#x/%#x cnt %d/%d pos %d/%d",
						kind, upd, iter, got, want, a.dif, b.dif, a.rng, b.rng,
						a.cnt, b.cnt, a.pos, b.pos)
				}
				for i := range ca {
					if ca[i] != cb[i] {
						t.Fatalf("kind=%d upd=%v iter=%d: cdf[%d] = %d, want %d",
							kind, upd, iter, i, ca[i], cb[i])
					}
				}
				cases++
			}
		}
	}

	t.Logf("%d cases", cases)
}
