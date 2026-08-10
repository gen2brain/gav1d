//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cflAcNormRVV(ac *int16, n int, log2sz int)

//go:noescape
func cflPredict8RVV(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)

//go:noescape
func cflPredict16RVV(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)

func init() {
	cflAcNorm = cflAcNormRVV
	cflPredict8Asm = cflPredict8RVV
	cflPredict16Asm = cflPredict16RVV
}
