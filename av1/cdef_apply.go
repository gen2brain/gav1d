package av1

const (
	backup2x8Y = 1 << iota
	backup2x8Uv
)

var cdefUvDirs = [2][8]uint8{
	{0, 1, 2, 3, 4, 5, 6, 7},
	{7, 0, 2, 4, 5, 6, 6, 6},
}

func backup2lines[P pixel, C coef](f *frameContext[P, C], dst *[3]int, src *[3]int) {
	yStride := f.stride[0]
	copy(f.lf.cdefLineBuf[dst[0]:dst[0]+2*yStride], f.cur[0][src[0]+6*yStride:])

	if f.layout == pixelLayoutI400 {
		return
	}

	uvStride := f.stride[1]
	uvOff := 6
	if f.layout == pixelLayoutI420 {
		uvOff = 2
	}
	for pl := 1; pl <= 2; pl++ {
		copy(f.lf.cdefLineBuf[dst[pl]:dst[pl]+2*uvStride], f.cur[pl][src[pl]+uvOff*uvStride:])
	}
}

func backup2x8[P pixel, C coef](f *frameContext[P, C], dst *[3][16]P, src *[3]int,
	xOff, flag int,
) {
	if flag&backup2x8Y != 0 {
		yOff := 0
		for y := range 8 {
			o := src[0] + yOff + xOff - 2
			*(*[2]P)(dst[0][y*2 : y*2+2]) = *(*[2]P)(f.cur[0][o : o+2])
			yOff += f.stride[0]
		}
	}

	if f.layout == pixelLayoutI400 || flag&backup2x8Uv == 0 {
		return
	}

	xOff >>= f.ssHor
	yOff := 0
	for y := range 8 >> f.ssVer {
		o1, o2 := src[1]+yOff+xOff-2, src[2]+yOff+xOff-2
		*(*[2]P)(dst[1][y*2 : y*2+2]) = *(*[2]P)(f.cur[1][o1 : o1+2])
		*(*[2]P)(dst[2][y*2 : y*2+2]) = *(*[2]P)(f.cur[2][o2 : o2+2])
		yOff += f.stride[1]
	}
}

func adjustStrength(strength int, variance uint32) int {
	if variance == 0 {
		return 0
	}

	i := 0
	if variance>>6 != 0 {
		i = min(ulog2(variance>>6), 12)
	}

	return (strength*(4+i) + 8) >> 4
}

func cdefBrow[P pixel, C coef](t *taskContext[P, C], p *[3]int, lflvl []av1Filter,
	byStart, byEnd int,
) {
	f := t.f
	bitdepthMin8 := f.bpc - 8
	edges := cdefHaveBottom
	if byStart > 0 {
		edges |= cdefHaveTop
	}
	ptrs := *p
	const sbsz = 16
	sb64w := f.sb128w << 1
	damping := int(f.frameHdr.cdef.damping) + bitdepthMin8
	ssVer, ssHor := f.ssVer, f.ssHor
	uvDir := &cdefUvDirs[b2i(f.layout == pixelLayoutI422)]
	yStride, uvStride := f.stride[0], f.stride[1]
	uvW, uvH := 8>>ssHor, 8>>ssVer

	for bit, by := 0, byStart; by < byEnd; by += 2 {
		tf := t.topPreCdefToggle
		byIdx := (by & 30) >> 1
		if by+2 >= f.bh {
			edges &^= cdefHaveBottom
		}

		if edges&cdefHaveBottom != 0 {
			backup2lines(f, &f.lf.cdefLine[1-tf], &ptrs)
		}

		lrBak := &t.scratch.cdefLrBak
		*lrBak = [2][3][16]P{}
		iptrs := ptrs
		edges &^= cdefHaveLeft
		edges |= cdefHaveRight
		prevFlag := 0

		for sbx := 0; sbx < sb64w; sbx++ {
			sb128x := sbx >> 1
			sb64Idx := ((by & sbsz) >> 3) + (sbx & 1)
			cdefIdx := int(lflvl[sb128x].cdefIdx[sb64Idx])

			yLvl, uvLvl := 0, 0
			if cdefIdx != -1 {
				yLvl = int(f.frameHdr.cdef.yStrength[cdefIdx])
				uvLvl = int(f.frameHdr.cdef.uvStrength[cdefIdx])
			}

			if cdefIdx == -1 || (yLvl == 0 && uvLvl == 0) {
				prevFlag = 0
			} else {
				noskipRow := &lflvl[sb128x].noskipMask[byIdx]
				noskipMask := uint32(noskipRow[1])<<16 | uint32(noskipRow[0])

				flag := b2i(yLvl != 0) + b2i(uvLvl != 0)<<1

				yPriLvl := (yLvl >> 2) << bitdepthMin8
				ySecLvl := yLvl & 3
				ySecLvl += b2i(ySecLvl == 3)
				ySecLvl <<= bitdepthMin8

				uvPriLvl := (uvLvl >> 2) << bitdepthMin8
				uvSecLvl := uvLvl & 3
				uvSecLvl += b2i(uvSecLvl == 3)
				uvSecLvl <<= bitdepthMin8

				bptrs := iptrs
				for bx := sbx * sbsz; bx < min((sbx+1)*sbsz, f.bw); bx += 2 {
					if bx+2 >= f.bw {
						edges &^= cdefHaveRight
					}

					if noskipMask&(3<<(bx&30)) == 0 {
						prevFlag = 0
					} else {
						doLeft := (prevFlag ^ flag) & flag
						prevFlag = flag
						if doLeft != 0 && edges&cdefHaveLeft != 0 {
							backup2x8(f, &lrBak[bit], &bptrs, 0, doLeft)
						}
						if edges&cdefHaveRight != 0 {
							backup2x8(f, &lrBak[1-bit], &bptrs, 8, flag)
						}

						dir := 0
						var variance uint32
						if yPriLvl != 0 || uvPriLvl != 0 {
							dir, variance = f.dsp.cdefDir(f.cur[0], bptrs[0], yStride, f.bitdepthMax)
						}

						topOff := f.lf.cdefLine[tf][0] + bx*4
						botOff := bptrs[0] + 8*yStride
						if yPriLvl != 0 {
							adjYPriLvl := adjustStrength(yPriLvl, variance)
							if adjYPriLvl != 0 || ySecLvl != 0 {
								f.dsp.cdefFilter(t.scratch.cdefTmp[:], f.cur[0], bptrs[0], yStride,
									lrBak[bit][0][:], 0, f.lf.cdefLineBuf, topOff,
									f.cur[0], botOff, adjYPriLvl, ySecLvl, dir,
									damping, 8, 8, edges, f.bitdepthMax)
							}
						} else if ySecLvl != 0 {
							f.dsp.cdefFilter(t.scratch.cdefTmp[:], f.cur[0], bptrs[0], yStride,
								lrBak[bit][0][:], 0, f.lf.cdefLineBuf, topOff,
								f.cur[0], botOff, 0, ySecLvl, 0,
								damping, 8, 8, edges, f.bitdepthMax)
						}

						if uvLvl != 0 {
							uvdir := 0
							if uvPriLvl != 0 {
								uvdir = int(uvDir[dir])
							}
							for pl := 1; pl <= 2; pl++ {
								topOff := f.lf.cdefLine[tf][pl] + (bx * 4 >> ssHor)
								botOff := bptrs[pl] + uvH*uvStride
								f.dsp.cdefFilter(t.scratch.cdefTmp[:], f.cur[pl], bptrs[pl], uvStride,
									lrBak[bit][pl][:], 0, f.lf.cdefLineBuf, topOff,
									f.cur[pl], botOff, uvPriLvl, uvSecLvl, uvdir,
									damping-1, uvW, uvH, edges, f.bitdepthMax)
							}
						}

						bit ^= 1
					}

					bptrs[0] += 8
					bptrs[1] += 8 >> ssHor
					bptrs[2] += 8 >> ssHor
					edges |= cdefHaveLeft
				}
			}

			iptrs[0] += sbsz * 4
			iptrs[1] += sbsz * 4 >> ssHor
			iptrs[2] += sbsz * 4 >> ssHor
			edges |= cdefHaveLeft
		}

		ptrs[0] += 8 * yStride
		ptrs[1] += 8 * uvStride >> ssVer
		ptrs[2] += 8 * uvStride >> ssVer
		t.topPreCdefToggle ^= 1
		edges |= cdefHaveTop
	}
}

func filterSbrowCdef[P pixel, C coef](t *taskContext[P, C], sby int) {
	f := t.f
	sbsz := f.sbStep
	y := sby * sbsz * 4
	isSb64 := b2i(f.seqHdr.sb128 == 0)

	p := [3]int{
		y * f.stride[0],
		y * f.stride[1] >> f.ssVer,
		y * f.stride[1] >> f.ssVer,
	}

	start := sby * sbsz
	if sby != 0 {
		pUp := [3]int{
			p[0] - 8*f.stride[0],
			p[1] - (8 * f.stride[1] >> f.ssVer),
			p[2] - (8 * f.stride[1] >> f.ssVer),
		}
		cdefBrow(t, &pUp, f.lflvl[((sby-1)>>isSb64)*f.sb128w:], start-2, start)
	}

	nBlks := sbsz - 2*b2i(sby+1 < f.sbh)
	end := min(start+nBlks, f.bh)
	cdefBrow(t, &p, f.lflvl[(sby>>isSb64)*f.sb128w:], start, end)
}
