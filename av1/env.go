package av1

func getIntraCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	if haveLeft {
		if haveTop {
			ctx := int(l.intra[yb4]) + int(a.intra[xb4])

			return ctx + b2i(ctx == 2)
		}

		return int(l.intra[yb4]) * 2
	}
	if haveTop {
		return int(a.intra[xb4]) * 2
	}

	return 0
}

func getTxCtx(a, l *blockContext, maxTx *txfmInfo, yb4, xb4 int) int {
	return b2i(l.txIntra[yb4] >= int8(maxTx.lh)) + b2i(a.txIntra[xb4] >= int8(maxTx.lw))
}

func getPartitionCtx(a, l *blockContext, bl, yb8, xb8 int) int {
	return int(a.partition[xb8]>>(4-bl))&1 |
		(int(l.partition[yb8]>>(4-bl))&1)<<1
}

func gatherTopPartitionProb(in []uint16, bl int) uint32 {
	out := uint32(in[partitionV-1]) - uint32(in[partitionTTopSplit])
	out += uint32(in[partitionTLeftSplit-1])
	if bl != bl128x128 {
		out += uint32(in[partitionV4-1]) - uint32(in[partitionTRightSplit])
	}

	return out
}

func gatherLeftPartitionProb(in []uint16, bl int) uint32 {
	out := uint32(in[partitionH-1]) - uint32(in[partitionH])
	out += uint32(in[partitionSplit-1]) - uint32(in[partitionTLeftSplit])
	if bl != bl128x128 {
		out += uint32(in[partitionH4-1]) - uint32(in[partitionH4])
	}

	return out
}

func getCurFrameSegid(by, bx int, haveTop, haveLeft bool, segMap []uint8,
	stride int,
) (uint32, int) {
	off := bx + by*stride
	if haveLeft && haveTop {
		l := int(segMap[off-1])
		a := int(segMap[off-stride])
		al := int(segMap[off-(stride+1)])

		segCtx := 0
		switch {
		case l == a && al == l:
			segCtx = 2
		case l == a || al == l || a == al:
			segCtx = 1
		}
		if a == al {
			return uint32(a), segCtx
		}

		return uint32(l), segCtx
	}
	if haveLeft {
		return uint32(segMap[off-1]), 0
	}
	if haveTop {
		return uint32(segMap[off-stride]), 0
	}

	return 0, 0
}

func getPrevFrameSegid(by, bx, w4, h4 int, refSegMap []uint8, stride int) uint32 {
	segID := uint32(8)
	off := by*stride + bx
	for {
		for x := range w4 {
			segID = min(segID, uint32(refSegMap[off+x]))
		}
		off += stride
		h4--
		if h4 <= 0 || segID == 0 {
			break
		}
	}

	return segID
}

func negDeinterleave(diff, ref, max int) int {
	if ref == 0 {
		return diff
	}
	if ref >= max-1 {
		return max - diff - 1
	}
	if 2*ref < max {
		if diff <= 2*ref {
			if diff&1 != 0 {
				return ref + (diff+1)>>1
			}

			return ref - diff>>1
		}

		return diff
	}
	if diff <= 2*(max-ref-1) {
		if diff&1 != 0 {
			return ref + (diff+1)>>1
		}

		return ref - diff>>1
	}

	return max - (diff + 1)
}
