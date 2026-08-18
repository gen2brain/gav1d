package av1

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Picture is a decoded frame.
type Picture struct {
	Width, Height int
	Layout        int
	BitDepth      int
	// Data holds the planes. Above 8 bits each sample is a native-endian
	// uint16, and Stride is in bytes.
	Data   [3][]byte
	Stride [3]int

	// CICP color description carried by the sequence header.
	ColorPrimaries int
	Transfer       int
	MatrixCoeffs   int
	FullRange      bool

	mem *picMem
}

// Release hands the picture's memory back to the decoder that produced it, to
// be reused by a later frame. It is optional: a picture that is never released
// is collected as any other value would be. Reading Data after releasing, or
// releasing twice, is a mistake the decoder will not survive quietly.
func (p *Picture) Release() {
	m := p.mem
	p.mem = nil
	p.Data = [3][]byte{}
	m.release()
}

const picPad = 128

// picPool hands back the sample memory of frames nobody holds any more. A
// decoder keeps one, so the buffers it recycles are all the same shape.
type picPool struct {
	mu   sync.Mutex
	free map[int][][]byte
}

// picPoolDepth bounds how many buffers of one size the pool keeps, so a stream
// that changes resolution cannot make it grow without end.
const picPoolDepth = 8

func (p *picPool) get(n int) []byte {
	if p == nil {
		return make([]byte, n)
	}

	p.mu.Lock()
	if bufs := p.free[n]; len(bufs) > 0 {
		b := bufs[len(bufs)-1]
		p.free[n] = bufs[:len(bufs)-1]
		p.mu.Unlock()

		return b
	}
	p.mu.Unlock()

	return make([]byte, n)
}

func (p *picPool) put(bufs [][]byte) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.free == nil {
		p.free = make(map[int][][]byte)
	}
	for _, b := range bufs {
		if len(p.free[len(b)]) < picPoolDepth {
			p.free[len(b)] = append(p.free[len(b)], b)
		}
	}
}

func (p *picPool) drop() {
	if p == nil {
		return
	}

	p.mu.Lock()
	p.free = nil
	p.mu.Unlock()
}

// picMem owns one frame's sample memory. The reference slots and the Picture
// handed to the caller each hold a count, and the memory goes back to the pool
// only when the last of them lets go.
type picMem struct {
	pool *picPool
	bufs [][]byte
	refs atomic.Int32
}

func (m *picMem) acquire() {
	if m != nil {
		m.refs.Add(1)
	}
}

func (m *picMem) release() {
	if m == nil {
		return
	}
	switch n := m.refs.Add(-1); {
	case n == 0:
		m.pool.put(m.bufs)
	case n < 0:
		panic("av1: picture released more times than it was held")
	}
}

func picAlloc[P pixel](p *picPool, m *picMem, n int) []P {
	var z P
	b := p.get(n * int(unsafe.Sizeof(z)))
	m.bufs = append(m.bufs, b)

	return unsafe.Slice((*P)(unsafe.Pointer(unsafe.SliceData(b))), n)
}

// frameDecoder is the half of a frame's decode that reads no decoder state.
type frameDecoder interface {
	decode(in, out *cdfPromise) (*Picture, error)
}

// setupFrame runs the half that does, in bitstream order, and reserves the
// reference slots the frame refreshes.
func setupFrame[P pixel, C coef](d *Decoder, tiles []tileGroup, p *pendingFrame) (frameDecoder, error) {
	seqHdr, frameHdr := d.seqHdr, d.frameHdr

	f := &frameContext[P, C]{
		seqHdr:   seqHdr,
		frameHdr: frameHdr,
		layout:   seqHdr.layout,
		bpc:      8 + 2*int(seqHdr.hbd),
		pool:     &d.pool,
		tc:       &d.tc,
		fb:       &d.fb,

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
	f.progress = newFrameProgress()
	f.dsp = newDsp[P]()
	f.lr = newLr[P, C]()

	f.resize = frameHdr.width[0] != frameHdr.width[1]
	f.srWidth = frameHdr.width[1]

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

	b, ok := d.fb.Get().(*frameBufs[P])
	if !ok {
		b = &frameBufs[P]{}
	}
	f.bufs = b

	for pl := range 3 {
		b.ipredEdge[pl] = reuse(b.ipredEdge[pl], f.sb128w*128*(sbh+1))
		f.ipredEdge[pl] = b.ipredEdge[pl]
	}

	nTs := int(frameHdr.tiling.cols) * int(frameHdr.tiling.rows)
	var err error
	if f.tileData, err = splitTiles(frameHdr, tiles, nTs); err != nil {
		return nil, err
	}

	b.a = reuse(b.a, f.sb128w*int(frameHdr.tiling.rows))
	f.a = b.a
	for i := range f.a {
		resetContext(&f.a[i], isKeyOrIntra(frameHdr))
	}

	sb128h := f.sb128h
	b.lflvl = reuse(b.lflvl, f.sb128w*sb128h)
	f.lflvl = b.lflvl
	b.level = reuse(b.level, f.sb128w*sb128h*32*32*4+3)
	f.lf.level = b.level
	calcEIH(&f.lf.limLut, int(frameHdr.loopfilter.sharpness))
	calcLfValues(&f.lf.lvl, frameHdr, &[4]int8{})

	alignH := (f.bh + 31) &^ 31
	b.txLpf[0] = reuse(b.txLpf[0], alignH*int(frameHdr.tiling.cols))
	b.txLpf[1] = reuse(b.txLpf[1], (alignH>>f.ssVer)*int(frameHdr.tiling.cols))
	f.lf.txLpfRightEdge = b.txLpf

	f.lf.restorePlanes = b2i(frameHdr.restoration.typ[0] != restorationNone) |
		b2i(frameHdr.restoration.typ[1] != restorationNone)<<1 |
		b2i(frameHdr.restoration.typ[2] != restorationNone)<<2
	f.srSb128w = (f.srWidth + 127) >> 7
	b.lrMask = reuse(b.lrMask, f.srSb128w*sb128h)
	f.lrMask = b.lrMask

	if seqHdr.cdef != 0 || f.lf.restorePlanes != 0 {
		const lrNumLines = 12
		b.lrLineBuf = reuse(b.lrLineBuf, 64+f.srStride[0]*lrNumLines+f.srStride[1]*lrNumLines*2)
		f.lf.lrLineBuf = b.lrLineBuf
		off := 64
		f.lf.lrLpfLine[0] = off
		off += f.srStride[0] * lrNumLines
		f.lf.lrLpfLine[1] = off
		f.lf.lrLpfLine[2] = off + f.srStride[1]*lrNumLines
	}

	if seqHdr.cdef != 0 {
		yStride, uvStride := f.stride[0], f.stride[1]
		b.cdefLineBuf = reuse(b.cdefLineBuf, 32+yStride*sbh*4+uvStride*sbh*8)
		f.lf.cdefLineBuf = b.cdefLineBuf
		off := 32
		f.lf.cdefLine[0][0] = off
		f.lf.cdefLine[1][0] = off + 2*yStride
		off += yStride * sbh * 4
		f.lf.cdefLine[0][1] = off
		f.lf.cdefLine[0][2] = off + 2*uvStride
		f.lf.cdefLine[1][1] = off + 4*uvStride
		f.lf.cdefLine[1][2] = off + 6*uvStride
	}

	f.saveRefs(d, p)

	return f, nil
}

func (f *frameContext[P, C]) decode(in, out *cdfPromise) (*Picture, error) {
	defer f.releasePlanes()

	seqHdr, frameHdr := f.seqHdr, f.frameHdr
	a := f.a

	inCdf, err := resolveCdf(frameHdr, in)
	if err != nil {
		f.progress.abort(err)
		out.finish(err)

		return nil, err
	}
	f.inCdf = inCdf
	out.cdf = *inCdf
	if frameHdr.refreshContext == 0 {
		out.finish(nil)
	}

	keyframe := isKeyOrIntra(frameHdr)

	nCols := int(frameHdr.tiling.cols)
	nRows := int(frameHdr.tiling.rows)
	tstates := make([]*tileState, nRows*nCols)
	for i := range tstates {
		ts := new(tileState)
		setupTile(ts, f, f.tileData[i], i/nCols, i%nCols)
		tstates[i] = ts
	}

	inter := isInterOrSwitch(frameHdr) || frameHdr.allowIntrabc != 0

	decodeSbrow := func(t *taskContext[P, C], tileRow, tileCol, sby int) error {
		ts := tstates[tileRow*nCols+tileCol]
		t.ts = ts
		colSb128Start := int(frameHdr.tiling.colStartSb[tileCol]) >> b2i(seqHdr.sb128 == 0)

		by := sby << f.sbShift
		byEnd := (by + f.sbStep) >> 1
		t.by = by

		if inter {
			refmvsTileSbrowInit(&t.rt, &f.rf, ts.tiling.colStart,
				ts.tiling.colEnd, ts.tiling.rowStart, ts.tiling.rowEnd,
				sby, tileRow)
		}
		if frameHdr.useRefFrameMvs != 0 || f.prevSegMap != nil {
			if err := f.waitRefRows(byEnd * 8); err != nil {
				return err
			}
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

		return nil
	}

	filterSbrow := func(pt *taskContext[P, C], sby int) {
		if f.lfEnabled {
			startOfTileRow := 0
			for tr := range nRows {
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
			lrSbrow(pt, sby)
		}
	}

	tileDone := func(i int) {
		if frameHdr.refreshContext != 0 && i == int(frameHdr.tiling.update) {
			cdfThreadUpdate(frameHdr, &out.cdf, &tstates[i].cdf)
			out.finish(nil)
		}
	}

	if err := f.runFrame(decodeSbrow, filterSbrow, tileDone, nRows, nCols); err != nil {
		out.finish(err)

		return nil, err
	}

	pic := f.picture()
	if hasFilmGrain(frameHdr) {
		pic = f.grainPicture()
	}

	return pic, nil
}

func (f *frameContext[P, C]) saveRefs(d *Decoder, out *pendingFrame) {
	hdr := f.frameHdr
	pic := f.refPicture()

	for i := range numRefFrames {
		if hdr.refreshFrameFlags&(1<<i) == 0 {
			continue
		}
		s := d.refs[i]
		s.out = out
		s.pic = pic
		s.codedW = hdr.width[0]
		s.segmap = f.curSegMap
		s.refmvs = nil
		if hdr.allowIntrabc == 0 {
			s.refmvs = f.mvs
		}
		s.refpoc = f.refPoc
		d.setRef(i, s)
	}
}

func (f *frameContext[P, C]) allocPicture(w int, planes *[3][]P, stride *[3]int) *picMem {
	h := f.frameHdr.height
	m := &picMem{pool: f.pool}
	m.refs.Store(1)

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
		buf := picAlloc[P](f.pool, m, s*(ph+2*picPad))
		if pl == 0 {
			stride[0] = s
		} else {
			stride[1] = s
		}
		planes[pl] = buf[picPad*s+picPad:]
	}

	return m
}

// releasePlanes drops the frame's own hold on its memory, which it keeps for as
// long as it decodes so a later frame cannot recycle it underneath.
// frameBufs holds the tables a frame sizes from its own dimensions. None of
// them outlive the frame, so the decoder keeps them for the next one, which
// reuses the memory whenever the geometry has not grown.
type frameBufs[P pixel] struct {
	ipredEdge   [3][]P
	a           []blockContext
	lflvl       []av1Filter
	level       []uint8
	txLpf       [2][]uint8
	lrMask      []av1Restoration
	lrLineBuf   []P
	cdefLineBuf []P
}

// reuse returns n zeroed elements, keeping the backing array when it is
// already large enough. It stands in for make, so a caller cannot tell the
// difference; what it saves is the allocation, not the zeroing.
func reuse[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	s = s[:n]
	clear(s)

	return s
}

func (f *frameContext[P, C]) releasePlanes() {
	if f.srMem != f.mem {
		f.srMem.release()
	}
	f.mem.release()
	f.mem, f.srMem = nil, nil

	if f.bufs != nil {
		f.fb.Put(f.bufs)
		f.bufs = nil
	}
}

func (f *frameContext[P, C]) allocPlanes() error {
	f.mem = f.allocPicture(f.frameHdr.width[0], &f.cur, &f.stride)

	if !f.resize {
		f.srCur, f.srStride = f.cur, f.stride
		f.srMem = f.mem

		return nil
	}

	f.srMem = f.allocPicture(f.srWidth, &f.srCur, &f.srStride)

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
		mem:      f.srMem,
	}
	f.srMem.acquire()
	f.setColor(p)

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

// runFrame decodes every tile, at most nThreads at a time, and post-filters
// each superblock row once every tile column of it is done.
func (f *frameContext[P, C]) runFrame(
	decodeSbrow func(*taskContext[P, C], int, int, int) error,
	filterSbrow func(*taskContext[P, C], int), tileDone func(int),
	nRows, nCols int,
) error {
	hdr := f.frameHdr
	pt := f.newTaskContext()
	defer f.putTaskContext(pt)

	if f.nThreads <= 1 {
		t := f.newTaskContext()
		defer f.putTaskContext(t)
		for tr := range nRows {
			sbhEnd := min(int(hdr.tiling.rowStartSb[tr+1]), f.sbh)
			for sby := int(hdr.tiling.rowStartSb[tr]); sby < sbhEnd; sby++ {
				for tc := range nCols {
					if err := decodeSbrow(t, tr, tc, sby); err != nil {
						f.progress.abort(err)

						return err
					}
				}
				f.progress.publishDecoded(f.decodedRows(sby))
				filterSbrow(pt, sby)
				f.progress.publishRows(f.finalRows(sby))
			}
			for tc := range nCols {
				tileDone(tr*nCols + tc)
			}
		}

		return nil
	}

	nTs := nRows * nCols
	errs := make([]error, nTs)
	sbs := newSbrowSync(f.sbh)

	next := make(chan int, nTs)
	for i := range nTs {
		next <- i
	}
	close(next)

	var wg sync.WaitGroup
	for range f.nThreads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := f.newTaskContext()
			defer f.putTaskContext(t)
			for i := range next {
				tr, tc := i/nCols, i%nCols
				sbhEnd := min(int(hdr.tiling.rowStartSb[tr+1]), f.sbh)
				for sby := int(hdr.tiling.rowStartSb[tr]); sby < sbhEnd; sby++ {
					if errs[i] = decodeSbrow(t, tr, tc, sby); errs[i] != nil {
						sbs.fail()

						break
					}
					sbs.done(sby, nCols)
				}
				if errs[i] == nil {
					tileDone(i)
				}
			}
		}()
	}

	for sby := range f.sbh {
		if !sbs.wait(sby, nCols) {
			break
		}
		f.progress.publishDecoded(f.decodedRows(sby))
		filterSbrow(pt, sby)
		f.progress.publishRows(f.finalRows(sby))
	}

	wg.Wait()

	err := errors.Join(errs...)
	if err != nil {
		f.progress.abort(err)
	}

	return err
}

// finalRows counts the rows of srCur that can no longer change once superblock
// row sby is post-filtered; the filters reach eight rows back across the edge.
func (f *frameContext[P, C]) finalRows(sby int) int {
	h := f.frameHdr.height
	if sby+1 >= f.sbh {
		return h
	}

	rows := (sby + 1) * f.sbStep * 4
	if f.lfEnabled || f.seqHdr.cdef != 0 || f.lf.restorePlanes != 0 || f.resize {
		rows -= 8
	}

	return min(rows, h)
}

// decodedRows counts the rows reconstructed once every tile column has
// finished superblock row sby.
func (f *frameContext[P, C]) decodedRows(sby int) int {
	return min((sby+1)*f.sbStep*4, f.frameHdr.height)
}

// frameProgress publishes how much of a frame a later frame may read: rows is
// post-filtered for motion compensation, decoded is enough for block data.
type frameProgress struct {
	rows    atomic.Int64
	decoded atomic.Int64
	mu      sync.Mutex
	cond    *sync.Cond
	err     error
}

const progressFailed = -1

func newFrameProgress() *frameProgress {
	p := &frameProgress{}
	p.cond = sync.NewCond(&p.mu)

	return p
}

func (p *frameProgress) publishRows(rows int)    { p.publish(&p.rows, rows) }
func (p *frameProgress) publishDecoded(rows int) { p.publish(&p.decoded, rows) }

func (p *frameProgress) waitRows(rows int) error    { return p.wait(&p.rows, rows) }
func (p *frameProgress) waitDecoded(rows int) error { return p.wait(&p.decoded, rows) }

func (p *frameProgress) publish(c *atomic.Int64, rows int) {
	p.mu.Lock()
	if v := c.Load(); v != progressFailed && int64(rows) > v {
		c.Store(int64(rows))
		p.cond.Broadcast()
	}
	p.mu.Unlock()
}

func (p *frameProgress) abort(err error) {
	p.mu.Lock()
	if p.err == nil {
		p.err = err
	}
	p.rows.Store(progressFailed)
	p.decoded.Store(progressFailed)
	p.cond.Broadcast()
	p.mu.Unlock()
}

func (p *frameProgress) wait(c *atomic.Int64, rows int) error {
	if c.Load() >= int64(rows) {
		return nil
	}

	p.mu.Lock()
	for c.Load() < int64(rows) && p.err == nil {
		p.cond.Wait()
	}
	err := p.err
	p.mu.Unlock()

	return err
}

// waitRef blocks until a reference has post-filtered every row a read needs.
// Edge extension clamps the read to the frame, so the bound is clamped too.
func (f *frameContext[P, C]) waitRef(refp *refPicture[P], bottom int) error {
	p := refp.progress
	if p == nil || p == f.progress {
		return nil
	}

	return p.waitRows(clip(bottom, 1, refp.h))
}

// waitRefRows blocks until the references this frame takes temporal motion
// vectors and its segmentation map from have decoded the co-located rows.
func (f *frameContext[P, C]) waitRefRows(rows int) error {
	priRef := int(f.frameHdr.primaryRefFrame)

	for i := range 7 {
		if f.refMvs[i] == nil && (f.prevSegMap == nil || i != priRef) {
			continue
		}
		if f.refp[i] == nil {
			continue
		}
		p := f.refp[i].progress
		if p == nil || p == f.progress {
			continue
		}
		if err := p.waitDecoded(min(rows, f.refp[i].h)); err != nil {
			return err
		}
	}

	return nil
}

// sbrowSync tracks how many tile columns have finished each superblock row.
type sbrowSync struct {
	mu     sync.Mutex
	cond   *sync.Cond
	n      []int
	failed bool
}

func newSbrowSync(sbh int) *sbrowSync {
	s := &sbrowSync{n: make([]int, sbh)}
	s.cond = sync.NewCond(&s.mu)

	return s
}

func (s *sbrowSync) done(sby, nCols int) {
	s.mu.Lock()
	s.n[sby]++
	if s.n[sby] == nCols {
		s.cond.Broadcast()
	}
	s.mu.Unlock()
}

func (s *sbrowSync) fail() {
	s.mu.Lock()
	s.failed = true
	s.cond.Broadcast()
	s.mu.Unlock()
}

func (s *sbrowSync) wait(sby, nCols int) bool {
	s.mu.Lock()
	for s.n[sby] < nCols && !s.failed {
		s.cond.Wait()
	}
	ok := !s.failed
	s.mu.Unlock()

	return ok
}

// newTaskContext takes the scratch from the decoder's pool, which is almost
// all of a task context by size. Only the scratch is reused; the rest is small
// and starts zeroed, so nothing that reads a field before writing it can carry
// over from the last frame.
func (f *frameContext[P, C]) newTaskContext() *taskContext[P, C] {
	t := &taskContext[P, C]{f: f}
	if s, ok := f.tc.Get().(*scratch[P]); ok {
		t.scratch = s
	} else {
		t.scratch = &scratch[P]{}
	}
	t.cf = make([]C, 32*32)

	return t
}

func (f *frameContext[P, C]) putTaskContext(t *taskContext[P, C]) {
	f.tc.Put(t.scratch)
	t.scratch = nil
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

// refSizeValid is the spec's reference scaling window: a reference may be at
// most twice the frame and at least a sixteenth of it.
func refSizeValid(fw, fh, rw, rh int) bool {
	return fw*2 >= rw && fh*2 >= rh && fw <= rw*16 && fh <= rh*16
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

		if !refSizeValid(hdr.width[0], hdr.height, p.w, p.h) {
			return errInvalid
		}
		if hdr.width[0] != p.w || hdr.height != p.h {
			f.svc[i][0].scale = scaleFac(p.w, hdr.width[0])
			f.svc[i][1].scale = scaleFac(p.h, hdr.height)
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
		planes:   f.srCur,
		stride:   f.srStride,
		w:        f.srWidth,
		h:        f.frameHdr.height,
		progress: f.progress,
		m:        f.srMem,
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

func (f *frameContext[P, C]) setColor(p *Picture) {
	p.ColorPrimaries = f.seqHdr.pri
	p.Transfer = f.seqHdr.trc
	p.MatrixCoeffs = f.seqHdr.mtrx
	p.FullRange = f.seqHdr.colorRange != 0
}

// Pixel layouts reported by Picture.Layout.
const (
	LayoutI400 = pixelLayoutI400
	LayoutI420 = pixelLayoutI420
	LayoutI422 = pixelLayoutI422
	LayoutI444 = pixelLayoutI444
)
