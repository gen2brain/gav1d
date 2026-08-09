//go:build amd64 && !noasm

package av1

//go:noescape
func itxColDct4AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct8AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct16AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxColDct32AVX2(tmp *int32, lanes, stride int, lo, hi int32)

//go:noescape
func itxIdentityAVX2(tmp *int32, count, stride, n int, p *[4]int32)

//go:noescape
func itxColDct64AVX2(tmp *int32, lanes, stride int, lo, hi int32, scratch *int32)

//go:noescape
func itxColAdst4AVX2(tmp *int32, lanes, stride int, lo, hi int32, flip int)

//go:noescape
func itxColAdst8AVX2(tmp *int32, lanes, stride int, lo, hi int32, flip int)

//go:noescape
func itxColAdst16AVX2(tmp *int32, lanes, stride int, lo, hi int32, flip int)

func itxIdentityVec(tmp []int32, lanes, stride, n int) bool {
	if n > 32 {
		return false
	}

	p := &itxIdentityParams[ulog2(uint32(n))-2]
	count, taps := lanes, n
	if lanes == stride {
		count, taps = lanes*n, 1
	}
	if count&7 != 0 {
		return false
	}
	itxIdentityAVX2(&tmp[0], count, stride, taps, p)

	return true
}

func itxColVec(tmp []int32, lanes, stride, n, txtp int, lo, hi int32, scratch []int32) bool {
	if txtp == tx1dIdentity {
		return itxIdentityVec(tmp, lanes, stride, n)
	}
	if lanes&7 != 0 {
		return false
	}
	if txtp == tx1dAdst || txtp == tx1dFlipadst {
		flip := b2i(txtp == tx1dFlipadst)
		switch n {
		case 4:
			itxColAdst4AVX2(&tmp[0], lanes, stride, lo, hi, flip)
		case 8:
			itxColAdst8AVX2(&tmp[0], lanes, stride, lo, hi, flip)
		case 16:
			itxColAdst16AVX2(&tmp[0], lanes, stride, lo, hi, flip)
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
		itxColDct4AVX2(&tmp[0], lanes, stride, lo, hi)
	case 8:
		itxColDct8AVX2(&tmp[0], lanes, stride, lo, hi)
	case 16:
		itxColDct16AVX2(&tmp[0], lanes, stride, lo, hi)
	case 32:
		itxColDct16AVX2(&tmp[0], lanes, stride*2, lo, hi)
		itxColDct32AVX2(&tmp[0], lanes, stride, lo, hi)
	case 64:
		itxColDct16AVX2(&tmp[0], lanes, stride*4, lo, hi)
		itxColDct32AVX2(&tmp[0], lanes, stride*2, lo, hi)
		itxColDct64AVX2(&tmp[0], lanes, stride, lo, hi, &scratch[0])
	default:
		return false
	}

	return true
}
