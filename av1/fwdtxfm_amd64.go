//go:build amd64 && !noasm

package av1

//go:noescape
func fwdDot4AVX2(dst *float64, b *float64, src *float64, n int)

func fwdDot(dst, b, src []float64, n int) {
	if hasAVX2 && n&3 == 0 {
		fwdDot4AVX2(&dst[0], &b[0], &src[0], n)

		return
	}
	fwdDotGo(dst, b, src, n)
}
