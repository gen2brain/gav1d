package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func comptypeCdfs(m *cdfModeInterContext) []*uint16 {
	var out []*uint16
	add := func(a []uint16) {
		for i := range a {
			out = append(out, &a[i])
		}
	}
	for i := range m.maskComp {
		add(m.maskComp[i][:])
	}
	for i := range m.jntComp {
		add(m.jntComp[i][:])
	}
	for i := range m.wedgeComp {
		add(m.wedgeComp[i][:])
	}
	for i := range m.wedgeIdx {
		add(m.wedgeIdx[i][:])
	}
	for i := range m.interintra {
		add(m.interintra[i][:])
	}
	for i := range m.interintraWedge {
		add(m.interintraWedge[i][:])
	}
	for i := range m.interintraMode {
		add(m.interintraMode[i][:])
	}

	return out
}

func TestCompoundType(t *testing.T) {
	sc, closeFixture := openFixture(t, "comptype.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "comptype" {
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
		bs, xb4, yb4 := num(fields[p]), num(fields[p+1]), num(fields[p+2])
		update := num(fields[p+3])
		maskedCompound, jntComp := num(fields[p+4]), num(fields[p+5])
		interIntra, orderHintNBits := num(fields[p+6]), num(fields[p+7])
		poc, ref0poc, ref1poc := num(fields[p+8]), num(fields[p+9]), num(fields[p+10])
		buf, err := hex.DecodeString(fields[p+11])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p += 12

		ts := new(tileState)
		f := &frameContext[uint8, int16]{
			seqHdr: &sequenceHeader{
				maskedCompound: uint8(maskedCompound),
				jntComp:        uint8(jntComp),
				interIntra:     uint8(interIntra),
				orderHintNBits: uint8(orderHintNBits),
			},
			frameHdr: &frameHeader{frameOffset: uint8(poc)},
		}
		tc := &taskContext[uint8, int16]{f: f, ts: ts}
		tc.a = new(blockContext)

		for i := range 32 {
			tc.a.compType[i] = uint8(num(fields[p]))
			tc.a.ref[0][i] = int8(num(fields[p+1]))
			tc.l.compType[i] = uint8(num(fields[p+2]))
			tc.l.ref[0][i] = int8(num(fields[p+3]))
			p += 4
		}

		cdfs := comptypeCdfs(&ts.cdf.mi)
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
		tc.readCompoundType(&b, bs, yb4, xb4, ref0poc, ref1poc)

		if int(b.compType) != num(fields[p]) || int(b.wedgeIdx) != num(fields[p+1]) ||
			int(b.maskSign) != num(fields[p+2]) {
			t.Fatalf("line %d: bs=%d comp = %d/%d/%d, want %s/%s/%s", line, bs,
				b.compType, b.wedgeIdx, b.maskSign,
				fields[p], fields[p+1], fields[p+2])
		}
		if ts.msac.rng != uint32(num(fields[p+3])) {
			t.Fatalf("line %d: comp rng = %d, want %s", line, ts.msac.rng, fields[p+3])
		}
		p += 4

		b = av1Block{}
		tc.readInterIntra(&b, bs)

		if int(b.interintraType) != num(fields[p]) ||
			int(b.interintraMode) != num(fields[p+1]) ||
			int(b.wedgeIdx) != num(fields[p+2]) {
			t.Fatalf("line %d: bs=%d ii = %d/%d/%d, want %s/%s/%s", line, bs,
				b.interintraType, b.interintraMode, b.wedgeIdx,
				fields[p], fields[p+1], fields[p+2])
		}
		if ts.msac.rng != uint32(num(fields[p+3])) {
			t.Fatalf("line %d: ii rng = %d, want %s", line, ts.msac.rng, fields[p+3])
		}
		p += 4

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
