//go:build arm64 && !noasm

package av1

//go:noescape
func itxClipNEON(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8NEON(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)

//go:noescape
func itxAdd16NEON(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) >= 4 {
		itxClipNEON(&tmp[0], len(tmp)&^3, rnd, shift, lo, hi)
	}
	itxClipGo(tmp[len(tmp)&^3:], rnd, shift, lo, hi)
}

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	itxAdd8NEON(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h int, bitdepthMax int32) {
	itxAdd16NEON(&dst[dstOff], stride, &tmp[0], w, h, bitdepthMax)
}

//go:noescape
func itxTransposeNEON(wide, out *int32, w, sh, n int)

func itxTransposeVec(wide, tmp []int32, w, sh, n int) {
	if w&3 != 0 || n&3 != 0 {
		itxTransposeGo(wide, tmp, w, sh, n)

		return
	}

	itxTransposeNEON(&wide[0], &tmp[0], w, sh, n)
}
