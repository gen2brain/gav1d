package av1

import "encoding/binary"

func readGolomb(msac *msacContext) uint32 {
	len := 0
	val := uint32(1)

	for msac.boolEqui() == 0 && len < 32 {
		len++
	}
	for ; len > 0; len-- {
		val = val<<1 + msac.boolEqui()
	}

	return val - 1
}

func le16(b []uint8) uint32 { return uint32(binary.LittleEndian.Uint16(b)) }
func le32(b []uint8) uint32 { return binary.LittleEndian.Uint32(b) }
func le64(b []uint8) uint64 { return binary.LittleEndian.Uint64(b) }

func getSkipCtx(tDim *txfmInfo, bs int, a []uint8, aOff int, l []uint8, lOff int,
	chroma int, layout int,
) uint32 {
	bDim := &blockDimensions[bs]

	if chroma != 0 {
		ssVer := b2u(layout == pixelLayoutI420)
		ssHor := b2u(layout != pixelLayoutI444)
		notOneBlk := bDim[2]-b2u8(bDim[2] != 0 && ssHor != 0) > tDim.lw ||
			bDim[3]-b2u8(bDim[3] != 0 && ssVer != 0) > tDim.lh

		var ca, cl uint32
		switch tDim.lw {
		case tx4x4:
			ca = b2u(a[aOff] != 0x40)
		case tx8x8:
			ca = b2u(le16(a[aOff:]) != 0x4040)
		case tx16x16:
			ca = b2u(le32(a[aOff:]) != 0x40404040)
		default:
			ca = b2u(le64(a[aOff:]) != 0x4040404040404040)
		}
		switch tDim.lh {
		case tx4x4:
			cl = b2u(l[lOff] != 0x40)
		case tx8x8:
			cl = b2u(le16(l[lOff:]) != 0x4040)
		case tx16x16:
			cl = b2u(le32(l[lOff:]) != 0x40404040)
		default:
			cl = b2u(le64(l[lOff:]) != 0x4040404040404040)
		}

		return 7 + b2u(notOneBlk)*3 + ca + cl
	}

	if bDim[2] == tDim.lw && bDim[3] == tDim.lh {
		return 0
	}

	merge := func(buf []uint8, off int, tx uint8) uint32 {
		var v uint32
		if tx == tx64x64 {
			tmp := le64(buf[off:]) | le64(buf[off+8:])
			v = uint32(tmp>>32) | uint32(tmp)
		} else {
			switch tx {
			case tx4x4:
				v = uint32(buf[off])
			case tx8x8:
				v = le16(buf[off:])
			default:
				v = le32(buf[off:])
			}
		}
		if tx == tx32x32 {
			v |= le32(buf[off+4:])
		}
		if tx >= tx16x16 {
			v |= v >> 16
		}
		if tx >= tx8x8 {
			v |= v >> 8
		}

		return v
	}

	la := merge(a, aOff, tDim.lw)
	ll := merge(l, lOff, tDim.lh)

	return uint32(skipCtx[min(la&0x3F, 4)][min(ll&0x3F, 4)])
}

func getDcSignCtx(tx int, a []uint8, aOff int, l []uint8, lOff int) uint32 {
	const mask64 = uint64(0xC0C0C0C0C0C0C0C0)
	const mul64 = uint64(0x0101010101010101)
	const mask32 = uint32(0xC0C0C0C0)
	const mul32 = uint32(0x01010101)

	var s int32

	switch tx {
	case tx4x4:
		t := uint32(a[aOff]) >> 6
		t += uint32(l[lOff]) >> 6
		s = int32(t) - 1 - 1
	case tx8x8:
		t := le16(a[aOff:]) & mask32
		t += le16(l[lOff:]) & mask32
		t *= 0x04040404
		s = int32(t>>24) - 2 - 2
	case tx16x16:
		t := (le32(a[aOff:]) & mask32) >> 6
		t += (le32(l[lOff:]) & mask32) >> 6
		t *= mul32
		s = int32(t>>24) - 4 - 4
	case tx32x32:
		t := (le64(a[aOff:]) & mask64) >> 6
		t += (le64(l[lOff:]) & mask64) >> 6
		t *= mul64
		s = int32(t>>56) - 8 - 8
	case tx64x64:
		t := (le64(a[aOff:]) & mask64) >> 6
		t += (le64(a[aOff+8:]) & mask64) >> 6
		t += (le64(l[lOff:]) & mask64) >> 6
		t += (le64(l[lOff+8:]) & mask64) >> 6
		t *= mul64
		s = int32(t>>56) - 16 - 16
	case rtx4x8:
		t := uint32(a[aOff]) & mask32
		t += le16(l[lOff:]) & mask32
		t *= 0x04040404
		s = int32(t>>24) - 1 - 2
	case rtx8x4:
		t := le16(a[aOff:]) & mask32
		t += uint32(l[lOff]) & mask32
		t *= 0x04040404
		s = int32(t>>24) - 2 - 1
	case rtx8x16:
		t := le16(a[aOff:]) & mask32
		t += le32(l[lOff:]) & mask32
		t = (t >> 6) * mul32
		s = int32(t>>24) - 2 - 4
	case rtx16x8:
		t := le32(a[aOff:]) & mask32
		t += le16(l[lOff:]) & mask32
		t = (t >> 6) * mul32
		s = int32(t>>24) - 4 - 2
	case rtx16x32:
		t := uint64(le32(a[aOff:]) & mask32)
		t += le64(l[lOff:]) & mask64
		t = (t >> 6) * mul64
		s = int32(t>>56) - 4 - 8
	case rtx32x16:
		t := le64(a[aOff:]) & mask64
		t += uint64(le32(l[lOff:]) & mask32)
		t = (t >> 6) * mul64
		s = int32(t>>56) - 8 - 4
	case rtx32x64:
		t := (le64(a[aOff:]) & mask64) >> 6
		t += (le64(l[lOff:]) & mask64) >> 6
		t += (le64(l[lOff+8:]) & mask64) >> 6
		t *= mul64
		s = int32(t>>56) - 8 - 16
	case rtx64x32:
		t := (le64(a[aOff:]) & mask64) >> 6
		t += (le64(a[aOff+8:]) & mask64) >> 6
		t += (le64(l[lOff:]) & mask64) >> 6
		t *= mul64
		s = int32(t>>56) - 16 - 8
	case rtx4x16:
		t := uint32(a[aOff]) & mask32
		t += le32(l[lOff:]) & mask32
		t = (t >> 6) * mul32
		s = int32(t>>24) - 1 - 4
	case rtx16x4:
		t := le32(a[aOff:]) & mask32
		t += uint32(l[lOff]) & mask32
		t = (t >> 6) * mul32
		s = int32(t>>24) - 4 - 1
	case rtx8x32:
		t := uint64(le16(a[aOff:]) & mask32)
		t += le64(l[lOff:]) & mask64
		t = (t >> 6) * mul64
		s = int32(t>>56) - 2 - 8
	case rtx32x8:
		t := le64(a[aOff:]) & mask64
		t += uint64(le16(l[lOff:]) & mask32)
		t = (t >> 6) * mul64
		s = int32(t>>56) - 8 - 2
	case rtx16x64:
		t := uint64(le32(a[aOff:]) & mask32)
		t += le64(l[lOff:]) & mask64
		t = (t >> 6) + ((le64(l[lOff+8:]) & mask64) >> 6)
		t *= mul64
		s = int32(t>>56) - 4 - 16
	case rtx64x16:
		t := le64(a[aOff:]) & mask64
		t += uint64(le32(l[lOff:]) & mask32)
		t = (t >> 6) + ((le64(a[aOff+8:]) & mask64) >> 6)
		t *= mul64
		s = int32(t>>56) - 16 - 4
	}

	return b2u(s != 0) + b2u(s > 0)
}

func getLoCtx2D(levels []uint8, off int, offset uint32, stride int) (uint32, uint32) {
	r0 := (*[3]uint8)(levels[off : off+3])
	r1 := (*[2]uint8)(levels[off+stride : off+stride+2])
	mag := uint32(r0[1]) + uint32(r1[0]) + uint32(r1[1])
	hiMag := mag
	mag += uint32(r0[2]) + uint32(levels[off+2*stride])

	return offset + min((mag+64)>>7, 4), hiMag
}

func getLoCtx1D(levels []uint8, off int, y uint32, stride int) (uint32, uint32) {
	r0 := (*[5]uint8)(levels[off : off+5])
	mag := uint32(r0[1]) + uint32(levels[off+stride]) + uint32(r0[2])
	hiMag := mag
	mag += uint32(r0[3]) + uint32(r0[4])

	offset := uint32(26)
	if y > 1 {
		offset += 10
	} else {
		offset += y * 5
	}

	return offset + min((mag+64)>>7, 4), hiMag
}

func b2u(v bool) uint32 {
	if v {
		return 1
	}

	return 0
}

func b2u8(v bool) uint8 {
	if v {
		return 1
	}

	return 0
}

func decodeCoefs[P pixel, C coef](t *taskContext[P, C], a []uint8, aOff int,
	l []uint8, lOff int, tx, bs int, b *av1Block, intra, plane int, cf []C,
	txtpIn int,
) (eob int, txtp int, resCtx uint8) {
	txtp = txtpIn
	ts := t.ts
	chroma := b2i(plane != 0)
	f := t.f
	lossless := f.frameHdr.segmentation.lossless[b.segID]
	tDim := &txfmDimensions[tx]

	sctx := getSkipCtx(tDim, bs, a, aOff, l, lOff, chroma, f.layout)
	allSkip := ts.msac.boolAdapt(ts.cdf.coef.skip[tDim.ctx][sctx][:])
	if allSkip != 0 {
		return -1, int(lossless) * whtWht, 0x40
	}

	switch {
	case lossless != 0:
		txtp = whtWht
	case int(tDim.max)+intra >= tx64x64:
		txtp = dctDct
	case chroma != 0:
		if intra != 0 {
			txtp = int(txtpFromUvmode[b.uvMode])
		} else {
			txtp = getUvInterTxtp(tDim, txtp)
		}
	case f.frameHdr.segmentation.qidx[b.segID] == 0:
		txtp = dctDct
	default:
		var idx uint32
		if intra != 0 {
			yModeNofilt := b.yMode
			if b.yMode == filterPred {
				yModeNofilt = filterModeToYMode[b.yAngle]
			}
			if f.frameHdr.reducedTxtpSet != 0 || tDim.min == tx16x16 {
				idx = ts.msac.symbolAdapt(ts.cdf.m.txtpIntra2[tDim.min][yModeNofilt][:], 4)
				txtp = int(txTypesPerSet[idx+0])
			} else {
				idx = ts.msac.symbolAdapt(ts.cdf.m.txtpIntra1[tDim.min][yModeNofilt][:], 6)
				txtp = int(txTypesPerSet[idx+5])
			}
		} else {
			switch {
			case f.frameHdr.reducedTxtpSet != 0 || tDim.max == tx32x32:
				idx = ts.msac.boolAdapt(ts.cdf.m.txtpInter3[tDim.min][:])
				txtp = int(idx-1) & idtx
			case tDim.min == tx16x16:
				idx = ts.msac.symbolAdapt(ts.cdf.m.txtpInter2[:], 11)
				txtp = int(txTypesPerSet[idx+12])
			default:
				idx = ts.msac.symbolAdapt(ts.cdf.m.txtpInter1[tDim.min][:], 15)
				txtp = int(txTypesPerSet[idx+24])
			}
		}
	}

	slw, slh := min(int(tDim.lw), tx32x32), min(int(tDim.lh), tx32x32)
	tx2dszctx := slw + slh
	txClass := int(txTypeClass[txtp])
	is1d := b2i(txClass != txClass2d)

	switch tx2dszctx {
	case 0:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin16[chroma][is1d][:], 4+0))
	case 1:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin32[chroma][is1d][:], 4+1))
	case 2:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin64[chroma][is1d][:], 4+2))
	case 3:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin128[chroma][is1d][:], 4+3))
	case 4:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin256[chroma][is1d][:], 4+4))
	case 5:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin512[chroma][:], 4+5))
	case 6:
		eob = int(ts.msac.symbolAdapt(ts.cdf.coef.eobBin1024[chroma][:], 4+6))
	}

	if eob > 1 {
		eobBin := eob - 2
		eobHiBit := ts.msac.boolAdapt(ts.cdf.coef.eobHiBit[tDim.ctx][chroma][eobBin][:])
		eob = int((eobHiBit|2)<<eobBin) | int(ts.msac.bools(eobBin))
	}

	eobCdf := &ts.cdf.coef.eobBaseTok[tDim.ctx][chroma]
	hiCdf := &ts.cdf.coef.brTok[min(int(tDim.ctx), 3)][chroma]

	var rc, dcTok uint32

	if eob != 0 {
		loCdf := &ts.cdf.coef.baseTok[tDim.ctx][chroma]
		levels := t.scratch.levels[:]

		ctx := uint32(1 + b2i(eob > 2<<tx2dszctx) + b2i(eob > 4<<tx2dszctx))
		eobTok := int(ts.msac.symbolAdapt4(&eobCdf[ctx], 2))
		tok := eobTok + 1
		levelTok := tok * 0x41
		var mag uint32

		var scan []uint16
		var loCtxOff []uint8
		var stride, shift, shift2 int
		var mask uint32

		switch txClass {
		case txClass2d:
			loCtxOff = loCtxOffFromScan[tx]
			scan = scans[tx]
			stride = 4 << slh
			shift, shift2 = slh+2, 0
			mask = uint32(4<<slh) - 1
			clear(levels[:stride*((4<<slw)+2)])
		case txClassH:
			stride = 16
			shift, shift2 = slh+2, 0
			mask = uint32(4<<slh) - 1
			clear(levels[:stride*((4<<slh)+2)])
		default:
			stride = 16
			shift, shift2 = slw+2, slh+2
			mask = uint32(4<<slw) - 1
			clear(levels[:stride*((4<<slw)+2)])
		}

		var x, y uint32
		var level int

		switch txClass {
		case txClass2d:
			rc = uint32(scan[eob])
			x, y = rc>>shift, rc&mask
		case txClassH:
			x, y = uint32(eob)&mask, uint32(eob)>>shift
			rc = uint32(eob)
		default:
			x, y = uint32(eob)&mask, uint32(eob)>>shift
			rc = x<<shift2 | y
		}

		if eobTok == 2 {
			var c uint32
			if txClass == txClass2d {
				c = 7
				if x|y > 1 {
					c = 14
				}
			} else {
				c = 7
				if y != 0 {
					c = 14
				}
			}
			tok = int(ts.msac.hiTok(&hiCdf[c]))
			levelTok = tok + (3 << 6)
		}

		cf[rc] = C(tok << 11)
		if txClass == txClass2d {
			level = int(rc)
		} else {
			level = int(x)*stride + int(y)
		}
		levels[level] = uint8(levelTok)

		for i := eob - 1; i > 0; i-- {
			var rcI uint32
			if txClass == txClass2d {
				rcI = uint32(scan[i])
				level = int(rcI)
				ctx, mag = getLoCtx2D(levels, level, uint32(loCtxOff[i]), stride)
			} else {
				x, y = uint32(i)&mask, uint32(i)>>shift
				if txClass == txClassH {
					rcI = uint32(i)
				} else {
					rcI = x<<shift2 | y
				}
				level = int(x)*stride + int(y)
				ctx, mag = getLoCtx1D(levels, level, y, stride)
			}
			tok = int(ts.msac.symbolAdapt4(&loCdf[ctx], 3))
			if tok == 3 {
				mag &= 63
				base := uint32(7)
				if txClass == txClass2d {
					if (rcI>>shift)|(rcI&mask) > 1 {
						base = 14
					}
				} else if y != 0 {
					base = 14
				}
				if mag > 12 {
					ctx = base + 6
				} else {
					ctx = base + (mag+1)>>1
				}
				tok = int(ts.msac.hiTok(&hiCdf[ctx]))
				levels[level] = uint8(tok + (3 << 6))
				cf[rcI] = C(uint32(tok<<11) | rc)
				rc = rcI
			} else {
				tokv := uint32(tok) * 0x17ff41
				levels[level] = uint8(tokv)
				tokv = (tokv >> 9) & (rc + ^uint32(0x7ff))
				if tokv != 0 {
					rc = rcI
				}
				cf[rcI] = C(tokv)
			}
		}

		if txClass == txClass2d {
			ctx = 0
		} else {
			ctx, mag = getLoCtx1D(levels, 0, 0, stride)
		}
		dcTok = ts.msac.symbolAdapt4(&loCdf[ctx], 3)
		if dcTok == 3 {
			if txClass == txClass2d {
				mag = uint32(levels[0*stride+1]) + uint32(levels[1*stride+0]) +
					uint32(levels[1*stride+1])
			}
			mag &= 63
			if mag > 12 {
				ctx = 6
			} else {
				ctx = (mag + 1) >> 1
			}
			dcTok = ts.msac.hiTok(&hiCdf[ctx])
		}
	} else {
		tokBr := ts.msac.symbolAdapt4(&eobCdf[0], 2)
		dcTok = 1 + tokBr
		if tokBr == 2 {
			dcTok = ts.msac.hiTok(&hiCdf[0])
		}
		rc = 0
	}

	dqTbl := &ts.dq[b.segID][plane]
	var qmTbl []uint8
	if txtp < idtx {
		qmTbl = f.qm[tx][plane]
	}
	dqShift := max(0, int(tDim.ctx)-2)
	cfMax := int32(^(^uint32(127) << f.bpc))

	var culLevel, dcSignLevel uint32

	acLoop := func(useQm bool) {
		acDq := uint32(dqTbl[1])
		for rc != 0 {
			sign := ts.msac.boolEqui()
			rcTok := uint32(cf[rc])
			var tok, dq uint32

			if useQm {
				dq = (acDq*uint32(qmTbl[rc]) + 16) >> 5
				if rcTok >= 15<<11 {
					tok = readGolomb(&ts.msac) + 15
					tok &= 0xfffff
					dq = (dq * tok) & 0xffffff
				} else {
					tok = rcTok >> 11
					dq *= tok
				}
				culLevel += tok
				dq >>= dqShift
				dqSat := min(int32(dq), cfMax+int32(sign))
				if sign != 0 {
					cf[rc] = C(-dqSat)
				} else {
					cf[rc] = C(dqSat)
				}
			} else {
				var d int32
				if rcTok >= 15<<11 {
					tok = readGolomb(&ts.msac) + 15
					tok &= 0xfffff
					d = int32(((acDq * tok) & 0xffffff) >> dqShift)
					d = min(d, cfMax+int32(sign))
				} else {
					tok = rcTok >> 11
					d = int32((acDq * tok) >> dqShift)
				}
				culLevel += tok
				if sign != 0 {
					cf[rc] = C(-d)
				} else {
					cf[rc] = C(d)
				}
			}

			rc = rcTok & 0x3ff
		}
	}

	if dcTok == 0 {
		culLevel = 0
		dcSignLevel = 1 << 6
		acLoop(qmTbl != nil)

		return eob, txtp, uint8(min(culLevel, 63) | dcSignLevel)
	}

	dcSignCtx := getDcSignCtx(tx, a, aOff, l, lOff)
	dcSign := ts.msac.boolAdapt(ts.cdf.coef.dcSign[chroma][dcSignCtx][:])

	dcDq := uint32(dqTbl[0])
	dcSignLevel = (dcSign - 1) & (2 << 6)

	if qmTbl != nil {
		dcDq = (dcDq*uint32(qmTbl[0]) + 16) >> 5

		if dcTok == 15 {
			dcTok = readGolomb(&ts.msac) + 15
			dcTok &= 0xfffff
			dcDq = (dcDq * dcTok) & 0xffffff
		} else {
			dcDq *= dcTok
		}
		culLevel = dcTok
		dcDq >>= dqShift
		v := min(int32(dcDq), cfMax+int32(dcSign))
		if dcSign != 0 {
			cf[0] = C(-v)
		} else {
			cf[0] = C(v)
		}
	} else {
		var v int32
		if dcTok == 15 {
			dcTok = readGolomb(&ts.msac) + 15
			dcTok &= 0xfffff
			v = int32(((dcDq * dcTok) & 0xffffff) >> dqShift)
			v = min(v, cfMax+int32(dcSign))
		} else {
			v = int32((dcDq * dcTok) >> dqShift)
		}
		culLevel = dcTok
		if dcSign != 0 {
			cf[0] = C(-v)
		} else {
			cf[0] = C(v)
		}
	}

	acLoop(qmTbl != nil)

	return eob, txtp, uint8(min(culLevel, 63) | dcSignLevel)
}

func b2i(v bool) int {
	if v {
		return 1
	}

	return 0
}
