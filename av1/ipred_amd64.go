//go:build amd64 && !noasm

package av1

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
