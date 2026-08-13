//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package av1

func z2LeftRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n, ypos, dy int) int {
	return 0
}

// z2TopRun fills the columns that read the top edge, where the fraction is
// constant across the row, and reports how many it took.
func z2TopRun[P pixel](dst []P, dstOff int, edge []P, edgeOff, n int, frac int32) int {
	return 0
}
