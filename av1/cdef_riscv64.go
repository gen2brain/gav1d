//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cdefFilterRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClipRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter4RVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip4RVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter16RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip16RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilter416RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

//go:noescape
func cdefFilterClip416RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)

func cdefFilterBlockRVV(tmp []int16, dst []uint8, dstOff, dstStride int,
	left []uint8, leftOff int, top []uint8, topOff int, bottom []uint8, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy8RVVRows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	clip := priStrength != 0 && secStrength != 0
	switch {
	case w == 4 && clip:
		cdefFilterClip4RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case w == 4:
		cdefFilter4RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case clip:
		cdefFilterClipRVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	default:
		cdefFilterRVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	}
}

func cdefFilterBlock16RVV(tmp []int16, dst []uint16, dstOff, dstStride int,
	left []uint16, leftOff int, top []uint16, topOff int, bottom []uint16, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	tmpOff := 2*tmpStride + 2

	cdefPadding(cdefCopy16RVVRows, tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	p := cdefParamsFor(priStrength, secStrength, dir, damping, w, h, bitdepthMax)

	clip := priStrength != 0 && secStrength != 0
	switch {
	case w == 4 && clip:
		cdefFilterClip416RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case w == 4:
		cdefFilter416RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	case clip:
		cdefFilterClip16RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	default:
		cdefFilter16RVV(&dst[dstOff], dstStride, &tmp[tmpOff], &p)
	}
}
