package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

const (
	binterStride = 128
	binterRows   = 37
	binterB4     = 64
)

func binterFill(r []refmvsBlock, iter int) {
	for i := range r {
		b := &r[i]
		b.mv[0].y = int16(warprefGen(iter, 0, i*4+0)&0x1fff) - 0x1000
		b.mv[0].x = int16(warprefGen(iter, 0, i*4+1)&0x1fff) - 0x1000
		b.mv[1].y = int16(warprefGen(iter, 0, i*4+2)&0x1fff) - 0x1000
		b.mv[1].x = int16(warprefGen(iter, 0, i*4+3)&0x1fff) - 0x1000
		switch warprefGen(iter, 1, i) % 3 {
		case 0:
			b.ref[0], b.ref[1] = 0, -1
		case 1:
			b.ref[0] = int8(1 + warprefGen(iter, 6, i)%7)
			b.ref[1] = -1
		default:
			b.ref[0] = int8(1 + warprefGen(iter, 6, i)%2)
			b.ref[1] = int8(5 + warprefGen(iter, 7, i)%2)
		}
		b.bs = uint8(warprefGen(iter, 2, i) % nBsSizes)
		b.mf = uint8(warprefGen(iter, 3, i) & 3)
	}
}

func binterHash(r []refmvsBlock) uint32 {
	h := uint32(2166136261)
	put := func(v uint32, n int) {
		for range n {
			h ^= v & 0xff
			h *= 16777619
			v >>= 8
		}
	}
	for i := range r {
		put(uint32(uint16(r[i].mv[0].y)), 2)
		put(uint32(uint16(r[i].mv[0].x)), 2)
		put(uint32(uint16(r[i].mv[1].y)), 2)
		put(uint32(uint16(r[i].mv[1].x)), 2)
		put(uint32(uint8(r[i].ref[0])), 1)
		put(uint32(uint8(r[i].ref[1])), 1)
		put(uint32(r[i].bs), 1)
		put(uint32(r[i].mf), 1)
	}

	return h
}

func binterCdfs(c *cdfContext) []*uint16 {
	var out []*uint16
	add := func(a []uint16, n int) {
		for i := range n + 1 {
			out = append(out, &a[i])
		}
	}
	add1 := func(a [][2]uint16) {
		for i := range a {
			add(a[i][:], 1)
		}
	}
	m, mi := &c.m, &c.mi
	add1(mi.maskComp[:])
	add1(mi.jntComp[:])
	add1(mi.wedgeComp[:])
	add1(mi.interintra[:])
	add1(mi.interintraWedge[:])
	add1(mi.comp[:])
	add1(mi.compDir[:])
	add1(mi.intra[:])
	add1(m.skip[:])
	add1(mi.skipMode[:])
	add1(mi.segPred[:])
	add1(mi.newmvMode[:])
	add1(mi.globalmvMode[:])
	add1(mi.refmvMode[:])
	add1(mi.drlBit[:])
	add1(mi.obmc[:])
	for i := range mi.compFwdRef {
		add1(mi.compFwdRef[i][:])
	}
	for i := range mi.compBwdRef {
		add1(mi.compBwdRef[i][:])
	}
	for i := range mi.compUniRef {
		add1(mi.compUniRef[i][:])
	}
	for i := range mi.ref {
		add1(mi.ref[i][:])
	}
	for i := range m.txpart {
		add1(m.txpart[i][:])
	}
	for i := range mi.wedgeIdx {
		add(mi.wedgeIdx[i][:], 15)
	}
	for i := range mi.interintraMode {
		add(mi.interintraMode[i][:], 3)
	}
	for i := range mi.compInterMode {
		add(mi.compInterMode[i][:], 7)
	}
	for i := range mi.filter {
		for j := range mi.filter[i] {
			add(mi.filter[i][j][:], nSwitchableFilters-1)
		}
	}
	for i := range mi.motionMode {
		add(mi.motionMode[i][:], 2)
	}
	for i := range mi.yMode {
		add(mi.yMode[i][:], nIntraPredModes-1)
	}
	for i := range m.segID {
		add(m.segID[i][:], maxSegments-1)
	}
	for i := range m.uvMode {
		for j := range m.uvMode[i] {
			add(m.uvMode[i][j][:], nUvIntraPredModes-1-b2i(i == 0))
		}
	}
	for i := range m.angleDelta {
		add(m.angleDelta[i][:], 6)
	}
	add(m.cflSign[:], 7)
	for i := range m.cflAlpha {
		add(m.cflAlpha[i][:], 15)
	}
	for i := range m.txsz {
		for j := range m.txsz[i] {
			add(m.txsz[i][j][:], min(i+1, 2))
		}
	}
	add(c.mv.joint[:], nMvJoints-1)
	for k := range c.mv.comp {
		cm := &c.mv.comp[k]
		add(cm.classes[:], 10)
		add(cm.sign[:], 1)
		add(cm.class0[:], 1)
		add(cm.class0Hp[:], 1)
		add(cm.classNHp[:], 1)
		add(cm.classNFp[:], 3)
		for i := range cm.class0Fp {
			add(cm.class0Fp[i][:], 3)
		}
		for i := range cm.classN {
			add(cm.classN[i][:], 1)
		}
	}

	return out
}

func binterReadBc(bc *blockContext, fields []string, num func(string) int) int {
	p := 0
	for i := range 32 {
		bc.mode[i] = uint8(num(fields[p]))
		bc.intra[i] = uint8(num(fields[p+1]))
		bc.skip[i] = uint8(num(fields[p+2]))
		bc.skipMode[i] = uint8(num(fields[p+3]))
		bc.segPred[i] = uint8(num(fields[p+4]))
		bc.palSz[i] = uint8(num(fields[p+5]))
		bc.compType[i] = uint8(num(fields[p+6]))
		bc.ref[0][i] = int8(num(fields[p+7]))
		bc.ref[1][i] = int8(num(fields[p+8]))
		bc.filter[0][i] = uint8(num(fields[p+9]))
		bc.filter[1][i] = uint8(num(fields[p+10]))
		bc.txIntra[i] = int8(num(fields[p+11]))
		p += 12
	}

	return p
}

func binterCheckBc(t *testing.T, line int, name string, bc *blockContext,
	fields []string, num func(string) int,
) int {
	t.Helper()

	p := 0
	for i := range 32 {
		got := []int{
			int(bc.mode[i]), int(bc.intra[i]), int(bc.skip[i]), int(bc.skipMode[i]),
			int(bc.segPred[i]), int(bc.palSz[i]), int(bc.compType[i]),
			int(bc.ref[0][i]), int(bc.ref[1][i]), int(bc.filter[0][i]),
			int(bc.filter[1][i]), int(bc.txIntra[i]),
		}
		for j, v := range got {
			if v != num(fields[p+j]) {
				t.Fatalf("line %d: %s[%d] field %d = %d, want %s",
					line, name, i, j, v, fields[p+j])
			}
		}
		p += 12
	}

	return p
}

func TestDecodeBInter(t *testing.T) {
	sc, closeFixture := openFixture(t, "binter.txt")
	defer closeFixture()

	var cases, inter, warp int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "binter" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}
		unum := func(s string) uint32 {
			v, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return uint32(v)
		}

		p := 1
		iter, layout := num(fields[p]), num(fields[p+1])
		bl, bp, bs := num(fields[p+2]), num(fields[p+3]), num(fields[p+4])
		bx4, by4 := num(fields[p+5]), num(fields[p+6])
		edgeFlags, update := num(fields[p+7]), num(fields[p+8])
		p += 9

		seqHdr := &sequenceHeader{
			layout:         layout,
			orderHintNBits: uint8(num(fields[p])),
			jntComp:        uint8(num(fields[p+1])),
			maskedCompound: uint8(num(fields[p+2])),
			interIntra:     uint8(num(fields[p+3])),
			dualFilter:     uint8(num(fields[p+4])),
		}
		p += 5

		frameHdr := &frameHeader{
			frameType:            frameTypeInter,
			frameOffset:          uint8(num(fields[p])),
			hp:                   uint8(num(fields[p+1])),
			forceIntegerMv:       uint8(num(fields[p+2])),
			switchableCompRefs:   uint8(num(fields[p+3])),
			switchableMotionMode: uint8(num(fields[p+4])),
			warpMotion:           uint8(num(fields[p+5])),
		}
		p += 6

		frameHdr.segmentation.enabled = uint8(num(fields[p]))
		frameHdr.segmentation.updateMap = uint8(num(fields[p+1]))
		frameHdr.segmentation.temporal = uint8(num(fields[p+2]))
		frameHdr.segmentation.segData.preskip = uint8(num(fields[p+3]))
		frameHdr.segmentation.segData.lastActiveSegid = int8(num(fields[p+4]))
		hasPrevSegmap := num(fields[p+5]) != 0
		p += 6
		for i := range maxSegments {
			d := &frameHdr.segmentation.segData.d[i]
			d.ref = int8(num(fields[p]))
			d.skip = uint8(num(fields[p+1]))
			d.globalmv = uint8(num(fields[p+2]))
			p += 3
		}

		frameHdr.skipModeEnabled = uint8(num(fields[p]))
		frameHdr.skipModeRefs[0] = int8(num(fields[p+1]))
		frameHdr.skipModeRefs[1] = int8(num(fields[p+2]))
		frameHdr.txfmMode = num(fields[p+3])
		frameHdr.subpelFilterMode = num(fields[p+4])
		frameHdr.width[0], frameHdr.height = 256, 256
		p += 5

		var parsed av1Block
		f := &frameContext[uint8, int16]{
			seqHdr: seqHdr, frameHdr: frameHdr, layout: layout,
			b4Stride: binterB4, bw: 64, bh: 64, w4: 64, h4: 64,
			reconIntra: func(_ *taskContext[uint8, int16], _, _ int, b *av1Block) {
				parsed = *b
			},
			reconInter: func(_ *taskContext[uint8, int16], _ int, b *av1Block) error {
				parsed = *b

				return nil
			},
		}
		f.curSegMap = make([]uint8, 64*binterB4)
		if hasPrevSegmap {
			f.prevSegMap = make([]uint8, 64*binterB4)
			for i := range f.prevSegMap {
				f.prevSegMap[i] = uint8(warprefGen(iter, 8, i) % 8)
			}
		}

		rf := &refmvsFrame{frmHdr: frameHdr, iw4: 64, ih4: 64, iw8: 32, ih8: 32}
		for i := range 7 {
			f.refPoc[i] = uint8(num(fields[p]))
			f.svc[i][0].scale = num(fields[p+1])
			f.gmvWarpAllowed[i] = num(fields[p+2]) != 0
			frameHdr.gmv[i].typ = num(fields[p+3])
			for j := range 6 {
				frameHdr.gmv[i].matrix[j] = int32(num(fields[p+4+j]))
			}
			getShearParams(&frameHdr.gmv[i])
			rf.signBias[i] = uint8(num(fields[p+10]))
			rf.pocdiff[i] = int8(num(fields[p+11]))
			p += 12
		}

		buf, err := hex.DecodeString(fields[p])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p++

		ts := new(tileState)
		ts.tiling.colStart, ts.tiling.colEnd = 0, 32
		ts.tiling.rowStart, ts.tiling.rowEnd = 0, 32
		ts.lastQidx = 128
		ts.lflvl = &f.lf.lvl

		tc := &taskContext[uint8, int16]{f: f, ts: ts, bx: bx4, by: by4}
		tc.a = new(blockContext)
		tc.lfMask = new(av1Filter)
		tc.curSbCdefIdx = []int8{-1, -1, -1, -1}
		tc.rt.rf = rf
		tc.rt.r = make([]refmvsBlock, binterRows*binterStride)
		for i := range binterRows {
			tc.rt.rows[i] = i * binterStride
		}
		tc.rt.tileCol.start, tc.rt.tileCol.end = 0, 32
		tc.rt.tileRow.start, tc.rt.tileRow.end = 0, 32
		binterFill(tc.rt.r, iter)

		p += binterReadBc(tc.a, fields[p:], num)
		p += binterReadBc(&tc.l, fields[p:], num)

		if h := binterHash(tc.rt.r); h != unum(fields[p]) {
			t.Fatalf("line %d: initial refmvs hash = %d, want %s", line, h, fields[p])
		}
		p++

		nCdf := num(fields[p])
		p++
		cdfs := binterCdfs(&ts.cdf)
		if len(cdfs) != nCdf {
			t.Fatalf("line %d: %d cdf words, want %d", line, len(cdfs), nCdf)
		}
		before := make([]uint16, nCdf)
		for i, c := range cdfs {
			*c = uint16(num(fields[p+i]))
			before[i] = *c
		}
		p += nCdf

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++
		wantRet := num(fields[p])
		p++

		ts.msac.init(buf, update == 0)

		err = tc.decodeB(bl, bs, bp, edgeFlags)
		gotRet := 0
		if err != nil {
			gotRet = -1
		}
		if gotRet != wantRet {
			t.Fatalf("line %d: ret = %d (%v), want %d", line, gotRet, err, wantRet)
		}
		if wantRet != 0 {
			cases++

			continue
		}

		b := &parsed
		ssVer, ssHor := b2i(layout == pixelLayoutI420), b2i(layout != pixelLayoutI444)
		bDim := &blockDimensions[bs]
		hasChroma := layout != pixelLayoutI400 &&
			(int(bDim[0]) > ssHor || bx4&1 != 0) &&
			(int(bDim[1]) > ssVer || by4&1 != 0)

		var got []int
		var names []string
		add := func(n string, v int) {
			got = append(got, v)
			names = append(names, n)
		}

		add("intra", int(b.intra))
		add("segID", int(b.segID))
		add("skipMode", int(b.skipMode))
		add("skip", int(b.skip))
		add("uvtx", int(b.uvtx))

		if b.intra != 0 {
			add("yMode", int(b.yMode))
			add("tx", int(b.tx))
			add("palSz0", int(b.palSz[0]))
			add("palSz1", int(b.palSz[1]))
			add("yAngle", int(b.yAngle))
			add("hasChroma", b2i(hasChroma))
			if hasChroma {
				add("uvMode", int(b.uvMode))
				add("uvAngle", int(b.uvAngle))
				if b.uvMode == cflPred {
					add("cflAlpha0", int(b.cflAlpha[0]))
					add("cflAlpha1", int(b.cflAlpha[1]))
				}
			}
		} else {
			add("compType", int(b.compType))
			add("interMode", int(b.interMode))
			add("filter2d", int(b.filter2d))
			add("maxYtx", int(b.maxYtx))
			add("txSplit0", int(b.txSplit0))
			add("txSplit1", int(b.txSplit1))
			add("ref0", int(b.ref[0]))
			add("ref1", int(b.ref[1]))
			add("mv0y", int(b.mv[0].y))
			add("mv0x", int(b.mv[0].x))
			if b.compType != compInterNone {
				add("mv1y", int(b.mv[1].y))
				add("mv1x", int(b.mv[1].x))
			}
			if b.compType >= compInterSeg {
				add("maskSign", int(b.maskSign))
			}
			if !(b.compType == compInterNone && b.interMode == globalmv) {
				add("drlIdx", int(b.drlIdx))
			}
			if b.compType == compInterNone {
				add("interintraType", int(b.interintraType))
				add("motionMode", int(b.motionMode))
				if b.interintraType != interIntraNone {
					add("interintraMode", int(b.interintraMode))
				}
			}
			if b.compType == compInterWedge ||
				(b.compType == compInterNone && b.interintraType == interIntraWedge) {
				add("wedgeIdx", int(b.wedgeIdx))
			}
			if b.compType == compInterNone && b.motionMode == mmWarp {
				add("wm.typ", tc.warpmv.typ)
				for i := range 6 {
					add("wm.m", int(tc.warpmv.matrix[i]))
				}
				for i := range 4 {
					add("wm.abcd", int(tc.warpmv.abcd[i]))
				}
			}
		}

		for i, v := range got {
			if v != num(fields[p+i]) {
				t.Fatalf("line %d: bs=%d %s = %d, want %s",
					line, bs, names[i], v, fields[p+i])
			}
		}
		p += len(got)

		p += binterCheckBc(t, line, "a", tc.a, fields[p:], num)
		p += binterCheckBc(t, line, "l", &tc.l, fields[p:], num)

		if h := binterHash(tc.rt.r); h != unum(fields[p]) {
			t.Fatalf("line %d: bs=%d refmvs hash = %d, want %s", line, bs, h, fields[p])
		}
		p++

		rb := &tc.rt.r[tc.rt.rows[(by4&31)+5]+tc.bx]
		rbGot := []int{
			int(rb.mv[0].y), int(rb.mv[0].x), int(rb.mv[1].y), int(rb.mv[1].x),
			int(rb.ref[0]), int(rb.ref[1]), int(rb.bs), int(rb.mf),
		}
		for i, v := range rbGot {
			if v != num(fields[p+i]) {
				t.Fatalf("line %d: bs=%d splat[%d] = %d, want %s",
					line, bs, i, v, fields[p+i])
			}
		}
		p += len(rbGot)

		if ts.msac.rng != unum(fields[p]) {
			t.Fatalf("line %d: rng = %d, want %s", line, ts.msac.rng, fields[p])
		}
		p++

		nDiff := num(fields[p])
		p++
		want := make([]uint16, nCdf)
		copy(want, before)
		for range nDiff {
			want[num(fields[p])] = uint16(num(fields[p+1]))
			p += 2
		}
		for i, c := range cdfs {
			if *c != want[i] {
				t.Fatalf("line %d: bs=%d cdf[%d] = %d, want %d",
					line, bs, i, *c, want[i])
			}
		}

		if b.intra == 0 {
			inter++
			if b.motionMode == mmWarp {
				warp++
			}
		}
		cases++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 || inter == 0 || warp == 0 {
		t.Fatalf("thin coverage: %d cases, %d inter, %d warp", cases, inter, warp)
	}

	t.Logf("%d cases, %d inter, %d warp", cases, inter, warp)
}
