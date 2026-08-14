package av1

import "encoding/binary"

func setCtx(dst []uint8, off, n int, v uint8) {
	d := dst[off : off+n]
	w := uint64(v) * 0x0101010101010101
	for len(d) >= 8 {
		binary.LittleEndian.PutUint64(d, w)
		d = d[8:]
	}
	for i := range d {
		d[i] = v
	}
}

func setCtxI8(dst []int8, off, n int, v int8) {
	d := dst[off : off+n]
	for i := range d {
		d[i] = v
	}
}

func initQuantTables(seqHdr *sequenceHeader, hdr *frameHeader, qidx int,
	dq *[maxSegments][3][2]uint16,
) {
	for i := range maxSegments {
		var yac int
		if hdr.segmentation.enabled != 0 {
			yac = clip(qidx+int(hdr.segmentation.segData.d[i].deltaQ), 0, 255)
		} else {
			yac = qidx
		}
		ydc := clip(yac+int(hdr.quant.ydcDelta), 0, 255)
		udc := clip(yac+int(hdr.quant.udcDelta), 0, 255)
		uac := clip(yac+int(hdr.quant.uacDelta), 0, 255)
		vdc := clip(yac+int(hdr.quant.vdcDelta), 0, 255)
		vac := clip(yac+int(hdr.quant.vacDelta), 0, 255)

		hbd := int(seqHdr.hbd)
		dq[i][0][0] = dqTbl[hbd][ydc][0]
		dq[i][0][1] = dqTbl[hbd][yac][1]
		dq[i][1][0] = dqTbl[hbd][udc][0]
		dq[i][1][1] = dqTbl[hbd][uac][1]
		dq[i][2][0] = dqTbl[hbd][vdc][0]
		dq[i][2][1] = dqTbl[hbd][vac][1]
	}
}

type blockGeom struct {
	bx4, by4, cbx4, cby4 int
	bw4, bh4, w4, h4     int
	cbw4, cbh4           int
	haveTop, haveLeft    bool
	hasChroma            bool
}

func (t *taskContext[P, C]) decodeB(bl, bs, bp int, intraEdgeFlags int) error {
	ts := t.ts
	f := t.f
	b := &t.blk
	*b = av1Block{}
	bDim := &blockDimensions[bs]
	bx4, by4 := t.bx&31, t.by&31
	ssVer := b2i(f.layout == pixelLayoutI420)
	ssHor := b2i(f.layout != pixelLayoutI444)
	cbx4, cby4 := bx4>>ssHor, by4>>ssVer
	bw4, bh4 := int(bDim[0]), int(bDim[1])
	w4, h4 := min(bw4, f.bw-t.bx), min(bh4, f.bh-t.by)
	cbw4, cbh4 := (bw4+ssHor)>>ssHor, (bh4+ssVer)>>ssVer
	cw4, ch4 := (w4+ssHor)>>ssHor, (h4+ssVer)>>ssVer
	haveLeft := t.bx > ts.tiling.colStart
	haveTop := t.by > ts.tiling.rowStart
	hasChroma := f.layout != pixelLayoutI400 &&
		(bw4 > ssHor || t.bx&1 != 0) &&
		(bh4 > ssVer || t.by&1 != 0)

	b.bl = uint8(bl)
	b.bp = uint8(bp)
	b.bs = uint8(bs)

	var seg *segmentationData
	segPred := 0

	prevSegid := func() (uint8, error) {
		if f.prevSegMap == nil {
			return 0, nil
		}
		id := getPrevFrameSegid(t.by, t.bx, w4, h4, f.prevSegMap, f.b4Stride)
		if id >= maxSegments {
			return 0, errInvalid
		}

		return uint8(id), nil
	}

	if f.frameHdr.segmentation.enabled != 0 {
		if f.frameHdr.segmentation.updateMap == 0 {
			id, err := prevSegid()
			if err != nil {
				return err
			}
			b.segID = id
			seg = &f.frameHdr.segmentation.segData.d[b.segID]
		} else if f.frameHdr.segmentation.segData.preskip != 0 {
			if f.frameHdr.segmentation.temporal != 0 {
				sctx := int(t.a.segPred[bx4]) + int(t.l.segPred[by4])
				segPred = int(ts.msac.boolAdapt(ts.cdf.mi.segPred[sctx][:]))
			}
			if segPred != 0 {
				id, err := prevSegid()
				if err != nil {
					return err
				}
				b.segID = id
			} else {
				predSegID, segCtx := getCurFrameSegid(t.by, t.bx, haveTop, haveLeft,
					f.curSegMap, f.b4Stride)
				diff := ts.msac.symbolAdapt(ts.cdf.m.segID[segCtx][:], maxSegments-1)
				lastActive := int(f.frameHdr.segmentation.segData.lastActiveSegid)
				id := negDeinterleave(int(diff), int(predSegID), lastActive+1)
				if id < 0 || id > lastActive || id >= maxSegments {
					id = 0
				}
				b.segID = uint8(id)
			}
			seg = &f.frameHdr.segmentation.segData.d[b.segID]
		}
	} else {
		b.segID = 0
	}

	if (seg == nil || (seg.globalmv == 0 && seg.ref == -1 && seg.skip == 0)) &&
		f.frameHdr.skipModeEnabled != 0 && min(bw4, bh4) > 1 {
		smctx := int(t.a.skipMode[bx4]) + int(t.l.skipMode[by4])
		b.skipMode = uint8(ts.msac.boolAdapt(ts.cdf.mi.skipMode[smctx][:]))
	} else {
		b.skipMode = 0
	}

	if b.skipMode != 0 || (seg != nil && seg.skip != 0) {
		b.skip = 1
	} else {
		sctx := int(t.a.skip[bx4]) + int(t.l.skip[by4])
		b.skip = uint8(ts.msac.boolAdapt(ts.cdf.m.skip[sctx][:]))
	}

	if f.frameHdr.segmentation.enabled != 0 &&
		f.frameHdr.segmentation.updateMap != 0 &&
		f.frameHdr.segmentation.segData.preskip == 0 {
		if b.skip == 0 && f.frameHdr.segmentation.temporal != 0 {
			sctx := int(t.a.segPred[bx4]) + int(t.l.segPred[by4])
			segPred = int(ts.msac.boolAdapt(ts.cdf.mi.segPred[sctx][:]))
		}
		if segPred != 0 {
			id, err := prevSegid()
			if err != nil {
				return err
			}
			b.segID = id
		} else {
			predSegID, segCtx := getCurFrameSegid(t.by, t.bx, haveTop, haveLeft,
				f.curSegMap, f.b4Stride)
			if b.skip != 0 {
				b.segID = uint8(predSegID)
			} else {
				diff := ts.msac.symbolAdapt(ts.cdf.m.segID[segCtx][:], maxSegments-1)
				lastActive := int(f.frameHdr.segmentation.segData.lastActiveSegid)
				id := negDeinterleave(int(diff), int(predSegID), lastActive+1)
				if id < 0 || id > lastActive {
					id = 0
				}
				b.segID = uint8(id)
			}
			if b.segID >= maxSegments {
				b.segID = 0
			}
		}
		seg = &f.frameHdr.segmentation.segData.d[b.segID]
	}

	if b.skip == 0 {
		idx := 0
		if f.seqHdr.sb128 != 0 {
			idx = (t.bx&16)>>4 + (t.by&16)>>3
		}
		if t.curSbCdefIdx[idx] == -1 {
			v := int8(ts.msac.bools(int(f.frameHdr.cdef.nBits)))
			t.curSbCdefIdx[idx] = v
			if bw4 > 16 {
				t.curSbCdefIdx[idx+1] = v
			}
			if bh4 > 16 {
				t.curSbCdefIdx[idx+2] = v
			}
			if bw4 == 32 && bh4 == 32 {
				t.curSbCdefIdx[idx+3] = v
			}
		}
	}

	if (t.bx|t.by)&(31>>b2i(f.seqHdr.sb128 == 0)) == 0 {
		prevQidx := ts.lastQidx
		sbBs := bs64x64
		if f.seqHdr.sb128 != 0 {
			sbBs = bs128x128
		}
		haveDeltaQ := f.frameHdr.delta.q.present != 0 && (bs != sbBs || b.skip == 0)

		prevDeltaLf := ts.lastDeltaLf

		if haveDeltaQ {
			deltaQ := int(ts.msac.symbolAdapt(ts.cdf.m.deltaQ[:], 3))
			if deltaQ == 3 {
				nBits := 1 + int(ts.msac.bools(3))
				deltaQ = int(ts.msac.bools(nBits)) + 1 + (1 << nBits)
			}
			if deltaQ != 0 {
				if ts.msac.boolEqui() != 0 {
					deltaQ = -deltaQ
				}
				deltaQ *= 1 << f.frameHdr.delta.q.resLog2
			}
			ts.lastQidx = clip(ts.lastQidx+deltaQ, 1, 255)

			if f.frameHdr.delta.lf.present != 0 {
				nLfs := 1
				if f.frameHdr.delta.lf.multi != 0 {
					nLfs = 2
					if f.layout != pixelLayoutI400 {
						nLfs = 4
					}
				}
				for i := range nLfs {
					deltaLf := int(ts.msac.symbolAdapt(
						ts.cdf.m.deltaLf[i+int(f.frameHdr.delta.lf.multi)][:], 3))
					if deltaLf == 3 {
						nBits := 1 + int(ts.msac.bools(3))
						deltaLf = int(ts.msac.bools(nBits)) + 1 + (1 << nBits)
					}
					if deltaLf != 0 {
						if ts.msac.boolEqui() != 0 {
							deltaLf = -deltaLf
						}
						deltaLf *= 1 << f.frameHdr.delta.lf.resLog2
					}
					ts.lastDeltaLf[i] = int8(clip(int(ts.lastDeltaLf[i])+deltaLf, -63, 63))
				}
			}
		}
		if ts.lastQidx == int(f.frameHdr.quant.yac) {
			ts.dq = &f.dq
		} else if ts.lastQidx != prevQidx {
			initQuantTables(f.seqHdr, f.frameHdr, ts.lastQidx, &ts.dqmem)
			ts.dq = &ts.dqmem
		}
		if ts.lastDeltaLf == ([4]int8{}) {
			ts.lflvl = &f.lf.lvl
		} else if ts.lastDeltaLf != prevDeltaLf {
			calcLfValues(&ts.lflvlmem, f.frameHdr, &ts.lastDeltaLf)
			ts.lflvl = &ts.lflvlmem
		}
	}

	switch {
	case b.skipMode != 0:
		b.intra = 0
	case isInterOrSwitch(f.frameHdr):
		if seg != nil && (seg.ref >= 0 || seg.globalmv != 0) {
			b.intra = uint8(b2i(seg.ref == 0))
		} else {
			ictx := getIntraCtx(t.a, &t.l, by4, bx4, haveTop, haveLeft)
			b.intra = uint8(1 - ts.msac.boolAdapt(ts.cdf.mi.intra[ictx][:]))
		}
	case f.frameHdr.allowIntrabc != 0:
		b.intra = uint8(1 - ts.msac.boolAdapt(ts.cdf.intrabc[:]))
	default:
		b.intra = 1
	}

	if b.intra != 0 {
		var ymodeCdf []uint16
		if isInterOrSwitch(f.frameHdr) {
			ymodeCdf = ts.cdf.mi.yMode[ymodeSizeContext[bs]][:]
		} else {
			ymodeCdf = ts.cdf.kfym[intraModeContext[t.a.mode[bx4]]][intraModeContext[t.l.mode[by4]]][:]
		}
		b.yMode = uint8(ts.msac.symbolAdapt(ymodeCdf, nIntraPredModes-1))

		if bDim[2]+bDim[3] >= 2 && b.yMode >= vertPred && b.yMode <= vertLeftPred {
			acdf := ts.cdf.m.angleDelta[b.yMode-vertPred][:]
			angle := int(ts.msac.symbolAdapt(acdf, 6))
			b.yAngle = int8(angle - 3)
		} else {
			b.yAngle = 0
		}

		if hasChroma {
			cflAllowed := 0
			if f.frameHdr.segmentation.lossless[b.segID] != 0 {
				cflAllowed = b2i(cbw4 == 1 && cbh4 == 1)
			} else {
				cflAllowed = b2i(cflAllowedMask&(1<<bs) != 0)
			}
			uvmodeCdf := ts.cdf.m.uvMode[cflAllowed][b.yMode][:]
			b.uvMode = uint8(ts.msac.symbolAdapt(uvmodeCdf, nUvIntraPredModes-1-(1-cflAllowed)))

			b.uvAngle = 0
			if b.uvMode == cflPred {
				sign := int(ts.msac.symbolAdapt(ts.cdf.m.cflSign[:], 7)) + 1
				signU := sign * 0x56 >> 8
				signV := sign - signU*3
				if signU != 0 {
					ctx := b2i(signU == 2)*3 + signV
					b.cflAlpha[0] = int8(ts.msac.symbolAdapt(ts.cdf.m.cflAlpha[ctx][:], 15) + 1)
					if signU == 1 {
						b.cflAlpha[0] = -b.cflAlpha[0]
					}
				} else {
					b.cflAlpha[0] = 0
				}
				if signV != 0 {
					ctx := b2i(signV == 2)*3 + signU
					b.cflAlpha[1] = int8(ts.msac.symbolAdapt(ts.cdf.m.cflAlpha[ctx][:], 15) + 1)
					if signV == 1 {
						b.cflAlpha[1] = -b.cflAlpha[1]
					}
				} else {
					b.cflAlpha[1] = 0
				}
			} else if bDim[2]+bDim[3] >= 2 && b.uvMode >= vertPred && b.uvMode <= vertLeftPred {
				acdf := ts.cdf.m.angleDelta[b.uvMode-vertPred][:]
				angle := int(ts.msac.symbolAdapt(acdf, 6))
				b.uvAngle = int8(angle - 3)
			}
		}

		b.palSz[0], b.palSz[1] = 0, 0
		if f.frameHdr.allowScreenContentTools != 0 &&
			max(bw4, bh4) <= 16 && bw4+bh4 >= 4 {
			szCtx := int(bDim[2]) + int(bDim[3]) - 2
			if b.yMode == dcPred {
				palCtx := b2i(t.a.palSz[bx4] > 0) + b2i(t.l.palSz[by4] > 0)
				if ts.msac.boolAdapt(ts.cdf.m.palY[szCtx][palCtx][:]) != 0 {
					readPalPlane(t, b, 0, szCtx, bx4, by4)
				}
			}
			if hasChroma && b.uvMode == dcPred {
				palCtx := b2i(b.palSz[0] > 0)
				if ts.msac.boolAdapt(ts.cdf.m.palUv[palCtx][:]) != 0 {
					readPalUv(t, b, szCtx, bx4, by4)
				}
			}
		}

		if b.yMode == dcPred && b.palSz[0] == 0 &&
			max(bDim[2], bDim[3]) <= 3 && f.seqHdr.filterIntra != 0 {
			isFilter := ts.msac.boolAdapt(ts.cdf.m.useFilterIntra[bs][:])
			if isFilter != 0 {
				b.yMode = filterPred
				b.yAngle = int8(ts.msac.symbolAdapt(ts.cdf.m.filterIntra[:], 4))
			}
		}

		if b.palSz[0] != 0 {
			readPalIndices(t, t.scratch.palIdxY[:], int(b.palSz[0]), 0, w4, h4, bw4, bh4)
		}
		if hasChroma && b.palSz[1] != 0 {
			readPalIndices(t, t.scratch.palIdxUv[:], int(b.palSz[1]), 1, cw4, ch4, cbw4, cbh4)
		}

		var tDim *txfmInfo
		if f.frameHdr.segmentation.lossless[b.segID] != 0 {
			b.tx, b.uvtx = tx4x4, tx4x4
			tDim = &txfmDimensions[tx4x4]
		} else {
			b.tx = maxTxfmSizeForBs[bs][0]
			b.uvtx = maxTxfmSizeForBs[bs][f.layout]
			tDim = &txfmDimensions[b.tx]
			if f.frameHdr.txfmMode == txSwitchable && tDim.max > tx4x4 {
				tctx := getTxCtx(t.a, &t.l, tDim, by4, bx4)
				txCdf := ts.cdf.m.txsz[tDim.max-1][tctx][:]
				depth := int(ts.msac.symbolAdapt(txCdf, min(int(tDim.max), 2)))
				for ; depth > 0; depth-- {
					b.tx = tDim.sub
					tDim = &txfmDimensions[b.tx]
				}
			}
		}

		f.reconIntra(t, bs, intraEdgeFlags, b)

		if f.lfEnabled {
			var auv, luv []uint8
			auvOff, luvOff := 0, 0
			if hasChroma {
				auv, luv = t.a.txLpfUv[:], t.l.txLpfUv[:]
				auvOff, luvOff = cbx4, cby4
			}
			createLfMaskIntra(t.lfMask, f.lf.level, f.b4Stride,
				&ts.lflvl[b.segID], t.bx, t.by, f.w4, f.h4, bs,
				int(b.tx), int(b.uvtx), f.layout,
				t.a.txLpfY[:], t.l.txLpfY[:], auv, luv,
				bx4, by4, auvOff, luvOff, hasChroma)
		}

		yModeNofilt := b.yMode
		if b.yMode == filterPred {
			yModeNofilt = dcPred
		}

		for i, edge := range [2]*blockContext{t.a, &t.l} {
			off := bx4
			n := bw4
			tLsz := tDim.lw
			if i == 1 {
				off = by4
				n = bh4
				tLsz = tDim.lh
			}
			setCtxI8(edge.txIntra[:], off, n, int8(tLsz))
			setCtxI8(edge.tx[:], off, n, int8(tLsz))
			setCtx(edge.mode[:], off, n, yModeNofilt)
			setCtx(edge.palSz[:], off, n, b.palSz[0])
			setCtx(edge.segPred[:], off, n, uint8(segPred))
			setCtx(edge.skipMode[:], off, n, b.skipMode)
			setCtx(edge.intra[:], off, n, 1)
			setCtx(edge.skip[:], off, n, b.skip)
			palSzUv := uint8(0)
			if hasChroma {
				palSzUv = b.palSz[1]
			}
			setCtx(t.palSzUv[i][:], off, n, palSzUv)
			if isInterOrSwitch(f.frameHdr) {
				setCtx(edge.compType[:], off, n, compInterNone)
				setCtxI8(edge.ref[0][:], off, n, -1)
				setCtxI8(edge.ref[1][:], off, n, -1)
				setCtx(edge.filter[0][:], off, n, nSwitchableFilters)
				setCtx(edge.filter[1][:], off, n, nSwitchableFilters)
			}
		}

		if b.palSz[0] != 0 {
			copyPalBlockY(t, bx4, by4, bw4, bh4)
		}

		if hasChroma {
			setCtx(t.a.uvmode[:], cbx4, cbw4, b.uvMode)
			setCtx(t.l.uvmode[:], cby4, cbh4, b.uvMode)
			if b.palSz[1] != 0 {
				copyPalBlockUv(t, bx4, by4, bw4, bh4)
			}
		}

		if isInterOrSwitch(f.frameHdr) || f.frameHdr.allowIntrabc != 0 {
			t.splatIntraref(bs, bw4, bh4)
		}
	} else {
		g := blockGeom{
			bx4: bx4, by4: by4, cbx4: cbx4, cby4: cby4,
			bw4: bw4, bh4: bh4, w4: w4, h4: h4, cbw4: cbw4, cbh4: cbh4,
			haveTop: haveTop, haveLeft: haveLeft, hasChroma: hasChroma,
		}
		if isKeyOrIntra(f.frameHdr) {
			if err := t.decodeBIntrabc(b, bs, intraEdgeFlags, segPred, &g); err != nil {
				return err
			}
		} else if err := t.decodeBInter(b, bs, intraEdgeFlags, seg, segPred, &g); err != nil {
			return err
		}
	}

	if f.frameHdr.segmentation.enabled != 0 && f.frameHdr.segmentation.updateMap != 0 {
		for y := range h4 {
			row := (t.by+y)*f.b4Stride + t.bx
			for x := range w4 {
				f.curSegMap[row+x] = b.segID
			}
		}
	}

	if b.skip == 0 {
		mask := uint16((^uint32(0) >> (32 - bw4)) << (bx4 & 15))
		bxIdx := (bx4 & 16) >> 4
		for y := 0; y < bh4; y += 2 {
			nm := &t.lfMask.noskipMask[(by4>>1)+(y>>1)]
			nm[bxIdx] |= mask
			if bw4 == 32 {
				nm[1] |= mask
			}
		}
	}

	return nil
}

func (t *taskContext[P, C]) decodeSb(bl int, node *edgeNode) error {
	f := t.f
	ts := t.ts
	hsz := 16 >> bl
	haveHSplit := f.bw > t.bx+hsz
	haveVSplit := f.bh > t.by+hsz

	if !haveHSplit && !haveVSplit {
		return t.decodeSb(bl+1, node.kids[0])
	}

	bx8, by8 := (t.bx&31)>>1, (t.by&31)>>1
	ctx := getPartitionCtx(t.a, &t.l, bl, by8, bx8)
	pc := ts.cdf.m.partition[bl][ctx][:]

	var bp int

	switch {
	case haveHSplit && haveVSplit:
		bp = int(ts.msac.symbolAdapt(pc, int(partitionTypeCount[bl])))
		if f.layout == pixelLayoutI422 &&
			(bp == partitionV || bp == partitionV4 ||
				bp == partitionTLeftSplit || bp == partitionTRightSplit) {
			return errInvalid
		}
		b := &blockSizes[bl][bp]

		switch bp {
		case partitionNone:
			if err := t.decodeB(bl, int(b[0]), partitionNone, int(node.o)); err != nil {
				return err
			}
		case partitionH:
			if err := t.decodeB(bl, int(b[0]), partitionH, int(node.h[0])); err != nil {
				return err
			}
			t.by += hsz
			if err := t.decodeB(bl, int(b[0]), partitionH, int(node.h[1])); err != nil {
				return err
			}
			t.by -= hsz
		case partitionV:
			if err := t.decodeB(bl, int(b[0]), partitionV, int(node.v[0])); err != nil {
				return err
			}
			t.bx += hsz
			if err := t.decodeB(bl, int(b[0]), partitionV, int(node.v[1])); err != nil {
				return err
			}
			t.bx -= hsz
		case partitionSplit:
			if bl == bl8x8 {
				if err := t.decodeB(bl, bs4x4, partitionSplit, edgeAllTrAndBl); err != nil {
					return err
				}
				tlFilter := t.tl4x4Filter
				t.bx++
				if err := t.decodeB(bl, bs4x4, partitionSplit, int(node.split[0])); err != nil {
					return err
				}
				t.bx--
				t.by++
				if err := t.decodeB(bl, bs4x4, partitionSplit, int(node.split[1])); err != nil {
					return err
				}
				t.bx++
				t.tl4x4Filter = tlFilter
				if err := t.decodeB(bl, bs4x4, partitionSplit, int(node.split[2])); err != nil {
					return err
				}
				t.bx--
				t.by--
			} else {
				if err := t.decodeSb(bl+1, node.kids[0]); err != nil {
					return err
				}
				t.bx += hsz
				if err := t.decodeSb(bl+1, node.kids[1]); err != nil {
					return err
				}
				t.bx -= hsz
				t.by += hsz
				if err := t.decodeSb(bl+1, node.kids[2]); err != nil {
					return err
				}
				t.bx += hsz
				if err := t.decodeSb(bl+1, node.kids[3]); err != nil {
					return err
				}
				t.bx -= hsz
				t.by -= hsz
			}
		case partitionTTopSplit:
			if err := t.decodeB(bl, int(b[0]), bp, edgeAllTrAndBl); err != nil {
				return err
			}
			t.bx += hsz
			if err := t.decodeB(bl, int(b[0]), bp, int(node.v[1])); err != nil {
				return err
			}
			t.bx -= hsz
			t.by += hsz
			if err := t.decodeB(bl, int(b[1]), bp, int(node.h[1])); err != nil {
				return err
			}
			t.by -= hsz
		case partitionTBottomSplit:
			if err := t.decodeB(bl, int(b[0]), bp, int(node.h[0])); err != nil {
				return err
			}
			t.by += hsz
			if err := t.decodeB(bl, int(b[1]), bp, int(node.v[0])); err != nil {
				return err
			}
			t.bx += hsz
			if err := t.decodeB(bl, int(b[1]), bp, 0); err != nil {
				return err
			}
			t.bx -= hsz
			t.by -= hsz
		case partitionTLeftSplit:
			if err := t.decodeB(bl, int(b[0]), bp, edgeAllTrAndBl); err != nil {
				return err
			}
			t.by += hsz
			if err := t.decodeB(bl, int(b[0]), bp, int(node.h[1])); err != nil {
				return err
			}
			t.by -= hsz
			t.bx += hsz
			if err := t.decodeB(bl, int(b[1]), bp, int(node.v[1])); err != nil {
				return err
			}
			t.bx -= hsz
		case partitionTRightSplit:
			if err := t.decodeB(bl, int(b[0]), bp, int(node.v[0])); err != nil {
				return err
			}
			t.bx += hsz
			if err := t.decodeB(bl, int(b[1]), bp, int(node.h[0])); err != nil {
				return err
			}
			t.by += hsz
			if err := t.decodeB(bl, int(b[1]), bp, 0); err != nil {
				return err
			}
			t.by -= hsz
			t.bx -= hsz
		case partitionH4:
			if err := t.decodeB(bl, int(b[0]), bp, int(node.h[0])); err != nil {
				return err
			}
			t.by += hsz >> 1
			if err := t.decodeB(bl, int(b[0]), bp, int(node.h4)); err != nil {
				return err
			}
			t.by += hsz >> 1
			if err := t.decodeB(bl, int(b[0]), bp, edgeAllLeftHasBottom); err != nil {
				return err
			}
			t.by += hsz >> 1
			if t.by < f.bh {
				if err := t.decodeB(bl, int(b[0]), bp, int(node.h[1])); err != nil {
					return err
				}
			}
			t.by -= hsz * 3 >> 1
		case partitionV4:
			if err := t.decodeB(bl, int(b[0]), bp, int(node.v[0])); err != nil {
				return err
			}
			t.bx += hsz >> 1
			if err := t.decodeB(bl, int(b[0]), bp, int(node.v4)); err != nil {
				return err
			}
			t.bx += hsz >> 1
			if err := t.decodeB(bl, int(b[0]), bp, edgeAllTopHasRight); err != nil {
				return err
			}
			t.bx += hsz >> 1
			if t.bx < f.bw {
				if err := t.decodeB(bl, int(b[0]), bp, int(node.v[1])); err != nil {
					return err
				}
			}
			t.bx -= hsz * 3 >> 1
		}

	case haveHSplit:
		isSplit := ts.msac.boolF(gatherTopPartitionProb(pc, bl))
		if isSplit != 0 {
			bp = partitionSplit
			if err := t.decodeSb(bl+1, node.kids[0]); err != nil {
				return err
			}
			t.bx += hsz
			if err := t.decodeSb(bl+1, node.kids[1]); err != nil {
				return err
			}
			t.bx -= hsz
		} else {
			bp = partitionH
			if err := t.decodeB(bl, int(blockSizes[bl][partitionH][0]),
				partitionH, int(node.h[0])); err != nil {
				return err
			}
		}

	default:
		isSplit := ts.msac.boolF(gatherLeftPartitionProb(pc, bl))
		if f.layout == pixelLayoutI422 && isSplit == 0 {
			return errInvalid
		}
		if isSplit != 0 {
			bp = partitionSplit
			if err := t.decodeSb(bl+1, node.kids[0]); err != nil {
				return err
			}
			t.by += hsz
			if err := t.decodeSb(bl+1, node.kids[2]); err != nil {
				return err
			}
			t.by -= hsz
		} else {
			bp = partitionV
			if err := t.decodeB(bl, int(blockSizes[bl][partitionV][0]),
				partitionV, int(node.v[0])); err != nil {
				return err
			}
		}
	}

	if bp != partitionSplit || bl == bl8x8 {
		n := 1 << ulog2(uint32(hsz))
		setCtx(t.a.partition[:], bx8, n, alPartCtx[0][bl][bp])
		setCtx(t.l.partition[:], by8, n, alPartCtx[1][bl][bp])
	}

	return nil
}

func resetContext(ctx *blockContext, keyframe bool) {
	memset(ctx.intra[:], b2u8(keyframe))
	memset(ctx.uvmode[:], dcPred)
	if keyframe {
		memset(ctx.mode[:], dcPred)
	}

	clear(ctx.partition[:])
	clear(ctx.skip[:])
	clear(ctx.skipMode[:])
	memset(ctx.txLpfY[:], 2)
	memset(ctx.txLpfUv[:], 1)
	for i := range ctx.txIntra {
		ctx.txIntra[i] = -1
	}
	for i := range ctx.tx {
		ctx.tx[i] = tx64x64
	}
	if !keyframe {
		for i := range ctx.ref {
			for j := range ctx.ref[i] {
				ctx.ref[i][j] = -1
			}
		}
		clear(ctx.compType[:])
		memset(ctx.mode[:], nearestmv)
	}
	memset(ctx.lcoef[:], 0x40)
	memset(ctx.ccoef[0][:], 0x40)
	memset(ctx.ccoef[1][:], 0x40)
	for i := range ctx.filter {
		memset(ctx.filter[i][:], nSwitchableFilters)
	}
	clear(ctx.segPred[:])
	clear(ctx.palSz[:])
}

func readRestorationInfo[P pixel, C coef](t *taskContext[P, C],
	lr *av1RestorationUnit, p, frameType int,
) {
	ts := t.ts
	lrRef := ts.lrRef[p]

	if frameType == restorationSwitchable {
		filter := int(ts.msac.symbolAdapt(ts.cdf.m.restoreSwitchable[:], 2))
		lr.typ = uint8(filter + b2i(filter != 0))
	} else {
		cdf := ts.cdf.m.restoreSgrproj[:]
		if frameType == restorationWiener {
			cdf = ts.cdf.m.restoreWiener[:]
		}
		if ts.msac.boolAdapt(cdf) != 0 {
			lr.typ = uint8(frameType)
		} else {
			lr.typ = restorationNone
		}
	}

	switch lr.typ {
	case restorationWiener:
		if p != 0 {
			lr.filterV[0], lr.filterH[0] = 0, 0
		} else {
			lr.filterV[0] = int8(ts.msac.subexp(int32(lrRef.filterV[0])+5, 16, 1) - 5)
		}
		lr.filterV[1] = int8(ts.msac.subexp(int32(lrRef.filterV[1])+23, 32, 2) - 23)
		lr.filterV[2] = int8(ts.msac.subexp(int32(lrRef.filterV[2])+17, 64, 3) - 17)

		if p == 0 {
			lr.filterH[0] = int8(ts.msac.subexp(int32(lrRef.filterH[0])+5, 16, 1) - 5)
		}
		lr.filterH[1] = int8(ts.msac.subexp(int32(lrRef.filterH[1])+23, 32, 2) - 23)
		lr.filterH[2] = int8(ts.msac.subexp(int32(lrRef.filterH[2])+17, 64, 3) - 17)
		lr.sgrWeights = lrRef.sgrWeights
		ts.lrRef[p] = lr

	case restorationSgrproj:
		idx := ts.msac.bools(4)
		sp := &sgrParams[idx]
		lr.typ += uint8(idx)
		if sp[0] != 0 {
			lr.sgrWeights[0] = int8(ts.msac.subexp(int32(lrRef.sgrWeights[0])+96, 128, 4) - 96)
		} else {
			lr.sgrWeights[0] = 0
		}
		if sp[1] != 0 {
			lr.sgrWeights[1] = int8(ts.msac.subexp(int32(lrRef.sgrWeights[1])+32, 128, 4) - 32)
		} else {
			lr.sgrWeights[1] = 95
		}
		lr.filterV = lrRef.filterV
		lr.filterH = lrRef.filterH
		ts.lrRef[p] = lr
	}
}

func readLrUnits[P pixel, C coef](t *taskContext[P, C]) {
	f := t.f

	for p := range 3 {
		if f.lf.restorePlanes>>p&1 == 0 {
			continue
		}

		ssVer := b2i(p != 0 && f.layout == pixelLayoutI420)
		ssHor := b2i(p != 0 && f.layout != pixelLayoutI444)
		unitSizeLog2 := int(f.frameHdr.restoration.unitSize[b2i(p != 0)])
		y := t.by * 4 >> ssVer
		h := (f.frameHdr.height + ssVer) >> ssVer

		unitSize := 1 << unitSizeLog2
		mask := unitSize - 1
		if y&mask != 0 {
			continue
		}
		halfUnit := unitSize >> 1
		if y != 0 && y+halfUnit > h {
			continue
		}

		frameType := f.frameHdr.restoration.typ[p]

		if f.resize {
			w := (f.srWidth + ssHor) >> ssHor
			nUnits := max(1, (w+halfUnit)>>unitSizeLog2)

			d := int(f.frameHdr.superRes.widthScaleDenominator)
			rnd, shift := unitSize*8-1, unitSizeLog2+3
			x0 := ((4 * t.bx * d >> ssHor) + rnd) >> shift
			x1 := ((4 * (t.bx + f.sbStep) * d >> ssHor) + rnd) >> shift

			for x := x0; x < min(x1, nUnits); x++ {
				pxX := x << (unitSizeLog2 + ssHor)
				sbIdx := (t.by>>5)*f.srSb128w + (pxX >> 7)
				unitIdx := (t.by&16)>>3 + (pxX&64)>>6
				readRestorationInfo(t, &f.lrMask[sbIdx].lr[p][unitIdx], p, frameType)
			}

			continue
		}

		x := 4 * t.bx >> ssHor
		if x&mask != 0 {
			continue
		}
		w := (f.frameHdr.width[0] + ssHor) >> ssHor
		if x != 0 && x+halfUnit > w {
			continue
		}
		sbIdx := (t.by>>5)*f.srSb128w + (t.bx >> 5)
		unitIdx := (t.by&16)>>3 + (t.bx&16)>>4

		readRestorationInfo(t, &f.lrMask[sbIdx].lr[p][unitIdx], p, frameType)
	}
}

func orderPalette(palIdx []uint8, stride, i, first, last int,
	order *[64][8]uint8, ctx *[64]uint8,
) {
	haveTop := i > first
	off := first + (i-first)*stride

	for j, n := first, 0; j >= last; j, n, off, haveTop = j-1, n+1, off+stride-1, true {
		haveLeft := j > 0

		var mask uint32
		oIdx := 0
		add := func(v uint8) {
			order[n][oIdx] = v
			oIdx++
			mask |= 1 << v
		}

		switch {
		case !haveLeft:
			ctx[n] = 0
			add(palIdx[off-stride])
		case !haveTop:
			ctx[n] = 0
			add(palIdx[off-1])
		default:
			l, tp, tl := palIdx[off-1], palIdx[off-stride], palIdx[off-stride-1]
			sameTL, sameTTl, sameLTl := tp == l, tp == tl, l == tl

			switch {
			case sameTL && sameTTl && sameLTl:
				ctx[n] = 4
				add(tp)
			case sameTL:
				ctx[n] = 3
				add(tp)
				add(tl)
			case sameTTl || sameLTl:
				ctx[n] = 2
				add(tl)
				if sameTTl {
					add(l)
				} else {
					add(tp)
				}
			default:
				ctx[n] = 1
				add(min(tp, l))
				add(max(tp, l))
				add(tl)
			}
		}

		for m, bit := uint32(1), uint8(0); m < 0x100; m, bit = m<<1, bit+1 {
			if mask&m == 0 {
				order[n][oIdx] = bit
				oIdx++
			}
		}
	}
}

func readPalIndices[P pixel, C coef](t *taskContext[P, C], palIdx []uint8,
	palSz, pl, w4, h4, bw4, bh4 int,
) {
	ts := t.ts
	stride := bw4 * 4
	palTmp := t.scratch.palIdxUv[:]

	palTmp[0] = uint8(ts.msac.uniform(uint32(palSz)))
	colorMapCdf := &ts.cdf.m.colorMap[pl][palSz-2]
	order := &t.scratch.palOrder
	ctx := &t.scratch.palCtx

	for i := 1; i < 4*(w4+h4)-1; i++ {
		first := min(i, w4*4-1)
		last := max(0, i-h4*4+1)
		orderPalette(palTmp, stride, i, first, last, order, ctx)
		for j, m := first, 0; j >= last; j, m = j-1, m+1 {
			colorIdx := int(ts.msac.symbolAdapt(colorMapCdf[ctx[m]][:], palSz-1))
			palTmp[(i-j)*stride+j] = order[m][colorIdx]
		}
	}

	palIdxFinish(palIdx, palTmp, bw4*4, bh4*4, w4*4, h4*4)
}

func readMvComponentDiff(msac *msacContext, comp *cdfMvComponent, mvPrec int) int {
	sign := msac.boolAdapt(comp.sign[:])
	cl := int(msac.symbolAdapt(comp.classes[:], 10))
	up, fp, hp := 0, 3, 1

	if cl == 0 {
		up = int(msac.boolAdapt(comp.class0[:]))
		if mvPrec >= 0 {
			fp = int(msac.symbolAdapt(comp.class0Fp[up][:], 3))
			if mvPrec > 0 {
				hp = int(msac.boolAdapt(comp.class0Hp[:]))
			}
		}
	} else {
		up = 1 << cl
		for n := range cl {
			up |= int(msac.boolAdapt(comp.classN[n][:])) << n
		}
		if mvPrec >= 0 {
			fp = int(msac.symbolAdapt(comp.classNFp[:], 3))
			if mvPrec > 0 {
				hp = int(msac.boolAdapt(comp.classNHp[:]))
			}
		}
	}

	diff := (up<<3 | fp<<1 | hp) + 1
	if sign != 0 {
		return -diff
	}

	return diff
}

func readMvResidual(ts *tileState, refMv *mv, mvPrec int) {
	joint := int(ts.msac.symbolAdapt(ts.cdf.mv.joint[:], nMvJoints-1))

	if joint&mvJointV != 0 {
		refMv.y += int16(readMvComponentDiff(&ts.msac, &ts.cdf.mv.comp[0], mvPrec))
	}
	if joint&mvJointH != 0 {
		refMv.x += int16(readMvComponentDiff(&ts.msac, &ts.cdf.mv.comp[1], mvPrec))
	}
}

func (t *taskContext[P, C]) readTxTree(from, depth int, masks *[2]uint16,
	xOff, yOff int,
) {
	f := t.f
	bx4, by4 := t.bx&31, t.by&31
	tDim := &txfmDimensions[from]
	txw, txh := int(tDim.lw), int(tDim.lh)

	isSplit := 0
	if depth < 2 && from > tx4x4 {
		cat := 2*(tx64x64-int(tDim.max)) - depth
		a := b2i(int(t.a.tx[bx4]) < txw)
		l := b2i(int(t.l.tx[by4]) < txh)

		isSplit = int(t.ts.msac.boolAdapt(t.ts.cdf.m.txpart[cat][a+l][:]))
		if isSplit != 0 {
			masks[depth] |= 1 << (yOff*4 + xOff)
		}
	}

	if isSplit != 0 && int(tDim.max) > tx8x8 {
		sub := int(tDim.sub)
		subTDim := &txfmDimensions[sub]
		txsw, txsh := int(subTDim.w), int(subTDim.h)

		t.readTxTree(sub, depth+1, masks, xOff*2+0, yOff*2+0)
		t.bx += txsw
		if txw >= txh && t.bx < f.bw {
			t.readTxTree(sub, depth+1, masks, xOff*2+1, yOff*2+0)
		}
		t.bx -= txsw
		t.by += txsh
		if txh >= txw && t.by < f.bh {
			t.readTxTree(sub, depth+1, masks, xOff*2+0, yOff*2+1)
			t.bx += txsw
			if txw >= txh && t.bx < f.bw {
				t.readTxTree(sub, depth+1, masks, xOff*2+1, yOff*2+1)
			}
			t.bx -= txsw
		}
		t.by -= txsh

		return
	}

	av, lv := int8(txw), int8(txh)
	if isSplit != 0 {
		av, lv = tx4x4, tx4x4
	}
	setCtxI8(t.a.tx[:], bx4, 1<<tDim.lw, av)
	setCtxI8(t.l.tx[:], by4, 1<<tDim.lh, lv)
}

func (t *taskContext[P, C]) readVartxTree(b *av1Block, bs, bx4, by4 int) {
	f := t.f
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])

	var txSplit [2]uint16
	b.maxYtx = maxTxfmSizeForBs[bs][0]

	switch {
	case b.skip == 0 && (f.frameHdr.segmentation.lossless[b.segID] != 0 ||
		b.maxYtx == tx4x4):
		b.maxYtx, b.uvtx = tx4x4, tx4x4
		if f.frameHdr.txfmMode == txSwitchable {
			setCtxI8(t.a.tx[:], bx4, 1<<bDim[2], tx4x4)
			setCtxI8(t.l.tx[:], by4, 1<<bDim[3], tx4x4)
		}

	case f.frameHdr.txfmMode != txSwitchable || b.skip != 0:
		if f.frameHdr.txfmMode == txSwitchable {
			setCtxI8(t.a.tx[:], bx4, 1<<bDim[2], int8(bDim[2]))
			setCtxI8(t.l.tx[:], by4, 1<<bDim[3], int8(bDim[3]))
		}
		b.uvtx = maxTxfmSizeForBs[bs][f.layout]

	default:
		ytx := &txfmDimensions[b.maxYtx]
		y, yOff := 0, 0
		for ; y < bh4; y, yOff = y+int(ytx.h), yOff+1 {
			x, xOff := 0, 0
			for ; x < bw4; x, xOff = x+int(ytx.w), xOff+1 {
				t.readTxTree(int(b.maxYtx), 0, &txSplit, xOff, yOff)
				t.bx += int(ytx.w)
			}
			t.bx -= x
			t.by += int(ytx.h)
		}
		t.by -= y
		b.uvtx = maxTxfmSizeForBs[bs][f.layout]
	}

	b.txSplit0 = uint8(txSplit[0])
	b.txSplit1 = txSplit[1]
}
