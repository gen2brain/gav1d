package av1

const (
	obuMetaHdrCll      = 1
	obuMetaHdrMdcv     = 2
	obuMetaScalability = 3
	obuMetaItutT35     = 4
	obuMetaTimecode    = 5
)

const (
	tx4x4 = iota
	tx8x8
	tx16x16
	tx32x32
	tx64x64
	nTxSizes
)

const (
	bl128x128 = iota
	bl64x64
	bl32x32
	bl16x16
	bl8x8
	nBlLevels
)

const (
	rtx4x8 = nTxSizes + iota
	rtx8x4
	rtx8x16
	rtx16x8
	rtx16x32
	rtx32x16
	rtx32x64
	rtx64x32
	rtx4x16
	rtx16x4
	rtx8x32
	rtx32x8
	rtx16x64
	rtx64x16
	nRectTxSizes
)

const (
	dctDct = iota
	adstDct
	dctAdst
	adstAdst
	flipadstDct
	dctFlipadst
	flipadstFlipadst
	adstFlipadst
	flipadstAdst
	idtx
	vDct
	hDct
	vAdst
	hAdst
	vFlipadst
	hFlipadst
	nTxTypes
	whtWht         = nTxTypes
	nTxTypesPlusLl = nTxTypes + 1
)

const (
	txClass2d = iota
	txClassH
	txClassV
)

const (
	dcPred = iota
	vertPred
	horPred
	diagDownLeftPred
	diagDownRightPred
	vertRightPred
	horDownPred
	horUpPred
	vertLeftPred
	smoothPred
	smoothVPred
	smoothHPred
	paethPred
	nIntraPredModes
	cflPred             = nIntraPredModes
	nUvIntraPredModes   = nIntraPredModes + 1
	nImplIntraPredModes = nUvIntraPredModes
	leftDcPred          = diagDownLeftPred
	topDcPred           = diagDownRightPred
	dc128Pred           = vertRightPred
	z1Pred              = horDownPred
	z2Pred              = horUpPred
	z3Pred              = vertLeftPred
	filterPred          = nIntraPredModes
)

const (
	iiDcPred = iota
	iiVertPred
	iiHorPred
	iiSmoothPred
	nInterIntraPredModes
)

const (
	partitionNone = iota
	partitionH
	partitionV
	partitionSplit
	partitionTTopSplit
	partitionTBottomSplit
	partitionTLeftSplit
	partitionTRightSplit
	partitionH4
	partitionV4
	nPartitions
	nSub8x8Partitions = partitionTTopSplit
)

const (
	bs128x128 = iota
	bs128x64
	bs64x128
	bs64x64
	bs64x32
	bs64x16
	bs32x64
	bs32x32
	bs32x16
	bs32x8
	bs16x64
	bs16x32
	bs16x16
	bs16x8
	bs16x4
	bs8x32
	bs8x16
	bs8x8
	bs8x4
	bs4x16
	bs4x8
	bs4x4
	nBsSizes
)

const (
	filter2d8tapRegular = iota
	filter2d8tapRegularSmooth
	filter2d8tapRegularSharp
	filter2d8tapSharpRegular
	filter2d8tapSharpSmooth
	filter2d8tapSharp
	filter2d8tapSmoothRegular
	filter2d8tapSmooth
	filter2d8tapSmoothSharp
	filter2dBilinear
	n2dFilters
)

const (
	mvJointZero = iota
	mvJointH
	mvJointV
	mvJointHv
	nMvJoints
)

const (
	nearestmv = iota
	nearmv
	globalmv
	newmv
	nInterPredModes
)

const (
	nearestDrl = iota
	nearerDrl
	nearDrl
	nearishDrl
)

const (
	nearestmvNearestmv = iota
	nearmvNearmv
	nearestmvNewmv
	newmvNearestmv
	nearmvNewmv
	newmvNearmv
	globalmvGlobalmv
	newmvNewmv
	nCompInterPredModes
)

const (
	compInterNone = iota
	compInterWeightedAvg
	compInterAvg
	compInterSeg
	compInterWedge
)

const (
	interIntraNone = iota
	interIntraBlend
	interIntraWedge
)

const (
	mmTranslation = iota
	mmObmc
	mmWarp
)

type mv struct {
	y, x int16
}

func (m mv) n() uint32 {
	return uint32(uint16(m.y)) | uint32(uint16(m.x))<<16
}

type txfmInfo struct {
	w, h, lw, lh, min, max, sub, ctx uint8
}
