package av1

const (
	invalidMv      = 0x80008000
	invalidRef2Cur = -32
)

type refmvsTemporalBlock struct {
	mv  mv
	ref uint8
}

type refmvsBlock struct {
	mv  [2]mv
	ref [2]int8
	bs  uint8
	mf  uint8
}

func splatMv(r []refmvsBlock, rows []int, rmv *refmvsBlock, bx4, bw4, bh4 int) {
	for y := range bh4 {
		off := rows[y] + bx4
		for x := range bw4 {
			r[off+x] = *rmv
		}
	}
}

func saveTmvs(rp []refmvsTemporalBlock, rpOff, stride int,
	r []refmvsBlock, rows []int, refSign *[7]uint8,
	colEnd8, rowEnd8, colStart8, rowStart8 int,
) {
	for y := rowStart8; y < rowEnd8; y++ {
		b := rows[(y&15)*2]

		for x := colStart8; x < colEnd8; {
			candB := &r[b+x*2+1]
			bw8 := (int(blockDimensions[candB.bs][0]) + 1) >> 1

			var tmv refmvsTemporalBlock
			switch {
			case candB.ref[1] > 0 && refSign[candB.ref[1]-1] != 0 &&
				abs32(int32(candB.mv[1].y))|abs32(int32(candB.mv[1].x)) < 4096:
				tmv = refmvsTemporalBlock{mv: candB.mv[1], ref: uint8(candB.ref[1])}
			case candB.ref[0] > 0 && refSign[candB.ref[0]-1] != 0 &&
				abs32(int32(candB.mv[0].y))|abs32(int32(candB.mv[0].x)) < 4096:
				tmv = refmvsTemporalBlock{mv: candB.mv[0], ref: uint8(candB.ref[0])}
			}

			for range bw8 {
				rp[rpOff+x] = tmv
				x++
			}
		}
		rpOff += stride
	}
}

type refmvsFrame struct {
	frmHdr         *frameHeader
	iw4, ih4       int
	iw8, ih8       int
	sbsz           int
	useRefFrameMvs int
	signBias       [7]uint8
	mfmvSign       [7]uint8
	pocdiff        [7]int8
	rpStride       int

	mfmvRef     [3]uint8
	mfmvRef2cur [3]int8
	mfmvRef2ref [3][7]uint8
	nMfmvs      int

	rpProj []refmvsTemporalBlock
	rpRef  [7][]refmvsTemporalBlock

	r  []refmvsBlock
	rp []refmvsTemporalBlock

	nTileThreads  int
	nFrameThreads int
	nBlocks       int
}

type refmvsTile struct {
	rf        *refmvsFrame
	r         []refmvsBlock
	rows      [32 + 5]int
	rpProj    []refmvsTemporalBlock
	rpProjOff int
	tileCol   struct{ start, end int }
	tileRow   struct{ start, end int }
}

type refmvsCandidate struct {
	mv     [2]mv
	weight int
}

func (c *refmvsCandidate) pairN() uint64 {
	return uint64(c.mv[0].n()) | uint64(c.mv[1].n())<<32
}

func fixIntMvPrecision(m *mv) {
	m.x = int16(uint16(m.x-(m.x>>15)+3) &^ 7)
	m.y = int16(uint16(m.y-(m.y>>15)+3) &^ 7)
}

func fixMvPrecision(hdr *frameHeader, m *mv) {
	if hdr.forceIntegerMv != 0 {
		fixIntMvPrecision(m)
	} else if hdr.hp == 0 {
		m.x = int16(uint16(m.x-(m.x>>15)) &^ 1)
		m.y = int16(uint16(m.y-(m.y>>15)) &^ 1)
	}
}

func getGmv2d(gmv *warpedMotionParams, bx4, by4, bw4, bh4 int, hdr *frameHeader) mv {
	switch gmv.typ {
	case wmTypeIdentity:
		return mv{}

	case wmTypeTranslation:
		res := mv{y: int16(gmv.matrix[0] >> 13), x: int16(gmv.matrix[1] >> 13)}
		if hdr.forceIntegerMv != 0 {
			fixIntMvPrecision(&res)
		}

		return res

	default:
		x := int32(bx4*4 + bw4*2 - 1)
		y := int32(by4*4 + bh4*2 - 1)
		xc := (gmv.matrix[2]-(1<<16))*x + gmv.matrix[3]*y + gmv.matrix[0]
		yc := (gmv.matrix[5]-(1<<16))*y + gmv.matrix[4]*x + gmv.matrix[1]
		notHp := int32(b2i(hdr.hp == 0))
		shift := 16 - (3 - notHp)
		round := int32(1<<shift) >> 1
		res := mv{
			y: int16(applySign(((abs32(yc)+round)>>shift)<<notHp, yc)),
			x: int16(applySign(((abs32(xc)+round)>>shift)<<notHp, xc)),
		}
		if hdr.forceIntegerMv != 0 {
			fixIntMvPrecision(&res)
		}

		return res
	}
}

func addSpatialCandidate(mvstack []refmvsCandidate, cnt *int, weight int,
	b *refmvsBlock, ref [2]int8, gmv *[2]mv, haveNewmvMatch, haveRefmvMatch *int,
) {
	if b.mv[0].n() == invalidMv {
		return
	}

	if ref[1] == -1 {
		for n := range 2 {
			if b.ref[n] != ref[0] {
				continue
			}
			candMv := b.mv[n]
			if b.mf&1 != 0 && gmv[0].n() != invalidMv {
				candMv = gmv[0]
			}

			*haveRefmvMatch = 1
			*haveNewmvMatch |= int(b.mf >> 1)

			last := *cnt
			for m := range last {
				if mvstack[m].mv[0].n() == candMv.n() {
					mvstack[m].weight += weight

					return
				}
			}
			if last < 8 {
				mvstack[last].mv[0] = candMv
				mvstack[last].weight = weight
				*cnt = last + 1
			}

			return
		}

		return
	}

	if b.ref != ref {
		return
	}

	var candMv [2]mv
	for i := range 2 {
		candMv[i] = b.mv[i]
		if b.mf&1 != 0 && gmv[i].n() != invalidMv {
			candMv[i] = gmv[i]
		}
	}

	*haveRefmvMatch = 1
	*haveNewmvMatch |= int(b.mf >> 1)

	last := *cnt
	cand := refmvsCandidate{mv: candMv}
	for n := range last {
		if mvstack[n].pairN() == cand.pairN() {
			mvstack[n].weight += weight

			return
		}
	}
	if last < 8 {
		mvstack[last].mv = candMv
		mvstack[last].weight = weight
		*cnt = last + 1
	}
}

func scanRow(mvstack []refmvsCandidate, cnt *int, ref [2]int8, gmv *[2]mv,
	r []refmvsBlock, bOff, bw4, w4, maxRows, step int,
	haveNewmvMatch, haveRefmvMatch *int,
) int {
	candB := &r[bOff]
	candBw4 := int(blockDimensions[candB.bs][0])
	l := max(step, min(bw4, candBw4))

	if bw4 <= candBw4 {
		weight := 2
		if bw4 != 1 {
			weight = max(2, min(2*maxRows, int(blockDimensions[candB.bs][1])))
		}
		addSpatialCandidate(mvstack, cnt, l*weight, candB, ref, gmv,
			haveNewmvMatch, haveRefmvMatch)

		return weight >> 1
	}

	for x := 0; ; {
		addSpatialCandidate(mvstack, cnt, l*2, candB, ref, gmv,
			haveNewmvMatch, haveRefmvMatch)
		x += l
		if x >= w4 {
			return 1
		}
		candB = &r[bOff+x]
		candBw4 = int(blockDimensions[candB.bs][0])
		l = max(step, candBw4)
	}
}

func scanCol(mvstack []refmvsCandidate, cnt *int, ref [2]int8, gmv *[2]mv,
	r []refmvsBlock, rows []int, bh4, h4, bx4, maxCols, step int,
	haveNewmvMatch, haveRefmvMatch *int,
) int {
	candB := &r[rows[0]+bx4]
	candBh4 := int(blockDimensions[candB.bs][1])
	l := max(step, min(bh4, candBh4))

	if bh4 <= candBh4 {
		weight := 2
		if bh4 != 1 {
			weight = max(2, min(2*maxCols, int(blockDimensions[candB.bs][0])))
		}
		addSpatialCandidate(mvstack, cnt, l*weight, candB, ref, gmv,
			haveNewmvMatch, haveRefmvMatch)

		return weight >> 1
	}

	for y := 0; ; {
		addSpatialCandidate(mvstack, cnt, l*2, candB, ref, gmv,
			haveNewmvMatch, haveRefmvMatch)
		y += l
		if y >= h4 {
			return 1
		}
		candB = &r[rows[y]+bx4]
		candBh4 = int(blockDimensions[candB.bs][1])
		l = max(step, candBh4)
	}
}

var mvProjDivMult = [32]uint16{
	0, 16384, 8192, 5461, 4096, 3276, 2730, 2340,
	2048, 1820, 1638, 1489, 1365, 1260, 1170, 1092,
	1024, 963, 910, 862, 819, 780, 744, 712,
	682, 655, 630, 606, 585, 564, 546, 528,
}

func mvProjection(m mv, num, den int) mv {
	frac := int32(num) * int32(mvProjDivMult[den])
	y := int32(m.y) * frac
	x := int32(m.x) * frac

	return mv{
		y: int16(clip((y+8192+(y>>31))>>14, -0x3fff, 0x3fff)),
		x: int16(clip((x+8192+(x>>31))>>14, -0x3fff, 0x3fff)),
	}
}

func addTemporalCandidate(rf *refmvsFrame, mvstack []refmvsCandidate, cnt *int,
	rb *refmvsTemporalBlock, ref [2]int8, globalmvCtx *int, tgmv *[2]mv,
) {
	if rb.mv.n() == invalidMv {
		return
	}

	m := mvProjection(rb.mv, int(rf.pocdiff[ref[0]-1]), int(rb.ref))
	fixMvPrecision(rf.frmHdr, &m)

	last := *cnt
	if ref[1] == -1 {
		if globalmvCtx != nil {
			*globalmvCtx = b2i(abs32(int32(m.x)-int32(tgmv[0].x))|
				abs32(int32(m.y)-int32(tgmv[0].y)) >= 16)
		}

		for n := range last {
			if mvstack[n].mv[0].n() == m.n() {
				mvstack[n].weight += 2

				return
			}
		}
		if last < 8 {
			mvstack[last].mv[0] = m
			mvstack[last].weight = 2
			*cnt = last + 1
		}

		return
	}

	m1 := mvProjection(rb.mv, int(rf.pocdiff[ref[1]-1]), int(rb.ref))
	fixMvPrecision(rf.frmHdr, &m1)
	cand := refmvsCandidate{mv: [2]mv{m, m1}}

	for n := range last {
		if mvstack[n].pairN() == cand.pairN() {
			mvstack[n].weight += 2

			return
		}
	}
	if last < 8 {
		mvstack[last].mv = cand.mv
		mvstack[last].weight = 2
		*cnt = last + 1
	}
}

func addCompoundExtendedCandidate(same []refmvsCandidate, sameCount *[4]int,
	candB *refmvsBlock, sign0, sign1 int, ref [2]int8, signBias *[7]uint8,
) {
	diff := same[2:]

	for n := range 2 {
		candRef := candB.ref[n]
		if candRef <= 0 {
			break
		}

		candMv := candB.mv[n]
		switch candRef {
		case ref[0]:
			if sameCount[0] < 2 {
				same[sameCount[0]].mv[0] = candMv
				sameCount[0]++
			}
			if sameCount[3] < 2 {
				if sign1^int(signBias[candRef-1]) != 0 {
					candMv.y, candMv.x = -candMv.y, -candMv.x
				}
				diff[sameCount[3]].mv[1] = candMv
				sameCount[3]++
			}
		case ref[1]:
			if sameCount[1] < 2 {
				same[sameCount[1]].mv[1] = candMv
				sameCount[1]++
			}
			if sameCount[2] < 2 {
				if sign0^int(signBias[candRef-1]) != 0 {
					candMv.y, candMv.x = -candMv.y, -candMv.x
				}
				diff[sameCount[2]].mv[0] = candMv
				sameCount[2]++
			}
		default:
			iCandMv := mv{y: -candMv.y, x: -candMv.x}

			if sameCount[2] < 2 {
				v := candMv
				if sign0^int(signBias[candRef-1]) != 0 {
					v = iCandMv
				}
				diff[sameCount[2]].mv[0] = v
				sameCount[2]++
			}
			if sameCount[3] < 2 {
				v := candMv
				if sign1^int(signBias[candRef-1]) != 0 {
					v = iCandMv
				}
				diff[sameCount[3]].mv[1] = v
				sameCount[3]++
			}
		}
	}
}

func addSingleExtendedCandidate(mvstack []refmvsCandidate, cnt *int,
	candB *refmvsBlock, sign int, signBias *[7]uint8,
) {
	for n := range 2 {
		candRef := candB.ref[n]
		if candRef <= 0 {
			break
		}

		candMv := candB.mv[n]
		if sign^int(signBias[candRef-1]) != 0 {
			candMv.y, candMv.x = -candMv.y, -candMv.x
		}

		last := *cnt
		m := 0
		for ; m < last; m++ {
			if candMv.n() == mvstack[m].mv[0].n() {
				break
			}
		}
		if m == last {
			mvstack[m].mv[0] = candMv
			mvstack[m].weight = 2
			*cnt = last + 1
		}
	}
}

func refmvsFind(rt *refmvsTile, mvstack []refmvsCandidate, cnt, ctx *int,
	ref [2]int8, bs, edgeFlags, by4, bx4 int,
) {
	rf := rt.rf
	bDim := &blockDimensions[bs]
	bw4 := int(bDim[0])
	w4 := min(min(bw4, 16), rt.tileCol.end-bx4)
	bh4 := int(bDim[1])
	h4 := min(min(bh4, 16), rt.tileRow.end-by4)
	var gmv, tgmv [2]mv

	*cnt = 0

	if ref[0] > 0 {
		tgmv[0] = getGmv2d(&rf.frmHdr.gmv[ref[0]-1], bx4, by4, bw4, bh4, rf.frmHdr)
		gmv[0] = mv{y: -32768, x: -32768}
		if rf.frmHdr.gmv[ref[0]-1].typ > wmTypeTranslation {
			gmv[0] = tgmv[0]
		}
	} else {
		tgmv[0] = mv{}
		gmv[0] = mv{y: -32768, x: -32768}
	}
	if ref[1] > 0 {
		tgmv[1] = getGmv2d(&rf.frmHdr.gmv[ref[1]-1], bx4, by4, bw4, bh4, rf.frmHdr)
		gmv[1] = mv{y: -32768, x: -32768}
		if rf.frmHdr.gmv[ref[1]-1].typ > wmTypeTranslation {
			gmv[1] = tgmv[1]
		}
	}

	haveNewmv, haveColMvs, haveRowMvs := 0, 0, 0
	maxRows, nRows := 0, -1
	bTop := 0
	if by4 > rt.tileRow.start {
		maxRows = min((by4-rt.tileRow.start+1)>>1, 2+b2i(bh4 > 1))
		bTop = rt.rows[(by4&31)-1+5] + bx4
		step := 1
		if bw4 >= 16 {
			step = 4
		}
		nRows = scanRow(mvstack, cnt, ref, &gmv, rt.r, bTop,
			bw4, w4, maxRows, step, &haveNewmv, &haveRowMvs)
	}

	maxCols, nCols := 0, -1
	bLeft := 0
	if bx4 > rt.tileCol.start {
		maxCols = min((bx4-rt.tileCol.start+1)>>1, 2+b2i(bw4 > 1))
		bLeft = (by4 & 31) + 5
		step := 1
		if bh4 >= 16 {
			step = 4
		}
		nCols = scanCol(mvstack, cnt, ref, &gmv, rt.r, rt.rows[bLeft:],
			bh4, h4, bx4-1, maxCols, step, &haveNewmv, &haveColMvs)
	}

	if nRows != -1 && edgeFlags&edgeI444TopHasRight != 0 &&
		max(bw4, bh4) <= 16 && bw4+bx4 < rt.tileCol.end {
		addSpatialCandidate(mvstack, cnt, 4, &rt.r[bTop+bw4], ref, &gmv,
			&haveNewmv, &haveRowMvs)
	}

	nearestMatch := haveColMvs + haveRowMvs
	nearestCnt := *cnt
	for n := range nearestCnt {
		mvstack[n].weight += 640
	}

	globalmvCtx := int(rf.frmHdr.useRefFrameMvs)
	if rf.useRefFrameMvs != 0 {
		stride := rf.rpStride
		by8, bx8 := by4>>1, bx4>>1
		rbi := rt.rpProjOff + (by8&15)*stride + bx8
		rb := rbi
		stepH, stepV := 1, 1
		if bw4 >= 16 {
			stepH = 2
		}
		if bh4 >= 16 {
			stepV = 2
		}
		w8, h8 := min((w4+1)>>1, 8), min((h4+1)>>1, 8)
		for y := 0; y < h8; y += stepV {
			for x := 0; x < w8; x += stepH {
				var g *int
				if x|y == 0 {
					g = &globalmvCtx
				}
				addTemporalCandidate(rf, mvstack, cnt, &rt.rpProj[rb+x], ref, g, &tgmv)
			}
			rb += stride * stepV
		}
		if min(bw4, bh4) >= 2 && max(bw4, bh4) < 16 {
			bh8, bw8 := bh4>>1, bw4>>1
			rb = rbi + bh8*stride
			hasBottom := by8+bh8 < min(rt.tileRow.end>>1, (by8&^7)+8)
			if hasBottom && bx8-1 >= max(rt.tileCol.start>>1, bx8&^7) {
				addTemporalCandidate(rf, mvstack, cnt, &rt.rpProj[rb-1], ref, nil, nil)
			}
			if bx8+bw8 < min(rt.tileCol.end>>1, (bx8&^7)+8) {
				if hasBottom {
					addTemporalCandidate(rf, mvstack, cnt, &rt.rpProj[rb+bw8], ref, nil, nil)
				}
				if by8+bh8-1 < min(rt.tileRow.end>>1, (by8&^7)+8) {
					addTemporalCandidate(rf, mvstack, cnt, &rt.rpProj[rb+bw8-stride],
						ref, nil, nil)
				}
			}
		}
	}

	var haveDummyNewmvMatch int
	if nRows != -1 && nCols != -1 {
		addSpatialCandidate(mvstack, cnt, 4, &rt.r[bTop-1], ref, &gmv,
			&haveDummyNewmvMatch, &haveRowMvs)
	}

	for n := 2; n <= 3; n++ {
		if nRows != -1 && n > nRows && n <= maxRows {
			step := 2
			if bw4 >= 16 {
				step = 4
			}
			off := rt.rows[(((by4&31)-2*n+1)|1)+5] + (bx4 | 1)
			nRows += scanRow(mvstack, cnt, ref, &gmv, rt.r, off,
				bw4, w4, 1+maxRows-n, step, &haveDummyNewmvMatch, &haveRowMvs)
		}

		if nCols != -1 && n > nCols && n <= maxCols {
			step := 2
			if bh4 >= 16 {
				step = 4
			}
			nCols += scanCol(mvstack, cnt, ref, &gmv, rt.r, rt.rows[((by4&31)|1)+5:],
				bh4, h4, (bx4-n*2+1)|1, 1+maxCols-n, step,
				&haveDummyNewmvMatch, &haveColMvs)
		}
	}

	refMatchCount := haveColMvs + haveRowMvs

	var refmvCtx, newmvCtx int
	switch nearestMatch {
	case 0:
		refmvCtx = min(2, refMatchCount)
		newmvCtx = b2i(refMatchCount > 0)
	case 1:
		refmvCtx = min(refMatchCount*3, 4)
		newmvCtx = 3 - haveNewmv
	default:
		refmvCtx = 5
		newmvCtx = 5 - haveNewmv
	}

	for l := nearestCnt; l != 0; {
		last := 0
		for n := 1; n < l; n++ {
			if mvstack[n-1].weight < mvstack[n].weight {
				mvstack[n-1], mvstack[n] = mvstack[n], mvstack[n-1]
				last = n
			}
		}
		l = last
	}
	for l := *cnt; l > nearestCnt; {
		last := nearestCnt
		for n := nearestCnt + 1; n < l; n++ {
			if mvstack[n-1].weight < mvstack[n].weight {
				mvstack[n-1], mvstack[n] = mvstack[n], mvstack[n-1]
				last = n
			}
		}
		l = last
	}

	if ref[1] > 0 {
		if *cnt < 2 {
			sign0 := int(rf.signBias[ref[0]-1])
			sign1 := int(rf.signBias[ref[1]-1])
			sz4 := min(w4, h4)
			same := mvstack[*cnt:]
			var sameCount [4]int

			if nRows != -1 {
				for x := 0; x < sz4; {
					candB := &rt.r[bTop+x]
					addCompoundExtendedCandidate(same, &sameCount, candB,
						sign0, sign1, ref, &rf.signBias)
					x += int(blockDimensions[candB.bs][0])
				}
			}

			if nCols != -1 {
				for y := 0; y < sz4; {
					candB := &rt.r[rt.rows[bLeft+y]+bx4-1]
					addCompoundExtendedCandidate(same, &sameCount, candB,
						sign0, sign1, ref, &rf.signBias)
					y += int(blockDimensions[candB.bs][1])
				}
			}

			diff := same[2:]

			for n := range 2 {
				m := sameCount[n]
				if m >= 2 {
					continue
				}

				l := sameCount[2+n]
				if l != 0 {
					same[m].mv[n] = diff[0].mv[n]
					m++
					if m == 2 {
						continue
					}
					if l == 2 {
						same[1].mv[n] = diff[1].mv[n]

						continue
					}
				}
				for ; m < 2; m++ {
					same[m].mv[n] = tgmv[n]
				}
			}

			n := *cnt
			if n == 1 && mvstack[0].pairN() == same[0].pairN() {
				mvstack[1].mv = mvstack[2].mv
			}
			for ; n < 2; n++ {
				mvstack[n].weight = 2
			}
			*cnt = 2
		}

		left := int32(-(bx4 + bw4 + 4) * 4 * 8)
		right := int32((rf.iw4 - bx4 + 4) * 4 * 8)
		top := int32(-(by4 + bh4 + 4) * 4 * 8)
		bottom := int32((rf.ih4 - by4 + 4) * 4 * 8)

		for n := range *cnt {
			for i := range 2 {
				mvstack[n].mv[i].x = int16(clip(int32(mvstack[n].mv[i].x), left, right))
				mvstack[n].mv[i].y = int16(clip(int32(mvstack[n].mv[i].y), top, bottom))
			}
		}

		switch refmvCtx >> 1 {
		case 0:
			*ctx = min(newmvCtx, 1)
		case 1:
			*ctx = 1 + min(newmvCtx, 3)
		default:
			*ctx = clip(3+newmvCtx, 4, 7)
		}

		return
	}

	if *cnt < 2 && ref[0] > 0 {
		sign := int(rf.signBias[ref[0]-1])
		sz4 := min(w4, h4)

		if nRows != -1 {
			for x := 0; x < sz4 && *cnt < 2; {
				candB := &rt.r[bTop+x]
				addSingleExtendedCandidate(mvstack, cnt, candB, sign, &rf.signBias)
				x += int(blockDimensions[candB.bs][0])
			}
		}

		if nCols != -1 {
			for y := 0; y < sz4 && *cnt < 2; {
				candB := &rt.r[rt.rows[bLeft+y]+bx4-1]
				addSingleExtendedCandidate(mvstack, cnt, candB, sign, &rf.signBias)
				y += int(blockDimensions[candB.bs][1])
			}
		}
	}

	if *cnt != 0 {
		left := int32(-(bx4 + bw4 + 4) * 4 * 8)
		right := int32((rf.iw4 - bx4 + 4) * 4 * 8)
		top := int32(-(by4 + bh4 + 4) * 4 * 8)
		bottom := int32((rf.ih4 - by4 + 4) * 4 * 8)

		for n := range *cnt {
			mvstack[n].mv[0].x = int16(clip(int32(mvstack[n].mv[0].x), left, right))
			mvstack[n].mv[0].y = int16(clip(int32(mvstack[n].mv[0].y), top, bottom))
		}
	}

	for n := *cnt; n < 2; n++ {
		mvstack[n].mv[0] = tgmv[0]
	}

	*ctx = refmvCtx<<4 | globalmvCtx<<3 | newmvCtx
}

func loadTmvs(rf *refmvsFrame, tileRowIdx, colStart8, colEnd8, rowStart8, rowEnd8 int) {
	if rf.nTileThreads == 1 {
		tileRowIdx = 0
	}
	rowEnd8 = min(rowEnd8, rf.ih8)
	colStart8i := max(colStart8-8, 0)
	colEnd8i := min(colEnd8+8, rf.iw8)

	stride := rf.rpStride
	rpProj := 16*stride*tileRowIdx + (rowStart8&15)*stride
	for y := rowStart8; y < rowEnd8; y++ {
		for x := colStart8; x < colEnd8; x++ {
			rf.rpProj[rpProj+x].mv = mv{y: -32768, x: -32768}
		}
		rpProj += stride
	}

	rpProj = 16 * stride * tileRowIdx
	for n := range rf.nMfmvs {
		ref2cur := int(rf.mfmvRef2cur[n])
		if ref2cur == invalidRef2Cur {
			continue
		}

		ref := int(rf.mfmvRef[n])
		refSign := ref - 4
		r := rf.rpRef[ref]
		rOff := rowStart8 * stride
		for y := rowStart8; y < rowEnd8; y++ {
			ySbAlign := y &^ 7
			yProjStart := max(ySbAlign, rowStart8)
			yProjEnd := min(ySbAlign+8, rowEnd8)
			for x := colStart8i; x < colEnd8i; x++ {
				rb := &r[rOff+x]
				bRef := int(rb.ref)
				if bRef == 0 {
					continue
				}
				ref2ref := int(rf.mfmvRef2ref[n][bRef-1])
				if ref2ref == 0 {
					continue
				}
				bMv := rb.mv
				offset := mvProjection(bMv, ref2cur, ref2ref)
				posX := x + int(applySign(abs32(int32(offset.x))>>6,
					int32(offset.x)^int32(refSign)))
				posY := y + int(applySign(abs32(int32(offset.y))>>6,
					int32(offset.y)^int32(refSign)))

				if posY >= yProjStart && posY < yProjEnd {
					pos := (posY & 15) * stride
					for {
						xSbAlign := x &^ 7
						if posX >= max(xSbAlign-8, colStart8) &&
							posX < min(xSbAlign+16, colEnd8) {
							rf.rpProj[rpProj+pos+posX].mv = rb.mv
							rf.rpProj[rpProj+pos+posX].ref = uint8(ref2ref)
						}
						x++
						if x >= colEnd8i {
							break
						}
						rb = &r[rOff+x]
						if int(rb.ref) != bRef || rb.mv.n() != bMv.n() {
							break
						}
						posX++
					}
				} else {
					for {
						x++
						if x >= colEnd8i {
							break
						}
						rb = &r[rOff+x]
						if int(rb.ref) != bRef || rb.mv.n() != bMv.n() {
							break
						}
					}
				}
				x--
			}
			rOff += stride
		}
	}
}

func refmvsTileSbrowInit(rt *refmvsTile, rf *refmvsFrame,
	tileColStart4, tileColEnd4, tileRowStart4, tileRowEnd4, sby, tileRowIdx int,
) {
	if rf.nTileThreads == 1 {
		tileRowIdx = 0
	}
	rt.rpProj = rf.rpProj
	rt.rpProjOff = 16 * rf.rpStride * tileRowIdx

	rStride := rf.rpStride * 2
	r := 35 * rStride * tileRowIdx
	sbsz := rf.sbsz
	off := (sbsz * sby) & 16

	for i := range sbsz {
		rt.rows[off+5+i] = r
		r += rStride
	}
	rt.rows[off+0] = r
	r += rStride
	rt.rows[off+1] = -1
	rt.rows[off+2] = r
	r += rStride
	rt.rows[off+3] = -1
	rt.rows[off+4] = r

	if sby&1 != 0 {
		rt.rows[off+0], rt.rows[off+sbsz+0] = rt.rows[off+sbsz+0], rt.rows[off+0]
		rt.rows[off+2], rt.rows[off+sbsz+2] = rt.rows[off+sbsz+2], rt.rows[off+2]
		rt.rows[off+4], rt.rows[off+sbsz+4] = rt.rows[off+sbsz+4], rt.rows[off+4]
	}

	rt.rf = rf
	rt.r = rf.r
	rt.tileRow.start = tileRowStart4
	rt.tileRow.end = min(tileRowEnd4, rf.ih4)
	rt.tileCol.start = tileColStart4
	rt.tileCol.end = min(tileColEnd4, rf.iw4)
}

func refmvsInitFrame(rf *refmvsFrame, seqHdr *sequenceHeader, frmHdr *frameHeader,
	refPoc *[7]uint8, rp []refmvsTemporalBlock, refRefPoc *[7][7]uint8,
	rpRef *[7][]refmvsTemporalBlock, nTileThreads, nFrameThreads int,
) {
	rpStride := ((frmHdr.width[0] + 127) &^ 127) >> 3
	nTileRows := 1
	if nTileThreads > 1 {
		nTileRows = int(frmHdr.tiling.rows)
	}
	nBlocks := rpStride * nTileRows

	rf.sbsz = 16 << seqHdr.sb128
	rf.frmHdr = frmHdr
	rf.iw8 = (frmHdr.width[0] + 7) >> 3
	rf.ih8 = (frmHdr.height + 7) >> 3
	rf.iw4 = rf.iw8 << 1
	rf.ih4 = rf.ih8 << 1
	rf.rp = rp
	rf.rpStride = rpStride
	rf.nTileThreads = nTileThreads
	rf.nFrameThreads = nFrameThreads

	if nBlocks != rf.nBlocks {
		rf.r = make([]refmvsBlock, 35*2*nBlocks*(1+b2i(nFrameThreads > 1)))
		rf.rpProj = make([]refmvsTemporalBlock, 16*nBlocks)
		rf.nBlocks = nBlocks
	}

	nBits := int(seqHdr.orderHintNBits)
	poc := int(frmHdr.frameOffset)
	for i := range 7 {
		pocDiff := getPocDiff(nBits, int(refPoc[i]), poc)
		rf.signBias[i] = uint8(b2i(pocDiff > 0))
		rf.mfmvSign[i] = uint8(b2i(pocDiff < 0))
		rf.pocdiff[i] = int8(clip(getPocDiff(nBits, poc, int(refPoc[i])), -31, 31))
	}

	rf.nMfmvs = 0
	rf.rpRef = *rpRef

	if frmHdr.useRefFrameMvs != 0 && nBits != 0 {
		total := 2
		if rpRef[0] != nil && refRefPoc[0][6] != refPoc[3] {
			rf.mfmvRef[rf.nMfmvs] = 0
			rf.nMfmvs++
			total = 3
		}
		if rpRef[4] != nil && getPocDiff(nBits, int(refPoc[4]), poc) > 0 {
			rf.mfmvRef[rf.nMfmvs] = 4
			rf.nMfmvs++
		}
		if rpRef[5] != nil && getPocDiff(nBits, int(refPoc[5]), poc) > 0 {
			rf.mfmvRef[rf.nMfmvs] = 5
			rf.nMfmvs++
		}
		if rf.nMfmvs < total && rpRef[6] != nil &&
			getPocDiff(nBits, int(refPoc[6]), poc) > 0 {
			rf.mfmvRef[rf.nMfmvs] = 6
			rf.nMfmvs++
		}
		if rf.nMfmvs < total && rpRef[1] != nil {
			rf.mfmvRef[rf.nMfmvs] = 1
			rf.nMfmvs++
		}

		for n := range rf.nMfmvs {
			rpoc := int(refPoc[rf.mfmvRef[n]])
			diff1 := getPocDiff(nBits, rpoc, poc)
			if abs32(int32(diff1)) > 31 {
				rf.mfmvRef2cur[n] = invalidRef2Cur

				continue
			}

			if rf.mfmvRef[n] < 4 {
				rf.mfmvRef2cur[n] = int8(-diff1)
			} else {
				rf.mfmvRef2cur[n] = int8(diff1)
			}
			for m := range 7 {
				rrpoc := int(refRefPoc[rf.mfmvRef[n]][m])
				diff2 := getPocDiff(nBits, rpoc, rrpoc)
				if uint32(diff2) > 31 {
					rf.mfmvRef2ref[n][m] = 0
				} else {
					rf.mfmvRef2ref[n][m] = uint8(diff2)
				}
			}
		}
	}

	rf.useRefFrameMvs = b2i(rf.nMfmvs > 0)
}
