//go:build arm64 && !noasm

package av1

//go:noescape
func cdefDirCostsNEON(s *cdefSums, cost *[8]uint32)

func cdefDirCosts(s *cdefSums, cost *[8]uint32) {
	cdefDirCostsNEON(s, cost)
}
