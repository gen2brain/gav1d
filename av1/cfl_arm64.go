//go:build arm64 && !noasm

package av1

//go:noescape
func cflAcNormNEON(ac *int16, n int, log2sz int)

//go:noescape
func cflPredict8NEON(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16NEON(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func z2Top8NEON(dst *uint8, edge *uint8, n int, frac int32)

//go:noescape
func z2Top16NEON(dst *uint16, edge *uint16, n int, frac int32)

func init() {
	cflAcNorm = cflAcNormNEON
	cflPredict8Asm = cflPredict8NEON
	cflPredict16Asm = cflPredict16NEON
	z2Top8Asm = z2Top8NEON
	z2Top16Asm = z2Top16NEON
}
