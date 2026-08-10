//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cflAcNormRVV(ac *int16, n int, log2sz int)

func init() {
	cflAcNorm = cflAcNormRVV
}
