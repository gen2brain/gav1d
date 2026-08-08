package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestVartxTree(t *testing.T) {
	sc, closeFixture := openFixture(t, "vartx.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "vartx" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		p := 2
		bs, layout := num(fields[p]), num(fields[p+1])
		txfmMode, skip := num(fields[p+2]), num(fields[p+3])
		lossless, update := num(fields[p+4]), num(fields[p+5])
		bx, by := num(fields[p+6]), num(fields[p+7])
		buf, err := hex.DecodeString(fields[p+8])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p += 9

		var seqHdr sequenceHeader
		var hdr frameHeader
		seqHdr.sb128 = 1
		hdr.txfmMode = txfmMode
		hdr.segmentation.lossless[0] = uint8(lossless)

		f := &frameContext[uint8, int16]{
			seqHdr:   &seqHdr,
			frameHdr: &hdr,
			layout:   layout,
			bw:       64,
			bh:       64,
		}
		ts := new(tileState)
		tc := &taskContext[uint8, int16]{f: f, ts: ts, bx: bx, by: by}
		tc.a = new(blockContext)

		ts.msac.init(buf, update == 0)

		for c := range 7 {
			for j := range 3 {
				for k := range 2 {
					ts.cdf.m.txpart[c][j][k] = uint16(num(fields[p]))
					p++
				}
			}
		}
		for i := range 32 {
			tc.a.tx[i] = int8(num(fields[p]))
			tc.l.tx[i] = int8(num(fields[p+1]))
			p += 2
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++

		var b av1Block
		b.skip = uint8(skip)

		tc.readVartxTree(&b, bs, bx&31, by&31)

		got := []int{int(b.maxYtx), int(b.uvtx), int(b.txSplit0), int(b.txSplit1)}
		want := []int{num(fields[p]), num(fields[p+1]), num(fields[p+2]), num(fields[p+3])}
		p += 5

		for i := range 32 {
			got = append(got, int(tc.a.tx[i]), int(tc.l.tx[i]))
			want = append(want, num(fields[p]), num(fields[p+1]))
			p += 2
		}
		for c := range 7 {
			for j := range 3 {
				for k := range 2 {
					got = append(got, int(ts.cdf.m.txpart[c][j][k]))
					want = append(want, num(fields[p]))
					p++
				}
			}
		}

		for i, v := range got {
			if v != want[i] {
				t.Fatalf("line %d: bs=%d mode=%d skip=%d out[%d] = %d, want %d",
					line, bs, txfmMode, skip, i, v, want[i])
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
