//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

// cdefGather holds the four permutations the direction search needs: reverse a
// row, take its even and odd samples, and reverse the four pairwise sums.
var cdefGather = [32]int32{
	7, 6, 5, 4, 3, 2, 1, 0,
	0, 2, 4, 6, 0, 2, 4, 6,
	1, 3, 5, 7, 1, 3, 5, 7,
	3, 2, 1, 0, 3, 2, 1, 0,
}

//go:noescape
func cdefSums8RVV(img *uint8, stride int, s *cdefSums, idx *[32]int32)

//go:noescape
func cdefSums16RVV(img *uint16, stride int, s *cdefSums, idx *[32]int32, shift int)

func cdefFindDir8RVV(img []uint8, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums8RVV(&img[imgOff], stride, &s, &cdefGather)

	return cdefDirCost(&s)
}

func cdefFindDir16RVV(img []uint16, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	var s cdefSums
	cdefSums16RVV(&img[imgOff], stride, &s, &cdefGather, bitdepthFromMax(bitdepthMax)-8)

	return cdefDirCost(&s)
}
