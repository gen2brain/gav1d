//go:build amd64 && !noasm

package av1

import "unsafe"

//go:noescape
func mcPut8Asm(dst *uint8, dstStride int, src *uint8, srcStride int, w, h int)

func mcPutFast[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride, w, h int) bool {
	if h&1 != 0 {
		return false
	}
	n := int(unsafe.Sizeof(*new(P)))
	mcPut8Asm((*uint8)(unsafe.Pointer(&dst[dstOff])), dstStride*n,
		(*uint8)(unsafe.Pointer(&src[srcOff])), srcStride*n, w*n, h)

	return true
}
