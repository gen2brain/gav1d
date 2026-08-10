//go:build amd64 && !noasm

package av1

//go:noescape
func lpf8AVX2(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8AVX2(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf8AVX2(&dst[dstOff], stridea, strideb, &p)
}

//go:noescape
func lpf8AVX512(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8AVX512(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf8AVX512(&dst[dstOff], stridea, strideb, &p)
}

//go:noescape
func lpf16AVX2(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16AVX512(dst *uint16, stridea, strideb int, p *lpfParams)

func loopFilter16AVX512(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf16AVX512(&dst[dstOff], stridea, strideb, &p)
}

func loopFilter16AVX2(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	lpf16AVX2(&dst[dstOff], stridea, strideb, &p)
}
