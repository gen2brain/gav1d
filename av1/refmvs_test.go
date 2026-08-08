package av1

import (
	"strconv"
	"strings"
	"testing"
)

const (
	rmvStride = 128
	rmvRows   = 37
	rmvTStr   = 128
	rmvTRows  = 32
	rmvRpRows = 64
)

func rmvGen(iter, kind, i int) uint16 {
	v := uint32(i)*2654435761 + uint32(kind)*2246822519 + uint32(iter)*3266489917
	v ^= v >> 15
	v *= 2654435761

	return uint16(v >> 16)
}

func rmvFill(iter int) ([]refmvsBlock, []int) {
	r := make([]refmvsBlock, rmvRows*rmvStride)
	for i := range r {
		b := &r[i]
		b.mv[0].y = int16(rmvGen(iter, 0, i*4+0))
		b.mv[0].x = int16(rmvGen(iter, 0, i*4+1))
		b.mv[1].y = int16(rmvGen(iter, 0, i*4+2))
		b.mv[1].x = int16(rmvGen(iter, 0, i*4+3))
		b.ref[0] = int8(rmvGen(iter, 1, i*2+0)%9) - 1
		b.ref[1] = int8(rmvGen(iter, 1, i*2+1)%9) - 1
		b.bs = uint8(rmvGen(iter, 2, i) % nBsSizes)
		b.mf = uint8(rmvGen(iter, 3, i) & 3)
	}
	rows := make([]int, rmvRows)
	for i := range rows {
		rows[i] = i * rmvStride
	}

	return r, rows
}

func TestRefmvs(t *testing.T) {
	sc, closeFixture := openFixture(t, "refmvs.txt")
	defer closeFixture()

	var cases int

	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] == "refmvs" {
			continue
		}

		num := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("line %d: %v", line, err)
			}

			return v
		}

		iter := num(fields[1])
		r, rows := rmvFill(iter)

		var got []int
		var want []string

		switch fields[0] {
		case "S":
			bx4, bw4, bh4, ry := num(fields[2]), num(fields[3]), num(fields[4]), num(fields[5])
			var tmpl refmvsBlock
			tmpl.mv[0].y = int16(num(fields[6]))
			tmpl.mv[0].x = int16(num(fields[7]))
			tmpl.mv[1].y = int16(num(fields[8]))
			tmpl.mv[1].x = int16(num(fields[9]))
			tmpl.ref[0] = int8(num(fields[10]))
			tmpl.ref[1] = int8(num(fields[11]))
			tmpl.bs = uint8(num(fields[12]))
			tmpl.mf = uint8(num(fields[13]))
			if fields[14] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want = fields[15:]

			splatMv(r, rows[ry:], &tmpl, bx4, bw4, bh4)

			for y := range bh4 {
				for x := range bw4 {
					b := &r[(ry+y)*rmvStride+bx4+x]
					got = append(got, int(b.mv[0].y), int(b.mv[0].x),
						int(b.mv[1].y), int(b.mv[1].x),
						int(b.ref[0]), int(b.ref[1]), int(b.bs), int(b.mf))
				}
			}

		case "T":
			rowStart8, rowEnd8 := num(fields[2]), num(fields[3])
			colStart8, colEnd8 := num(fields[4]), num(fields[5])
			var refSign [7]uint8
			for i := range 7 {
				refSign[i] = uint8(num(fields[6+i]))
			}
			if fields[13] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want = fields[14:]

			rp := make([]refmvsTemporalBlock, rmvTRows*rmvTStr)
			saveTmvs(rp, 0, rmvTStr, r, rows, &refSign,
				colEnd8, rowEnd8, colStart8, rowStart8)

			for y := range rowEnd8 - rowStart8 {
				for x := colStart8; x < colEnd8; x++ {
					b := &rp[y*rmvTStr+x]
					got = append(got, int(b.mv.y), int(b.mv.x), int(b.ref))
				}
			}

		case "I":
			p := 2
			var seq sequenceHeader
			var hdr frameHeader
			seq.sb128 = uint8(num(fields[p]))
			seq.orderHintNBits = uint8(num(fields[p+1]))
			hdr.width[0] = num(fields[p+2])
			hdr.height = num(fields[p+3])
			hdr.tiling.rows = 1
			hdr.frameOffset = uint8(num(fields[p+4]))
			hdr.useRefFrameMvs = uint8(num(fields[p+5]))
			present := num(fields[p+6])
			p += 7

			var refPoc [7]uint8
			var refRefPoc [7][7]uint8
			for i := range 7 {
				refPoc[i] = uint8(num(fields[p]))
				p++
			}
			for i := range 7 {
				for j := range 7 {
					refRefPoc[i][j] = uint8(num(fields[p]))
					p++
				}
			}

			var rpRef [7][]refmvsTemporalBlock
			for i := range 7 {
				if present>>i&1 != 0 {
					rpRef[i] = make([]refmvsTemporalBlock, 1)
				}
			}

			if fields[p] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want = fields[p+1:]

			var rf refmvsFrame
			refmvsInitFrame(&rf, &seq, &hdr, &refPoc, nil, &refRefPoc, &rpRef, 1, 1)

			got = append(got, rf.sbsz, rf.iw8, rf.ih8, rf.iw4, rf.ih4,
				rf.rpStride, rf.nMfmvs, rf.useRefFrameMvs)
			for i := range 7 {
				got = append(got, int(rf.signBias[i]), int(rf.mfmvSign[i]),
					int(rf.pocdiff[i]))
			}
			for n := range 3 {
				got = append(got, int(rf.mfmvRef[n]), int(rf.mfmvRef2cur[n]))
				for m := range 7 {
					got = append(got, int(rf.mfmvRef2ref[n][m]))
				}
			}

		case "L":
			p := 2
			nMfmvs := num(fields[p])
			rowStart8, rowEnd8 := num(fields[p+1]), num(fields[p+2])
			colStart8, colEnd8 := num(fields[p+3]), num(fields[p+4])
			p += 5

			rf := &refmvsFrame{
				iw8:          64,
				ih8:          64,
				rpStride:     rmvTStr,
				nTileThreads: 1,
				nMfmvs:       nMfmvs,
			}
			for n := range 3 {
				rf.mfmvRef[n] = uint8(num(fields[p]))
				rf.mfmvRef2cur[n] = int8(num(fields[p+1]))
				p += 2
				for i := range 7 {
					rf.mfmvRef2ref[n][i] = uint8(num(fields[p]))
					p++
				}
			}

			for i := range 7 {
				buf := make([]refmvsTemporalBlock, rmvRpRows*rmvTStr)
				for j := range buf {
					buf[j].mv.y = int16(rmvGen(iter, 10+i, j*2+0)&0x1fff) - 0x1000
					buf[j].mv.x = int16(rmvGen(iter, 10+i, j*2+1)&0x1fff) - 0x1000
					buf[j].ref = uint8(rmvGen(iter, 20+i, j) % 8)
				}
				rf.rpRef[i] = buf
			}
			rf.rpProj = make([]refmvsTemporalBlock, 16*rmvTStr)

			if fields[p] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want = fields[p+1:]

			loadTmvs(rf, 0, colStart8, colEnd8, rowStart8, rowEnd8)

			for y := range rowEnd8 - rowStart8 {
				for x := colStart8; x < colEnd8; x++ {
					b := &rf.rpProj[((rowStart8&15)+y)*rmvTStr+x]
					got = append(got, int(b.mv.y), int(b.mv.x), int(b.ref))
				}
			}

		case "F":
			p := 2
			bs, edgeFlags := num(fields[p]), num(fields[p+1])
			by4, bx4 := num(fields[p+2]), num(fields[p+3])
			r0, r1 := int8(num(fields[p+4])), int8(num(fields[p+5]))
			p += 6

			var hdr frameHeader
			hdr.hp = uint8(num(fields[p]))
			hdr.forceIntegerMv = uint8(num(fields[p+1]))
			hdr.useRefFrameMvs = uint8(num(fields[p+2]))
			p += 3

			rf := &refmvsFrame{
				frmHdr:         &hdr,
				iw4:            128,
				ih4:            128,
				iw8:            64,
				ih8:            64,
				rpStride:       rmvTStr,
				useRefFrameMvs: num(fields[p]),
			}
			p++
			for i := range 7 {
				rf.signBias[i] = uint8(num(fields[p]))
				rf.pocdiff[i] = int8(num(fields[p+1]))
				p += 2
			}
			for i := range 7 {
				hdr.gmv[i].typ = num(fields[p])
				p++
				for j := range 6 {
					hdr.gmv[i].matrix[j] = int32(num(fields[p]))
					p++
				}
			}

			if fields[p] != "=" {
				t.Fatalf("line %d: malformed record", line)
			}
			want = fields[p+1:]

			rp := make([]refmvsTemporalBlock, rmvTRows*rmvTStr)
			for i := range rp {
				rp[i].mv.y = int16(rmvGen(iter, 4, i*2+0)&0x1fff) - 0x1000
				rp[i].mv.x = int16(rmvGen(iter, 4, i*2+1)&0x1fff) - 0x1000
				rp[i].ref = uint8(rmvGen(iter, 5, i) % 8)
			}

			rt := &refmvsTile{rf: rf, r: r, rpProj: rp}
			for i := range rt.rows {
				rt.rows[i] = i * rmvStride
			}
			rt.tileCol.start, rt.tileCol.end = 0, 32
			rt.tileRow.start, rt.tileRow.end = 0, 32

			mvstack := make([]refmvsCandidate, 8)
			var cnt, ctx int
			refmvsFind(rt, mvstack, &cnt, &ctx, [2]int8{r0, r1},
				bs, edgeFlags, by4, bx4)

			got = append(got, cnt, ctx)
			for i := range 8 {
				got = append(got, int(mvstack[i].mv[0].y), int(mvstack[i].mv[0].x),
					int(mvstack[i].mv[1].y), int(mvstack[i].mv[1].x),
					mvstack[i].weight)
			}

		default:
			t.Fatalf("line %d: unknown record %q", line, fields[0])
		}

		if len(got) != len(want) {
			t.Fatalf("line %d: %s: %d outputs, want %d",
				line, fields[0], len(got), len(want))
		}
		for i, v := range got {
			if v != num(want[i]) {
				t.Fatalf("line %d: %s: out[%d] = %d, want %s",
					line, fields[0], i, v, want[i])
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
