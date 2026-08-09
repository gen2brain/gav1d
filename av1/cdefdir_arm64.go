//go:build arm64 && !noasm

package av1

//go:noescape
func cdefSums8NEON(img *uint8, stride int, s *cdefSums)

//go:noescape
func cdefSums16NEON(img *uint16, stride int, s *cdefSums, shift int)

func cdefFindDir8NEON(img []uint8, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums8NEON(&img[imgOff], stride, &s)

	return cdefDirCost(&s)
}

func cdefFindDir16NEON(img []uint16, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums16NEON(&img[imgOff], stride, &s, bitdepthFromMax(bitdepthMax)-8)

	return cdefDirCost(&s)
}
