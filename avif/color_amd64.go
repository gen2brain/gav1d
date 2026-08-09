//go:build amd64 && !noasm

package avif

//go:noescape
func convertRow8AVX2(dst *uint8, yf, u0, u1, v0, v1 *float32, a *uint8, n int, c *rowConsts)

func rowFn() convertRow {
	if !hasAVX2 {
		return convertRowGo
	}

	return convertRowAVX2
}

func convertRowAVX2(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts,
) {
	if n8 := n &^ 7; n8 > 0 {
		convertRow8AVX2(&dst[0], &yf[0], &u0[0], &u1[0], &v0[0], &v1[0], &a[0], n8, c)
		if n8 == n {
			return
		}
		dst, yf, a, n = dst[4*n8:], yf[n8:], a[n8:], n-n8
		u0, u1 = u0[n8/2:], u1[n8/2:]
		v0, v1 = v0[n8/2:], v1[n8/2:]
	}

	convertRowGo(dst, yf, u0, u1, v0, v1, a, n, c)
}
