package av1

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

const (
	recintFW     = 72
	recintFH     = 72
	recintBW     = 18
	recintStride = 128
	recintPlaneH = 128
	recintRefMax = 128
)

var recintRefDims = [7][2]int{
	{recintFW, recintFH}, {48, 48}, {96, 96}, {96, 48},
	{recintFW, 48}, {48, recintFH}, {recintFW, recintFH},
}

func recintHash(p []uint8) uint32 {
	h := uint32(2166136261)
	for _, v := range p {
		h ^= uint32(v)
		h *= 16777619
	}

	return h
}

func TestReconBInter(t *testing.T) {
	sc, closeFixture := openFixture(t, "recinter.txt")
	defer closeFixture()

	var cases, comp, warp, ii, obmc int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "recinter" {
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
		iter, layout, qidx, qmlvl := num(fields[p]), num(fields[p+1]),
			num(fields[p+2]), num(fields[p+3])
		useQm, reducedTxtp := num(fields[p+4]), num(fields[p+5])
		ohb, frameOffset := num(fields[p+6]), num(fields[p+7])
		p += 8

		bs, bx, by := num(fields[p]), num(fields[p+1]), num(fields[p+2])
		tlFilter := num(fields[p+3])
		lossless := num(fields[p+4])
		p += 5

		var b av1Block
		b.bs = uint8(bs)
		b.skip = uint8(num(fields[p]))
		b.maxYtx = uint8(num(fields[p+1]))
		b.uvtx = uint8(num(fields[p+2]))
		b.txSplit0 = uint8(num(fields[p+3]))
		b.txSplit1 = uint16(num(fields[p+4]))
		b.filter2d = uint8(num(fields[p+5]))
		b.compType = uint8(num(fields[p+6]))
		b.ref[0] = int8(num(fields[p+7]))
		b.ref[1] = int8(num(fields[p+8]))
		b.interMode = uint8(num(fields[p+9]))
		b.motionMode = uint8(num(fields[p+10]))
		b.interintraType = uint8(num(fields[p+11]))
		b.interintraMode = uint8(num(fields[p+12]))
		p += 13
		b.wedgeIdx = uint8(num(fields[p]))
		b.maskSign = uint8(num(fields[p+1]))
		b.mv[0] = mv{y: int16(num(fields[p+2])), x: int16(num(fields[p+3]))}
		b.mv[1] = mv{y: int16(num(fields[p+4])), x: int16(num(fields[p+5]))}
		p += 6

		ssVer := b2i(layout == pixelLayoutI420)
		ssHor := b2i(layout != pixelLayoutI444)

		seqHdr := &sequenceHeader{layout: layout, orderHintNBits: uint8(ohb)}
		frameHdr := &frameHeader{
			frameType:      frameTypeInter,
			frameOffset:    uint8(frameOffset),
			reducedTxtpSet: uint8(reducedTxtp),
		}
		frameHdr.quant.qm = uint8(useQm)
		frameHdr.segmentation.lossless[0] = uint8(lossless)
		frameHdr.width[0], frameHdr.height = recintFW, recintFH
		for i := range maxSegments {
			frameHdr.segmentation.qidx[i] = uint8(qidx)
		}

		f := &frameContext[uint8, int16]{
			seqHdr: seqHdr, frameHdr: frameHdr, layout: layout,
			bpc: 8, bitdepthMax: 255, ssHor: ssHor, ssVer: ssVer,
			bw: recintBW, bh: recintBW,
			sbStep: 16, sbShift: 4, sb128w: 1,
			dsp: newDspGo[uint8](),
		}
		f.stride[0] = recintStride
		f.stride[1] = recintStride >> ssHor
		f.stride[2] = f.stride[1]
		for pl := range 3 {
			buf := make([]uint8, recintPlaneH*recintStride)
			for i := range buf {
				buf[i] = recmcGen(iter, 50+pl, i)
			}
			f.cur[pl] = buf
			edge := make([]uint8, 128*4)
			for i := range edge {
				edge[i] = recmcGen(iter, 53+pl, i)
			}
			f.ipredEdge[pl] = edge
		}
		f.srCur = f.cur
		f.srStride = f.stride
		f.srWidth = recintFW

		if useQm != 0 {
			for t2 := range nRectTxSizes {
				for pl := range 3 {
					f.qm[t2][pl] = qmTbl[qmlvl][b2i(pl > 0)][t2]
				}
			}
		}

		for r := range 7 {
			rw, rh := recintRefDims[r][0], recintRefDims[r][1]
			f.refPoc[r] = uint8(num(fields[p]))
			f.gmvWarpAllowed[r] = num(fields[p+1]) != 0
			frameHdr.gmv[r].typ = num(fields[p+2])
			for i := range 6 {
				frameHdr.gmv[r].matrix[i] = int32(num(fields[p+3+i]))
			}
			getShearParams(&frameHdr.gmv[r])
			p += 9

			pic := &refPicture[uint8]{w: rw, h: rh}
			pic.stride[0] = rw
			pic.stride[1] = (rw + ssHor) >> ssHor
			pic.stride[2] = pic.stride[1]
			for pl := range 3 {
				buf := make([]uint8, recintRefMax*recintRefMax)
				for i := range buf {
					buf[i] = recmcGen(iter, 3*r+pl, i)
				}
				pic.planes[pl] = buf
			}
			f.refp[r] = pic
			if rw != recintFW || rh != recintFH {
				f.svc[r][0].scale = ((rw << 14) + recintFW>>1) / recintFW
				f.svc[r][1].scale = ((rh << 14) + recintFH>>1) / recintFH
				f.svc[r][0].step = (f.svc[r][0].scale + 8) >> 4
				f.svc[r][1].step = (f.svc[r][1].scale + 8) >> 4
			}
		}
		f.frameHdr.switchableCompRefs = 1
		f.setupJntWeights()

		ts := new(tileState)
		ts.tiling.colEnd = recintBW
		ts.tiling.rowEnd = recintBW
		ts.cdf.initStatic(uint8(qidx))
		for i := range maxSegments {
			for pl := range 3 {
				ts.dqmem[i][pl][0] = dqTbl[0][qidx][0]
				ts.dqmem[i][pl][1] = dqTbl[0][qidx][1]
			}
		}
		ts.dq = &ts.dqmem

		tc := &taskContext[uint8, int16]{scratch: &scratch[uint8]{}, f: f, ts: ts, bx: bx, by: by}
		tc.a = new(blockContext)
		tc.cf = make([]int16, 32*32)
		tc.tl4x4Filter = uint8(tlFilter)
		tc.rt.rf = &f.rf
		f.rf.frmHdr = frameHdr
		f.rf.iw4, f.rf.ih4 = recintBW, recintBW
		tc.rt.r = make([]refmvsBlock, 37*128)
		recmcTileGrid(tc.rt.r, 37, 128, iter, 5, recintBW)
		for i := range 37 {
			tc.rt.rows[i] = i * 128
		}
		tc.rt.tileCol.end = recintBW
		tc.rt.tileRow.end = recintBW
		for i := range 32 {
			tc.a.filter[0][i] = recmcGen(iter, 43, i) % 3
			tc.a.filter[1][i] = recmcGen(iter, 43, 32+i) % 3
			tc.l.filter[0][i] = recmcGen(iter, 44, i) % 3
			tc.l.filter[1][i] = recmcGen(iter, 44, 32+i) % 3
			tc.a.lcoef[i] = recmcGen(iter, 46, i) & 0x3f
			tc.l.lcoef[i] = recmcGen(iter, 46, 32+i) & 0x3f
			for pl := range 2 {
				tc.a.ccoef[pl][i] = recmcGen(iter, 47+pl, i) & 0x3f
				tc.l.ccoef[pl][i] = recmcGen(iter, 47+pl, 32+i) & 0x3f
			}
		}

		tc.warpmv.typ = num(fields[p])
		for i := range 6 {
			tc.warpmv.matrix[i] = int32(num(fields[p+1+i]))
		}
		getShearParams(&tc.warpmv)
		tc.warpmv.typ = num(fields[p])
		p += 7

		buf, err := hex.DecodeString(fields[p])
		if err != nil {
			t.Fatalf("line %d: %v", line, err)
		}
		p++
		ts.msac.init(buf, false)

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed record", line)
		}
		p++
		wantRet := num(fields[p])
		p++

		gotRet := 0
		if err := reconBInter(tc, bs, &b); err != nil {
			gotRet = -1
		}
		if gotRet != wantRet {
			t.Fatalf("line %d: ret = %d, want %d", line, gotRet, wantRet)
		}

		var wantHash [3]uint32
		for pl := range 3 {
			wantHash[pl] = unum(fields[p])
			p++
		}

		bDim := &blockDimensions[bs]
		bw4, bh4 := int(bDim[0]), int(bDim[1])
		for y := range bh4 * 4 {
			for x := range bw4 * 4 {
				got := int(f.cur[0][(by*4+y)*recintStride+bx*4+x])
				if got != num(fields[p]) {
					t.Fatalf("line %d: bs=%d y[%d][%d] = %d, want %s",
						line, bs, y, x, got, fields[p])
				}
				p++
			}
		}
		if layout != pixelLayoutI400 {
			cw := (bw4*4 + ssHor) >> ssHor
			ch := (bh4*4 + ssVer) >> ssVer
			cstride := recintStride >> ssHor
			cx, cy := (bx*4)>>ssHor, (by*4)>>ssVer
			for pl := 1; pl < 3; pl++ {
				for y := range ch {
					for x := range cw {
						got := int(f.cur[pl][(cy+y)*cstride+cx+x])
						if got != num(fields[p]) {
							t.Fatalf("line %d: bs=%d uv[%d][%d][%d] = %d, want %s",
								line, bs, pl, y, x, got, fields[p])
						}
						p++
					}
				}
			}
		}

		for pl := range 3 {
			if h := recintHash(f.cur[pl]); h != wantHash[pl] {
				t.Fatalf("line %d: bs=%d layout=%d comp=%d mm=%d ii=%d plane %d hash = %d, want %d",
					line, bs, layout, b.compType, b.motionMode, b.interintraType,
					pl, h, wantHash[pl])
			}
		}

		for i := range 32 {
			got := []int{
				int(tc.a.lcoef[i]), int(tc.l.lcoef[i]),
				int(tc.a.ccoef[0][i]), int(tc.l.ccoef[0][i]),
				int(tc.a.ccoef[1][i]), int(tc.l.ccoef[1][i]),
			}
			for j, v := range got {
				if v != num(fields[p+j]) {
					t.Fatalf("line %d: bs=%d coefctx[%d][%d] = %d, want %s",
						line, bs, j, i, v, fields[p+j])
				}
			}
			p += 6
		}

		if ts.msac.rng != unum(fields[p]) {
			t.Fatalf("line %d: rng = %d, want %s", line, ts.msac.rng, fields[p])
		}
		p++
		if int(tc.tl4x4Filter) != num(fields[p]) {
			t.Fatalf("line %d: tl4x4Filter = %d, want %s",
				line, tc.tl4x4Filter, fields[p])
		}

		if b.compType != compInterNone {
			comp++
		}
		if b.motionMode == mmWarp {
			warp++
		}
		if b.motionMode == mmObmc {
			obmc++
		}
		if b.interintraType != 0 {
			ii++
		}
		cases++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if cases == 0 || comp == 0 || warp == 0 || obmc == 0 || ii == 0 {
		t.Fatalf("thin coverage: %d cases, %d comp, %d warp, %d obmc, %d ii",
			cases, comp, warp, obmc, ii)
	}

	t.Logf("%d cases, %d comp, %d warp, %d obmc, %d interintra",
		cases, comp, warp, obmc, ii)
}
