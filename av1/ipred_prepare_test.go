package av1

import (
	"strconv"
	"strings"
	"testing"
)

func TestPrepareIntraEdges(t *testing.T) {
	sc, closeFixture := openFixture(t, "prep.txt")
	defer closeFixture()

	const (
		stride = 160
		pad    = 72
		rows   = 208
		topN   = 66
		leftN  = 64
	)

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "prep" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		mode := num(fields[1])
		haveLeft, haveTop := num(fields[2]) != 0, num(fields[3]) != 0
		w, h := num(fields[4]), num(fields[5])
		x, y := num(fields[6]), num(fields[7])
		edgeFlags := num(fields[8])
		filterEdge := num(fields[9]) != 0
		useSbEdge := num(fields[10]) != 0
		tw, th := num(fields[11]), num(fields[12])
		angleDelta := num(fields[13])

		p := 14
		plane := make([]uint8, rows*stride)
		dstOff := pad*stride + pad
		for i := range topN {
			plane[dstOff-stride+i-1] = uint8(num(fields[p+i]))
		}
		p += topN
		for i := range leftN {
			plane[dstOff+i*stride-1] = uint8(num(fields[p+i]))
		}
		p += leftN

		sbBuf := make([]uint8, stride+8)
		sbEdgeOff := 8
		for i := range topN {
			sbBuf[sbEdgeOff+x*4+i-1] = uint8(num(fields[p+i]))
		}
		p += topN

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		want := fields[p+1:]

		tl := make([]uint8, 257)
		angle := angleDelta

		var sb []uint8
		var sbOff int
		if useSbEdge {
			sb, sbOff = sbBuf, sbEdgeOff
		}

		got := prepareIntraEdges(x, haveLeft, y, haveTop, w, h, edgeFlags,
			plane, dstOff, stride, sb, sbOff, mode, &angle, tw, th, filterEdge,
			tl, 128, 0xff)

		if got != num(want[0]) || angle != num(want[1]) {
			t.Fatalf("line %d: mode=%d hl=%v ht=%v tw=%d th=%d = (mode %d, angle %d), want (%s, %s)",
				line, mode, haveLeft, haveTop, tw, th, got, angle, want[0], want[1])
		}
		for i := range 257 {
			if int(tl[i]) != num(want[2+i]) {
				t.Fatalf("line %d: mode=%d->%d hl=%v ht=%v tw=%d th=%d tl[%d] = %d, want %s",
					line, mode, got, haveLeft, haveTop, tw, th, i-128, tl[i], want[2+i])
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

func BenchmarkPrepareIntraEdges(b *testing.B) {
	const (
		stride = 160
		pad    = 72
		rows   = 208
	)

	plane := make([]uint8, rows*stride)
	for i := range plane {
		plane[i] = uint8(i * 7)
	}
	dstOff := pad*stride + pad
	tl := make([]uint8, 257)

	for _, tw := range []int{1, 2, 4, 8, 16} {
		b.Run(strconv.Itoa(tw*4), func(b *testing.B) {
			angle := 0
			for b.Loop() {
				prepareIntraEdges(4, true, 4, true, 64, 64, edgeAllTrAndBl,
					plane, dstOff, stride, nil, 0, paethPred, &angle, tw, tw,
					false, tl, 128, 0xff)
			}
		})
	}
}
