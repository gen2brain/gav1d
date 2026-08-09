//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func itxColDct4RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct8RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct16RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct32RVV(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxIdentityRVV(tmp *int32, count, stride, n int, p *[4]int32)

func itxIdentityVec(tmp []int32, lanes, stride, n int) bool {
	if n > 32 {
		return false
	}

	p := &itxIdentityParams[ulog2(uint32(n))-2]
	count, taps := lanes, n
	if lanes == stride {
		count, taps = lanes*n, 1
	}
	itxIdentityRVV(&tmp[0], count, stride, taps, p)

	return true
}

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32) bool {
	if txtp == tx1dIdentity {
		return itxIdentityVec(tmp, lanes, stride, n)
	}
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
	case 32:
		itxColDct16RVV(&tmp[0], lanes, stride*2, lo, hi)
		itxColDct32RVV(&tmp[0], lanes, stride, lo, hi)
	default:
		return false
	}

	return true
}
