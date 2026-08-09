//go:build amd64 && !noasm

package av1

//go:noescape
func itxColDct4AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct8AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct16AVX2(tmp *int32, lanes, stride int, lo, hi int32)

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32) bool {
	if txtp != tx1dDct || lanes&7 != 0 {
		return false
	}

	switch n {
	case 4:
		itxColDct4AVX2(&tmp[0], lanes, stride, lo, hi)
	case 8:
		itxColDct8AVX2(&tmp[0], lanes, stride, lo, hi)
	case 16:
		itxColDct16AVX2(&tmp[0], lanes, stride, lo, hi)
	default:
		return false
	}

	return true
}
