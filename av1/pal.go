package av1

func palIdxFinish(dst, src []uint8, bw, bh, w, h int) {
	dstW := w / 2
	dstBw := bw / 2

	dstOff, srcOff := 0, 0
	for range h {
		for x := range dstW {
			dst[dstOff+x] = src[srcOff+x*2] | src[srcOff+x*2+1]<<4
		}
		if dstW < dstBw {
			memset(dst[dstOff+dstW:dstOff+dstBw], src[srcOff+w-1]*0x11)
		}
		srcOff += bw
		dstOff += dstBw
	}

	if h < bh {
		lastRow := dstOff - dstBw
		for y := h; y < bh; y++ {
			copy(dst[dstOff:dstOff+dstBw], dst[lastRow:lastRow+dstBw])
			dstOff += dstBw
		}
	}
}
