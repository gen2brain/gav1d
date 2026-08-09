//go:build arm64 && !noasm

package av1

//go:noescape
func lpf8NEON(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8NEON(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf8NEON(&dst[dstOff], stridea, strideb, &p)
}

//go:noescape
func lpf16NEON(dst *uint16, stridea, strideb int, p *lpfParams)

func loopFilter16NEON(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf16NEON(&dst[dstOff], stridea, strideb, &p)
}
