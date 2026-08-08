package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func mvCdfFields(c *cdfMvContext) []*uint16 {
	var out []*uint16
	for i := range c.joint {
		out = append(out, &c.joint[i])
	}
	for k := range c.comp {
		m := &c.comp[k]
		for i := range m.classes {
			out = append(out, &m.classes[i])
		}
		for i := range m.sign {
			out = append(out, &m.sign[i])
		}
		for i := range m.class0 {
			out = append(out, &m.class0[i])
		}
		for i := range m.class0Fp {
			for j := range m.class0Fp[i] {
				out = append(out, &m.class0Fp[i][j])
			}
		}
		for i := range m.class0Hp {
			out = append(out, &m.class0Hp[i])
		}
		for i := range m.classN {
			for j := range m.classN[i] {
				out = append(out, &m.classN[i][j])
			}
		}
		for i := range m.classNFp {
			out = append(out, &m.classNFp[i])
		}
		for i := range m.classNHp {
			out = append(out, &m.classNHp[i])
		}
	}

	return out
}

func TestMvResidual(t *testing.T) {
	sc, closeFixture := openFixture(t, "mvres.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "mvres" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		buf, err := hex.DecodeString(fields[1])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		update := num(fields[2])
		mvPrec := num(fields[3])
		nOps := num(fields[4])

		var ts tileState
		ts.msac.init(buf, update == 0)

		cdfs := mvCdfFields(&ts.cdf.mv)
		p := 5
		for _, f := range cdfs {
			*f = uint16(num(fields[p]))
			p++
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		for op := range nOps {
			m := mv{y: int16(num(fields[p])), x: int16(num(fields[p+1]))}
			wantY, wantX := int16(num(fields[p+2])), int16(num(fields[p+3]))
			p += 4

			readMvResidual(&ts, &m, mvPrec)

			if m.y != wantY || m.x != wantX {
				t.Fatalf("line %d op %d: mv = %d/%d, want %d/%d",
					line, op, m.y, m.x, wantY, wantX)
			}
		}

		for i, f := range cdfs {
			if *f != uint16(num(fields[p])) {
				t.Fatalf("line %d: cdf[%d] = %d, want %s", line, i, *f, fields[p])
			}
			p++
		}
		if ts.msac.rng != uint32(num(fields[p])) {
			t.Fatalf("line %d: rng = %d, want %s", line, ts.msac.rng, fields[p])
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
