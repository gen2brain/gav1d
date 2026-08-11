//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func cdefDirCostsRVV(s *cdefSums, cost *[8]uint32)
