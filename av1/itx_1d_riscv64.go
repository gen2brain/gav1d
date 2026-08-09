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

//go:noescape
func itxColDct64RVV(tmp *int32, lanes, stride int, lo, hi int32, scratch *int32)

//go:noescape
func itxColAdst4RVV(tmp *int32, lanes, stride int, lo, hi int32, flip int)

//go:noescape
func itxColAdst8RVV(tmp *int32, lanes, stride int, lo, hi int32, flip int)

//go:noescape
func itxColAdst16RVV(tmp *int32, lanes, stride int, lo, hi int32, flip int)

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

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32, scratch []int32) bool {
	if txtp == tx1dIdentity {
		return itxIdentityVec(tmp, lanes, stride, n)
	}
	if txtp == tx1dAdst || txtp == tx1dFlipadst {
		flip := b2i(txtp == tx1dFlipadst)
		switch n {
		case 4:
			itxColAdst4RVV(&tmp[0], lanes, stride, lo, hi, flip)
		case 8:
			itxColAdst8RVV(&tmp[0], lanes, stride, lo, hi, flip)
		case 16:
			itxColAdst16RVV(&tmp[0], lanes, stride, lo, hi, flip)
		default:
			return false
		}

		return true
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
	case 64:
		itxColDct16RVV(&tmp[0], lanes, stride*4, lo, hi)
		itxColDct32RVV(&tmp[0], lanes, stride*2, lo, hi)
		itxColDct64RVV(&tmp[0], lanes, stride, lo, hi, &scratch[0])
	default:
		return false
	}

	return true
}
