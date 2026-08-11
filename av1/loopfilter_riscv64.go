//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func lpfVL16RVV() int

// The eight bit kernels widen every tap of every line at once and the wide one
// takes sixteen lines to a register, both of which need VLEN 256.
var lpfWideRVV = lpfVL16RVV() == 16

//go:noescape
func lpf8RVV(dst *uint8, stridea, strideb int, p *lpfParams, n int)

//go:noescape
func lpf8WideRVV(dst *uint8, stridea, strideb int, p *lpfParams)

func loopFilter8RVV(dst []uint8, dstOff int, e, i, h int32,
	stridea, strideb int, wd, lines int, bitdepthMax int32,
) {
	if !lpfWideRVV {
		loopFilter(dst, dstOff, e, i, h, stridea, strideb, wd, lines, bitdepthMax)

		return
	}

	p := lpfParamsFor(e, i, h, wd, bitdepthMax)
	for ; lines >= 16; lines, dstOff = lines-16, dstOff+16*stridea {
		lpf8WideRVV(&dst[dstOff], stridea, strideb, &p)
	}
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
