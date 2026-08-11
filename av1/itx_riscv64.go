//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func itxClipRVV(tmp *int32, n int, rnd int32, shift int, lo, hi int32)

//go:noescape
func itxAdd8RVV(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

//go:noescape
func itxAdd16RVV(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)

func itxClipVec(tmp []int32, rnd int32, shift int, lo, hi int32) {
	if len(tmp) > 0 {
		itxClipRVV(&tmp[0], len(tmp), rnd, shift, lo, hi)
	}
}

func itxAdd8Vec(dst []uint8, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	itxAdd8RVV(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

func itxAdd16Vec(dst []uint16, dstOff, stride int, tmp []int32, w, h, ts int, bitdepthMax int32) {
	itxAdd16RVV(&dst[dstOff], stride, &tmp[0], w, h, ts, bitdepthMax)
}

//go:noescape
func itxTransposeRVV(wide, out *int32, w, ws, ts, n int)

func itxTransposeVec(wide, tmp []int32, w, ws, ts, n int) {
	itxTransposeRVV(&wide[0], &tmp[0], w, ws, ts, n)
}

//go:noescape
func widenCoefs16RVV(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func widenCoefs16Rect2RVV(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)

//go:noescape
func itxDC8RVV(dst *uint8, stride int, dc int32, w, h int, bitdepthMax int32)

//go:noescape
func itxDC16RVV(dst *uint16, stride int, dc int32, w, h int, bitdepthMax int32)

func itxDC8Vec(dst []uint8, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC8RVV(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}

func itxDC16Vec(dst []uint16, dstOff, stride int, dc int32, w, h int, bitdepthMax int32) {
	itxDC16RVV(&dst[dstOff], stride, dc, w, h, bitdepthMax)
}
