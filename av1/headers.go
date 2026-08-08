package av1

const (
	maxCdefStrengths   = 8
	maxOperatingPoints = 32
	maxTileCols        = 64
	maxTileRows        = 64
	maxSegments        = 8
	numRefFrames       = 8
	primaryRefNone     = 7
	refsPerFrame       = 7
	totalRefsPerFrame  = refsPerFrame + 1
)

const (
	obuSeqHdr            = 1
	obuTd                = 2
	obuFrameHdr          = 3
	obuTileGrp           = 4
	obuMetadata          = 5
	obuFrame             = 6
	obuRedundantFrameHdr = 7
	obuPadding           = 15
)

const (
	tx4x4Only = iota
	txLargest
	txSwitchable
	nTxModes
)

const (
	filterRegular8tap = iota
	filterSmooth8tap
	filterSharp8tap
	nSwitchableFilters
	filterBilinear   = nSwitchableFilters
	nFilters         = nSwitchableFilters + 1
	filterSwitchable = nFilters
)

const (
	adaptiveOff = 0
	adaptiveOn  = 1
	adaptive    = 2
)

const (
	restorationNone = iota
	restorationSwitchable
	restorationWiener
	restorationSgrproj
)

const (
	wmTypeIdentity = iota
	wmTypeTranslation
	wmTypeRotZoom
	wmTypeAffine
)

const (
	pixelLayoutI400 = iota
	pixelLayoutI420
	pixelLayoutI422
	pixelLayoutI444
)

const (
	frameTypeKey = iota
	frameTypeInter
	frameTypeIntra
	frameTypeSwitch
)

const (
	colorPriBt709    = 1
	colorPriUnknown  = 2
	colorPriBt470m   = 4
	colorPriBt470bg  = 5
	colorPriBt601    = 6
	colorPriSmpte240 = 7
	colorPriFilm     = 8
	colorPriBt2020   = 9
	colorPriXyz      = 10
	colorPriSmpte431 = 11
	colorPriSmpte432 = 12
	colorPriEbu3213  = 22
	colorPriReserved = 255
)

const (
	trcBt709        = 1
	trcUnknown      = 2
	trcBt470m       = 4
	trcBt470bg      = 5
	trcBt601        = 6
	trcSmpte240     = 7
	trcLinear       = 8
	trcLog100       = 9
	trcLog100Sqrt10 = 10
	trcIec61966     = 11
	trcBt1361       = 12
	trcSrgb         = 13
	trcBt202010bit  = 14
	trcBt202012bit  = 15
	trcSmpte2084    = 16
	trcSmpte428     = 17
	trcHlg          = 18
	trcReserved     = 255
)

const (
	mcIdentity   = 0
	mcBt709      = 1
	mcUnknown    = 2
	mcFcc        = 4
	mcBt470bg    = 5
	mcBt601      = 6
	mcSmpte240   = 7
	mcSmpteYcgco = 8
	mcBt2020Ncl  = 9
	mcBt2020Cl   = 10
	mcSmpte2085  = 11
	mcChromatNcl = 12
	mcChromatCl  = 13
	mcIctcp      = 14
	mcReserved   = 255
)

const (
	chrUnknown   = 0
	chrVertical  = 1
	chrColocated = 2
)

type warpedMotionParams struct {
	typ    int
	matrix [6]int32
	abcd   [4]int16
}

type sequenceHeaderOperatingPoint struct {
	majorLevel, minorLevel   uint8
	initialDisplayDelay      uint8
	idc                      uint16
	tier                     uint8
	decoderModelParamPresent uint8
	displayModelParamPresent uint8
}

type sequenceHeaderOperatingParameterInfo struct {
	decoderBufferDelay uint32
	encoderBufferDelay uint32
	lowDelayMode       uint8
}

type sequenceHeader struct {
	profile                         uint8
	maxWidth, maxHeight             int
	layout                          int
	pri                             int
	trc                             int
	mtrx                            int
	chr                             int
	hbd                             uint8
	colorRange                      uint8
	numOperatingPoints              uint8
	operatingPoints                 [maxOperatingPoints]sequenceHeaderOperatingPoint
	stillPicture                    uint8
	reducedStillPictureHeader       uint8
	timingInfoPresent               uint8
	numUnitsInTick                  uint32
	timeScale                       uint32
	equalPictureInterval            uint8
	numTicksPerPicture              uint32
	decoderModelInfoPresent         uint8
	encoderDecoderBufferDelayLength uint8
	numUnitsInDecodingTick          uint32
	bufferRemovalDelayLength        uint8
	framePresentationDelayLength    uint8
	displayModelInfoPresent         uint8
	widthNBits, heightNBits         uint8
	frameIDNumbersPresent           uint8
	deltaFrameIDNBits               uint8
	frameIDNBits                    uint8
	sb128                           uint8
	filterIntra                     uint8
	intraEdgeFilter                 uint8
	interIntra                      uint8
	maskedCompound                  uint8
	warpedMotion                    uint8
	dualFilter                      uint8
	orderHint                       uint8
	jntComp                         uint8
	refFrameMvs                     uint8
	screenContentTools              int
	forceIntegerMv                  int
	orderHintNBits                  uint8
	superRes                        uint8
	cdef                            uint8
	restoration                     uint8
	ssHor, ssVer, monochrome        uint8
	colorDescriptionPresent         uint8
	separateUvDeltaQ                uint8
	filmGrainPresent                uint8
	operatingParameterInfo          [maxOperatingPoints]sequenceHeaderOperatingParameterInfo
}

type segmentationData struct {
	deltaQ                                   int16
	deltaLfYV, deltaLfYH, deltaLfU, deltaLfV int8
	ref                                      int8
	skip                                     uint8
	globalmv                                 uint8
}

type segmentationDataSet struct {
	d               [maxSegments]segmentationData
	preskip         uint8
	lastActiveSegid int8
}

type loopfilterModeRefDeltas struct {
	modeDelta [2]int8
	refDelta  [totalRefsPerFrame]int8
}

type filmGrainData struct {
	seed                  uint32
	numYPoints            int
	yPoints               [14][2]uint8
	chromaScalingFromLuma int
	numUvPoints           [2]int
	uvPoints              [2][10][2]uint8
	scalingShift          int
	arCoeffLag            int
	arCoeffsY             [24]int8
	arCoeffsUv            [2][28]int8
	arCoeffShift          uint64
	grainScaleShift       int
	uvMult                [2]int
	uvLumaMult            [2]int
	uvOffset              [2]int
	overlapFlag           int
	clipToRestrictedRange int
}

type frameHeaderOperatingPoint struct {
	bufferRemovalTime uint32
}

type frameHeaderTiling struct {
	uniform                                  uint8
	nBytes                                   uint8
	minLog2Cols, maxLog2Cols, log2Cols, cols uint8
	minLog2Rows, maxLog2Rows, log2Rows, rows uint8
	colStartSb                               [maxTileCols + 1]uint16
	rowStartSb                               [maxTileRows + 1]uint16
	update                                   uint16
}

type frameHeaderQuant struct {
	yac                                    uint8
	ydcDelta                               int8
	udcDelta, uacDelta, vdcDelta, vacDelta int8
	qm, qmY, qmU, qmV                      uint8
}

type frameHeaderSegmentation struct {
	enabled, updateMap, temporal, updateData uint8
	segData                                  segmentationDataSet
	lossless                                 [maxSegments]uint8
	qidx                                     [maxSegments]uint8
}

type frameHeaderDelta struct {
	q struct {
		present uint8
		resLog2 uint8
	}
	lf struct {
		present uint8
		resLog2 uint8
		multi   uint8
	}
}

type frameHeaderLoopfilter struct {
	levelY              [2]uint8
	levelU, levelV      uint8
	modeRefDeltaEnabled uint8
	modeRefDeltaUpdate  uint8
	modeRefDeltas       loopfilterModeRefDeltas
	sharpness           uint8
}

type frameHeaderCdef struct {
	damping    uint8
	nBits      uint8
	yStrength  [maxCdefStrengths]uint8
	uvStrength [maxCdefStrengths]uint8
}

type frameHeaderRestoration struct {
	typ      [3]int
	unitSize [2]uint8
}

type frameHeaderSuperRes struct {
	widthScaleDenominator uint8
	enabled               uint8
}

type frameHeaderFilmGrain struct {
	data            filmGrainData
	present, update uint8
}

type frameHeader struct {
	filmGrain                 frameHeaderFilmGrain
	frameType                 int
	width                     [2]int
	height                    int
	frameOffset               uint8
	temporalID                uint8
	spatialID                 uint8
	showExistingFrame         uint8
	existingFrameIdx          uint8
	frameID                   uint32
	framePresentationDelay    uint32
	showFrame                 uint8
	showableFrame             uint8
	errorResilientMode        uint8
	disableCdfUpdate          uint8
	allowScreenContentTools   uint8
	forceIntegerMv            uint8
	frameSizeOverride         uint8
	primaryRefFrame           uint8
	bufferRemovalTimePresent  uint8
	operatingPoints           [maxOperatingPoints]frameHeaderOperatingPoint
	refreshFrameFlags         uint8
	renderWidth, renderHeight int
	superRes                  frameHeaderSuperRes
	haveRenderSize            uint8
	allowIntrabc              uint8
	frameRefShortSignaling    uint8
	refidx                    [refsPerFrame]int8
	hp                        uint8
	subpelFilterMode          int
	switchableMotionMode      uint8
	useRefFrameMvs            uint8
	refreshContext            uint8
	tiling                    frameHeaderTiling
	quant                     frameHeaderQuant
	segmentation              frameHeaderSegmentation
	delta                     frameHeaderDelta
	allLossless               uint8
	loopfilter                frameHeaderLoopfilter
	cdef                      frameHeaderCdef
	restoration               frameHeaderRestoration
	txfmMode                  int
	switchableCompRefs        uint8
	skipModeAllowed           uint8
	skipModeEnabled           uint8
	skipModeRefs              [2]int8
	warpMotion                uint8
	reducedTxtpSet            uint8
	gmv                       [refsPerFrame]warpedMotionParams
}
