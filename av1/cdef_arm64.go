//go:build arm64 && !noasm

package av1

//go:noescape
func cdefFilterNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClipNEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter4NEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip4NEON(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip16NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter416NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip416NEON(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

func cdefFilterBlockNEON(tmp []int16, dst []uint8, dstOff, dstStride int,
	left []uint8, leftOff int, top []uint8, topOff int, bottom []uint8, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy8NEONRows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	clip := priStrength != 0 && secStrength != 0
	switch {
	case w == 4 && clip:
		cdefFilterClip4NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case w == 4:
		cdefFilter4NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case clip:
		cdefFilterClipNEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	default:
		cdefFilterNEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	}
}

func cdefFilterBlock16NEON(tmp []int16, dst []uint16, dstOff, dstStride int,
	left []uint16, leftOff int, top []uint16, topOff int, bottom []uint16, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy16NEONRows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	clip := priStrength != 0 && secStrength != 0
	switch {
	case w == 4 && clip:
		cdefFilterClip416NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case w == 4:
		cdefFilter416NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case clip:
		cdefFilterClip16NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	default:
		cdefFilter16NEON(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	}
}
