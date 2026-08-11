package av1

var scans = [nRectTxSizes][]uint16{
	tx4x4:    scan4x4[:],
	tx8x8:    scan8x8[:],
	tx16x16:  scan16x16[:],
	tx32x32:  scan32x32[:],
	tx64x64:  scan32x32[:],
	rtx4x8:   scan4x8[:],
	rtx8x4:   scan8x4[:],
	rtx8x16:  scan8x16[:],
	rtx16x8:  scan16x8[:],
	rtx16x32: scan16x32[:],
	rtx32x16: scan32x16[:],
	rtx32x64: scan32x32[:],
	rtx64x32: scan32x32[:],
	rtx4x16:  scan4x16[:],
	rtx16x4:  scan16x4[:],
	rtx8x32:  scan8x32[:],
	rtx32x8:  scan32x8[:],
	rtx16x64: scan16x32[:],
	rtx64x16: scan32x16[:],
}

var lastNonzeroColFromEob [nRectTxSizes][]uint8

var loCtxOffFromScan [nRectTxSizes][]uint8

func initLoCtxOffFromScan(tx int) []uint8 {
	scan := scans[tx]
	if scan == nil {
		return nil
	}

	slh := min(int(txfmDimensions[tx].lh), tx32x32)
	shift, mask := slh+2, uint32(4<<slh)-1
	nonsquare := b2i(tx >= rtx4x8)
	offsets := &loCtxOffsets[nonsquare+(tx&nonsquare)]

	tbl := make([]uint8, len(scan))
	for i, rc := range scan {
		x, y := uint32(rc)>>shift, uint32(rc)&mask
		tbl[i] = offsets[min(y, 4)][min(x, 4)]
	}

	return tbl
}

func initLastNonzeroColFromEob(scan []uint16, w, h int) []uint8 {
	tbl := make([]uint8, w*h)
	maxCol := 0
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rc := int(scan[n])
			rcx := rc & (h - 1)
			maxCol = max(maxCol, rcx)
			tbl[n] = uint8(maxCol)
			n++
		}
	}

	return tbl
}

func init() {
	t4x4 := initLastNonzeroColFromEob(scan4x4[:], 4, 4)
	t8x8 := initLastNonzeroColFromEob(scan8x8[:], 8, 8)
	t16x16 := initLastNonzeroColFromEob(scan16x16[:], 16, 16)
	t32x32 := initLastNonzeroColFromEob(scan32x32[:], 32, 32)
	t4x8 := initLastNonzeroColFromEob(scan4x8[:], 4, 8)
	t8x4 := initLastNonzeroColFromEob(scan8x4[:], 8, 4)
	t8x16 := initLastNonzeroColFromEob(scan8x16[:], 8, 16)
	t16x8 := initLastNonzeroColFromEob(scan16x8[:], 16, 8)
	t16x32 := initLastNonzeroColFromEob(scan16x32[:], 16, 32)
	t32x16 := initLastNonzeroColFromEob(scan32x16[:], 32, 16)
	t4x16 := initLastNonzeroColFromEob(scan4x16[:], 4, 16)
	t16x4 := initLastNonzeroColFromEob(scan16x4[:], 16, 4)
	t8x32 := initLastNonzeroColFromEob(scan8x32[:], 8, 32)
	t32x8 := initLastNonzeroColFromEob(scan32x8[:], 32, 8)

	lastNonzeroColFromEob = [nRectTxSizes][]uint8{
		tx4x4:    t4x4,
		tx8x8:    t8x8,
		tx16x16:  t16x16,
		tx32x32:  t32x32,
		tx64x64:  t32x32,
		rtx4x8:   t4x8,
		rtx8x4:   t8x4,
		rtx8x16:  t8x16,
		rtx16x8:  t16x8,
		rtx16x32: t16x32,
		rtx32x16: t32x16,
		rtx32x64: t32x32,
		rtx64x32: t32x32,
		rtx4x16:  t4x16,
		rtx16x4:  t16x4,
		rtx8x32:  t8x32,
		rtx32x8:  t32x8,
		rtx16x64: t16x32,
		rtx64x16: t32x16,
	}

	for tx := range loCtxOffFromScan {
		loCtxOffFromScan[tx] = initLoCtxOffFromScan(tx)
	}
}
