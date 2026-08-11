//go:build amd64 && !noasm

package av1

//go:noescape
func itxClipAVX2(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8AVX2(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

//go:noescape
func itxAdd16AVX2(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) >= 8 {
		itxClipAVX2(&tmp[0], len(tmp)&^7, rnd, shift, lo, hi)
	}
	itxClipGo(tmp[len(tmp)&^7:], rnd, shift, lo, hi)
}

//go:noescape
func itxAdd8AVX512(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	if hasAVX512 && w >= 16 {
		itxAdd8AVX512(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)

		return
	}
	itxAdd8AVX2(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	itxAdd16AVX2(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

//go:noescape
func itxTransposeAVX2(wide, out *int32, w, ws, ts, n int)

func itxTransposeVec(wide, tmp []int32, w, ws, ts, n int) {
	if w&7 != 0 || n&7 != 0 {
		itxTransposeGo(wide, tmp, w, ws, ts, n)

		return
	}

	itxTransposeAVX2(&wide[0], &tmp[0], w, ws, ts, n)
}

//go:noescape
func widenCoefs16AVX2(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func widenCoefs16Rect2AVX2(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func widenCoefs16AVX512(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func widenCoefs16Rect2AVX512(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func itxDC8AVX2(dst *uint8, stride int, dc int32, w, h int, bitdepthMax int32)

func itxDC8Vec(dst []uint8, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC8AVX2(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}

//go:noescape
func itxDC16AVX2(dst *uint16, stride int, dc int32, w, h int, bitdepthMax int32)

func itxDC16Vec(dst []uint16, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC16AVX2(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}
