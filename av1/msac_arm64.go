//go:build arm64 && !noasm

package av1

//go:noescape
func msacBoolEquiNEON(s *msacContext) uint32

//go:noescape
func msacBoolFNEON(s *msacContext, f uint32) uint32

//go:noescape
func msacBoolAdaptNEON(s *msacContext, cdf *uint16) uint32

func (s *msacContext) boolEqui() uint32 { return msacBoolEquiNEON(s) }

func (s *msacContext) boolF(f uint32) uint32 { return msacBoolFNEON(s, f) }

func (s *msacContext) boolAdapt(cdf []uint16) uint32 {
	if len(cdf) < 2 {
		return s.boolAdaptGo(cdf)
	}

	return msacBoolAdaptNEON(s, &cdf[0])
}

func (s *msacContext) symbolAdapt(cdf []uint16, nSymbols int) uint32 {
	return s.symbolAdaptGo(cdf, nSymbols)
}

func (s *msacContext) symbolAdapt4(cdf *[4]uint16, nSymbols int) uint32 {
	return s.symbolAdaptGo(cdf[:], nSymbols)
}
