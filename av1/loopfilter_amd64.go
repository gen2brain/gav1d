//go:build amd64 && !noasm

package av1

//go:noescape
func lpf8AVX2(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8WAVX2(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8HWAVX2(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8AVX2(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	switch {
	case stridea == 1:
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8 {
			lpf8WAVX2(&dst[dstOff], stridea, strideb, &p)
		}
	case wd < 16:
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8*stridea {
			lpf8HWAVX2(&dst[dstOff], stridea, strideb, &p)
		}
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf8AVX2(&dst[dstOff], stridea, strideb, &p)
	}
}

//go:noescape
func lpf8AVX512(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8WAVX512(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8HWAVX512(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8ZAVX512(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8HZAVX512(dst *uint8, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf8HZ14AVX512(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8AVX512(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	switch {
	case stridea == 1:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16 {
			lpf8ZAVX512(&dst[dstOff], stridea, strideb, &p)
		}
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8 {
			lpf8WAVX512(&dst[dstOff], stridea, strideb, &p)
		}
	case wd < 16:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16*stridea {
			lpf8HZAVX512(&dst[dstOff], stridea, strideb, &p)
		}
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8*stridea {
			lpf8HWAVX512(&dst[dstOff], stridea, strideb, &p)
		}
	default:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16*stridea {
			lpf8HZ14AVX512(&dst[dstOff], stridea, strideb, &p)
		}
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf8AVX512(&dst[dstOff], stridea, strideb, &p)
	}
}

//go:noescape
func lpf16AVX2(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16WAVX2(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16AVX512(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16WAVX512(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16ZAVX512(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16HZAVX512(dst *uint16, stridea, strideb int, p *lpfParams)

//go:noescape
func lpf16HZ14AVX512(dst *uint16, stridea, strideb int, p *lpfParams)

func loopFilter16AVX512(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	switch {
	case stridea == 1:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16 {
			lpf16ZAVX512(&dst[dstOff], stridea, strideb, &p)
		}
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8 {
			lpf16WAVX512(&dst[dstOff], stridea, strideb, &p)
		}
	case wd < 16:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16*stridea {
			lpf16HZAVX512(&dst[dstOff], stridea, strideb, &p)
		}
	default:
		for ; lines >= 16; lines, dstOff = lines-16, dstOff+16*stridea {
			lpf16HZ14AVX512(&dst[dstOff], stridea, strideb, &p)
		}
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf16AVX512(&dst[dstOff], stridea, strideb, &p)
	}
}

func loopFilter16AVX2(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	if stridea == 1 {
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8 {
			lpf16WAVX2(&dst[dstOff], stridea, strideb, &p)
		}
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf16AVX2(&dst[dstOff], stridea, strideb, &p)
	}
}
