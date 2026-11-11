//go:build (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64)) || noasm

package av1

func cdefDirCosts(s *cdefSums, cost *[8]uint32) {
	cdefDirCostsGo(s, cost)
}
