package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

const (
	rbFW      = 64
	rbPad     = 64
	rbStride  = rbFW + 2*rbPad
	rbPlaneH  = rbFW + 2*rbPad
	rbPlaneSz = rbPlaneH * rbStride
)

func rbGen(iter, kind, i int) uint8 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint8(v >> 24)
}

func TestReconBIntra(t *testing.T) {
	sc, closeFixture := openFixture(t, "rbintra.txt")
	defer closeFixture()

	var cases int
	iter := 0

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "rbintra" {
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
		useQm, qmlvl := num(fields[3]), num(fields[4])
		edgeFilter := num(fields[5])
		intraEdgeFlags := num(fields[6])
		bs := num(fields[7])
		bx, by := num(fields[8]), num(fields[9])
		skip := num(fields[10])
		btx, buvtx := num(fields[11]), num(fields[12])
		yMode, yAngle := num(fields[13]), num(fields[14])
		uvMode, uvAngle := num(fields[15]), num(fields[16])
		variant := num(fields[17])
		palSz0, palSz1 := num(fields[18]), num(fields[19])
		cfl0, cfl1 := num(fields[20]), num(fields[21])
		reducedTxtp := num(fields[22])

		if fields[23] != "a" {
			t.Fatalf("line %d: malformed record at a", line)
		}
		p := 24

		var a, l blockContext
		rd := func(dst []uint8) {
			for i := range 32 {
				dst[i] = uint8(num(fields[p+i]))
			}
			p += 32
		}
		rd(a.lcoef[:])
		rd(l.lcoef[:])
		rd(a.intra[:])
		rd(l.intra[:])
		rd(a.mode[:])
		rd(l.mode[:])
		rd(a.uvmode[:])
		rd(l.uvmode[:])
		for pl := range 2 {
			rd(a.ccoef[pl][:])
			rd(l.ccoef[pl][:])
		}

		if fields[p] != "b" {
			t.Fatalf("line %d: malformed record at b", line)
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

		ssVer := b2i(layout == pixelLayoutI420)
		ssHor := b2i(layout != pixelLayoutI444)

		var f frameContext[uint8, int16]
		var seqHdr sequenceHeader
		var frameHdr frameHeader

		seqHdr.intraEdgeFilter = uint8(edgeFilter)
		frameHdr.reducedTxtpSet = uint8(reducedTxtp)
		for i := range maxSegments {
			frameHdr.segmentation.qidx[i] = uint8(qidx)
		}

		f.dsp = newDspGo[uint8]()
		f.seqHdr = &seqHdr
		f.frameHdr = &frameHdr
		f.bw, f.bh = 16, 16
		f.sbStep, f.sbShift, f.sb128w = 16, 4, 1
		f.layout = layout
		f.bpc = 8
		f.bitdepthMax = 0xff
		f.stride[0], f.stride[1] = rbStride, rbStride

		if useQm != 0 {
			for t2 := range nRectTxSizes {
				for pl := range 3 {
					f.qm[t2][pl] = qmTbl[qmlvl][b2i(pl > 0)][t2]
				}
			}
		}

		for pl := range 3 {
			plane := make([]uint8, rbPlaneSz)
			for i := range plane {
				plane[i] = rbGen(iter, pl, i)
			}
			f.cur[pl] = plane
			edge := make([]uint8, 128)
			for i := range edge {
				edge[i] = rbGen(iter, 3+pl, i)
			}
			f.ipredEdge[pl] = edge
		}

		var ts tileState
		for i := range maxSegments {
			for pl := range 3 {
				ts.dqmem[i][pl][0] = dqTbl[0][qidx][0]
				ts.dqmem[i][pl][1] = dqTbl[0][qidx][1]
			}
		}
		ts.dq = &ts.dqmem
		ts.tiling.colEnd, ts.tiling.rowEnd = 16, 16
		ts.cdf.initStatic(uint8(qidx))
		ts.msac.init(buf, false)

		tc := &taskContext[uint8, int16]{f: &f, ts: &ts, bx: bx, by: by}
		tc.a = &a
		tc.l = l
		tc.cf = make([]int16, 32*32)

		if variant == 1 {
			for pl := range 3 {
				for i := range 8 {
					tc.scratch.pal[pl][i] = rbGen(iter, 6+pl, i)
				}
			}
			for i := range tc.scratch.palIdxY {
				tc.scratch.palIdxY[i] = rbGen(iter, 9, i) & 0x77
			}
			for i := range tc.scratch.palIdxUv {
				tc.scratch.palIdxUv[i] = rbGen(iter, 10, i) & 0x77
			}
		}

		b := &av1Block{
			skip:     uint8(skip),
			tx:       uint8(btx),
			uvtx:     uint8(buvtx),
			yMode:    uint8(yMode),
			yAngle:   int8(yAngle),
			uvMode:   uint8(uvMode),
			uvAngle:  int8(uvAngle),
			palSz:    [2]uint8{uint8(palSz0), uint8(palSz1)},
			cflAlpha: [2]int8{int8(cfl0), int8(cfl1)},
		}

		// dst offsets are relative to the visible origin
		for pl := range 3 {
			f.cur[pl] = f.cur[pl][rbPad*rbStride+rbPad:]
		}

		reconBIntra(tc, bs, intraEdgeFlags, b)

		if tc.bx != num(want[0]) || tc.by != num(want[1]) {
			t.Fatalf("line %d: bx/by = (%d, %d), want (%s, %s)",
				line, tc.bx, tc.by, want[0], want[1])
		}
		if int(ts.msac.rng) != num(want[2]) || ts.msac.cnt != num(want[3]) ||
			ts.msac.pos != num(want[4]) {
			t.Fatalf("line %d: bs=%d variant=%d msac = (rng=%d cnt=%d pos=%d), want (%s %s %s)",
				line, bs, variant, ts.msac.rng, ts.msac.cnt, ts.msac.pos,
				want[2], want[3], want[4])
		}

		bDim := &blockDimensions[bs]
		bw4, bh4 := int(bDim[0]), int(bDim[1])
		cbw4, cbh4 := (bw4+ssHor)>>ssHor, (bh4+ssVer)>>ssVer

		w := 5
		for r := range bh4 * 4 {
			for c := range bw4 * 4 {
				got := int(f.cur[0][(by*4+r)*rbStride+bx*4+c])
				if got != num(want[w]) {
					t.Fatalf("line %d: bs=%d variant=%d y[%d][%d] = %d, want %s",
						line, bs, variant, r, c, got, want[w])
				}
				w++
			}
		}
		for pl := 1; pl < 3; pl++ {
			for r := range cbh4 * 4 {
				for c := range cbw4 * 4 {
					got := int(f.cur[pl][((by*4>>ssVer)+r)*rbStride+(bx*4>>ssHor)+c])
					if got != num(want[w]) {
						t.Fatalf("line %d: bs=%d variant=%d uv%d[%d][%d] = %d, want %s",
							line, bs, variant, pl, r, c, got, want[w])
					}
					w++
				}
			}
		}

		chk := func(name string, src []uint8) {
			for i := range 32 {
				if int(src[i]) != num(want[w]) {
					t.Fatalf("line %d: bs=%d variant=%d %s[%d] = %d, want %s",
						line, bs, variant, name, i, src[i], want[w])
				}
				w++
			}
		}
		chk("a.lcoef", a.lcoef[:])
		chk("l.lcoef", tc.l.lcoef[:])
		for pl := range 2 {
			chk("a.ccoef", a.ccoef[pl][:])
			chk("l.ccoef", tc.l.ccoef[pl][:])
		}

		cases++
		iter++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 {
		t.Fatal("no fixture records")
	}

	t.Logf("%d cases", cases)
}
