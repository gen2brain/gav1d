package av1

import (
	"sync"
	"unsafe"
)

type blockContext struct {
	mode      [32]uint8
	lcoef     [32]uint8
	ccoef     [2][32]uint8
	segPred   [32]uint8
	skip      [32]uint8
	skipMode  [32]uint8
	intra     [32]uint8
	compType  [32]uint8
	ref       [2][32]int8
	filter    [2][32]uint8
	txIntra   [32]int8
	tx        [32]int8
	txLpfY    [32]uint8
	txLpfUv   [32]uint8
	partition [16]uint8
	uvmode    [32]uint8
	palSz     [32]uint8
}

type av1Block struct {
	bl, bs, bp uint8
	intra      uint8
	segID      uint8
	skipMode   uint8
	skip       uint8
	uvtx       uint8

	yMode    uint8
	uvMode   uint8
	tx       uint8
	maxYtx   uint8
	txSplit0 uint8
	txSplit1 uint16
	palSz    [2]uint8
	yAngle   int8
	uvAngle  int8
	cflAlpha [2]int8

	ref            [2]int8
	mv             [2]mv
	compType       uint8
	interintraType uint8
	interintraMode uint8
	wedgeIdx       uint8
	maskSign       uint8
	interMode      uint8
	motionMode     uint8
	drlIdx         uint8
	filter2d       uint8
}

type tileTiling struct {
	colStart, colEnd, rowStart, rowEnd int
	col, row                           int
}

type tileState struct {
	cdf  cdfContext
	msac msacContext

	tiling tileTiling

	dqmem [maxSegments][3][2]uint16
	dq    *[maxSegments][3][2]uint16

	lastQidx    int
	lastDeltaLf [4]int8

	lflvlmem [8][4][8][2]uint8
	lflvl    *[8][4][8][2]uint8

	lrRef [3]*av1RestorationUnit
}

type scratch[P pixel] struct {
	levels   [32 * 34]uint8
	ac       [32 * 32]int16
	palIdxY  [32 * 64]uint8
	palIdxUv [64 * 64]uint8
	palOrder [64][8]uint8
	palCtx   [64]uint8
	edge     [257]P
	pal      [3][8]P

	lap        [128 * 32]P
	emuEdge    [320 * (256 + 7)]P
	compinter  [2][128 * 128]int16
	segMask    [128 * 128]uint8
	interintra [64 * 64]P
	txtpMap    [32 * 32]uint8
	itx        [itxScratchLen]int32
	mcMid      [135 * mcMidStride]int16
	lfTxa      txaBuf
	cdefTmp    [144]int16
	cdefLrBak  [2][3][16]P
	lrBorder   [2][(128 + 8) * 4]P
	lrParams   looprestorationParams
	lrw        lrScratch
}

type frameContext[P pixel, C coef] struct {
	seqHdr   *sequenceHeader
	frameHdr *frameHeader

	pool  *picPool
	tc    *sync.Pool
	fb    *sync.Pool
	bufs  *frameBufs[P]
	mem   *picMem
	srMem *picMem

	bitdepthMax int32
	bpc         int
	layout      int
	ssHor       int
	ssVer       int

	bw, bh   int
	w4, h4   int
	sb128w   int
	srSb128w int
	sb128h   int
	sbh      int
	sbSz     int
	sbShift  int
	sbStep   int
	b4Stride int

	dq        [maxSegments][3][2]uint16
	curSegMap []uint8
	inCdf     *cdfContext

	qm [nRectTxSizes][3][]uint8

	cur       [3][]P
	stride    [3]int
	srCur     [3][]P
	srStride  [3]int
	srWidth   int
	ipredEdge [3][]P

	resize      bool
	resizeStep  [2]int
	resizeStart [2]int

	refp       [7]*refPicture[P]
	refPoc     [7]uint8
	refRefPoc  [7][7]uint8
	refMvs     [7][]refmvsTemporalBlock
	reconIntra func(t *taskContext[P, C], bs, intraEdgeFlags int, b *av1Block)
	reconInter func(t *taskContext[P, C], bs int, b *av1Block) error

	jntWeights     [7][7]uint8
	svc            [7][2]struct{ scale, step int }
	gmvWarpAllowed [7]bool

	rf         refmvsFrame
	mvs        []refmvsTemporalBlock
	prevSegMap []uint8

	nThreads  int
	lfEnabled bool
	progress  *frameProgress
	tileData  [][]byte
	dsp       *dspContext[P]
	lr        *lrContext[P, C]

	a      []blockContext
	lflvl  []av1Filter
	lrMask []av1Restoration

	lf struct {
		level          []uint8
		limLut         filterLUT
		lvl            [8][4][8][2]uint8
		txLpfRightEdge [2][]uint8
		cdefLineBuf    []P
		cdefLine       [2][3]int
		lrLineBuf      []P
		lrLpfLine      [3]int
		restorePlanes  int
	}
}

type taskContext[P pixel, C coef] struct {
	f  *frameContext[P, C]
	ts *tileState

	bx, by int

	l blockContext
	a *blockContext

	cf           []C
	scratch      *scratch[P]
	blk          av1Block
	curSbCdefIdx []int8
	lfMask       *av1Filter

	topPreCdefToggle int

	alPal   [2][32][3][8]P
	palSzUv [2][32]uint8

	warpmv      warpedMotionParams
	rt          refmvsTile
	tl4x4Filter uint8
}

func getUvInterTxtp(uvtDim *txfmInfo, ytxtp int) int {
	if uvtDim.max == tx32x32 {
		if ytxtp == idtx {
			return idtx
		}

		return dctDct
	}
	if uvtDim.min == tx16x16 &&
		(1<<ytxtp)&((1<<hFlipadst)|(1<<vFlipadst)|(1<<hAdst)|(1<<vAdst)) != 0 {
		return dctDct
	}

	return ytxtp
}

type tileGroup struct {
	data       []byte
	start, end int
}

func pixelsToBytes[P pixel](p []P) []byte {
	var z P
	if unsafe.Sizeof(z) == 1 {
		return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(p))), len(p))
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(p))), len(p)*2)
}
