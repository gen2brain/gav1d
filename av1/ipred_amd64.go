//go:build amd64 && !noasm

package av1

var smWeightsPm = func() [128][2]int8 {
	var t [128][2]int8
	for i, w := range smWeights {
		t[i][0] = int8(int(w) - 128)
		t[i][1] = int8(127 - int(w))
	}

	return t
}()

var filterIntraTapsPm = func() [5][64]int8 {
	var t [5][64]int8
	for f := range filterIntraTaps {
		for j := range 8 {
			for k := range 4 {
				t[f][16*k+2*j] = filterIntraTaps[f][(2*k)*8+j]
				if 2*k+1 < 7 {
					t[f][16*k+2*j+1] = filterIntraTaps[f][(2*k+1)*8+j]
				}
			}
		}
	}

	return t
}()

//go:noescape
func smoothV8AVX2(dst *uint8, stride int, tl *uint8, weights *int8, w, h int)

//go:noescape
func smoothH8AVX2(dst *uint8, stride int, tl *uint8, weights *int8, w, h int)

//go:noescape
func smooth8AVX2(dst *uint8, stride int, tl *uint8, vw, hw *int8, w, h int)

func ipredSmoothV8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothV8AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPm[height][0], width, height)
}

func ipredSmoothH8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothH8AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPm[width][0], width, height)
}

func ipredSmooth8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smooth8AVX2(&dst[dstOff], stride, &tl[tlOff], &smWeightsPm[height][0],
		&smWeightsPm[width][0], width, height)
}

//go:noescape
func paeth8AVX2(dst *uint8, stride int, tl *uint8, w, h int)

//go:noescape
func splatDc8AVX2(dst *uint8, stride, w, h int, dc uint8)

//go:noescape
func vpred8AVX2(dst *uint8, stride int, tl *uint8, w, h int)

//go:noescape
func hpred8AVX2(dst *uint8, stride int, tl *uint8, w, h int)

func ipredPaeth8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	paeth8AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredV8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	vpred8AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredH8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	hpred8AVX2(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredDc8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8AVX2(&dst[dstOff], stride, width, height,
		uint8(dcGen(tl, tlOff, width, height, bitdepthMax)))
}

func ipredDcTop8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8AVX2(&dst[dstOff], stride, width, height, uint8(dcGenTop(tl, tlOff, width)))
}

func ipredDcLeft8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8AVX2(&dst[dstOff], stride, width, height, uint8(dcGenLeft(tl, tlOff, height)))
}

func ipredDc1288AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8AVX2(&dst[dstOff], stride, width, height, uint8((bitdepthMax+1)>>1))
}

//go:noescape
func filterIntra8AVX2(dst *uint8, stride int, tl *uint8, f *int8, w, h int)

func ipredFilter8AVX2(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, filtIdx, maxWidth, maxHeight int, bitdepthMax int32,
) {
	filterIntra8AVX2(&dst[dstOff], stride, &tl[tlOff],
		&filterIntraTapsPm[filtIdx&511][0], width, height)
}

//go:noescape
func z1Full8AVX2(dst *uint8, stride int, top *uint8, w, rows, dx, xpos int)

func ipredZ18AVX2(dst []uint8, dstOff, stride int, tlIn []uint8, tlInOff,
	width, height, angle, maxWidth, maxHeight int, bitdepthMax int32,
) {
	var topOut [64 + 64]uint8
	top, topOff, maxBaseX, dx, baseInc := z1Edge(tlIn, tlInOff, width, height,
		angle, bitdepthMax, topOut[:])

	if baseInc != 1 || dx <= 0 {
		z1Sample(dst, dstOff, stride, top, topOff, width, height, dx, baseInc, maxBaseX, dx)

		return
	}

	y1 := z1Rows(maxBaseX-width, dx, height)
	y2 := z1Rows(maxBaseX-1, dx, height)

	if y1 > 0 {
		z1Full8AVX2(&dst[dstOff], stride, &top[topOff], width, y1, dx, dx)
	}
	if y2 > y1 {
		z1Sample(dst, dstOff+y1*stride, stride, top, topOff, width, y2-y1,
			dx, 1, maxBaseX, (y1+1)*dx)
	}
	if y2 < height {
		splatDc8AVX2(&dst[dstOff+y2*stride], stride, width, height-y2,
			top[topOff+maxBaseX])
	}
}
