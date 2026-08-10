//go:build amd64 && !noasm

package av1

//go:noescape
func cdefFilterAVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClipAVX2(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip16AVX2(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

func cdefFilterBlockAVX2(tmp []int16, dst []uint8, dstOff, dstStride int,
	left []uint8, leftOff int, top []uint8, topOff int, bottom []uint8, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy8AVX2Rows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	if priStrength != 0 && secStrength != 0 {
		cdefFilterClipAVX2(&dst[dstOff], dstStride, &tmp[tmpOff], &p)

		return
	}

	cdefFilterAVX2(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
}

func cdefFilterBlock16AVX2(tmp []int16, dst []uint16, dstOff, dstStride int,
	left []uint16, leftOff int, top []uint16, topOff int, bottom []uint16, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy16AVX2Rows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	if priStrength != 0 && secStrength != 0 {
		cdefFilterClip16AVX2(&dst[dstOff], dstStride, &tmp[tmpOff], &p)

		return
	}

	cdefFilter16AVX2(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
}
