package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	recmcCurW   = 64
	recmcCurH   = 64
	recmcRefMax = 128
	recmcDstStr = 128
)

var recmcRefDims = [7][2]int{
	{recmcCurW, recmcCurH}, {48, 48}, {96, 96}, {96, 48},
	{recmcCurW, 48}, {48, recmcCurH}, {recmcCurW, recmcCurH},
}

func recmcGen(iter, kind, i int) uint8 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint8(v >> 24)
}

func recmcFrame(iter, layout int) (*frameContext[uint8, int16], *taskContext[uint8, int16]) {
	ssHor := b2i(layout != pixelLayoutI444)

	f := &frameContext[uint8, int16]{
		seqHdr:      &sequenceHeader{layout: layout},
		frameHdr:    &frameHeader{},
		layout:      layout,
		bpc:         8,
		bitdepthMax: 255,
		bw:          recmcCurW / 4,
		bh:          recmcCurH / 4,
	}
	f.frameHdr.width[0] = recmcCurW
	f.frameHdr.height = recmcCurH
	f.stride[0] = recmcCurW
	f.stride[1] = (recmcCurW + ssHor) >> ssHor
	f.stride[2] = f.stride[1]
	for pl := range 3 {
		buf := make([]uint8, recmcCurH*recmcCurW)
		for i := range buf {
			buf[i] = recmcGen(iter, 30+pl, i)
		}
		f.cur[pl] = buf
	}

	for r := range 7 {
		rw, rh := recmcRefDims[r][0], recmcRefDims[r][1]
		p := &refPicture[uint8]{w: rw, h: rh}
		p.stride[0] = rw
		p.stride[1] = (rw + ssHor) >> ssHor
		p.stride[2] = p.stride[1]
		for pl := range 3 {
			buf := make([]uint8, recmcRefMax*recmcRefMax)
			for i := range buf {
				buf[i] = recmcGen(iter, 3*r+pl, i)
			}
			p.planes[pl] = buf
		}
		f.refp[r] = p
		if rw != recmcCurW || rh != recmcCurH {
			f.svc[r][0].scale = ((rw << 14) + recmcCurW>>1) / recmcCurW
			f.svc[r][1].scale = ((rh << 14) + recmcCurH>>1) / recmcCurH
			f.svc[r][0].step = (f.svc[r][0].scale + 8) >> 4
			f.svc[r][1].step = (f.svc[r][1].scale + 8) >> 4
		}
	}

	ts := new(tileState)
	ts.tiling.colEnd = recmcCurW / 4
	ts.tiling.rowEnd = recmcCurH / 4
	tc := &taskContext[uint8, int16]{f: f, ts: ts}
	tc.a = new(blockContext)

	return f, tc
}

func TestReconMc(t *testing.T) {
	sc, closeFixture := openFixture(t, "recmc.txt")
	defer closeFixture()

	var nMc, nWarp, nObmc, nScaled int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "recmc" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		kind := fields[0]
		p := 1
		iter, layout, pl, bs := num(fields[p]), num(fields[p+1]),
			num(fields[p+2]), num(fields[p+3])
		p += 4

		ssVer := b2i(pl != 0 && layout == pixelLayoutI420)
		ssHor := b2i(pl != 0 && layout != pixelLayoutI444)
		hMul, vMul := 4>>ssHor, 4>>ssVer

		f, tc := recmcFrame(iter, layout)
		dst8 := make([]uint8, recmcDstStr*recmcDstStr)
		dst16 := make([]int16, recmcDstStr*recmcDstStr)

		var w, h int

		switch kind {
		case "M":
			bw4, bh4 := num(fields[p]), num(fields[p+1])
			bx, by := num(fields[p+2]), num(fields[p+3])
			m := mv{y: int16(num(fields[p+4])), x: int16(num(fields[p+5]))}
			refidx, isPrep := num(fields[p+6]), num(fields[p+7]) != 0
			f2d := num(fields[p+8])
			p += 9
			w, h = bw4*hMul, bh4*vMul
			if f.refp[refidx].w != recmcCurW || f.refp[refidx].h != recmcCurH {
				nScaled++
			}

			var d8 []uint8
			var d16 []int16
			if isPrep {
				d16 = dst16
			} else {
				d8 = dst8
			}
			if err := recMc(tc, d8, 0, d16, recmcDstStr, bw4, bh4, bx, by, pl,
				m, f.refp[refidx], refidx, f2d); err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			if !isPrep {
				dst16 = nil
			}
			nMc++

		case "W":
			bx, by := num(fields[p]), num(fields[p+1])
			refidx, isPrep := num(fields[p+2]), num(fields[p+3]) != 0
			p += 4
			var wm warpedMotionParams
			for i := range 6 {
				wm.matrix[i] = int32(num(fields[p+i]))
			}
			p += 6
			for i := range 4 {
				wm.abcd[i] = int16(num(fields[p+i]))
			}
			p += 4
			tc.bx, tc.by = bx, by
			bDim := &blockDimensions[bs]
			w, h = int(bDim[0])*hMul, int(bDim[1])*vMul

			var d8 []uint8
			var d16 []int16
			if isPrep {
				d16 = dst16
			} else {
				d8 = dst8
			}
			if err := recWarpAffine(tc, d8, 0, d16, recmcDstStr, bDim, pl,
				f.refp[refidx], &wm); err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			if !isPrep {
				dst16 = nil
			}
			nWarp++

		case "O":
			bx, by := num(fields[p]), num(fields[p+1])
			w4, h4 := num(fields[p+2]), num(fields[p+3])
			p += 4
			tc.bx, tc.by = bx, by
			bx4, by4 := bx&31, by&31
			bDim := &blockDimensions[bs]
			w, h = int(bDim[0])*hMul, int(bDim[1])*vMul

			tc.rt.r = make([]refmvsBlock, 37*128)
			recmcTileGrid(tc.rt.r, 37, 128, iter, 5, 16)
			for i := range 37 {
				tc.rt.rows[i] = i * 128
			}
			for i := range 32 {
				tc.a.filter[0][i] = recmcGen(iter, 43, i) % 3
				tc.a.filter[1][i] = recmcGen(iter, 43, 32+i) % 3
				tc.l.filter[0][i] = recmcGen(iter, 44, i) % 3
				tc.l.filter[1][i] = recmcGen(iter, 44, 32+i) % 3
			}
			for i := range dst8 {
				dst8[i] = recmcGen(iter, 45, i)
			}

			if err := recObmc(tc, dst8, 0, recmcDstStr, bDim, pl, bx4, by4,
				w4, h4); err != nil {
				t.Fatalf("line %d: %v", line, err)
			}
			dst16 = nil
			nObmc++

		default:
			t.Fatalf("line %d: unknown record %q", line, kind)
		}

		if fields[p] != "=" {
			t.Fatalf("line %d: malformed %s record", line, kind)
		}
		p++

		for y := range h {
			for x := range w {
				var got int
				if dst16 != nil {
					got = int(dst16[y*recmcDstStr+x])
				} else {
					got = int(dst8[y*recmcDstStr+x])
				}
				if got != num(fields[p]) {
					t.Fatalf("line %d: %s bs=%d pl=%d out[%d][%d] = %d, want %s",
						line, kind, bs, pl, y, x, got, fields[p])
				}
				p++
			}
		}
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if nMc == 0 || nWarp == 0 || nObmc == 0 || nScaled == 0 {
		t.Fatalf("thin coverage: %d mc (%d scaled), %d warp, %d obmc",
			nMc, nScaled, nWarp, nObmc)
	}

	t.Logf("%d mc (%d scaled), %d warp, %d obmc", nMc, nScaled, nWarp, nObmc)
}

func recmcBsForDims(w4, h4 int) uint8 {
	for bs := range nBsSizes {
		if int(blockDimensions[bs][0]) == w4 && int(blockDimensions[bs][1]) == h4 {
			return uint8(bs)
		}
	}

	return nBsSizes
}

func recmcTileGrid(blk []refmvsBlock, rows, stride, iter, off, nrows int) {
	bs4x4 := recmcBsForDims(1, 1)
	for i := range blk {
		b := &blk[i]
		b.bs = bs4x4
		b.ref[0] = int8(recmcGen(iter, 41, i) % 8)
		b.ref[1] = -1
		b.mv[0].y = int16(uint16(recmcGen(iter, 42, i*2))<<3) - 1024
		b.mv[0].x = int16(uint16(recmcGen(iter, 42, i*2+1))<<3) - 1024
		b.mv[1] = mv{}
		b.mf = 0
	}

	for y := 0; y < nrows; {
		lh := int(recmcGen(iter, 60, off+y) % 5)
		for lh != 0 && y&((1<<lh)-1) != 0 {
			lh--
		}
		for y+(1<<lh) > nrows {
			lh--
		}
		h4 := 1 << lh
		lwMin, lwMax := max(lh-2, 0), min(lh+2, 4)

		for x := 0; x < stride; {
			lw := clip(int(recmcGen(iter, 61, (off+y)*stride+x)%5), lwMin, lwMax)
			for lw > lwMin && x&((1<<lw)-1) != 0 {
				lw--
			}
			w4 := 1 << lw
			bs := recmcBsForDims(w4, h4)

			for yy := range h4 {
				for xx := range w4 {
					i := (off+y+yy)*stride + x + xx
					b := &blk[i]
					b.bs = bs
					b.ref[0] = int8(recmcGen(iter, 41, i) % 8)
					b.ref[1] = -1
					b.mv[0].y = int16(uint16(recmcGen(iter, 42, i*2))<<3) - 1024
					b.mv[0].x = int16(uint16(recmcGen(iter, 42, i*2+1))<<3) - 1024
					b.mv[1] = mv{}
					b.mf = 0
				}
			}
			x += w4
		}
		y += h4
	}
}
