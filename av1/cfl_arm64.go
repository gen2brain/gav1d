//go:build arm64 && !noasm

package av1

import "unsafe"

//go:noescape
func cflAcNormNEON(ac *int16, n int, log2sz int)

//go:noescape
func cflPredict8NEON(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16NEON(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflAcMain8NEON(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)

//go:noescape
func z2Top8NEON(dst *uint8, edge *uint8, n int, frac int32)

//go:noescape
func z2Top16NEON(dst *uint16, edge *uint16, n int, frac int32)

func init() {
	cflAcNorm = cflAcNormNEON
	cflAcMain8Asm = cflAcMain8NEON
	cflPredict8Asm = cflPredict8NEON
	cflPredict16Asm = cflPredict16NEON
}

func z2LeftRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n, ypos, dy int) int {
	return 0
}

// z2TopRun fills the columns that read the top edge, where the fraction is
// constant across the row, and reports how many it took.
func z2TopRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n int, frac int32) int {
	var z P
	if unsafe.Sizeof(z) == 1 {
		if n16 := n &^ 15; n16 > 0 {
			z2Top8NEON((*uint8)(unsafe.Pointer(&dst[dstOff])),
				(*uint8)(unsafe.Pointer(&edge[edgeOff])), n16, frac)

			return n16
		}
	}
	if unsafe.Sizeof(z) == 2 {
		if n8 := n &^ 7; n8 > 0 {
			z2Top16NEON((*uint16)(unsafe.Pointer(&dst[dstOff])),
				(*uint16)(unsafe.Pointer(&edge[edgeOff])), n8, frac)

			return n8
		}
	}

	return 0
}
