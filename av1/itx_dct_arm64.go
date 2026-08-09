//go:build arm64 && !noasm

package av1

//go:noescape
func itxColDct4NEON(tmp *int32, w int, lo, hi int32)

//go:noescape
func itxColDct8NEON(tmp *int32, w int, lo, hi int32)

//go:noescape
func itxColDct16NEON(tmp *int32, w int, lo, hi int32)

// itxColVec runs the column pass with one column per lane, which needs no
// transpose because consecutive columns are contiguous in the scratch.
func itxColVec(tmp []int32, w, h, txtp int, lo, hi int32) bool {
	if txtp != tx1dDct || w&3 != 0 {
		return false
	}

	switch h {
	case 4:
		itxColDct4NEON(&tmp[0], w, lo, hi)
	case 8:
		itxColDct8NEON(&tmp[0], w, lo, hi)
	case 16:
		itxColDct16NEON(&tmp[0], w, lo, hi)
	default:
		return false
	}

	return true
}
