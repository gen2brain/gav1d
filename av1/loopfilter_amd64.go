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
