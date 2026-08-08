package av1

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// Picture is a decoded frame.
type Picture struct {
	Width, Height int
	Layout        int
	BitDepth      int
	Data          [3][]byte
	Stride        [3]int
}

const picPad = 128

func decodeFrame[P pixel, C coef](d *Decoder, inCdf, outCdf *cdfContext, tiles []tileGroup) (*Picture, error) {
	seqHdr, frameHdr := d.seqHdr, d.frameHdr

	f := &frameContext[P, C]{
		seqHdr:   seqHdr,
		frameHdr: frameHdr,
		layout:   seqHdr.layout,
		bpc:      8 + 2*int(seqHdr.hbd),

		reconIntra: reconBIntra[P, C],
		reconInter: reconBInter[P, C],
	}
	f.bitdepthMax = int32(1<<f.bpc - 1)
	f.ssHor = b2i(f.layout != pixelLayoutI444)
	f.ssVer = b2i(f.layout == pixelLayoutI420)

	f.w4 = (frameHdr.width[0] + 3) >> 2
	f.h4 = (frameHdr.height + 3) >> 2
	f.bw = ((frameHdr.width[0] + 7) >> 3) << 1
	f.bh = ((frameHdr.height + 7) >> 3) << 1
	f.sb128w = (f.bw + 31) >> 5
	f.sb128h = (f.bh + 31) >> 5
	f.sbShift = 4 + int(seqHdr.sb128)
	f.sbStep = 16 << seqHdr.sb128
	f.sbh = (f.bh + f.sbStep - 1) >> f.sbShift
	sbh := f.sbh
	f.b4Stride = (f.bw + 31) &^ 31

	f.lfEnabled = frameHdr.loopfilter.levelY[0] != 0 || frameHdr.loopfilter.levelY[1] != 0

	f.resize = frameHdr.width[0] != frameHdr.width[1]
	f.srWidth = frameHdr.width[1]

	f.inCdf = inCdf

	initQuantTables(seqHdr, frameHdr, int(frameHdr.quant.yac), &f.dq)

	if frameHdr.quant.qm != 0 {
		for i := range nRectTxSizes {
			f.qm[i][0] = qmTbl[frameHdr.quant.qmY][0][i]
			f.qm[i][1] = qmTbl[frameHdr.quant.qmU][1][i]
			f.qm[i][2] = qmTbl[frameHdr.quant.qmV][1][i]
		}
	}

	f.nThreads = d.Threads
	if f.nThreads <= 0 {
		f.nThreads = runtime.GOMAXPROCS(0)
	}
	f.nThreads = min(f.nThreads, int(frameHdr.tiling.cols)*int(frameHdr.tiling.rows))

	if err := f.allocPlanes(); err != nil {
		return nil, err
	}
	if err := f.setupRefs(d); err != nil {
		return nil, err
	}
	f.setupSegMap(d)
	f.setupRefmvs(d, seqHdr)
	f.setupJntWeights()

	for pl := range 3 {
		f.ipredEdge[pl] = make([]P, f.sb128w*128*(sbh+1))
	}

	nTs := int(frameHdr.tiling.cols) * int(frameHdr.tiling.rows)
	tileData, err := splitTiles(frameHdr, tiles, nTs)
	if err != nil {
		return nil, err
	}

	f.a = make([]blockContext, f.sb128w*int(frameHdr.tiling.rows))
	for i := range f.a {
		resetContext(&f.a[i], isKeyOrIntra(frameHdr))
	}
	a := f.a

	sb128h := f.sb128h
	f.lflvl = make([]av1Filter, f.sb128w*sb128h)
	f.lf.level = make([]uint8, f.sb128w*sb128h*32*32*4+3)
	calcEIH(&f.lf.limLut, int(frameHdr.loopfilter.sharpness))
	calcLfValues(&f.lf.lvl, frameHdr, &[4]int8{})

	alignH := (f.bh + 31) &^ 31
	f.lf.txLpfRightEdge[0] = make([]uint8, alignH*int(frameHdr.tiling.cols))
	f.lf.txLpfRightEdge[1] = make([]uint8, (alignH>>f.ssVer)*int(frameHdr.tiling.cols))

	f.lf.restorePlanes = b2i(frameHdr.restoration.typ[0] != restorationNone) |
		b2i(frameHdr.restoration.typ[1] != restorationNone)<<1 |
		b2i(frameHdr.restoration.typ[2] != restorationNone)<<2
	f.srSb128w = (f.srWidth + 127) >> 7
	f.lrMask = make([]av1Restoration, f.srSb128w*sb128h)

	if seqHdr.cdef != 0 || f.lf.restorePlanes != 0 {
		const lrNumLines = 12
		f.lf.lrLineBuf = make([]P, 64+f.srStride[0]*lrNumLines+f.srStride[1]*lrNumLines*2)
		off := 64
		f.lf.lrLpfLine[0] = off
		off += f.srStride[0] * lrNumLines
		f.lf.lrLpfLine[1] = off
		f.lf.lrLpfLine[2] = off + f.srStride[1]*lrNumLines
	}

	if seqHdr.cdef != 0 {
		yStride, uvStride := f.stride[0], f.stride[1]
		f.lf.cdefLineBuf = make([]P, 32+yStride*sbh*4+uvStride*sbh*8)
		off := 32
		f.lf.cdefLine[0][0] = off
		f.lf.cdefLine[1][0] = off + 2*yStride
		off += yStride * sbh * 4
		f.lf.cdefLine[0][1] = off
		f.lf.cdefLine[0][2] = off + 2*uvStride
		f.lf.cdefLine[1][1] = off + 4*uvStride
		f.lf.cdefLine[1][2] = off + 6*uvStride
	}

	keyframe := isKeyOrIntra(frameHdr)

	nCols := int(frameHdr.tiling.cols)
	nRows := int(frameHdr.tiling.rows)
	tstates := make([]*tileState, nRows*nCols)
	for i := range tstates {
		ts := new(tileState)
		setupTile(ts, f, tileData[i], i/nCols, i%nCols)
		tstates[i] = ts
	}

	inter := isInterOrSwitch(frameHdr) || frameHdr.allowIntrabc != 0

	decodeTile := func(t *taskContext[P, C], tileRow, tileCol int) error {
		ts := tstates[tileRow*nCols+tileCol]
		t.ts = ts
		colSb128Start := int(frameHdr.tiling.colStartSb[tileCol]) >> b2i(seqHdr.sb128 == 0)
		sbhEnd := min(int(frameHdr.tiling.rowStartSb[tileRow+1]), f.sbh)

		for sby := int(frameHdr.tiling.rowStartSb[tileRow]); sby < sbhEnd; sby++ {
			by := sby << f.sbShift
			byEnd := (by + f.sbStep) >> 1
			t.by = by

			if inter {
				refmvsTileSbrowInit(&t.rt, &f.rf, ts.tiling.colStart,
					ts.tiling.colEnd, ts.tiling.rowStart, ts.tiling.rowEnd,
					sby, tileRow)
			}
			if frameHdr.useRefFrameMvs != 0 {
				loadTmvs(&f.rf, tileRow, ts.tiling.colStart>>1,
					ts.tiling.colEnd>>1, by>>1, byEnd)
			}
			resetContext(&t.l, keyframe)
			clear(t.palSzUv[1][:])

			sb128y := by >> 5
			aIdx := colSb128Start + tileRow*f.sb128w
			for bx := ts.tiling.colStart; bx < ts.tiling.colEnd; bx += f.sbStep {
				t.bx = bx
				t.a = &a[aIdx]

				lfIdx := sb128y*f.sb128w + (bx >> 5)
				t.lfMask = &f.lflvl[lfIdx]
				if seqHdr.sb128 != 0 {
					t.curSbCdefIdx = t.lfMask.cdefIdx[:]
					for i := range 4 {
						t.curSbCdefIdx[i] = -1
					}
				} else {
					off := (bx&16)>>4 + (by&16)>>3
					t.curSbCdefIdx = t.lfMask.cdefIdx[off:]
					t.curSbCdefIdx[0] = -1
				}

				readLrUnits(t)

				rootBl := bl64x64
				treeIdx := 1
				if seqHdr.sb128 != 0 {
					rootBl = bl128x128
					treeIdx = 0
				}
				if err := t.decodeSb(rootBl, intraEdgeTree[treeIdx]); err != nil {
					return err
				}

				if bx&16 != 0 || seqHdr.sb128 != 0 {
					aIdx++
				}
			}

			alignH := (f.bh + 31) &^ 31
			n := min(f.sbStep, f.bh-by)
			copy(f.lf.txLpfRightEdge[0][alignH*tileCol+by:][:n],
				t.l.txLpfY[by&16:])
			copy(f.lf.txLpfRightEdge[1][(alignH>>f.ssVer)*tileCol+(by>>f.ssVer):][:n>>f.ssVer],
				t.l.txLpfUv[(by&16)>>f.ssVer:])

			backupIpredEdge(t)

			if isInterOrSwitch(frameHdr) {
				f.saveTmvsSbrow(&t.rt, ts.tiling.colStart>>1,
					ts.tiling.colEnd>>1, by>>1, byEnd)
			}
		}

		return nil
	}

	if err := f.runTiles(decodeTile, nRows, nCols); err != nil {
		return nil, err
	}

	if frameHdr.refreshContext != 0 {
		cdfThreadUpdate(frameHdr, outCdf, &tstates[frameHdr.tiling.update].cdf)
	}

	pt := f.newTaskContext()
	for sby := range (f.bh + f.sbStep - 1) / f.sbStep {
		if f.lfEnabled {
			startOfTileRow := 0
			for tr := range int(frameHdr.tiling.rows) {
				if sby == int(frameHdr.tiling.rowStartSb[tr]) {
					startOfTileRow = tr
				}
			}
			loopfilterSbrowCols(f, sby, startOfTileRow)
			loopfilterSbrowRows(f, sby)
		}
		if seqHdr.cdef != 0 || f.lf.restorePlanes != 0 {
			y := sby * f.sbStep * 4
			p := [3]int{
				y * f.stride[0],
				y * f.stride[1] >> f.ssVer,
				y * f.stride[1] >> f.ssVer,
			}
			copyLpf(f, &p, sby)
		}
		if seqHdr.cdef != 0 {
			filterSbrowCdef(pt, sby)
		}
		if f.resize {
			filterSbrowResize(f, sby)
		}
		if f.lf.restorePlanes != 0 {
			lrSbrow(f, sby)
		}
	}

	out := f.picture()
	if hasFilmGrain(frameHdr) {
		out = f.grainPicture()
	}
	f.saveRefs(d, out)

	return out, nil
}

func (f *frameContext[P, C]) saveRefs(d *Decoder, out *Picture) {
	hdr := f.frameHdr
	pic := f.refPicture()

	for i := range numRefFrames {
		if hdr.refreshFrameFlags&(1<<i) == 0 {
			continue
		}
		ref := &d.refs[i]
		ref.out = out
		ref.pic = pic
		ref.codedW = hdr.width[0]
		ref.segmap = f.curSegMap
		ref.refmvs = nil
		if hdr.allowIntrabc == 0 {
			ref.refmvs = f.mvs
		}
		ref.refpoc = f.refPoc
	}
}

func (f *frameContext[P, C]) allocPicture(w int, planes *[3][]P, stride *[3]int) {
	h := f.frameHdr.height

	for pl := range 3 {
		if pl != 0 && f.layout == pixelLayoutI400 {
			break
		}
		pw, ph := w, h
		if pl != 0 {
			pw = (w + f.ssHor) >> f.ssHor
			ph = (h + f.ssVer) >> f.ssVer
		}
		s := (pw + 2*picPad + 63) &^ 63
		buf := make([]P, s*(ph+2*picPad))
		if pl == 0 {
			stride[0] = s
		} else {
			stride[1] = s
		}
		planes[pl] = buf[picPad*s+picPad:]
	}
}

func (f *frameContext[P, C]) allocPlanes() error {
	f.allocPicture(f.frameHdr.width[0], &f.cur, &f.stride)

	if !f.resize {
		f.srCur, f.srStride = f.cur, f.stride

		return nil
	}

	f.allocPicture(f.srWidth, &f.srCur, &f.srStride)

	f.resizeStep[0] = scaleFac(f.frameHdr.width[0], f.srWidth)
	inCw := (f.frameHdr.width[0] + f.ssHor) >> f.ssHor
	outCw := (f.srWidth + f.ssHor) >> f.ssHor
	f.resizeStep[1] = scaleFac(inCw, outCw)
	f.resizeStart[0] = getUpscaleX0(f.frameHdr.width[0], f.srWidth, f.resizeStep[0])
	f.resizeStart[1] = getUpscaleX0(inCw, outCw, f.resizeStep[1])

	return nil
}

func (f *frameContext[P, C]) picture() *Picture {
	p := &Picture{
		Width:    f.srWidth,
		Height:   f.frameHdr.height,
		Layout:   f.layout,
		BitDepth: f.bpc,
	}

	for pl := range 3 {
		if f.srCur[pl] == nil {
			continue
		}
		stride := f.srStride[0]
		ph := f.frameHdr.height
		if pl != 0 {
			stride = f.srStride[1]
			ph = (ph + f.ssVer) >> f.ssVer
		}
		var z P
		p.Stride[pl] = stride * int(unsafe.Sizeof(z))
		p.Data[pl] = pixelsToBytes(f.srCur[pl][:stride*ph])
	}

	return p
}

func setupTile[P pixel, C coef](ts *tileState, f *frameContext[P, C],
	data []byte, tileRow, tileCol int,
) {
	hdr := f.frameHdr
	colSbStart := int(hdr.tiling.colStartSb[tileCol])
	colSbEnd := int(hdr.tiling.colStartSb[tileCol+1])
	rowSbStart := int(hdr.tiling.rowStartSb[tileRow])
	rowSbEnd := int(hdr.tiling.rowStartSb[tileRow+1])

	ts.cdf = *f.inCdf
	ts.lastQidx = int(hdr.quant.yac)
	ts.lastDeltaLf = [4]int8{}

	ts.msac.init(data, hdr.disableCdfUpdate != 0)

	ts.tiling.row = tileRow
	ts.tiling.col = tileCol
	ts.tiling.colStart = colSbStart << f.sbShift
	ts.tiling.colEnd = min(colSbEnd<<f.sbShift, f.bw)
	ts.tiling.rowStart = rowSbStart << f.sbShift
	ts.tiling.rowEnd = min(rowSbEnd<<f.sbShift, f.bh)

	ts.dq = &f.dq
	ts.lflvl = &f.lf.lvl

	var sbIdx, unitIdx int
	if f.resize {
		sbIdx = (ts.tiling.rowStart >> 5) * f.srSb128w
		unitIdx = (ts.tiling.rowStart & 16) >> 3
	} else {
		sbIdx = (ts.tiling.rowStart>>5)*f.sb128w + (colSbStart >> b2i(f.seqHdr.sb128 == 0))
		unitIdx = (ts.tiling.rowStart&16)>>3 + (ts.tiling.colStart&16)>>4
	}
	for p := range 3 {
		if f.lf.restorePlanes>>p&1 == 0 {
			continue
		}
		idx, uIdx := sbIdx, unitIdx
		if f.resize {
			ssHor := b2i(p != 0 && f.layout != pixelLayoutI444)
			d := int(hdr.superRes.widthScaleDenominator)
			unitSizeLog2 := int(hdr.restoration.unitSize[b2i(p != 0)])
			rnd, shift := (8<<unitSizeLog2)-1, unitSizeLog2+3
			x := ((4 * ts.tiling.colStart * d >> ssHor) + rnd) >> shift
			pxX := x << (unitSizeLog2 + ssHor)
			uIdx += (pxX & 64) >> 6
			sb128x := pxX >> 7
			if sb128x >= f.srSb128w {
				continue
			}
			idx += sb128x
		}
		ref := &f.lrMask[idx].lr[p][uIdx]
		ts.lrRef[p] = ref
		ref.filterV = [3]int8{3, -7, 15}
		ref.filterH = [3]int8{3, -7, 15}
		ref.sgrWeights = [2]int8{-32, 31}
	}
}

// runTiles decodes every tile of the frame, at most nThreads at a time.
func (f *frameContext[P, C]) runTiles(
	fn func(*taskContext[P, C], int, int) error, nRows, nCols int,
) error {
	nTs := nRows * nCols
	errs := make([]error, nTs)

	if f.nThreads <= 1 {
		t := f.newTaskContext()
		for i := range nTs {
			if errs[i] = fn(t, i/nCols, i%nCols); errs[i] != nil {
				break
			}
		}
	} else {
		var wg sync.WaitGroup
		next := make(chan int, nTs)
		for i := range nTs {
			next <- i
		}
		close(next)

		for range f.nThreads {
			wg.Add(1)
			go func() {
				defer wg.Done()
				t := f.newTaskContext()
				for i := range next {
					errs[i] = fn(t, i/nCols, i%nCols)
				}
			}()
		}
		wg.Wait()
	}

	return errors.Join(errs...)
}

func (f *frameContext[P, C]) newTaskContext() *taskContext[P, C] {
	t := &taskContext[P, C]{f: f}
	t.cf = make([]C, 32*32)

	return t
}

func backupIpredEdge[P pixel, C coef](t *taskContext[P, C]) {
	f := t.f
	ts := t.ts
	sby := t.by >> f.sbShift
	sbyOff := f.sb128w * 128 * sby
	xOff := ts.tiling.colStart

	yOff := xOff*4 + ((t.by+f.sbStep)*4-1)*f.stride[0]
	n := 4 * (ts.tiling.colEnd - xOff)
	copy(f.ipredEdge[0][sbyOff+xOff*4:][:n], f.cur[0][yOff:yOff+n])

	if f.layout == pixelLayoutI400 {
		return
	}

	uvOff := (xOff * 4 >> f.ssHor) + (((t.by+f.sbStep)*4>>f.ssVer)-1)*f.stride[1]
	nUv := 4 * (ts.tiling.colEnd - xOff) >> f.ssHor
	for pl := 1; pl <= 2; pl++ {
		copy(f.ipredEdge[pl][sbyOff+(xOff*4>>f.ssHor):][:nUv],
			f.cur[pl][uvOff:uvOff+nUv])
	}
}

func splitTiles(hdr *frameHeader, tiles []tileGroup, nTs int) ([][]byte, error) {
	out := make([][]byte, nTs)
	nBytes := int(hdr.tiling.nBytes)

	for _, tg := range tiles {
		data := tg.data
		for j := tg.start; j <= tg.end; j++ {
			if j >= nTs {
				return nil, errInvalid
			}
			var sz int
			if j == tg.end {
				sz = len(data)
			} else {
				if nBytes > len(data) {
					return nil, errInvalid
				}
				for k := range nBytes {
					sz |= int(data[k]) << (k * 8)
				}
				sz++
				data = data[nBytes:]
				if sz > len(data) {
					return nil, errInvalid
				}
			}
			out[j] = data[:sz]
			data = data[sz:]
		}
	}

	for _, t := range out {
		if t == nil {
			return nil, errInvalid
		}
	}

	return out, nil
}

func scaleFacRef(refSz, thisSz int) int {
	return ((refSz << 14) + (thisSz >> 1)) / thisSz
}

func (f *frameContext[P, C]) setupRefs(d *Decoder) error {
	hdr := f.frameHdr

	if !isInterOrSwitch(hdr) {
		return nil
	}

	for i := range 7 {
		ref := &d.refs[hdr.refidx[i]]
		if ref.pic == nil {
			return errInvalid
		}
		p, ok := ref.pic.(*refPicture[P])
		if !ok {
			return errInvalid
		}
		f.refp[i] = p

		if p.w<<14 < f.frameHdr.width[0] || p.h<<14 < f.frameHdr.height ||
			p.w > f.frameHdr.width[0]<<4 || p.h > f.frameHdr.height<<4 {
			return errInvalid
		}
		if hdr.width[0] != p.w || hdr.height != p.h {
			f.svc[i][0].scale = scaleFacRef(p.w, hdr.width[0])
			f.svc[i][1].scale = scaleFacRef(p.h, hdr.height)
			f.svc[i][0].step = (f.svc[i][0].scale + 8) >> 4
			f.svc[i][1].step = (f.svc[i][1].scale + 8) >> 4
		} else {
			f.svc[i][0].scale, f.svc[i][1].scale = 0, 0
		}

		f.gmvWarpAllowed[i] = hdr.gmv[i].typ > wmTypeTranslation &&
			hdr.forceIntegerMv == 0 && !getShearParams(&hdr.gmv[i]) &&
			f.svc[i][0].scale == 0
	}

	return nil
}

func (f *frameContext[P, C]) setupRefmvs(d *Decoder, seqHdr *sequenceHeader) {
	hdr := f.frameHdr

	if !isInterOrSwitch(hdr) && hdr.allowIntrabc == 0 {
		refmvsInitFrame(&f.rf, seqHdr, hdr, &f.refPoc, nil, &f.refRefPoc,
			&f.refMvs, f.nThreads, 1)

		return
	}

	f.mvs = make([]refmvsTemporalBlock, f.sb128h*16*(f.b4Stride>>1))

	if hdr.allowIntrabc == 0 {
		for i := range 7 {
			if f.refp[i] != nil {
				f.refPoc[i] = d.refs[hdr.refidx[i]].frameHdr.frameOffset
			}
		}
	}

	if hdr.useRefFrameMvs != 0 {
		for i := range 7 {
			refidx := hdr.refidx[i]
			ref := &d.refs[refidx]
			refW := ((ref.codedW + 7) >> 3) << 1
			refH := 0
			if p, ok := ref.pic.(*refPicture[P]); ok {
				refH = ((p.h + 7) >> 3) << 1
			}
			if ref.refmvs != nil && refW == f.bw && refH == f.bh {
				f.refMvs[i] = ref.refmvs
			}
			f.refRefPoc[i] = ref.refpoc
		}
	}

	refmvsInitFrame(&f.rf, seqHdr, hdr, &f.refPoc, f.mvs, &f.refRefPoc,
		&f.refMvs, f.nThreads, 1)
}

func (f *frameContext[P, C]) setupSegMap(d *Decoder) {
	hdr := f.frameHdr
	size := f.b4Stride * 32 * f.sb128h

	if hdr.segmentation.enabled == 0 {
		f.curSegMap = make([]uint8, size)

		return
	}

	if hdr.segmentation.temporal != 0 || hdr.segmentation.updateMap == 0 {
		priRef := int(hdr.primaryRefFrame)
		ref := &d.refs[hdr.refidx[priRef]]
		refW := ((ref.codedW + 7) >> 3) << 1
		refH := 0
		if p, ok := ref.pic.(*refPicture[P]); ok {
			refH = ((p.h + 7) >> 3) << 1
		}
		if refW == f.bw && refH == f.bh {
			f.prevSegMap = ref.segmap
		}
	}

	switch {
	case hdr.segmentation.updateMap != 0:
		f.curSegMap = make([]uint8, size)
	case f.prevSegMap != nil:
		f.curSegMap = f.prevSegMap
	default:
		f.curSegMap = make([]uint8, size)
	}
}

func (f *frameContext[P, C]) refPicture() *refPicture[P] {
	return &refPicture[P]{
		planes: f.srCur,
		stride: f.srStride,
		w:      f.srWidth,
		h:      f.frameHdr.height,
	}
}

var (
	quantDistWeight      = [3][2]uint8{{2, 3}, {2, 5}, {2, 7}}
	quantDistLookupTable = [4][2]uint8{{9, 7}, {11, 5}, {12, 4}, {13, 3}}
)

func (f *frameContext[P, C]) setupJntWeights() {
	if f.frameHdr.switchableCompRefs == 0 {
		return
	}

	nbits := int(f.seqHdr.orderHintNBits)
	poc := int(f.frameHdr.frameOffset)

	for i := range 7 {
		ref0poc := int(f.refPoc[i])
		for j := i + 1; j < 7; j++ {
			ref1poc := int(f.refPoc[j])
			d1 := min(int(abs32(int32(getPocDiff(nbits, ref0poc, poc)))), 31)
			d0 := min(int(abs32(int32(getPocDiff(nbits, ref1poc, poc)))), 31)
			order := b2i(d0 <= d1)

			k := 0
			for ; k < 3; k++ {
				c0 := int(quantDistWeight[k][order])
				c1 := int(quantDistWeight[k][1-order])
				if (d0 > d1 && d0*c0 < d1*c1) || (d0 <= d1 && d0*c0 > d1*c1) {
					break
				}
			}

			f.jntWeights[i][j] = quantDistLookupTable[k][order]
		}
	}
}

func (f *frameContext[P, C]) saveTmvsSbrow(rt *refmvsTile,
	colStart8, colEnd8, rowStart8, rowEnd8 int,
) {
	rf := rt.rf
	rowEnd8 = min(rowEnd8, rf.ih8)
	colEnd8 = min(colEnd8, rf.iw8)

	saveTmvs(rf.rp, rowStart8*rf.rpStride, rf.rpStride, rt.r, rt.rows[6:],
		&rf.mfmvSign, colEnd8, rowEnd8, colStart8, rowStart8)
}

// Pixel layouts reported by Picture.Layout.
const (
	LayoutI400 = pixelLayoutI400
	LayoutI420 = pixelLayoutI420
	LayoutI422 = pixelLayoutI422
	LayoutI444 = pixelLayoutI444
)
