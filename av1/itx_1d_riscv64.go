//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func itxColDct4RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct8RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct16RVV(tmp *int32, lanes, stride int, lo, hi int32)

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32) bool {
	if txtp != tx1dDct {
		return false
	}

	switch n {
	case 4:
		itxColDct4RVV(&tmp[0], lanes, stride, lo, hi)
	case 8:
		itxColDct8RVV(&tmp[0], lanes, stride, lo, hi)
	case 16:
		itxColDct16RVV(&tmp[0], lanes, stride, lo, hi)
	default:
		return false
	}

	return true
}
