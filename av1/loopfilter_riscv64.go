//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func lpf8RVV(dst *uint8, stridea, strideb int, p *lpfParams, n int)

func loopFilter8RVV(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	for ; lines >= 8; lines, dstOff = lines-8, dstOff+8*stridea {
		lpf8RVV(&dst[dstOff], stridea, strideb, &p, 8)
	}
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf8RVV(&dst[dstOff], stridea, strideb, &p, 4)
	}
}

//go:noescape
func lpf16RVV(dst *uint16, stridea, strideb int, p *lpfParams)

func loopFilter16RVV(dst []uint16, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	for ; lines > 0; lines, dstOff = lines-4, dstOff+4*stridea {
		lpf16RVV(&dst[dstOff], stridea, strideb, &p)
	}
}
