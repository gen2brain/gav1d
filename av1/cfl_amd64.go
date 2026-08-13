//go:build amd64 && !noasm

package av1

import "unsafe"

//go:noescape
func cflPredict8AVX2(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16AVX2(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflAcNormAVX2(ac *int16, n int, log2sz int)

//go:noescape
func cflAcMain8AVX2(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)

//go:noescape
func z2Left8AVX2(dst *uint8, edge *uint8, n int, ypos, dy int32)

//go:noescape
func z2Top8AVX2(dst *uint8, edge *uint8, n int, frac int32)

//go:noescape
func z2Top16AVX2(dst *uint16, edge *uint16, n int, frac int32)

func init() {
	if !hasAVX2 {
		return
	}

	cflPredict8Asm = cflPredict8AVX2
	cflPredict16Asm = cflPredict16AVX2
	cflAcNorm = cflAcNormAVX2
	cflAcMain8Asm = cflAcMain8AVX2
}

func z2LeftRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n, ypos, dy int) int {
	var z P
	if unsafe.Sizeof(z) == 1 && hasAVX2 {
		if n8 := n &^ 7; n8 > 0 {
			z2Left8AVX2((*uint8)(unsafe.Pointer(&dst[dstOff])),
				(*uint8)(unsafe.Pointer(&edge[edgeOff])), n8, int32(ypos), int32(dy))

			return n8
		}
	}

	return 0
}

// z2TopRun fills the columns that read the top edge, where the fraction is
// constant across the row, and reports how many it took.
func z2TopRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n int, frac int32) int {
	var z P
	if unsafe.Sizeof(z) == 1 && hasAVX2 {
		if n16 := n &^ 15; n16 > 0 {
			z2Top8AVX2((*uint8)(unsafe.Pointer(&dst[dstOff])),
				(*uint8)(unsafe.Pointer(&edge[edgeOff])), n16, frac)

			return n16
		}
	}
	if unsafe.Sizeof(z) == 2 && hasAVX2 {
		if n8 := n &^ 7; n8 > 0 {
			z2Top16AVX2((*uint16)(unsafe.Pointer(&dst[dstOff])),
				(*uint16)(unsafe.Pointer(&edge[edgeOff])), n8, frac)

			return n8
		}
	}

	return 0
}
