//go:build amd64 && !noasm

package av1

type cdefZParams struct {
	pri, sec       int32
	priShr, secShr int32
	priTapOff      int32
	dirPri         int32
	dirSec0        int32
	dirSec1        int32
	edge           int32
}

//go:noescape
func cdef4x4AVX512(dst *uint8, stride int, left *uint8, top *uint8, bot *uint8, p *cdefZParams)

func cdefZParamsFor(p *cdefZParams, priStrength, secStrength, dir, damping, edges, bitdepthMin8 int) {
	*p = cdefZParams{
		pri:     int32(priStrength),
		sec:     int32(secStrength),
		dirPri:  int32(dir+2) * 4,
		dirSec0: int32(dir+4) * 4,
		dirSec1: int32(dir+0) * 4,
		edge:    int32(edges),
	}
	if priStrength != 0 {
		p.priShr = int32(max(0, damping-ulog2(uint32(priStrength))))
		p.priTapOff = int32(priStrength>>bitdepthMin8&1) * 4
	}
	if secStrength != 0 {
		p.secShr = int32(damping - ulog2(uint32(secStrength)))
	}
}

//go:noescape
func cdef8x8AVX512(dst *uint8, stride int, left *uint8, top *uint8, bot *uint8, p *cdefZParams)

//go:noescape
func cdef4x4_16AVX512(dst *uint16, stride int, left *uint16, top *uint16, bot *uint16, p *cdefZParams)

//go:noescape
func cdef8x8_16AVX512(dst *uint16, stride int, left *uint16, top *uint16, bot *uint16, p *cdefZParams)
