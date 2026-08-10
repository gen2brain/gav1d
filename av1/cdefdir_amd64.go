//go:build amd64 && !noasm

package av1

//go:noescape
func cdefSums8AVX2(img *uint8, stride int, s *cdefSums)

//go:noescape
func cdefDirCostsAVX2(s *cdefSums, cost *[8]uint32)

func init() {
	if hasAVX2 {
		cdefDirCosts = cdefDirCostsAVX2
	}
}

//go:noescape
func cdefSums16AVX2(img *uint16, stride int, s *cdefSums, shift int)

func cdefFindDir8AVX2(img []uint8, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums8AVX2(&img[imgOff], stride, &s)

	return cdefDirCost(&s)
}

func cdefFindDir16AVX2(img []uint16, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums16AVX2(&img[imgOff], stride, &s, bitdepthFromMax(bitdepthMax)-8)

	return cdefDirCost(&s)
}
