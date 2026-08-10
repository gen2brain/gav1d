//go:build arm64 && !noasm

package av1

var filterIntraTapsW = func() [5][56]int16 {
	var t [5][56]int16
	for f := range filterIntraTaps {
		for i := range 56 {
			t[f][i] = int16(filterIntraTaps[f][i])
		}
	}

	return t
}()

//go:noescape
func smoothV8NEON(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)

//go:noescape
func smoothH8NEON(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)

//go:noescape
func smooth8NEON(dst *uint8, stride int, tl *uint8, vw, hw *uint8, w, h int)

//go:noescape
func paeth8NEON(dst *uint8, stride int, tl *uint8, w, h int)

//go:noescape
func splatDc8NEON(dst *uint8, stride, w, h int, dc uint8)

//go:noescape
func vpred8NEON(dst *uint8, stride int, tl *uint8, w, h int)

//go:noescape
func hpred8NEON(dst *uint8, stride int, tl *uint8, w, h int)

func ipredSmoothV8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothV8NEON(&dst[dstOff], stride, &tl[tlOff], &smWeights[height], width, height)
}

func ipredSmoothH8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smoothH8NEON(&dst[dstOff], stride, &tl[tlOff], &smWeights[width], width, height)
}

func ipredSmooth8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	smooth8NEON(&dst[dstOff], stride, &tl[tlOff], &smWeights[height],
		&smWeights[width], width, height)
}

func ipredPaeth8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	paeth8NEON(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredV8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	vpred8NEON(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredH8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	hpred8NEON(&dst[dstOff], stride, &tl[tlOff], width, height)
}

func ipredDc8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8NEON(&dst[dstOff], stride, width, height,
		uint8(dcGen(tl, tlOff, width, height, bitdepthMax)))
}

func ipredDcTop8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8NEON(&dst[dstOff], stride, width, height, uint8(dcGenTop(tl, tlOff, width)))
}

func ipredDcLeft8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8NEON(&dst[dstOff], stride, width, height, uint8(dcGenLeft(tl, tlOff, height)))
}

func ipredDc1288NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc8NEON(&dst[dstOff], stride, width, height, uint8((bitdepthMax+1)>>1))
}

//go:noescape
func filterIntra8NEON(dst *uint8, stride int, tl *uint8, f *int16, w, h int)

func ipredFilter8NEON(dst []uint8, dstOff, stride int, tl []uint8, tlOff,
	width, height, filtIdx, maxWidth, maxHeight int, bitdepthMax int32,
) {
	filterIntra8NEON(&dst[dstOff], stride, &tl[tlOff],
		&filterIntraTapsW[filtIdx&511][0], width, height)
}

//go:noescape
func z1Full8NEON(dst *uint8, stride int, top *uint8, w, rows, dx, xpos int)

func ipredZ18NEON(dst []uint8, dstOff, stride int, tlIn []uint8, tlInOff,
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
		z1Full8NEON(&dst[dstOff], stride, &top[topOff], width, y1, dx, dx)
	}
	if y2 > y1 {
		z1Sample(dst, dstOff+y1*stride, stride, top, topOff, width, y2-y1,
			dx, 1, maxBaseX, (y1+1)*dx)
	}
	if y2 < height {
		splatDc8NEON(&dst[dstOff+y2*stride], stride, width, height-y2,
			top[topOff+maxBaseX])
	}
}
