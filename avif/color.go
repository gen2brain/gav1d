package avif

import "github.com/gen2brain/gav1d/av1"

const (
	mcIdentity  = 0
	mcBT709     = 1
	mcUnspec    = 2
	mcFCC       = 4
	mcBT470BG   = 5
	mcBT601     = 6
	mcSMPTE240  = 7
	mcYCgCo     = 8
	mcBT2020NCL = 9
)

// yuvCoefficients returns kr, kg, kb for a CICP matrix_coefficients value.
func yuvCoefficients(matrix int) (float32, float32, float32) {
	kr, kb := float32(0.299), float32(0.114)

	switch matrix {
	case mcBT709:
		kr, kb = 0.2126, 0.0722
	case mcFCC:
		kr, kb = 0.30, 0.11
	case mcBT470BG, mcBT601:
		kr, kb = 0.299, 0.114
	case mcSMPTE240:
		kr, kb = 0.212, 0.087
	case mcBT2020NCL:
		kr, kb = 0.2627, 0.0593
	}

	return kr, 1 - kr - kb, kb
}

type colorState struct {
	matrix     int
	fullRange  bool
	depth      int
	maxChannel int

	kr, kg, kb       float32
	tableY, tableUV  []float32
	outMax           float32
	ssHor, ssVer     int
	hasColor         bool
	unsupportedSpace bool
}

// newColorState resolves the color description, preferring nclx over the sequence header.
func newColorState(pic *av1.Picture, nclx *colorInfo, outDepth int) *colorState {
	s := &colorState{
		matrix:    pic.MatrixCoeffs,
		fullRange: pic.FullRange,
		depth:     pic.BitDepth,
	}
	if nclx != nil && nclx.hasNCLX {
		s.matrix = int(nclx.matrix)
		s.fullRange = nclx.fullRange
	}
	if s.matrix == mcUnspec {
		s.matrix = mcBT601
	}

	s.maxChannel = 1<<s.depth - 1
	s.outMax = float32(int(1)<<outDepth - 1)
	s.ssHor, s.ssVer = subsampling(pic.Layout)
	s.hasColor = pic.Layout != av1.LayoutI400
	s.kr, s.kg, s.kb = yuvCoefficients(s.matrix)

	switch s.matrix {
	case mcIdentity:
		if pic.Layout != av1.LayoutI444 && pic.Layout != av1.LayoutI400 {
			s.unsupportedSpace = true
		}
	case mcYCgCo:
		if !s.fullRange {
			s.unsupportedSpace = true
		}
	case 3, 10, 11, 12, 13, 14:
		s.unsupportedSpace = true
	}

	shift := s.depth - 8
	biasY, rangeY := float32(0), float32(s.maxChannel)
	if !s.fullRange {
		biasY = float32(int(16) << shift)
		rangeY = float32(int(219) << shift)
	}
	biasUV := float32(int(1) << (s.depth - 1))
	rangeUV := float32(s.maxChannel)
	if !s.fullRange {
		rangeUV = float32(int(224) << shift)
	}

	n := 1 << s.depth
	s.tableY = make([]float32, n)
	for i := range n {
		s.tableY[i] = (float32(i) - biasY) / rangeY
	}
	if s.matrix == mcIdentity {
		s.tableUV = s.tableY
	} else {
		s.tableUV = make([]float32, n)
		for i := range n {
			s.tableUV[i] = (float32(i) - biasUV) / rangeUV
		}
	}

	return s
}

func (s *colorState) sample(data []byte, stride, x, y int) int {
	if s.depth == 8 {
		return int(data[y*stride+x])
	}
	i := y*stride + x*2
	v := int(data[i]) | int(data[i+1])<<8

	return min(v, s.maxChannel)
}

func clampF(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}

	return v
}

// rgb converts one pixel, returning components scaled to the output depth.
func (s *colorState) rgb(pic *av1.Picture, x, y int) (uint16, uint16, uint16) {
	unormY := s.sample(pic.Data[0], pic.Stride[0], x, y)
	yy := s.tableY[unormY]

	if !s.hasColor {
		g := uint16(0.5 + clampF(yy)*s.outMax)

		return g, g, g
	}

	uvX, uvY := x>>s.ssHor, y>>s.ssVer
	cb := s.tableUV[s.sample(pic.Data[1], pic.Stride[1], uvX, uvY)]
	cr := s.tableUV[s.sample(pic.Data[2], pic.Stride[2], uvX, uvY)]

	var r, g, b float32
	switch s.matrix {
	case mcIdentity:
		g, b, r = yy, cb, cr
	case mcYCgCo:
		t := yy - cb
		g, b, r = yy+cb, t-cr, t+cr
	default:
		r = yy + 2*(1-s.kr)*cr
		b = yy + 2*(1-s.kb)*cb
		g = yy - (2*((s.kr*(1-s.kr)*cr)+(s.kb*(1-s.kb)*cb)))/s.kg
	}

	return uint16(0.5 + clampF(r)*s.outMax),
		uint16(0.5 + clampF(g)*s.outMax),
		uint16(0.5 + clampF(b)*s.outMax)
}

func (s *colorState) alpha(pic *av1.Picture, x, y int) uint16 {
	v := s.tableY[s.sample(pic.Data[0], pic.Stride[0], x, y)]

	return uint16(0.5 + clampF(v)*s.outMax)
}

// lumaTable builds the luma normalization, which alpha uses too.
func lumaTable(depth int, full bool) []float32 {
	maxChannel := int(1)<<depth - 1
	shift := depth - 8
	bias, rng := float32(0), float32(maxChannel)
	if !full {
		bias, rng = float32(int(16)<<shift), float32(int(219)<<shift)
	}
	t := make([]float32, maxChannel+1)
	for i := range t {
		t[i] = (float32(i) - bias) / rng
	}

	return t
}
