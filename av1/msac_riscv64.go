//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

//go:noescape
func msacBoolEquiRVV(s *msacContext) uint32

//go:noescape
func msacBoolFRVV(s *msacContext, f uint32) uint32

//go:noescape
func msacBoolAdaptRVV(s *msacContext, cdf *uint16) uint32

//go:noescape
func msacSymbolAdaptRVV(s *msacContext, cdf *uint16, n int) uint32

func (s *msacContext) boolEqui() uint32 { return msacBoolEquiRVV(s) }

func (s *msacContext) boolF(f uint32) uint32 { return msacBoolFRVV(s, f) }

func (s *msacContext) boolAdapt(cdf []uint16) uint32 {
	if len(cdf) < 2 {
		return s.boolAdaptGo(cdf)
	}

	return msacBoolAdaptRVV(s, &cdf[0])
}

func (s *msacContext) symbolAdapt(cdf []uint16, nSymbols int) uint32 {
	if nSymbols <= 15 && len(cdf) > nSymbols {
		return msacSymbolAdaptRVV(s, &cdf[0], nSymbols)
	}

	return s.symbolAdaptGo(cdf, nSymbols)
}

func (s *msacContext) symbolAdapt4(cdf *[4]uint16, nSymbols int) uint32 {
	return msacSymbolAdaptRVV(s, &cdf[0], nSymbols)
}
