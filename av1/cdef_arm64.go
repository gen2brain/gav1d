//go:build arm64 && !noasm

package av1

type cdefParams struct {
	off      [6]int32
	priStr   int32
	secStr   int32
	priShift int32
	secShift int32
	priTap   [2]int32
	secTap   [2]int32
	w        int32
	h        int32
}

//go:noescape
func cdefFilterNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClipNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

func cdefFilterBlockNEON(dst []uint8, dstOff, dstStride int,
	left []uint8, leftOff int, top []uint8, topOff int, bottom []uint8, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	var tmpBuf [144]int16
	tmpOff := 2*tmpStride + 2

	cdefPadding(tmpBuf[:], tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	dirs := &cdefDirections
	p := cdefParams{
		off: [6]int32{
			int32(dirs[dir+2][0]), int32(dirs[dir+2][1]),
			int32(dirs[dir+4][0]), int32(dirs[dir+4][1]),
			int32(dirs[dir+0][0]), int32(dirs[dir+0][1]),
		},
		priStr: int32(priStrength),
		secStr: int32(secStrength),
		secTap: [2]int32{2, 1},
		w:      int32(w),
		h:      int32(h),
	}

	if priStrength != 0 {
		bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
		tap := int32(4 - ((priStrength >> bitdepthMin8) & 1))
		p.priTap = [2]int32{tap, (tap & 3) | 2}
		p.priShift = int32(max(0, damping-ulog2(uint32(priStrength))))
	}
	if secStrength != 0 {
		p.secShift = int32(damping - ulog2(uint32(secStrength)))
	}

	if priStrength != 0 && secStrength != 0 {
		cdefFilterClipNEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)

		return
	}

	cdefFilterNEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)
}
