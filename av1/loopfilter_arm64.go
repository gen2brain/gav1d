//go:build arm64 && !noasm

package av1

//go:noescape
func lpf8ColNEON(dst *uint8, stridea, strideb int, p *lpfParams, n int)

//go:noescape
func lpf8RowNEON(dst *uint8, stridea, strideb int, p *lpfParams, back int)

func loopFilter8NEON(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	if stridea == 1 {
		for ; lines >= 8; lines, dstOff = lines-8, dstOff+8 {
			lpf8ColNEON(&dst[dstOff], stridea, strideb, &p, 8)
		}
		for ; lines > 0; lines, dstOff = lines-4, dstOff+4 {
			lpf8ColNEON(&dst[dstOff], stridea, strideb, &p, 4)
		}

		return
	}
	for ; lines >= 8; lines, dstOff = lines-8, dstOff+8*stridea {
		lpf8RowNEON(&dst[dstOff], stridea, strideb, &p, stridea)
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf8RowNEON(&dst[dstOff], stridea, strideb, &p, -3*stridea)
	}
}

//go:noescape
func lpf16NEON(dst *uint16, stridea, strideb int, p *lpfParams)

func loopFilter16NEON(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf16NEON(&dst[dstOff], stridea, strideb, &p)
	}
}
