package av1

func dcPredict(recon []uint8, off, stride, w, h int, haveTop, haveLeft bool, mid uint8) uint8 {
	switch {
	case haveTop && haveLeft:
		sum := 0
		for x := range w {
			sum += int(recon[off-stride+x])
		}
		for y := range h {
			sum += int(recon[off+y*stride-1])
		}

		return uint8((sum + (w+h)>>1) / (w + h))
	case haveTop:
		sum := 0
		for x := range w {
			sum += int(recon[off-stride+x])
		}

		return uint8((sum + w>>1) / w)
	case haveLeft:
		sum := 0
		for y := range h {
			sum += int(recon[off+y*stride-1])
		}

		return uint8((sum + h>>1) / h)
	}

	return mid
}

func (e *encoder) reconBlock(bx, by, bs int) uint8 {
	tx := int(maxTxfmSizeForBs[bs][0])
	tDim := &txfmDimensions[tx]
	tw, th := 4*int(tDim.w), 4*int(tDim.h)
	bDim := &blockDimensions[bs]
	bw4, bh4 := int(bDim[0]), int(bDim[1])

	shift := quantShift(tx)
	cfMax := int32(1<<15 - 1)

	var res [64 * 64]int32
	var coeff [32 * 32]int32
	var lv [32 * 32]int32
	cf := make([]int16, 32*32)

	ctx := uint8(0x40)
	for y := 0; y < bh4 && by+y < e.bh; y += int(tDim.h) {
		for x := 0; x < bw4 && bx+x < e.bw; x += int(tDim.w) {
			px, py := (bx+x)*4, (by+y)*4
			off := py*e.stride + px
			w := min(tw, e.cfg.Width-px)
			h := min(th, e.cfg.Height-py)

			pred := dcPredict(e.recon, off, e.stride, w, h, py > 0, px > 0, 128)
			for j := range th {
				row := off + j*e.stride
				for i := range tw {
					e.recon[row+i] = pred
				}
			}
			for j := range th {
				for i := range tw {
					sx := min(px+i, e.cfg.Width-1)
					sy := min(py+j, e.cfg.Height-1)
					res[j*tw+i] = int32(e.cfg.Src[sy*e.cfg.SrcStride+sx]) - int32(pred)
				}
			}

			fwdTxfm(coeff[:], res[:], tw, tx, dctDct)
			sw, sh := min(tw, 32), min(th, 32)
			for i := range sw * sh {
				d := e.dq[1]
				if i == 0 {
					d = e.dq[0]
				}
				lv[i] = quantOne(coeff[i], uint32(d), shift)
			}

			ctx = e.writeCoefs(&coefCtx{a: e.a[bx>>5].lcoef[:], l: e.l.lcoef[:]},
				(bx+x)&31, (by+y)&31, tx, bs, 0, dcPred, lv[:], e.levels[:])

			aw := min(int(tDim.w), e.bw-(bx+x))
			lh := min(int(tDim.h), e.bh-(by+y))
			memset(e.a[bx>>5].lcoef[(bx+x)&31:][:aw], ctx)
			memset(e.l.lcoef[(by+y)&31:][:lh], ctx)

			eob := -1
			for i := sw*sh - 1; i >= 0; i-- {
				if lv[scans[tx][i]] != 0 {
					eob = i

					break
				}
			}
			if eob >= 0 {
				clear(cf[:sw*sh])
				for i := range sw * sh {
					if lv[i] == 0 {
						continue
					}
					dqv := uint32(e.dq[1])
					if i == 0 {
						dqv = uint32(e.dq[0])
					}
					d := int32(uint32(abs32(lv[i])) * dqv >> uint(shift))
					d = min(d, cfMax)
					if lv[i] < 0 {
						d = -d
					}
					cf[i] = int16(d)
				}
				invTxfmAdd(e.dsp, e.recon, off, e.stride, cf, e.itx[:], eob, tx, dctDct, 255)
			}
		}
	}

	return ctx
}
