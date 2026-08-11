//go:build amd64 && !noasm

package av1

//go:noescape
func msacSymbolAdapt4SSE(s *msacContext, cdf *uint16, n int) uint32

//go:noescape
func msacSymbolAdapt8SSE(s *msacContext, cdf *uint16, n int) uint32

//go:noescape
func msacSymbolAdapt16AVX2(s *msacContext, cdf *uint16, n int) uint32

func (s *msacContext) symbolAdapt(cdf []uint16, nSymbols int) uint32 {
	if nSymbols <= 3 && len(cdf) >= 4 {
		return msacSymbolAdapt4SSE(s, &cdf[0], nSymbols)
	}
	if nSymbols <= 7 && len(cdf) >= 8 {
		return msacSymbolAdapt8SSE(s, &cdf[0], nSymbols)
	}
	if nSymbols <= 15 && len(cdf) >= 16 && hasAVX2 {
		return msacSymbolAdapt16AVX2(s, &cdf[0], nSymbols)
	}

	return s.symbolAdaptGo(cdf, nSymbols)
}

//go:noescape
func msacBoolEquiSSE(s *msacContext) uint32

//go:noescape
func msacBoolFSSE(s *msacContext, f uint32) uint32

//go:noescape
func msacBoolAdaptSSE(s *msacContext, cdf *uint16) uint32

func (s *msacContext) boolEqui() uint32 { return msacBoolEquiSSE(s) }

func (s *msacContext) boolF(f uint32) uint32 { return msacBoolFSSE(s, f) }

func (s *msacContext) boolAdapt(cdf []uint16) uint32 {
	if len(cdf) < 2 {
		return s.boolAdaptGo(cdf)
	}

	return msacBoolAdaptSSE(s, &cdf[0])
}

func (s *msacContext) symbolAdapt4(cdf *[4]uint16, nSymbols int) uint32 {
	return msacSymbolAdapt4SSE(s, &cdf[0], nSymbols)
}
