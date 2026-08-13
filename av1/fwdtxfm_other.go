//go:build !amd64 || noasm

package av1

func fwdDot(dst, b, src []float64, n int) {
	fwdDotGo(dst, b, src, n)
}
