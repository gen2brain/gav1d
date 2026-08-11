//go:build arm64 && !noasm

package av1

//go:noescape
func itxClipNEON(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8NEON(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

//go:noescape
func itxAdd16NEON(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) >= 4 {
		itxClipNEON(&tmp[0], len(tmp)&^3, rnd, shift, lo, hi)
	}
	itxClipGo(tmp[len(tmp)&^3:], rnd, shift, lo, hi)
}

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	itxAdd8NEON(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	itxAdd16NEON(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

//go:noescape
func itxTransposeNEON(wide, out *int32, w, ws, ts, n int)

func itxTransposeVec(wide, tmp []int32, w, ws, ts, n int) {
	if w&3 != 0 || n&3 != 0 {
		itxTransposeGo(wide, tmp, w, ws, ts, n)

		return
	}

	itxTransposeNEON(&wide[0], &tmp[0], w, ws, ts, n)
}

//go:noescape
func widenCoefs16NEON(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func widenCoefs16Rect2NEON(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func itxDC8NEON(dst *uint8, stride int, dc int32, w, h int, bitdepthMax int32)

//go:noescape
func itxDC16NEON(dst *uint16, stride int, dc int32, w, h int, bitdepthMax int32)

func itxDC8Vec(dst []uint8, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC8NEON(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}

func itxDC16Vec(dst []uint16, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC16NEON(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}
