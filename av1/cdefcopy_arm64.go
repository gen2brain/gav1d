//go:build arm64 && !noasm

package av1

//go:noescape
func cdefCopy8NEON(tmp *int16, tmpStride int, src *uint8, srcStride, n, h int)

//go:noescape
func cdefCopy16NEON(tmp *int16, tmpStride int, src *uint16, srcStride, n, h int)

func cdefCopy8NEONRows(tmp []int16, tmpOff, tmpStride int,
	src []uint8, srcOff, srcStride, n, h int,
) {
	if n == 0 || h == 0 {
		return
	}
	cdefCopy8NEON(&tmp[tmpOff], tmpStride, &src[srcOff], srcStride, n, h)
}

func cdefCopy16NEONRows(tmp []int16, tmpOff, tmpStride int,
	src []uint16, srcOff, srcStride, n, h int,
) {
	if n == 0 || h == 0 {
		return
	}
	cdefCopy16NEON(&tmp[tmpOff], tmpStride, &src[srcOff], srcStride, n, h)
}
