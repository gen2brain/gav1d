package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeCoefs(t *testing.T) {
	sc, closeFixture := openFixture(t, "coefs.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "coefs" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		qidx := num(fields[1])
		layout := num(fields[2])
		intra := num(fields[3])
		plane := num(fields[4])
		useQm := num(fields[5])
		reducedTxtp := num(fields[6])
		tx, bs := num(fields[7]), num(fields[8])
		uvMode, yMode, yAngle := num(fields[9]), num(fields[10]), num(fields[11])
		txtpIn := num(fields[12])

		p := 13
		a := make([]uint8, 32)
		l := make([]uint8, 32)
		for i := range 32 {
			a[i] = uint8(num(fields[p+i]))
		}
		p += 32
		for i := range 32 {
			l[i] = uint8(num(fields[p+i]))
		}
		p += 32

		tDim := &txfmDimensions[tx]
		sw, sh := min(4*int(tDim.w), 32), min(4*int(tDim.h), 32)

		var qm []uint8
		if useQm != 0 {
			qm = make([]uint8, sw*sh)
			for i := range qm {
				qm[i] = uint8(num(fields[p+i]))
			}
			p += sw * sh
		}

		if fields[p] != "b" {
			t.Fatalf("line %d: malformed record at b (got %q)", line, fields[p])
		}
		buf, err := hex.DecodeString(fields[p+1])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p += 2
		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record at =", line)
		}
		want := fields[p+1:]

		var f frameContext[uint8, int16]
		var seqHdr sequenceHeader
		var frameHdr frameHeader

		frameHdr.reducedTxtpSet = uint8(reducedTxtp)
		for i := range maxSegments {
			frameHdr.segmentation.qidx[i] = uint8(qidx)
		}
		f.seqHdr = &seqHdr
		f.frameHdr = &frameHdr
		f.layout = layout
		f.bpc = 8
		if useQm != 0 {
			f.qm[tx][plane] = qm
		}

		var ts tileState
		for i := range maxSegments {
			for pl := range 3 {
				ts.dqmem[i][pl][0] = dqTbl[0][qidx][0]
				ts.dqmem[i][pl][1] = dqTbl[0][qidx][1]
			}
		}
		ts.dq = &ts.dqmem
		ts.cdf.initStatic(uint8(qidx))
		ts.msac.init(buf, false)

		tc := &taskContext[uint8, int16]{scratch: &scratch[uint8]{}, f: &f, ts: &ts}

		b := &av1Block{
			uvMode: uint8(uvMode),
			yMode:  uint8(yMode),
			yAngle: int8(yAngle),
		}

		cf := make([]int16, 32*32)
		eob, txtp, resCtx := decodeCoefs(tc, a, 0, l, 0, tx, bs, b, intra, plane, cf, txtpIn)

		if eob != num(want[0]) || txtp != num(want[1]) || int(resCtx) != num(want[2]) {
			t.Fatalf("line %d: tx=%d plane=%d intra=%d qm=%d = (eob=%d txtp=%d res=%d), want (%s %s %s)",
				line, tx, plane, intra, useQm, eob, txtp, resCtx, want[0], want[1], want[2])
		}
		if int(ts.msac.rng) != num(want[3]) || ts.msac.cnt != num(want[4]) ||
			ts.msac.pos != num(want[5]) {
			t.Fatalf("line %d: msac state = (rng=%d cnt=%d pos=%d), want (%s %s %s)",
				line, ts.msac.rng, ts.msac.cnt, ts.msac.pos, want[3], want[4], want[5])
		}

		if eob >= 0 {
			for i := range sw * sh {
				if int(cf[i]) != num(want[6+i]) {
					t.Fatalf("line %d: tx=%d plane=%d cf[%d] = %d, want %s",
						line, tx, plane, i, cf[i], want[6+i])
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
