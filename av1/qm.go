package av1

func qmUntriangled(src []uint8, sz int) []uint8 {
	dst := make([]uint8, sz*sz)

	dstOff, srcOff := 0, 0
	for y := range sz {
		for x := range y + 1 {
			dst[dstOff+x] = src[srcOff+x]
		}
		srcPtrOff := y
		for x := y + 1; x < sz; x++ {
			srcPtrOff += x
			dst[dstOff+x] = src[srcOff+srcPtrOff]
		}
		dstOff += sz
		srcOff += y + 1
	}

	return dst
}

func qmSubsampled(src []uint8, h, hstep, vstep int) []uint8 {
	dst := make([]uint8, h*32/(hstep*vstep))

	i := 0
	srcOff := (vstep-1)/2*32 + (hstep-1)/2
	for y := 0; y < h; y += vstep {
		for x := 0; x < 32; x += hstep {
			dst[i] = src[srcOff+y*32+x]
			i++
		}
	}

	return dst
}

func qmTransposed(src []uint8, w, h int) []uint8 {
	dst := make([]uint8, w*h)

	for y := range h {
		for x := range w {
			dst[x*h+y] = src[y*w+x]
		}
	}

	return dst
}

var qmTbl [16][2][nRectTxSizes][]uint8

func init() {
	for i := range 15 {
		for j := range 2 {
			t32x32 := qmUntriangled(qmTbl32x32T[i][j][:], 32)
			t32x16 := qmTbl32x16[i][j][:]

			t4x4 := qmSubsampled(t32x32, 32, 8, 8)
			t8x8 := qmSubsampled(t32x32, 32, 4, 4)
			t16x16 := qmSubsampled(t32x32, 32, 2, 2)
			t8x4 := qmSubsampled(t32x16, 16, 4, 4)
			t16x4 := qmSubsampled(t32x16, 16, 2, 4)
			t16x8 := qmSubsampled(t32x16, 16, 2, 2)
			t32x8 := qmSubsampled(t32x16, 16, 1, 2)

			t4x8 := qmTransposed(t8x4, 8, 4)
			t4x16 := qmTransposed(t16x4, 16, 4)
			t8x16 := qmTransposed(t16x8, 16, 8)
			t8x32 := qmTransposed(t32x8, 32, 8)
			t16x32 := qmTransposed(t32x16, 32, 16)

			row := &qmTbl[i][j]

			row[rtx4x8] = t8x4
			row[rtx8x4] = t4x8
			row[rtx4x16] = t16x4
			row[rtx16x4] = t4x16
			row[rtx8x16] = t16x8
			row[rtx16x8] = t8x16
			row[rtx8x32] = t32x8
			row[rtx32x8] = t8x32
			row[rtx16x32] = t32x16
			row[rtx32x16] = t16x32

			row[tx4x4] = t4x4
			row[tx8x8] = t8x8
			row[tx16x16] = t16x16
			row[tx32x32] = t32x32

			row[tx64x64] = row[tx32x32]
			row[rtx64x32] = row[tx32x32]
			row[rtx64x16] = row[rtx32x16]
			row[rtx32x64] = row[tx32x32]
			row[rtx16x64] = row[rtx16x32]
		}
	}
}
