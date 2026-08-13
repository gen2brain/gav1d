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

	dq := uint32(e.dq[0])
	shift := quantShift(tx)
	cfMax := int32(1<<15 - 1)

	var res [64 * 64]int32
	var coeff [32 * 32]int32
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
			level := quantOne(coeff[0], dq, shift)

			ctx = e.writeDCOnly(&coefCtx{a: e.a[bx>>5].lcoef[:], l: e.l.lcoef[:]},
				(bx+x)&31, (by+y)&31, tx, bs, 0, dcPred, level)

			memset(e.a[bx>>5].lcoef[(bx+x)&31:][:int(tDim.w)], ctx)
			memset(e.l.lcoef[(by+y)&31:][:int(tDim.h)], ctx)

			if level != 0 {
				clear(cf[:tw*th])
				d := int32(uint32(abs32(level)) * dq >> uint(shift))
				d = min(d, cfMax)
				if level < 0 {
					d = -d
				}
				cf[0] = int16(d)
				invTxfmAdd(e.dsp, e.recon, off, e.stride, cf, e.itx[:], 0, tx, dctDct, 255)
			}
		}
	}

	return ctx
}
