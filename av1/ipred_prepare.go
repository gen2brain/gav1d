package av1

import "math/bits"

const (
	edgeI444TopHasRight   = 1 << 0
	edgeI422TopHasRight   = 1 << 1
	edgeI420TopHasRight   = 1 << 2
	edgeI444LeftHasBottom = 1 << 3
	edgeI422LeftHasBottom = 1 << 4
	edgeI420LeftHasBottom = 1 << 5

	edgeAllTopHasRight   = edgeI444TopHasRight | edgeI422TopHasRight | edgeI420TopHasRight
	edgeAllLeftHasBottom = edgeI444LeftHasBottom | edgeI422LeftHasBottom | edgeI420LeftHasBottom
	edgeAllTrAndBl       = edgeAllTopHasRight | edgeAllLeftHasBottom
)

const (
	angleUseEdgeFilterFlag = 1024
	angleSmoothEdgeFlag    = 512
)

var av1ModeConv = [nIntraPredModes][2][2]uint8{
	dcPred:    {{dc128Pred, topDcPred}, {leftDcPred, dcPred}},
	paethPred: {{dc128Pred, vertPred}, {horPred, paethPred}},
}

var av1ModeToAngleMap = [8]int{90, 180, 45, 135, 113, 157, 203, 67}

type intraEdgeNeeds struct {
	left, top, topleft, topright, bottomleft bool
}

var av1IntraPredictionEdges = [nImplIntraPredModes]intraEdgeNeeds{
	dcPred:      {top: true, left: true},
	vertPred:    {top: true},
	horPred:     {left: true},
	leftDcPred:  {left: true},
	topDcPred:   {top: true},
	dc128Pred:   {},
	z1Pred:      {top: true, topright: true, topleft: true},
	z2Pred:      {left: true, top: true, topleft: true},
	z3Pred:      {left: true, bottomleft: true, topleft: true},
	smoothPred:  {left: true, top: true},
	smoothVPred: {left: true, top: true},
	smoothHPred: {left: true, top: true},
	paethPred:   {left: true, top: true, topleft: true},
	filterPred:  {left: true, top: true, topleft: true},
}

func fill[P pixel](dst []P, v P) {
	for i := range dst {
		dst[i] = v
	}
}

func prepareIntraEdges[P pixel](x int, haveLeft bool, y int, haveTop bool,
	w, h, edgeFlags int, dst []P, dstOff, stride int,
	sbEdge []P, sbEdgeOff int, mode int, angle *int, tw, th int,
	filterEdge bool, tlOut []P, tlOutOff int, bitdepthMax int32,
) int {
	bitdepth := bitdepthFromMax(bitdepthMax)

	switch mode {
	case vertPred, horPred, diagDownLeftPred, diagDownRightPred,
		vertRightPred, horDownPred, horUpPred, vertLeftPred:
		*angle = av1ModeToAngleMap[mode-vertPred] + 3**angle

		switch {
		case *angle <= 90:
			if *angle < 90 && haveTop {
				mode = z1Pred
			} else {
				mode = vertPred
			}
		case *angle < 180:
			mode = z2Pred
		default:
			if *angle > 180 && haveLeft {
				mode = z3Pred
			} else {
				mode = horPred
			}
		}
	case dcPred, paethPred:
		mode = int(av1ModeConv[mode][b2i(haveLeft)][b2i(haveTop)])
	}

	needs := &av1IntraPredictionEdges[mode]

	var dstTop []P
	var dstTopOff int
	if haveTop && (needs.top || needs.topleft || (needs.left && !haveLeft)) {
		if sbEdge != nil {
			dstTop, dstTopOff = sbEdge, sbEdgeOff+x*4
		} else {
			dstTop, dstTopOff = dst, dstOff-stride
		}
	}

	if needs.left {
		sz := th << 2
		leftOff := tlOutOff - sz

		if haveLeft {
			pxHave := min(sz, (h-y)<<2)
			gatherColumn(tlOut, leftOff+sz-1, dst, dstOff-1, stride, pxHave)
			if pxHave < sz {
				fill(tlOut[leftOff:leftOff+sz-pxHave], tlOut[leftOff+sz-pxHave])
			}
		} else {
			v := P(int32(1<<bitdepth)>>1 + 1)
			if haveTop {
				v = dstTop[dstTopOff]
			}
			fill(tlOut[leftOff:leftOff+sz], v)
		}

		if needs.bottomleft {
			haveBottomleft := haveLeft && y+th < h && edgeFlags&edgeI444LeftHasBottom != 0

			if haveBottomleft {
				pxHave := min(sz, (h-y-th)<<2)
				gatherColumn(tlOut, leftOff-1, dst, dstOff+sz*stride-1, stride, pxHave)
				if pxHave < sz {
					fill(tlOut[leftOff-sz:leftOff-pxHave], tlOut[leftOff-pxHave])
				}
			} else {
				fill(tlOut[leftOff-sz:leftOff], tlOut[leftOff])
			}
		}
	}

	if needs.top {
		sz := tw << 2
		topOff := tlOutOff + 1

		if haveTop {
			pxHave := min(sz, (w-x)<<2)
			copy(tlOut[topOff:topOff+pxHave], dstTop[dstTopOff:dstTopOff+pxHave])
			if pxHave < sz {
				fill(tlOut[topOff+pxHave:topOff+sz], tlOut[topOff+pxHave-1])
			}
		} else {
			v := P(int32(1<<bitdepth)>>1 - 1)
			if haveLeft {
				v = dst[dstOff-1]
			}
			fill(tlOut[topOff:topOff+sz], v)
		}

		if needs.topright {
			haveTopright := haveTop && x+tw < w && edgeFlags&edgeI444TopHasRight != 0

			if haveTopright {
				pxHave := min(sz, (w-x-tw)<<2)
				copy(tlOut[topOff+sz:topOff+sz+pxHave], dstTop[dstTopOff+sz:dstTopOff+sz+pxHave])
				if pxHave < sz {
					fill(tlOut[topOff+sz+pxHave:topOff+2*sz], tlOut[topOff+sz+pxHave-1])
				}
			} else {
				fill(tlOut[topOff+sz:topOff+2*sz], tlOut[topOff+sz-1])
			}
		}
	}

	if needs.topleft {
		switch {
		case haveLeft && haveTop:
			tlOut[tlOutOff] = dstTop[dstTopOff-1]
		case haveLeft:
			tlOut[tlOutOff] = dst[dstOff-1]
		case haveTop:
			tlOut[tlOutOff] = dstTop[dstTopOff]
		default:
			tlOut[tlOutOff] = P(int32(1<<bitdepth) >> 1)
		}

		if mode == z2Pred && tw+th >= 6 && filterEdge {
			tlOut[tlOutOff] = P(((int32(tlOut[tlOutOff-1])+int32(tlOut[tlOutOff+1]))*5 +
				int32(tlOut[tlOutOff])*6 + 8) >> 4)
		}
	}

	return mode
}

func bitdepthFromMax(bitdepthMax int32) int {
	return bits.Len32(uint32(bitdepthMax))
}

func gatherColumn[P pixel](out []P, outOff int, src []P, srcOff, stride, n int) {
	out = out[outOff-(n-1) : outOff+1]
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = src[srcOff]
		srcOff += stride
	}
}
