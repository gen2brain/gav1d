//go:build (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64)) || noasm

package av1

func (s *msacContext) symbolAdapt(cdf []uint16, nSymbols int) uint32 {
	return s.symbolAdaptGo(cdf, nSymbols)
}

func (s *msacContext) boolEqui() uint32 { return s.boolEquiGo() }

func (s *msacContext) boolF(f uint32) uint32 { return s.boolFGo(f) }

func (s *msacContext) boolAdapt(cdf []uint16) uint32 { return s.boolAdaptGo(cdf) }

func (s *msacContext) symbolAdapt4(cdf *[4]uint16, nSymbols int) uint32 {
	return s.symbolAdaptGo(cdf[:], nSymbols)
}
