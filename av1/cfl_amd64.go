//go:build amd64 && !noasm

package av1

//go:noescape
func cflPredict8AVX2(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16AVX2(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflAcNormAVX2(ac *int16, n int, log2sz int)

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
	z2Top8Asm = z2Top8AVX2
	z2Top16Asm = z2Top16AVX2
}
