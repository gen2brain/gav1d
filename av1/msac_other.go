//go:build !amd64 || noasm

package av1

func (s *msacContext) symbolAdapt(cdf []uint16, nSymbols int) uint32 {
	return s.symbolAdaptGo(cdf, nSymbols)
}

func (s *msacContext) boolEqui() uint32 { return s.boolEquiGo() }

func (s *msacContext) boolF(f uint32) uint32 { return s.boolFGo(f) }

func (s *msacContext) boolAdapt(cdf []uint16) uint32 { return s.boolAdaptGo(cdf) }
