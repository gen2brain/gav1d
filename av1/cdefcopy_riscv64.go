//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cdefCopy8RVV(tmp *int16, tmpStride int, src *uint8, srcStride, n, h int)

//go:noescape
func cdefCopy16RVV(tmp *int16, tmpStride int, src *uint16, srcStride, n, h int)

func cdefCopy8RVVRows(tmp []int16, tmpOff, tmpStride int,
	src []uint8, srcOff, srcStride, n, h int,
) {
	if n == 0 || h == 0 {
		return
	}
	cdefCopy8RVV(&tmp[tmpOff], tmpStride, &src[srcOff], srcStride, n, h)
}

func cdefCopy16RVVRows(tmp []int16, tmpOff, tmpStride int,
	src []uint16, srcOff, srcStride, n, h int,
) {
	if n == 0 || h == 0 {
		return
	}
	cdefCopy16RVV(&tmp[tmpOff], tmpStride, &src[srcOff], srcStride, n, h)
}
