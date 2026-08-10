//go:build amd64 && !noasm

package av1

var smWeightsPw = func() [128][2]int16 {
	var t [128][2]int16
	for i, w := range smWeights {
		t[i][0] = int16(w)
		t[i][1] = int16(256 - int(w))
	}

	return t
}()

//go:noescape
func smoothV16AVX2(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)

//go:noescape
func smoothH16AVX2(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)

//go:noescape
func smooth16AVX2(dst *uint16, stride int, tl *uint16, vw, hw *int16, w, h int)

//go:noescape
func paeth16AVX2(dst *uint16, stride int, tl *uint16, w, h int)

//go:noescape
func splatDc16AVX2(dst *uint16, stride, w, h int, dc uint16)

//go:noescape
func vpred16AVX2(dst *uint16, stride int, tl *uint16, w, h int)

//go:noescape
func hpred16AVX2(dst *uint16, stride int, tl *uint16, w, h int)

func ipredSmoothV16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothV16AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPw[height][0], width, height)
}

func ipredSmoothH16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothH16AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPw[width][0], width, height)
}

func ipredSmooth16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smooth16AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPw[height][0],
		&smWeightsPw[width][0], width, height)
}

func ipredPaeth16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	paeth16AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredV16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	vpred16AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredH16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	hpred16AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredDc16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc16AVX2(&dst[dstOff], stride, width, height,
		uint16(dcGen(tl, tlOff, width, height, bitdepthMax)))
}

func ipredDcTop16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc16AVX2(&dst[dstOff], stride, width, height, uint16(dcGenTop(tl, tlOff, width)))
}

func ipredDcLeft16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc16AVX2(&dst[dstOff], stride, width, height, uint16(dcGenLeft(tl, tlOff, height)))
}

func ipredDc12816AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc16AVX2(&dst[dstOff], stride, width, height, uint16((bitdepthMax+1)>>1))
}

var filterIntraTapsPd = func() [5][64]int16 {
	var t [5][64]int16
	for f := range filterIntraTaps {
		for j := range 8 {
			for k := range 4 {
				t[f][16*k+2*j] = int16(filterIntraTaps[f][(2*k)*8+j])
				if 2*k+1 < 7 {
					t[f][16*k+2*j+1] = int16(filterIntraTaps[f][(2*k+1)*8+j])
				}
			}
		}
	}

	return t
}()

//go:noescape
func filterIntra16AVX2(dst *uint16, stride int, tl *uint16, f *int16, w, h int, bitdepthMax int32)

//go:noescape
func z1Full16AVX2(dst *uint16, stride int, top *uint16, w, rows, dx, xpos int)

func ipredFilter16AVX2(dst []uint16, dstOff, stride int, tl []uint16, tlOff,
	width, height, filtIdx, maxWidth, maxHeight int, bitdepthMax int32,
) {
	filterIntra16AVX2(&dst[dstOff], stride, &tl[tlOff],
		&filterIntraTapsPd[filtIdx&511][0], width, height, bitdepthMax)
}

func ipredZ116AVX2(dst []uint16, dstOff, stride int, tlIn []uint16, tlInOff,
	width, height, angle, maxWidth, maxHeight int, bitdepthMax int32,
) {
	var topOut [64 + 64]uint16
	top, topOff, maxBaseX, dx, baseInc := z1Edge(tlIn, tlInOff, width, height,
		angle, bitdepthMax, topOut[:])

	if baseInc != 1 || dx <= 0 {
		z1Sample(dst, dstOff, stride, top, topOff, width, height, dx, baseInc, maxBaseX, dx)

		return
	}

	y1 := z1Rows(maxBaseX-width, dx, height)
	y2 := z1Rows(maxBaseX-1, dx, height)

	if y1 > 0 {
		z1Full16AVX2(&dst[dstOff], stride, &top[topOff], width, y1, dx, dx)
	}
	if y2 > y1 {
		z1Sample(dst, dstOff+y1*stride, stride, top, topOff, width, y2-y1,
			dx, 1, maxBaseX, (y1+1)*dx)
	}
	if y2 < height {
		splatDc16AVX2(&dst[dstOff+y2*stride], stride, width, height-y2,
			top[topOff+maxBaseX])
	}
}

const ipredHas16Deep = true
