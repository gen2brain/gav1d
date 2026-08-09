//go:build arm64 && !noasm

package av1

//go:noescape
func cdefFilterNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClipNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

func cdefFilterBlockNEON(dst []uint8, dstOff, dstStride int,
	left []uint8, leftOff int, top []uint8, topOff int, bottom []uint8, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	var tmpBuf [144]int16
	tmpOff := 2*tmpStride + 2

	cdefPadding(tmpBuf[:], tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	if priStrength != 0 && secStrength != 0 {
		cdefFilterClipNEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)

		return
	}

	cdefFilterNEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)
}

func cdefFilterBlock16NEON(dst []uint16, dstOff, dstStride int,
	left []uint16, leftOff int, top []uint16, topOff int, bottom []uint16, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	var tmpBuf [144]int16
	tmpOff := 2*tmpStride + 2

	cdefPadding(tmpBuf[:], tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	if priStrength != 0 && secStrength != 0 {
		cdefFilterClip16NEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)

		return
	}

	cdefFilter16NEON(&dst[dstOff], dstStride, &tmpBuf[tmpOff], &p)
}
