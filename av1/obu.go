/*
Package av1 decodes the AV1 bitstream.

It is a port of dav1d and decodes bit-exactly against it, in pure Go with no
dependencies: 8, 10 and 12 bits, 4:2:0, 4:2:2, 4:4:4 and monochrome, intra and
inter prediction, film grain, superres, and scalable streams.

A [Decoder] is fed whole temporal units, which is what a container hands over:

	d := av1.Decoder{}

	for _, tu := range temporalUnits {
		pics, err := d.DecodeOBUs(tu)
		...
	}

[Decoder.DecodeOBUs] returns the pictures a temporal unit shows, in output
order, and is the simpler of the two interfaces. It decodes one frame at a
time, which is all a still image needs.

# Video

For video, [Decoder.Send] and [Decoder.Receive] keep several frames in flight:

	d := av1.Decoder{}

	for _, tu := range temporalUnits {
		if err := d.Send(tu); err != nil {
			...
		}
		pic, err := d.Receive()
		...
	}

Receive hands back pictures in output order and returns a nil picture once the
queue is empty. Call the two from one goroutine. [Decoder.Flush] drops what is
queued and forgets the reference frames, which is what seeking needs.

# Threads

[Decoder.Threads] bounds the goroutines decoding a frame's tiles and defaults
to GOMAXPROCS. It only helps streams that carry more than one tile.

[Decoder.FrameThreads] bounds how many frames decode at once, each holding its
own buffers. It is the one that helps most video, and only Send and Receive
benefit from it.

# Pictures

A [Picture] holds its planes as bytes, native-endian uint16 samples above 8
bits, with strides in bytes. [Picture.Release] hands the memory back to the
decoder to be reused by a later frame; it is optional, and a picture that is
never released is collected as any other value would be.

# Assembly

SIMD support for amd64, arm64 and riscv64 is written in Go assembly and
selected at run time from what the CPU reports. Building with the noasm tag
leaves pure Go everywhere, which decodes the same pixels more slowly.
*/
package av1

import (
	"errors"
	"math"
	"runtime"
	"sync"
)

var errInvalid = errors.New("av1: invalid bitstream")

// ErrFrameTooLarge is returned when a frame's area exceeds
// Decoder.FrameSizeLimit.
var ErrFrameTooLarge = errors.New("av1: frame exceeds size limit")

// DefaultFrameSizeLimit is the frame area, in pixels, accepted when
// Decoder.FrameSizeLimit is zero.
const DefaultFrameSizeLimit = min(16384*16384, math.MaxInt>>6)

var defaultModeRefDeltas = loopfilterModeRefDeltas{
	modeDelta: [2]int8{0, 0},
	refDelta:  [totalRefsPerFrame]int8{1, 0, 0, 0, -1, 0, -1, -1},
}

var defaultWmParams = warpedMotionParams{
	typ:    wmTypeIdentity,
	matrix: [6]int32{0, 0, 1 << 16, 0, 0, 1 << 16},
}

type refPicture[P pixel] struct {
	planes   [3][]P
	stride   [3]int
	w, h     int
	progress *frameProgress
	m        *picMem
}

func (p *refPicture[P]) mem() *picMem { return p.m }

// picHolder lets the reference slots count a frame's memory without knowing
// which pixel type it was decoded at.
type picHolder interface {
	mem() *picMem
}

// setRef replaces one reference slot, counting the memory in and out.
func (d *Decoder) setRef(i int, s refState) {
	if h, ok := d.refs[i].pic.(picHolder); ok {
		h.mem().release()
	}
	if h, ok := s.pic.(picHolder); ok {
		h.mem().acquire()
	}
	d.refs[i] = s
}

type refState struct {
	frameHdr *frameHeader
	out      *pendingFrame
	pic      any
	segmap   []uint8
	refmvs   []refmvsTemporalBlock
	refpoc   [7]uint8
	codedW   int
}

// pendingFrame is the output slot a frame is given when it is dispatched.
type pendingFrame struct {
	done chan struct{}
	pic  *Picture
	err  error
}

func newPendingFrame() *pendingFrame {
	return &pendingFrame{done: make(chan struct{})}
}

func (p *pendingFrame) finish(pic *Picture, err error) {
	p.pic, p.err = pic, err
	close(p.done)
}

func (p *pendingFrame) wait() (*Picture, error) {
	<-p.done

	return p.pic, p.err
}

type Decoder struct {
	// Threads bounds the goroutines used to decode a frame's tiles.
	// Zero means GOMAXPROCS.
	Threads int

	// OperatingPoint selects a scalable stream's operating point.
	OperatingPoint int

	// FrameThreads bounds how many frames decode at once, each holding its
	// own buffers. Zero derives a default from GOMAXPROCS, one decodes a
	// frame at a time. Only Send and Receive benefit from it.
	FrameThreads int

	// FrameSizeLimit bounds a frame's area in pixels, so that a corrupt or
	// hostile header cannot ask for an unbounded allocation. Zero means
	// DefaultFrameSizeLimit; a negative value removes the limit.
	FrameSizeLimit int

	seqHdr   *sequenceHeader
	frameHdr *frameHeader
	refs     [numRefFrames]refState

	strictStdCompliance bool
	operatingPointIdc   uint
	nTiles              int

	headersOnly  bool
	seqHdrHook   func(*sequenceHeader)
	frameHdrHook func(*frameHeader)

	tiles []tileGroup
	cdf   [numRefFrames]*cdfPromise
	queue []*pendingFrame
	out   []*Picture

	tc sync.Pool
	fb sync.Pool

	sem  chan struct{}
	wg   sync.WaitGroup
	pool picPool
}

func seqHdrEqual(a, b *sequenceHeader) bool {
	x, y := *a, *b
	x.operatingParameterInfo = [maxOperatingPoints]sequenceHeaderOperatingParameterInfo{}
	y.operatingParameterInfo = x.operatingParameterInfo

	return x == y
}

func isInterOrSwitch(hdr *frameHeader) bool {
	return hdr.frameType&1 != 0
}

func isKeyOrIntra(hdr *frameHeader) bool {
	return hdr.frameType&1 == 0
}

func getPocDiff(orderHintNBits, poc0, poc1 int) int {
	if orderHintNBits == 0 {
		return 0
	}
	mask := 1 << (orderHintNBits - 1)
	diff := poc0 - poc1

	return diff&(mask-1) - diff&mask
}

func tileLog2(sz, tgt int) int {
	k := 0
	for sz<<k < tgt {
		k++
	}

	return k
}

func checkTrailingBits(gb *getBits, strict bool) error {
	trailingOneBit := gb.bit()

	if gb.err {
		return errInvalid
	}

	if !strict {
		return nil
	}

	if trailingOneBit == 0 || gb.state != 0 {
		return errInvalid
	}

	size := gb.remaining()
	for size > 0 && gb.data[gb.index+size-1] == 0 {
		size--
	}

	if size != 0 {
		return errInvalid
	}

	return nil
}

func parseSeqHdr(hdr *sequenceHeader, gb *getBits, strict bool) error {
	*hdr = sequenceHeader{}

	hdr.profile = uint8(gb.bits(3))
	if hdr.profile > 2 {
		return errInvalid
	}

	hdr.stillPicture = uint8(gb.bit())
	hdr.reducedStillPictureHeader = uint8(gb.bit())
	if hdr.reducedStillPictureHeader != 0 && hdr.stillPicture == 0 {
		return errInvalid
	}

	if hdr.reducedStillPictureHeader != 0 {
		hdr.numOperatingPoints = 1
		hdr.operatingPoints[0].majorLevel = uint8(gb.bits(3))
		hdr.operatingPoints[0].minorLevel = uint8(gb.bits(2))
		hdr.operatingPoints[0].initialDisplayDelay = 10
	} else {
		hdr.timingInfoPresent = uint8(gb.bit())
		if hdr.timingInfoPresent != 0 {
			hdr.numUnitsInTick = gb.bits(32)
			hdr.timeScale = gb.bits(32)
			if strict && (hdr.numUnitsInTick == 0 || hdr.timeScale == 0) {
				return errInvalid
			}
			hdr.equalPictureInterval = uint8(gb.bit())
			if hdr.equalPictureInterval != 0 {
				numTicksPerPicture := gb.vlc()
				if numTicksPerPicture == 0xffffffff {
					return errInvalid
				}
				hdr.numTicksPerPicture = numTicksPerPicture + 1
			}

			hdr.decoderModelInfoPresent = uint8(gb.bit())
			if hdr.decoderModelInfoPresent != 0 {
				hdr.encoderDecoderBufferDelayLength = uint8(gb.bits(5)) + 1
				hdr.numUnitsInDecodingTick = gb.bits(32)
				if strict && hdr.numUnitsInDecodingTick == 0 {
					return errInvalid
				}
				hdr.bufferRemovalDelayLength = uint8(gb.bits(5)) + 1
				hdr.framePresentationDelayLength = uint8(gb.bits(5)) + 1
			}
		}

		hdr.displayModelInfoPresent = uint8(gb.bit())
		hdr.numOperatingPoints = uint8(gb.bits(5)) + 1
		for i := range int(hdr.numOperatingPoints) {
			op := &hdr.operatingPoints[i]
			op.idc = uint16(gb.bits(12))
			if op.idc != 0 && (op.idc&0xff == 0 || op.idc&0xf00 == 0) {
				return errInvalid
			}
			op.majorLevel = 2 + uint8(gb.bits(3))
			op.minorLevel = uint8(gb.bits(2))
			if op.majorLevel > 3 {
				op.tier = uint8(gb.bit())
			}
			if hdr.decoderModelInfoPresent != 0 {
				op.decoderModelParamPresent = uint8(gb.bit())
				if op.decoderModelParamPresent != 0 {
					opi := &hdr.operatingParameterInfo[i]
					opi.decoderBufferDelay = gb.bits(int(hdr.encoderDecoderBufferDelayLength))
					opi.encoderBufferDelay = gb.bits(int(hdr.encoderDecoderBufferDelayLength))
					opi.lowDelayMode = uint8(gb.bit())
				}
			}
			if hdr.displayModelInfoPresent != 0 {
				op.displayModelParamPresent = uint8(gb.bit())
			}
			if op.displayModelParamPresent != 0 {
				op.initialDisplayDelay = uint8(gb.bits(4)) + 1
			} else {
				op.initialDisplayDelay = 10
			}
		}
	}

	hdr.widthNBits = uint8(gb.bits(4)) + 1
	hdr.heightNBits = uint8(gb.bits(4)) + 1
	hdr.maxWidth = int(gb.bits(int(hdr.widthNBits))) + 1
	hdr.maxHeight = int(gb.bits(int(hdr.heightNBits))) + 1

	if hdr.reducedStillPictureHeader == 0 {
		hdr.frameIDNumbersPresent = uint8(gb.bit())
		if hdr.frameIDNumbersPresent != 0 {
			hdr.deltaFrameIDNBits = uint8(gb.bits(4)) + 2
			hdr.frameIDNBits = uint8(gb.bits(3)) + hdr.deltaFrameIDNBits + 1
		}
	}

	hdr.sb128 = uint8(gb.bit())
	hdr.filterIntra = uint8(gb.bit())
	hdr.intraEdgeFilter = uint8(gb.bit())
	if hdr.reducedStillPictureHeader != 0 {
		hdr.screenContentTools = adaptive
		hdr.forceIntegerMv = adaptive
	} else {
		hdr.interIntra = uint8(gb.bit())
		hdr.maskedCompound = uint8(gb.bit())
		hdr.warpedMotion = uint8(gb.bit())
		hdr.dualFilter = uint8(gb.bit())
		hdr.orderHint = uint8(gb.bit())
		if hdr.orderHint != 0 {
			hdr.jntComp = uint8(gb.bit())
			hdr.refFrameMvs = uint8(gb.bit())
		}
		if gb.bit() != 0 {
			hdr.screenContentTools = adaptive
		} else {
			hdr.screenContentTools = int(gb.bit())
		}
		if hdr.screenContentTools != 0 {
			if gb.bit() != 0 {
				hdr.forceIntegerMv = adaptive
			} else {
				hdr.forceIntegerMv = int(gb.bit())
			}
		} else {
			hdr.forceIntegerMv = 2
		}
		if hdr.orderHint != 0 {
			hdr.orderHintNBits = uint8(gb.bits(3)) + 1
		}
	}
	hdr.superRes = uint8(gb.bit())
	hdr.cdef = uint8(gb.bit())
	hdr.restoration = uint8(gb.bit())

	hdr.hbd = uint8(gb.bit())
	if hdr.profile == 2 && hdr.hbd != 0 {
		hdr.hbd += uint8(gb.bit())
	}
	if hdr.profile != 1 {
		hdr.monochrome = uint8(gb.bit())
	}
	hdr.colorDescriptionPresent = uint8(gb.bit())
	if hdr.colorDescriptionPresent != 0 {
		hdr.pri = int(gb.bits(8))
		hdr.trc = int(gb.bits(8))
		hdr.mtrx = int(gb.bits(8))
	} else {
		hdr.pri = colorPriUnknown
		hdr.trc = trcUnknown
		hdr.mtrx = mcUnknown
	}
	switch {
	case hdr.monochrome != 0:
		hdr.colorRange = uint8(gb.bit())
		hdr.layout = pixelLayoutI400
		hdr.ssHor, hdr.ssVer = 1, 1
		hdr.chr = chrUnknown
	case hdr.pri == colorPriBt709 && hdr.trc == trcSrgb && hdr.mtrx == mcIdentity:
		hdr.layout = pixelLayoutI444
		hdr.colorRange = 1
		if hdr.profile != 1 && !(hdr.profile == 2 && hdr.hbd == 2) {
			return errInvalid
		}
	default:
		hdr.colorRange = uint8(gb.bit())
		switch hdr.profile {
		case 0:
			hdr.layout = pixelLayoutI420
			hdr.ssHor, hdr.ssVer = 1, 1
		case 1:
			hdr.layout = pixelLayoutI444
		case 2:
			if hdr.hbd == 2 {
				hdr.ssHor = uint8(gb.bit())
				if hdr.ssHor != 0 {
					hdr.ssVer = uint8(gb.bit())
				}
			} else {
				hdr.ssHor = 1
			}
			switch {
			case hdr.ssHor == 0:
				hdr.layout = pixelLayoutI444
			case hdr.ssVer != 0:
				hdr.layout = pixelLayoutI420
			default:
				hdr.layout = pixelLayoutI422
			}
		}
		if hdr.ssHor&hdr.ssVer != 0 {
			hdr.chr = int(gb.bits(2))
		} else {
			hdr.chr = chrUnknown
		}
	}
	if strict && hdr.mtrx == mcIdentity && hdr.layout != pixelLayoutI444 {
		return errInvalid
	}
	if hdr.monochrome == 0 {
		hdr.separateUvDeltaQ = uint8(gb.bit())
	}

	hdr.filmGrainPresent = uint8(gb.bit())

	return checkTrailingBits(gb, strict)
}

func (d *Decoder) readFrameSize(gb *getBits, useRef bool) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	if useRef {
		for i := range refsPerFrame {
			if gb.bit() == 0 {
				continue
			}
			ref := d.refs[hdr.refidx[i]].frameHdr
			if ref == nil {
				return errInvalid
			}
			hdr.width[1] = ref.width[1]
			hdr.height = ref.height
			hdr.renderWidth = ref.renderWidth
			hdr.renderHeight = ref.renderHeight
			if seqhdr.superRes != 0 && gb.bit() != 0 {
				hdr.superRes.enabled = 1
			}
			if hdr.superRes.enabled != 0 {
				dd := 9 + int(gb.bits(3))
				hdr.superRes.widthScaleDenominator = uint8(dd)
				hdr.width[0] = max((hdr.width[1]*8+dd>>1)/dd, min(16, hdr.width[1]))
			} else {
				hdr.superRes.widthScaleDenominator = 8
				hdr.width[0] = hdr.width[1]
			}

			return nil
		}
	}

	if hdr.frameSizeOverride != 0 {
		hdr.width[1] = int(gb.bits(int(seqhdr.widthNBits))) + 1
		hdr.height = int(gb.bits(int(seqhdr.heightNBits))) + 1
	} else {
		hdr.width[1] = seqhdr.maxWidth
		hdr.height = seqhdr.maxHeight
	}
	if seqhdr.superRes != 0 && gb.bit() != 0 {
		hdr.superRes.enabled = 1
	}
	if hdr.superRes.enabled != 0 {
		dd := 9 + int(gb.bits(3))
		hdr.superRes.widthScaleDenominator = uint8(dd)
		hdr.width[0] = max((hdr.width[1]*8+dd>>1)/dd, min(16, hdr.width[1]))
	} else {
		hdr.superRes.widthScaleDenominator = 8
		hdr.width[0] = hdr.width[1]
	}
	hdr.haveRenderSize = uint8(gb.bit())
	if hdr.haveRenderSize != 0 {
		hdr.renderWidth = int(gb.bits(16)) + 1
		hdr.renderHeight = int(gb.bits(16)) + 1
	} else {
		hdr.renderWidth = hdr.width[1]
		hdr.renderHeight = hdr.height
	}

	return nil
}

func (d *Decoder) parseFrameHdr(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	if seqhdr.reducedStillPictureHeader == 0 {
		hdr.showExistingFrame = uint8(gb.bit())
	}
	if hdr.showExistingFrame != 0 {
		hdr.existingFrameIdx = uint8(gb.bits(3))
		if seqhdr.decoderModelInfoPresent != 0 && seqhdr.equalPictureInterval == 0 {
			hdr.framePresentationDelay = gb.bits(int(seqhdr.framePresentationDelayLength))
		}
		if seqhdr.frameIDNumbersPresent != 0 {
			hdr.frameID = gb.bits(int(seqhdr.frameIDNBits))
			refFrameHdr := d.refs[hdr.existingFrameIdx].frameHdr
			if refFrameHdr == nil || refFrameHdr.frameID != hdr.frameID {
				return errInvalid
			}
		}

		return nil
	}

	if seqhdr.reducedStillPictureHeader != 0 {
		hdr.frameType = frameTypeKey
		hdr.showFrame = 1
	} else {
		hdr.frameType = int(gb.bits(2))
		hdr.showFrame = uint8(gb.bit())
	}
	if hdr.showFrame != 0 {
		if seqhdr.decoderModelInfoPresent != 0 && seqhdr.equalPictureInterval == 0 {
			hdr.framePresentationDelay = gb.bits(int(seqhdr.framePresentationDelayLength))
		}
		if hdr.frameType != frameTypeKey {
			hdr.showableFrame = 1
		}
	} else {
		hdr.showableFrame = uint8(gb.bit())
	}
	if (hdr.frameType == frameTypeKey && hdr.showFrame != 0) ||
		hdr.frameType == frameTypeSwitch ||
		seqhdr.reducedStillPictureHeader != 0 || gb.bit() != 0 {
		hdr.errorResilientMode = 1
	}

	hdr.disableCdfUpdate = uint8(gb.bit())
	if seqhdr.screenContentTools == adaptive {
		hdr.allowScreenContentTools = uint8(gb.bit())
	} else {
		hdr.allowScreenContentTools = uint8(seqhdr.screenContentTools)
	}
	if hdr.allowScreenContentTools != 0 {
		if seqhdr.forceIntegerMv == adaptive {
			hdr.forceIntegerMv = uint8(gb.bit())
		} else {
			hdr.forceIntegerMv = uint8(seqhdr.forceIntegerMv)
		}
	}

	if isKeyOrIntra(hdr) {
		hdr.forceIntegerMv = 1
	}

	if seqhdr.frameIDNumbersPresent != 0 {
		hdr.frameID = gb.bits(int(seqhdr.frameIDNBits))
	}

	if seqhdr.reducedStillPictureHeader == 0 {
		if hdr.frameType == frameTypeSwitch {
			hdr.frameSizeOverride = 1
		} else {
			hdr.frameSizeOverride = uint8(gb.bit())
		}
	}
	if seqhdr.orderHint != 0 {
		hdr.frameOffset = uint8(gb.bits(int(seqhdr.orderHintNBits)))
	}
	if hdr.errorResilientMode == 0 && isInterOrSwitch(hdr) {
		hdr.primaryRefFrame = uint8(gb.bits(3))
	} else {
		hdr.primaryRefFrame = primaryRefNone
	}

	if seqhdr.decoderModelInfoPresent != 0 {
		hdr.bufferRemovalTimePresent = uint8(gb.bit())
		if hdr.bufferRemovalTimePresent != 0 {
			for i := range int(seqhdr.numOperatingPoints) {
				seqop := &seqhdr.operatingPoints[i]
				op := &hdr.operatingPoints[i]
				if seqop.decoderModelParamPresent == 0 {
					continue
				}
				inTemporalLayer := seqop.idc>>hdr.temporalID&1 != 0
				inSpatialLayer := seqop.idc>>(hdr.spatialID+8)&1 != 0
				if seqop.idc == 0 || (inTemporalLayer && inSpatialLayer) {
					op.bufferRemovalTime = gb.bits(int(seqhdr.bufferRemovalDelayLength))
				}
			}
		}
	}

	if isKeyOrIntra(hdr) {
		if hdr.frameType == frameTypeKey && hdr.showFrame != 0 {
			hdr.refreshFrameFlags = 0xff
		} else {
			hdr.refreshFrameFlags = uint8(gb.bits(8))
		}
		if hdr.refreshFrameFlags != 0xff && hdr.errorResilientMode != 0 && seqhdr.orderHint != 0 {
			for range numRefFrames {
				gb.bits(int(seqhdr.orderHintNBits))
			}
		}
		if d.strictStdCompliance &&
			hdr.frameType == frameTypeIntra && hdr.refreshFrameFlags == 0xff {
			return errInvalid
		}
		if err := d.readFrameSize(gb, false); err != nil {
			return err
		}
		if hdr.allowScreenContentTools != 0 && hdr.superRes.enabled == 0 {
			hdr.allowIntrabc = uint8(gb.bit())
		}
	} else {
		if hdr.frameType == frameTypeSwitch {
			hdr.refreshFrameFlags = 0xff
		} else {
			hdr.refreshFrameFlags = uint8(gb.bits(8))
		}
		if hdr.errorResilientMode != 0 && seqhdr.orderHint != 0 {
			for range numRefFrames {
				gb.bits(int(seqhdr.orderHintNBits))
			}
		}
		if seqhdr.orderHint != 0 {
			hdr.frameRefShortSignaling = uint8(gb.bit())
			if hdr.frameRefShortSignaling != 0 {
				if err := d.setRefShortSignaling(gb); err != nil {
					return err
				}
			}
		}
		for i := range refsPerFrame {
			if hdr.frameRefShortSignaling == 0 {
				hdr.refidx[i] = int8(gb.bits(3))
			}
			if seqhdr.frameIDNumbersPresent != 0 {
				deltaRefFrameID := gb.bits(int(seqhdr.deltaFrameIDNBits)) + 1
				refFrameID := (hdr.frameID + 1<<seqhdr.frameIDNBits - deltaRefFrameID) &
					(1<<seqhdr.frameIDNBits - 1)
				refFrameHdr := d.refs[hdr.refidx[i]].frameHdr
				if refFrameHdr == nil || refFrameHdr.frameID != refFrameID {
					return errInvalid
				}
			}
		}
		useRef := hdr.errorResilientMode == 0 && hdr.frameSizeOverride != 0
		if err := d.readFrameSize(gb, useRef); err != nil {
			return err
		}
		if hdr.forceIntegerMv == 0 {
			hdr.hp = uint8(gb.bit())
		}
		if gb.bit() != 0 {
			hdr.subpelFilterMode = filterSwitchable
		} else {
			hdr.subpelFilterMode = int(gb.bits(2))
		}
		hdr.switchableMotionMode = uint8(gb.bit())
		if hdr.errorResilientMode == 0 && seqhdr.refFrameMvs != 0 &&
			seqhdr.orderHint != 0 && isInterOrSwitch(hdr) {
			hdr.useRefFrameMvs = uint8(gb.bit())
		}
	}

	if seqhdr.reducedStillPictureHeader == 0 && hdr.disableCdfUpdate == 0 {
		if gb.bit() == 0 {
			hdr.refreshContext = 1
		}
	}

	if err := d.parseTiling(gb); err != nil {
		return err
	}
	d.parseQuant(gb)
	if err := d.parseSegmentation(gb); err != nil {
		return err
	}

	if hdr.quant.yac != 0 {
		hdr.delta.q.present = uint8(gb.bit())
		if hdr.delta.q.present != 0 {
			hdr.delta.q.resLog2 = uint8(gb.bits(2))
			if hdr.allowIntrabc == 0 {
				hdr.delta.lf.present = uint8(gb.bit())
				if hdr.delta.lf.present != 0 {
					hdr.delta.lf.resLog2 = uint8(gb.bits(2))
					hdr.delta.lf.multi = uint8(gb.bit())
				}
			}
		}
	}

	deltaLossless := hdr.quant.ydcDelta == 0 && hdr.quant.udcDelta == 0 &&
		hdr.quant.uacDelta == 0 && hdr.quant.vdcDelta == 0 && hdr.quant.vacDelta == 0
	hdr.allLossless = 1
	for i := range maxSegments {
		if hdr.segmentation.enabled != 0 {
			hdr.segmentation.qidx[i] = uint8(clipU8(int(hdr.quant.yac) +
				int(hdr.segmentation.segData.d[i].deltaQ)))
		} else {
			hdr.segmentation.qidx[i] = hdr.quant.yac
		}
		if hdr.segmentation.qidx[i] == 0 && deltaLossless {
			hdr.segmentation.lossless[i] = 1
		}
		hdr.allLossless &= hdr.segmentation.lossless[i]
	}

	if err := d.parseLoopfilter(gb); err != nil {
		return err
	}
	d.parseCdef(gb)
	d.parseRestoration(gb)

	if hdr.allLossless == 0 {
		if gb.bit() != 0 {
			hdr.txfmMode = txSwitchable
		} else {
			hdr.txfmMode = txLargest
		}
	}
	if isInterOrSwitch(hdr) {
		hdr.switchableCompRefs = uint8(gb.bit())
	}
	if err := d.deriveSkipMode(gb); err != nil {
		return err
	}
	if hdr.skipModeAllowed != 0 {
		hdr.skipModeEnabled = uint8(gb.bit())
	}
	if hdr.errorResilientMode == 0 && isInterOrSwitch(hdr) && seqhdr.warpedMotion != 0 {
		hdr.warpMotion = uint8(gb.bit())
	}
	hdr.reducedTxtpSet = uint8(gb.bit())

	for i := range refsPerFrame {
		hdr.gmv[i] = defaultWmParams
	}
	if isInterOrSwitch(hdr) {
		if err := d.parseGmv(gb); err != nil {
			return err
		}
	}

	return d.parseFilmGrain(gb)
}

func (d *Decoder) setRefShortSignaling(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	hdr.refidx[0] = int8(gb.bits(3))
	hdr.refidx[1], hdr.refidx[2] = -1, -1
	hdr.refidx[3] = int8(gb.bits(3))

	var frameOffset [numRefFrames + 1]int32

	earliestRef := -1
	earliestOffset := int32(math.MaxInt32)
	for i := range numRefFrames {
		refhdr := d.refs[i].frameHdr
		if refhdr == nil {
			return errInvalid
		}
		diff := int32(getPocDiff(int(seqhdr.orderHintNBits), int(refhdr.frameOffset), int(hdr.frameOffset)))
		frameOffset[i+1] = diff
		if diff < earliestOffset {
			earliestOffset = diff
			earliestRef = i
		}
	}
	const used = int32(math.MinInt32)
	frameOffset[hdr.refidx[0]+1] = used
	frameOffset[hdr.refidx[3]+1] = used

	refidx := -1
	for i, latestOffset := 0, int32(0); i < numRefFrames; i++ {
		hint := frameOffset[i+1]
		if hint >= latestOffset {
			latestOffset = hint
			refidx = i
		}
	}
	frameOffset[refidx+1] = used
	hdr.refidx[6] = int8(refidx)

	for i := 4; i < 6; i++ {
		earliest := uint32(math.MaxUint8)
		refidx = -1
		for j := range numRefFrames {
			hint := uint32(frameOffset[j+1])
			if hint < earliest {
				earliest = hint
				refidx = j
			}
		}
		frameOffset[refidx+1] = used
		hdr.refidx[i] = int8(refidx)
	}

	for i := 1; i < refsPerFrame; i++ {
		refidx = int(hdr.refidx[i])
		if refidx >= 0 {
			continue
		}
		latest := ^uint32(math.MaxUint8)
		for j := range numRefFrames {
			hint := uint32(frameOffset[j+1])
			if hint >= latest {
				latest = hint
				refidx = j
			}
		}
		frameOffset[refidx+1] = used
		if refidx >= 0 {
			hdr.refidx[i] = int8(refidx)
		} else {
			hdr.refidx[i] = int8(earliestRef)
		}
	}

	return nil
}

func (d *Decoder) parseTiling(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr
	t := &hdr.tiling

	t.uniform = uint8(gb.bit())
	sbszMin1 := (64 << seqhdr.sb128) - 1
	sbszLog2 := 6 + int(seqhdr.sb128)
	sbw := (hdr.width[0] + sbszMin1) >> sbszLog2
	sbh := (hdr.height + sbszMin1) >> sbszLog2
	maxTileWidthSb := 4096 >> sbszLog2
	maxTileAreaSb := 4096 * 2304 >> (2 * sbszLog2)
	t.minLog2Cols = uint8(tileLog2(maxTileWidthSb, sbw))
	t.maxLog2Cols = uint8(tileLog2(1, min(sbw, maxTileCols)))
	t.maxLog2Rows = uint8(tileLog2(1, min(sbh, maxTileRows)))
	minLog2Tiles := max(tileLog2(maxTileAreaSb, sbw*sbh), int(t.minLog2Cols))

	if t.uniform != 0 {
		for t.log2Cols = t.minLog2Cols; t.log2Cols < t.maxLog2Cols && gb.bit() != 0; t.log2Cols++ {
		}
		tileW := 1 + (sbw-1)>>t.log2Cols
		t.cols = 0
		for sbx := 0; sbx < sbw; sbx, t.cols = sbx+tileW, t.cols+1 {
			t.colStartSb[t.cols] = uint16(sbx)
		}
		t.minLog2Rows = uint8(max(minLog2Tiles-int(t.log2Cols), 0))

		for t.log2Rows = t.minLog2Rows; t.log2Rows < t.maxLog2Rows && gb.bit() != 0; t.log2Rows++ {
		}
		tileH := 1 + (sbh-1)>>t.log2Rows
		t.rows = 0
		for sby := 0; sby < sbh; sby, t.rows = sby+tileH, t.rows+1 {
			t.rowStartSb[t.rows] = uint16(sby)
		}
	} else {
		t.cols = 0
		widestTile, maxTileAreaSb := 0, sbw*sbh
		for sbx := 0; sbx < sbw && t.cols < maxTileCols; t.cols++ {
			tileWidthSb := min(sbw-sbx, maxTileWidthSb)
			tileW := 1
			if tileWidthSb > 1 {
				tileW = 1 + int(gb.uniform(uint32(tileWidthSb)))
			}
			t.colStartSb[t.cols] = uint16(sbx)
			sbx += tileW
			widestTile = max(widestTile, tileW)
		}
		t.log2Cols = uint8(tileLog2(1, int(t.cols)))
		if minLog2Tiles != 0 {
			maxTileAreaSb >>= minLog2Tiles + 1
		}
		maxTileHeightSb := max(maxTileAreaSb/widestTile, 1)

		t.rows = 0
		for sby := 0; sby < sbh && t.rows < maxTileRows; t.rows++ {
			tileHeightSb := min(sbh-sby, maxTileHeightSb)
			tileH := 1
			if tileHeightSb > 1 {
				tileH = 1 + int(gb.uniform(uint32(tileHeightSb)))
			}
			t.rowStartSb[t.rows] = uint16(sby)
			sby += tileH
		}
		t.log2Rows = uint8(tileLog2(1, int(t.rows)))
	}
	t.colStartSb[t.cols] = uint16(sbw)
	t.rowStartSb[t.rows] = uint16(sbh)
	if t.log2Cols != 0 || t.log2Rows != 0 {
		t.update = uint16(gb.bits(int(t.log2Cols) + int(t.log2Rows)))
		if t.update >= uint16(t.cols)*uint16(t.rows) {
			return errInvalid
		}
		t.nBytes = uint8(gb.bits(2)) + 1
	}

	return nil
}

func (d *Decoder) parseQuant(gb *getBits) {
	seqhdr := d.seqHdr
	q := &d.frameHdr.quant

	q.yac = uint8(gb.bits(8))
	if gb.bit() != 0 {
		q.ydcDelta = int8(gb.sbits(7))
	}
	if seqhdr.monochrome == 0 {
		diffUvDelta := uint32(0)
		if seqhdr.separateUvDeltaQ != 0 {
			diffUvDelta = gb.bit()
		}
		if gb.bit() != 0 {
			q.udcDelta = int8(gb.sbits(7))
		}
		if gb.bit() != 0 {
			q.uacDelta = int8(gb.sbits(7))
		}
		if diffUvDelta != 0 {
			if gb.bit() != 0 {
				q.vdcDelta = int8(gb.sbits(7))
			}
			if gb.bit() != 0 {
				q.vacDelta = int8(gb.sbits(7))
			}
		} else {
			q.vdcDelta = q.udcDelta
			q.vacDelta = q.uacDelta
		}
	}
	q.qm = uint8(gb.bit())
	if q.qm != 0 {
		q.qmY = uint8(gb.bits(4))
		q.qmU = uint8(gb.bits(4))
		if seqhdr.separateUvDeltaQ != 0 {
			q.qmV = uint8(gb.bits(4))
		} else {
			q.qmV = q.qmU
		}
	}
}

func (d *Decoder) parseSegmentation(gb *getBits) error {
	hdr := d.frameHdr
	s := &hdr.segmentation

	s.enabled = uint8(gb.bit())
	if s.enabled == 0 {
		for i := range maxSegments {
			s.segData.d[i].ref = -1
		}

		return nil
	}

	if hdr.primaryRefFrame == primaryRefNone {
		s.updateMap = 1
		s.updateData = 1
	} else {
		s.updateMap = uint8(gb.bit())
		if s.updateMap != 0 {
			s.temporal = uint8(gb.bit())
		}
		s.updateData = uint8(gb.bit())
	}

	if s.updateData == 0 {
		priRef := hdr.refidx[hdr.primaryRefFrame]
		if d.refs[priRef].frameHdr == nil {
			return errInvalid
		}
		s.segData = d.refs[priRef].frameHdr.segmentation.segData

		return nil
	}

	s.segData.lastActiveSegid = -1
	for i := range maxSegments {
		seg := &s.segData.d[i]
		if gb.bit() != 0 {
			seg.deltaQ = int16(gb.sbits(9))
			s.segData.lastActiveSegid = int8(i)
		}
		if gb.bit() != 0 {
			seg.deltaLfYV = int8(gb.sbits(7))
			s.segData.lastActiveSegid = int8(i)
		}
		if gb.bit() != 0 {
			seg.deltaLfYH = int8(gb.sbits(7))
			s.segData.lastActiveSegid = int8(i)
		}
		if gb.bit() != 0 {
			seg.deltaLfU = int8(gb.sbits(7))
			s.segData.lastActiveSegid = int8(i)
		}
		if gb.bit() != 0 {
			seg.deltaLfV = int8(gb.sbits(7))
			s.segData.lastActiveSegid = int8(i)
		}
		if gb.bit() != 0 {
			seg.ref = int8(gb.bits(3))
			s.segData.lastActiveSegid = int8(i)
			s.segData.preskip = 1
		} else {
			seg.ref = -1
		}
		if seg.skip = uint8(gb.bit()); seg.skip != 0 {
			s.segData.lastActiveSegid = int8(i)
			s.segData.preskip = 1
		}
		if seg.globalmv = uint8(gb.bit()); seg.globalmv != 0 {
			s.segData.lastActiveSegid = int8(i)
			s.segData.preskip = 1
		}
	}

	return nil
}

func (d *Decoder) parseLoopfilter(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr
	lf := &hdr.loopfilter

	if hdr.allLossless != 0 || hdr.allowIntrabc != 0 {
		lf.modeRefDeltaEnabled = 1
		lf.modeRefDeltaUpdate = 1
		lf.modeRefDeltas = defaultModeRefDeltas

		return nil
	}

	lf.levelY[0] = uint8(gb.bits(6))
	lf.levelY[1] = uint8(gb.bits(6))
	if seqhdr.monochrome == 0 && (lf.levelY[0] != 0 || lf.levelY[1] != 0) {
		lf.levelU = uint8(gb.bits(6))
		lf.levelV = uint8(gb.bits(6))
	}
	lf.sharpness = uint8(gb.bits(3))

	if hdr.primaryRefFrame == primaryRefNone {
		lf.modeRefDeltas = defaultModeRefDeltas
	} else {
		ref := hdr.refidx[hdr.primaryRefFrame]
		if d.refs[ref].frameHdr == nil {
			return errInvalid
		}
		lf.modeRefDeltas = d.refs[ref].frameHdr.loopfilter.modeRefDeltas
	}
	lf.modeRefDeltaEnabled = uint8(gb.bit())
	if lf.modeRefDeltaEnabled != 0 {
		lf.modeRefDeltaUpdate = uint8(gb.bit())
		if lf.modeRefDeltaUpdate != 0 {
			for i := range totalRefsPerFrame {
				if gb.bit() != 0 {
					lf.modeRefDeltas.refDelta[i] = int8(gb.sbits(7))
				}
			}
			for i := range 2 {
				if gb.bit() != 0 {
					lf.modeRefDeltas.modeDelta[i] = int8(gb.sbits(7))
				}
			}
		}
	}

	return nil
}

func (d *Decoder) parseCdef(gb *getBits) {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	if hdr.allLossless != 0 || seqhdr.cdef == 0 || hdr.allowIntrabc != 0 {
		return
	}

	hdr.cdef.damping = uint8(gb.bits(2)) + 3
	hdr.cdef.nBits = uint8(gb.bits(2))
	for i := range 1 << hdr.cdef.nBits {
		hdr.cdef.yStrength[i] = uint8(gb.bits(6))
		if seqhdr.monochrome == 0 {
			hdr.cdef.uvStrength[i] = uint8(gb.bits(6))
		}
	}
}

func (d *Decoder) parseRestoration(gb *getBits) {
	seqhdr := d.seqHdr
	hdr := d.frameHdr
	r := &hdr.restoration

	if (hdr.allLossless != 0 && hdr.superRes.enabled == 0) ||
		seqhdr.restoration == 0 || hdr.allowIntrabc != 0 {
		return
	}

	r.typ[0] = int(gb.bits(2))
	if seqhdr.monochrome == 0 {
		r.typ[1] = int(gb.bits(2))
		r.typ[2] = int(gb.bits(2))
	}

	if r.typ[0] != 0 || r.typ[1] != 0 || r.typ[2] != 0 {
		r.unitSize[0] = 6 + seqhdr.sb128
		if gb.bit() != 0 {
			r.unitSize[0]++
			if seqhdr.sb128 == 0 {
				r.unitSize[0] += uint8(gb.bit())
			}
		}
		r.unitSize[1] = r.unitSize[0]
		if (r.typ[1] != 0 || r.typ[2] != 0) && seqhdr.ssHor == 1 && seqhdr.ssVer == 1 {
			r.unitSize[1] -= uint8(gb.bit())
		}
	} else {
		r.unitSize[0] = 8
	}
}

func (d *Decoder) deriveSkipMode(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	if hdr.switchableCompRefs == 0 || !isInterOrSwitch(hdr) || seqhdr.orderHint == 0 {
		return nil
	}

	poc := int(hdr.frameOffset)
	offBefore, offAfter := -1, -1
	offBeforeIdx, offAfterIdx := 0, 0
	for i := range refsPerFrame {
		if d.refs[hdr.refidx[i]].frameHdr == nil {
			return errInvalid
		}
		refpoc := int(d.refs[hdr.refidx[i]].frameHdr.frameOffset)

		diff := getPocDiff(int(seqhdr.orderHintNBits), refpoc, poc)
		switch {
		case diff > 0:
			if offAfter < 0 || getPocDiff(int(seqhdr.orderHintNBits), offAfter, refpoc) > 0 {
				offAfter = refpoc
				offAfterIdx = i
			}
		case diff < 0 && (offBefore < 0 ||
			getPocDiff(int(seqhdr.orderHintNBits), refpoc, offBefore) > 0):
			offBefore = refpoc
			offBeforeIdx = i
		}
	}

	if (offBefore | offAfter) >= 0 {
		hdr.skipModeRefs[0] = int8(min(offBeforeIdx, offAfterIdx))
		hdr.skipModeRefs[1] = int8(max(offBeforeIdx, offAfterIdx))
		hdr.skipModeAllowed = 1

		return nil
	}

	if offBefore < 0 {
		return nil
	}

	offBefore2 := -1
	offBefore2Idx := 0
	for i := range refsPerFrame {
		if d.refs[hdr.refidx[i]].frameHdr == nil {
			return errInvalid
		}
		refpoc := int(d.refs[hdr.refidx[i]].frameHdr.frameOffset)
		if getPocDiff(int(seqhdr.orderHintNBits), refpoc, offBefore) >= 0 {
			continue
		}
		if offBefore2 < 0 || getPocDiff(int(seqhdr.orderHintNBits), refpoc, offBefore2) > 0 {
			offBefore2 = refpoc
			offBefore2Idx = i
		}
	}

	if offBefore2 >= 0 {
		hdr.skipModeRefs[0] = int8(min(offBeforeIdx, offBefore2Idx))
		hdr.skipModeRefs[1] = int8(max(offBeforeIdx, offBefore2Idx))
		hdr.skipModeAllowed = 1
	}

	return nil
}

func (d *Decoder) parseGmv(gb *getBits) error {
	hdr := d.frameHdr

	for i := range refsPerFrame {
		switch {
		case gb.bit() == 0:
			hdr.gmv[i].typ = wmTypeIdentity
		case gb.bit() != 0:
			hdr.gmv[i].typ = wmTypeRotZoom
		case gb.bit() != 0:
			hdr.gmv[i].typ = wmTypeTranslation
		default:
			hdr.gmv[i].typ = wmTypeAffine
		}

		if hdr.gmv[i].typ == wmTypeIdentity {
			continue
		}

		refGmv := &defaultWmParams
		if hdr.primaryRefFrame != primaryRefNone {
			priRef := hdr.refidx[hdr.primaryRefFrame]
			if d.refs[priRef].frameHdr == nil {
				return errInvalid
			}
			refGmv = &d.refs[priRef].frameHdr.gmv[i]
		}
		mat := &hdr.gmv[i].matrix
		refMat := &refGmv.matrix

		var bits, shift int
		if hdr.gmv[i].typ >= wmTypeRotZoom {
			mat[2] = 1<<16 + 2*gb.subexp((refMat[2]-1<<16)>>1, 12)
			mat[3] = 2 * gb.subexp(refMat[3]>>1, 12)
			bits, shift = 12, 10
		} else {
			bits = 9
			shift = 13
			if hdr.hp == 0 {
				bits--
				shift++
			}
		}

		if hdr.gmv[i].typ == wmTypeAffine {
			mat[4] = 2 * gb.subexp(refMat[4]>>1, 12)
			mat[5] = 1<<16 + 2*gb.subexp((refMat[5]-1<<16)>>1, 12)
		} else {
			mat[4] = -mat[3]
			mat[5] = mat[2]
		}

		mat[0] = gb.subexp(refMat[0]>>shift, uint32(bits)) * (1 << shift)
		mat[1] = gb.subexp(refMat[1]>>shift, uint32(bits)) * (1 << shift)
	}

	return nil
}

func (d *Decoder) parseFilmGrain(gb *getBits) error {
	seqhdr := d.seqHdr
	hdr := d.frameHdr

	if seqhdr.filmGrainPresent == 0 || (hdr.showFrame == 0 && hdr.showableFrame == 0) {
		return nil
	}

	hdr.filmGrain.present = uint8(gb.bit())
	if hdr.filmGrain.present == 0 {
		return nil
	}

	seed := gb.bits(16)
	if hdr.frameType != frameTypeInter || gb.bit() != 0 {
		hdr.filmGrain.update = 1
	}
	if hdr.filmGrain.update == 0 {
		refidx := int(gb.bits(3))
		i := 0
		for ; i < refsPerFrame; i++ {
			if int(hdr.refidx[i]) == refidx {
				break
			}
		}
		if i == refsPerFrame || d.refs[refidx].frameHdr == nil {
			return errInvalid
		}
		hdr.filmGrain.data = d.refs[refidx].frameHdr.filmGrain.data
		hdr.filmGrain.data.seed = seed

		return nil
	}

	fgd := &hdr.filmGrain.data
	fgd.seed = seed

	fgd.numYPoints = int(gb.bits(4))
	if fgd.numYPoints > 14 {
		return errInvalid
	}
	for i := range fgd.numYPoints {
		fgd.yPoints[i][0] = uint8(gb.bits(8))
		if i != 0 && fgd.yPoints[i-1][0] >= fgd.yPoints[i][0] {
			return errInvalid
		}
		fgd.yPoints[i][1] = uint8(gb.bits(8))
	}

	if seqhdr.monochrome == 0 {
		fgd.chromaScalingFromLuma = int(gb.bit())
	}
	if seqhdr.monochrome != 0 || fgd.chromaScalingFromLuma != 0 ||
		(seqhdr.ssVer == 1 && seqhdr.ssHor == 1 && fgd.numYPoints == 0) {
		fgd.numUvPoints[0], fgd.numUvPoints[1] = 0, 0
	} else {
		for pl := range 2 {
			fgd.numUvPoints[pl] = int(gb.bits(4))
			if fgd.numUvPoints[pl] > 10 {
				return errInvalid
			}
			for i := range fgd.numUvPoints[pl] {
				fgd.uvPoints[pl][i][0] = uint8(gb.bits(8))
				if i != 0 && fgd.uvPoints[pl][i-1][0] >= fgd.uvPoints[pl][i][0] {
					return errInvalid
				}
				fgd.uvPoints[pl][i][1] = uint8(gb.bits(8))
			}
		}
	}

	if seqhdr.ssHor == 1 && seqhdr.ssVer == 1 &&
		(fgd.numUvPoints[0] != 0) != (fgd.numUvPoints[1] != 0) {
		return errInvalid
	}

	fgd.scalingShift = int(gb.bits(2)) + 8
	fgd.arCoeffLag = int(gb.bits(2))
	numYPos := 2 * fgd.arCoeffLag * (fgd.arCoeffLag + 1)
	if fgd.numYPoints != 0 {
		for i := range numYPos {
			fgd.arCoeffsY[i] = int8(int(gb.bits(8)) - 128)
		}
	}
	for pl := range 2 {
		if fgd.numUvPoints[pl] == 0 && fgd.chromaScalingFromLuma == 0 {
			continue
		}
		numUvPos := numYPos
		if fgd.numYPoints != 0 {
			numUvPos++
		}
		for i := range numUvPos {
			fgd.arCoeffsUv[pl][i] = int8(int(gb.bits(8)) - 128)
		}
		if fgd.numYPoints == 0 {
			fgd.arCoeffsUv[pl][numUvPos] = 0
		}
	}
	fgd.arCoeffShift = uint64(gb.bits(2)) + 6
	fgd.grainScaleShift = int(gb.bits(2))
	for pl := range 2 {
		if fgd.numUvPoints[pl] == 0 {
			continue
		}
		fgd.uvMult[pl] = int(gb.bits(8)) - 128
		fgd.uvLumaMult[pl] = int(gb.bits(8)) - 128
		fgd.uvOffset[pl] = int(gb.bits(9)) - 256
	}
	fgd.overlapFlag = int(gb.bit())
	fgd.clipToRestrictedRange = int(gb.bit())

	return nil
}

func (d *Decoder) parseTileHdr(gb *getBits) (start, end int) {
	nTiles := int(d.frameHdr.tiling.cols) * int(d.frameHdr.tiling.rows)
	haveTilePos := uint32(0)
	if nTiles > 1 {
		haveTilePos = gb.bit()
	}

	if haveTilePos != 0 {
		nBits := int(d.frameHdr.tiling.log2Cols) + int(d.frameHdr.tiling.log2Rows)
		start = int(gb.bits(nBits))
		end = int(gb.bits(nBits))
	} else {
		end = nTiles - 1
	}

	return start, end
}

func (d *Decoder) checkFrameSize() error {
	lim := int64(d.FrameSizeLimit)
	if lim == 0 {
		lim = DefaultFrameSizeLimit
	}
	if lim < 0 {
		return nil
	}
	if int64(d.frameHdr.width[1])*int64(d.frameHdr.height) > lim {
		return ErrFrameTooLarge
	}

	return nil
}

func (d *Decoder) refreshRefs() {
	for i := range numRefFrames {
		if d.frameHdr.refreshFrameFlags&(1<<i) != 0 {
			d.refs[i].frameHdr = d.frameHdr
		}
	}
}

// parseOBU consumes one OBU from data and returns the number of bytes consumed.
func (d *Decoder) parseOBU(data []byte) (int, error) {
	var gb getBits
	gb.init(data)

	obuForbiddenBit := gb.bit()
	if d.strictStdCompliance && obuForbiddenBit != 0 {
		return 0, errInvalid
	}
	typ := gb.bits(4)
	hasExtension := gb.bit()
	hasLengthField := gb.bit()
	gb.bit()

	temporalID, spatialID := 0, 0
	if hasExtension != 0 {
		temporalID = int(gb.bits(3))
		spatialID = int(gb.bits(2))
		gb.bits(3)
	}

	if hasLengthField != 0 {
		l := int(gb.uleb128())
		if l > gb.remaining() {
			return 0, errInvalid
		}
		if !gb.setRemaining(l) {
			return 0, errInvalid
		}
	}
	if gb.err {
		return 0, errInvalid
	}

	if typ != obuSeqHdr && typ != obuTd && hasExtension != 0 && d.operatingPointIdc != 0 {
		inTemporalLayer := d.operatingPointIdc>>temporalID&1 != 0
		inSpatialLayer := d.operatingPointIdc>>(spatialID+8)&1 != 0
		if !inTemporalLayer || !inSpatialLayer {
			return len(gb.data), nil
		}
	}

	switch typ {
	case obuSeqHdr:
		var seqHdr sequenceHeader
		if err := parseSeqHdr(&seqHdr, &gb, d.strictStdCompliance); err != nil {
			return 0, err
		}

		opIdx := 0
		if d.OperatingPoint < int(seqHdr.numOperatingPoints) {
			opIdx = d.OperatingPoint
		}
		d.operatingPointIdc = uint(seqHdr.operatingPoints[opIdx].idc)

		if d.seqHdr == nil {
			d.frameHdr = nil
		} else if !seqHdrEqual(&seqHdr, d.seqHdr) {
			d.frameHdr = nil
			for i := range numRefFrames {
				d.setRef(i, refState{})
				d.cdf[i] = nil
			}
		}
		d.seqHdr = &seqHdr
		if d.seqHdrHook != nil {
			d.seqHdrHook(&seqHdr)
		}

	case obuRedundantFrameHdr, obuFrame, obuFrameHdr:
		if typ == obuRedundantFrameHdr && d.frameHdr != nil {
			break
		}
		if d.seqHdr == nil {
			return 0, errInvalid
		}
		d.frameHdr = &frameHeader{
			temporalID: uint8(temporalID),
			spatialID:  uint8(spatialID),
		}
		d.tiles = d.tiles[:0]
		if err := d.parseFrameHdr(&gb); err != nil {
			d.frameHdr = nil

			return 0, err
		}
		if err := d.checkFrameSize(); err != nil {
			d.frameHdr = nil

			return 0, err
		}
		if d.frameHdrHook != nil {
			d.frameHdrHook(d.frameHdr)
		}
		d.nTiles = 0
		if typ != obuFrame {
			if err := checkTrailingBits(&gb, d.strictStdCompliance); err != nil {
				d.frameHdr = nil

				return 0, err
			}

			break
		}
		if d.frameHdr.showExistingFrame != 0 {
			d.frameHdr = nil

			return 0, errInvalid
		}
		gb.bytealign()

		fallthrough

	case obuTileGrp:
		if d.frameHdr == nil {
			return 0, errInvalid
		}
		start, end := d.parseTileHdr(&gb)
		gb.bytealign()
		if gb.err {
			return 0, errInvalid
		}
		if start > end || start != d.nTiles {
			d.nTiles = 0
			d.tiles = d.tiles[:0]

			return 0, errInvalid
		}
		d.tiles = append(d.tiles, tileGroup{
			data:  gb.data[gb.index:],
			start: start,
			end:   end,
		})
		d.nTiles += 1 + end - start
	}

	if d.frameHdr != nil {
		if d.frameHdr.showExistingFrame != 0 {
			err := d.showExistingFrame()
			d.frameHdr = nil
			if err != nil {
				return 0, err
			}
		} else if d.nTiles == int(d.frameHdr.tiling.cols)*int(d.frameHdr.tiling.rows) {
			if err := d.submitFrame(); err != nil {
				d.frameHdr = nil
				d.nTiles = 0
				d.tiles = d.tiles[:0]

				return 0, err
			}
			d.refreshRefs()
			d.frameHdr = nil
			d.nTiles = 0
			d.tiles = d.tiles[:0]
		}
	}

	return len(gb.data), nil
}

func (d *Decoder) showExistingFrame() error {
	r := d.frameHdr.existingFrameIdx
	ref := &d.refs[r]
	if ref.frameHdr == nil {
		if d.headersOnly {
			return nil
		}

		return errInvalid
	}

	if !d.headersOnly {
		if ref.out == nil {
			return errInvalid
		}
		d.queue = append(d.queue, ref.out)
	}

	if ref.frameHdr.frameType != frameTypeKey {
		return nil
	}

	for i := range numRefFrames {
		if uint8(i) == r {
			continue
		}
		s := *ref
		s.refmvs = nil
		d.setRef(i, s)
		d.cdf[i] = d.cdf[r]
	}

	return nil
}

// parseSequenceHeader parses a sequence header OBU from a raw AV1 bitstream.
func parseSequenceHeader(data []byte) error {
	if len(data) == 0 {
		return errInvalid
	}

	var gb getBits
	gb.init(data)

	var hdr sequenceHeader
	found := false

	for {
		gb.bit()
		typ := gb.bits(4)
		hasExtension := gb.bit()
		hasLengthField := gb.bit()
		gb.bits(1 + 8*int(hasExtension))

		obuEnd := len(gb.data)
		if hasLengthField != 0 {
			l := int(gb.uleb128())
			if l > obuEnd-gb.index {
				return errInvalid
			}
			obuEnd = gb.index + l
		}

		if typ == obuSeqHdr {
			if err := parseSeqHdr(&hdr, &gb, false); err != nil {
				return err
			}
			if gb.index > obuEnd {
				return errInvalid
			}
			gb.bytealign()
			found = true
		}

		if gb.err {
			return errInvalid
		}
		gb.index = obuEnd

		if gb.index >= len(gb.data) {
			break
		}
	}

	if !found {
		return errInvalid
	}

	return nil
}

func (d *Decoder) submitFrame() error {
	hdr := d.frameHdr

	if d.headersOnly {
		return nil
	}

	var in *cdfPromise
	if hdr.primaryRefFrame != primaryRefNone {
		if in = d.cdf[hdr.refidx[hdr.primaryRefFrame]]; in == nil {
			return errInvalid
		}
	}
	out := newCdfPromise()

	p := newPendingFrame()

	var fd frameDecoder
	var err error
	if d.seqHdr.hbd != 0 {
		fd, err = setupFrame[uint16, int32](d, d.tiles, p)
	} else {
		fd, err = setupFrame[uint8, int16](d, d.tiles, p)
	}
	if err != nil {
		return err
	}

	if hdr.showFrame != 0 {
		d.queue = append(d.queue, p)
	}

	for i := range numRefFrames {
		if hdr.refreshFrameFlags&(1<<i) != 0 {
			d.cdf[i] = out
		}
	}

	d.dispatch(fd, p, in, out)

	return nil
}

const maxFrameThreads = 8

// frameThreads picks a frame count for n cores, kept well under n because tile
// and frame threading multiply.
func frameThreads(n int) int {
	f := 1
	for f*f < n {
		f++
	}

	return min(f, maxFrameThreads)
}

// dispatch hands a frame to a worker, blocking while FrameThreads frames are
// already in flight.
func (d *Decoder) dispatch(fd frameDecoder, p *pendingFrame, in, out *cdfPromise) {
	n := d.FrameThreads
	if n == 0 {
		n = frameThreads(runtime.GOMAXPROCS(0))
	}
	if n <= 1 {
		p.finish(fd.decode(in, out))

		return
	}

	if cap(d.sem) != n {
		d.wg.Wait()
		d.sem = make(chan struct{}, n)
	}

	d.sem <- struct{}{}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() { <-d.sem }()
		p.finish(fd.decode(in, out))
	}()
}

// Send decodes a temporal unit. The pictures it produces are queued for
// Receive, in output order.
func (d *Decoder) Send(data []byte) error {
	for len(data) > 0 {
		n, err := d.parseOBU(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return errInvalid
		}
		data = data[n:]
	}

	return nil
}

// Receive returns the next decoded picture, blocking until it is ready. It
// returns a nil picture when nothing more has been sent.
func (d *Decoder) Receive() (*Picture, error) {
	if len(d.queue) == 0 {
		return nil, nil
	}

	p := d.queue[0]
	d.queue = d.queue[1:]

	return p.wait()
}

// Flush drops every queued picture and forgets the reference frames, so that
// decoding can resume at the next keyframe.
func (d *Decoder) Flush() {
	d.wg.Wait()
	d.queue = nil

	d.seqHdr = nil
	d.frameHdr = nil
	d.tiles = d.tiles[:0]
	d.nTiles = 0
	for i := range numRefFrames {
		d.setRef(i, refState{})
		d.cdf[i] = nil
	}
	d.pool.drop()
}

// DecodeOBUs decodes a temporal unit and returns the pictures it shows, in output order.
func (d *Decoder) DecodeOBUs(data []byte) ([]*Picture, error) {
	d.out = d.out[:0]
	serr := d.Send(data)

	for {
		pic, err := d.Receive()
		if err != nil {
			d.queue = nil

			return nil, err
		}
		if pic == nil {
			break
		}
		d.out = append(d.out, pic)
	}

	if serr != nil {
		return nil, serr
	}

	return d.out, nil
}
