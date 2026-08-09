//go:build arm64 && !noasm

package av1

//go:noescape
func itxColDct4NEON(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct8NEON(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct16NEON(tmp *int32, lanes, stride int, lo, hi int32)

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32) bool {
	if txtp != tx1dDct || lanes&3 != 0 {
		return false
	}

	switch n {
	case 4:
		itxColDct4NEON(&tmp[0], lanes, stride, lo, hi)
	case 8:
		itxColDct8NEON(&tmp[0], lanes, stride, lo, hi)
	case 16:
		itxColDct16NEON(&tmp[0], lanes, stride, lo, hi)
	default:
		return false
	}

	return true
}
