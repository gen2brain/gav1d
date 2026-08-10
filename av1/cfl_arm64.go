//go:build arm64 && !noasm

package av1

//go:noescape
func cflAcNormNEON(ac *int16, n int, log2sz int)

func init() {
	cflAcNorm = cflAcNormNEON
}
