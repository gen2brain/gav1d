//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cflAcNormRVV(ac *int16, n int, log2sz int)

//go:noescape
func cflPredict8RVV(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16RVV(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflAcMain8RVV(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)

//go:noescape
func z2Top8RVV(dst *uint8, edge *uint8, n int, frac int32)

//go:noescape
func z2Top16RVV(dst *uint16, edge *uint16, n int, frac int32)

func init() {
	cflAcNorm = cflAcNormRVV
	cflAcMain8Asm = cflAcMain8RVV
	cflPredict8Asm = cflPredict8RVV
	cflPredict16Asm = cflPredict16RVV
	z2Top8Asm = z2Top8RVV
	z2Top16Asm = z2Top16RVV
}
