//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func itxClipRVV(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8RVV(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)

//go:noescape
func itxAdd16RVV(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) > 0 {
		itxClipRVV(&tmp[0], len(tmp), rnd, shift, lo, hi)
	}
}

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	itxAdd8RVV(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	itxAdd16RVV(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}
