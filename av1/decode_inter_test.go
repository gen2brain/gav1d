package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func refselCdfs(m *cdfModeInterContext) []*uint16 {
	var out []*uint16
	add2 := func(a [][2]uint16) {
		for i := range a {
			out = append(out, &a[i][0], &a[i][1])
		}
	}
	add2(m.compDir[:])
	for i := range m.compFwdRef {
		add2(m.compFwdRef[i][:])
	}
	for i := range m.compBwdRef {
		add2(m.compBwdRef[i][:])
	}
	for i := range m.compUniRef {
		add2(m.compUniRef[i][:])
	}
	for i := range m.ref {
		add2(m.ref[i][:])
	}

	return out
}

func TestRefSelection(t *testing.T) {
	sc, closeFixture := openFixture(t, "refsel.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "refsel" {
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
		isComp, update := num(fields[p]) != 0, num(fields[p+1])
		xb4, yb4 := num(fields[p+2]), num(fields[p+3])
		haveTop, haveLeft := num(fields[p+4]) != 0, num(fields[p+5]) != 0
		buf, err := hex.DecodeString(fields[p+6])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p += 7

		ts := new(tileState)
		tc := &taskContext[uint8, int16]{scratch: &scratch[uint8]{}, ts: ts}
		tc.a = new(blockContext)

		for i := range 32 {
			tc.a.intra[i] = uint8(num(fields[p]))
			tc.a.compType[i] = uint8(num(fields[p+1]))
			tc.a.ref[0][i] = int8(num(fields[p+2]))
			tc.a.ref[1][i] = int8(num(fields[p+3]))
			tc.l.intra[i] = uint8(num(fields[p+4]))
			tc.l.compType[i] = uint8(num(fields[p+5]))
			tc.l.ref[0][i] = int8(num(fields[p+6]))
			tc.l.ref[1][i] = int8(num(fields[p+7]))
			p += 8
		}

		cdfs := refselCdfs(&ts.cdf.mi)
		for _, c := range cdfs {
			*c = uint16(num(fields[p]))
			p++
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		ts.msac.init(buf, update == 0)

		var b av1Block
		if isComp {
			tc.readCompRefs(&b, yb4, xb4, haveTop, haveLeft)
		} else {
			tc.readSingleRef(&b, yb4, xb4, haveTop, haveLeft)
		}

		if int(b.ref[0]) != num(fields[p]) || int(b.ref[1]) != num(fields[p+1]) {
			t.Fatalf("line %d: comp=%v refs = %d/%d, want %s/%s",
				line, isComp, b.ref[0], b.ref[1], fields[p], fields[p+1])
		}
		if ts.msac.rng != uint32(num(fields[p+2])) {
			t.Fatalf("line %d: rng = %d, want %s", line, ts.msac.rng, fields[p+2])
		}
		p += 3

		for i, c := range cdfs {
			if *c != uint16(num(fields[p])) {
				t.Fatalf("line %d: cdf[%d] = %d, want %s", line, i, *c, fields[p])
			}
			p++
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
