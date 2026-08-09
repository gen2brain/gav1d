//go:build arm64 && !noasm

package avif

//go:noescape
func convertRow4NEON(dst *uint8, yf, u0, u1, v0, v1 *float32, a *uint8, n int, c *rowConsts)

func rowFn() convertRow { return convertRowNEON }

func convertRowNEON(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts,
) {
	if n4 := n &^ 3; n4 > 0 {
		convertRow4NEON(&dst[0], &yf[0], &u0[0], &u1[0], &v0[0], &v1[0], &a[0], n4, c)
		if n4 == n {
			return
		}
		dst, yf, a, n = dst[4*n4:], yf[n4:], a[n4:], n-n4
		u0, u1 = u0[n4/2:], u1[n4/2:]
		v0, v1 = v0[n4/2:], v1[n4/2:]
	}

	convertRowGo(dst, yf, u0, u1, v0, v1, a, n, c)
}
