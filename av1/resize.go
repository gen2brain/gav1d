package av1

func resize[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride int,
	dstW, h, srcW, dx, mx0 int, bitdepthMax int32,
) {
	for range h {
		mx, srcX := mx0, -1
		for x := range dstW {
			f := &resizeFilter[mx>>8]
			sum := int32(0)
			for i := range 8 {
				sum += int32(f[i]) * int32(src[srcOff+clip(srcX-3+i, 0, srcW-1)])
			}
			dst[dstOff+x] = P(clip((-sum+64)>>7, 0, bitdepthMax))

			mx += dx
			srcX += mx >> 14
			mx &= 0x3fff
		}
		dstOff += dstStride
		srcOff += srcStride
	}
}

func scaleFac(refSz, thisSz int) int {
	return ((refSz << 14) + (thisSz >> 1)) / thisSz
}

func getUpscaleX0(inW, outW, step int) int {
	err := outW*step - (inW << 14)

	return ((-((outW-inW)<<13)+(outW>>1))/outW + 128 - err/2) & 0x3fff
}

func filterSbrowResize[P pixel, C coef](f *frameContext[P, C], sby int) {
	sbsz := f.sbStep
	y := sby * sbsz * 4
	hasChroma := f.layout != pixelLayoutI400

	nPl := 1
	if hasChroma {
		nPl = 3
	}

	for pl := range nPl {
		chroma := b2i(pl != 0)
		ssVer := chroma & b2i(f.layout == pixelLayoutI420)
		ssHor := chroma & b2i(f.layout != pixelLayoutI444)
		hStart := 8 * b2i(sby != 0) >> ssVer

		dstStride := f.srStride[chroma]
		srcStride := f.stride[chroma]
		dstOff := (y * dstStride >> ssVer) - hStart*dstStride
		srcOff := (y * srcStride >> ssVer) - hStart*srcStride

		hEnd := 4 * (sbsz - 2*b2i(sby+1 < f.sbh)) >> ssVer
		dstW := (f.srWidth + ssHor) >> ssHor
		srcW := (4*f.bw + ssHor) >> ssHor
		imgH := (f.frameHdr.height - sbsz*4*sby + ssVer) >> ssVer

		resize(f.srCur[pl], dstOff, dstStride, f.cur[pl], srcOff, srcStride,
			dstW, min(imgH, hEnd)+hStart, srcW,
			f.resizeStep[chroma], f.resizeStart[chroma], f.bitdepthMax)
	}
}
