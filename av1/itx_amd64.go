//go:build amd64 && !noasm

package av1

//go:noescape
func itxClipAVX2(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8AVX2(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)

//go:noescape
func itxAdd16AVX2(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) >= 8 {
		itxClipAVX2(&tmp[0], len(tmp)&^7, rnd, shift, lo, hi)
	}
	itxClipGo(tmp[len(tmp)&^7:], rnd, shift, lo, hi)
}

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	if w < 8 {
		itxAddGo(dst, dstOff, stride, tmp, w, h, bitdepthMax)

		return
	}

	itxAdd8AVX2(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	if w < 8 {
		itxAddGo(dst, dstOff, stride, tmp, w, h, bitdepthMax)

		return
	}

	itxAdd16AVX2(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}
