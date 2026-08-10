//go:build riscv64 && riscv64.rva23u64 && !noasm

package avif

//go:noescape
func convertRowRVV(dst *uint8, yf, u0, u1, v0, v1 *float32, a *uint8, n int, c *rowConsts)

func rowFn(ssHor int) convertRow {
	if ssHor == 0 {
		return convertRow444Go
	}

	return convertRowVec
}

func convertRowVec(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts,
) {
	if n4 := n &^ 3; n4 > 0 {
		convertRowRVV(&dst[0], &yf[0], &u0[0], &u1[0], &v0[0], &v1[0], &a[0], n4, c)
		if n4 == n {
			return
		}
		dst, yf, a, n = dst[4*n4:], yf[n4:], a[n4:], n-n4
		u0, u1 = u0[n4/2:], u1[n4/2:]
		v0, v1 = v0[n4/2:], v1[n4/2:]
	}

	convertRowGo(dst, yf, u0, u1, v0, v1, a, n, c)
}

//go:noescape
func convertRow16RVV(dst *uint8, yf, u0, u1, v0, v1 *float32, a *uint16, n int, c *rowConsts)

func rowFn16(ssHor int) convertRow16 {
	if ssHor == 0 {
		return convertRow16x444Go
	}

	return convertRow16Vec
}

func convertRow16Vec(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint16,
	n int, c *rowConsts,
) {
	if n4 := n &^ 3; n4 > 0 {
		convertRow16RVV(&dst[0], &yf[0], &u0[0], &u1[0], &v0[0], &v1[0], &a[0], n4, c)
		if n4 == n {
			return
		}
		dst, yf, a, n = dst[8*n4:], yf[n4:], a[n4:], n-n4
		u0, u1 = u0[n4/2:], u1[n4/2:]
		v0, v1 = v0[n4/2:], v1[n4/2:]
	}

	convertRow16Go(dst, yf, u0, u1, v0, v1, a, n, c)
}
