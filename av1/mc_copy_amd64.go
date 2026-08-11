//go:build amd64 && !noasm

package av1

import "unsafe"

//go:noescape
func mcPut8Asm(dst *uint8, dstStride int, src *uint8, srcStride int, w, h int)

func mcPutFast[P pixel](dst []P, dstOff, dstStride int, src []P, srcOff, srcStride, w, h int) bool {
	if unsafe.Sizeof(*new(P)) != 1 || h&1 != 0 {
		return false
	}
	mcPut8Asm((*uint8)(unsafe.Pointer(&dst[dstOff])), dstStride,
		(*uint8)(unsafe.Pointer(&src[srcOff])), srcStride, w, h)

	return true
}
