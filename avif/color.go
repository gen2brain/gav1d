package avif

import (
	"encoding/binary"

	"github.com/gen2brain/gav1d/av1"
)

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
	biasY, rangeY    float32
	biasUV, rangeUV  float32
	tableY, tableUV  []float32
	tw0, tw1, tw2    []float32
	pixLUT           []uint16
	crCoef, cbCoef   float32
	gcr, gcb         float32
	outMax           float32
	ssHor, ssVer     int
	hasColor         bool
	unsupportedSpace bool

	uvIdx, uvAdj []int32
	uPad, vPad   [2][]float32
	yf           []float32
	row          convertRow
	row16        convertRow16
	consts       rowConsts
	yRow         []uint16
	uRow, vRow   [2][]uint16
	cbRow, crRow []float32
}

func (f *file) colorInfo(it *item, pic *av1.Picture) ColorInfo {
	ci := ColorInfo{
		Primaries: uint16(pic.ColorPrimaries),
		Transfer:  uint16(pic.Transfer),
		Matrix:    uint16(pic.MatrixCoeffs),
		FullRange: pic.FullRange,
	}

	p := f.meta.prop(it, "colr")
	if p == nil || p.colr == nil {
		return ci
	}
	if p.colr.hasNCLX {
		ci.Primaries = p.colr.primaries
		ci.Transfer = p.colr.transfer
		ci.Matrix = p.colr.matrix
		ci.FullRange = p.colr.fullRange
	}
	ci.ICCP = p.colr.icc

	return ci
}

func newColorState(pic *av1.Picture, ci ColorInfo, outDepth int) *colorState {
	s := &colorState{
		matrix:    int(ci.Matrix),
		fullRange: ci.FullRange,
		depth:     pic.BitDepth,
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

	s.biasY, s.rangeY = biasY, rangeY
	s.biasUV, s.rangeUV = biasUV, rangeUV

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

	s.derive()

	return s
}

func (s *colorState) derive() {
	const w0, w1, w2 = 9.0 / 16.0, 3.0 / 16.0, 1.0 / 16.0

	n := len(s.tableUV)
	s.tw0 = make([]float32, n)
	s.tw1 = make([]float32, n)
	s.tw2 = make([]float32, n)
	for i := range n {
		s.tw0[i] = s.tableUV[i] * w0
		s.tw1[i] = s.tableUV[i] * w1
		s.tw2[i] = s.tableUV[i] * w2
	}

	s.pixLUT = make([]uint16, len(s.tableY))
	for i := range s.tableY {
		s.pixLUT[i] = uint16(0.5 + clampF(s.tableY[i])*s.outMax)
	}

	s.crCoef, s.cbCoef = 2*(1-s.kr), 2*(1-s.kb)
	s.gcr, s.gcb = s.kr*(1-s.kr), s.kb*(1-s.kb)
}

func (s *colorState) prepare(w int) {
	uw := (w + s.ssHor) >> s.ssHor

	s.yRow = make([]uint16, w)
	for i := range s.uRow {
		s.uRow[i] = make([]uint16, uw)
		s.vRow[i] = make([]uint16, uw)
	}
	s.cbRow = make([]float32, w)
	s.crRow = make([]float32, w)

	for i := range s.uPad {
		s.uPad[i] = make([]float32, uw+8)
		s.vPad[i] = make([]float32, uw+8)
	}
	s.yf = make([]float32, w)

	s.uvIdx = make([]int32, w)
	s.uvAdj = make([]int32, w)
	for x := range w {
		uvX := x >> s.ssHor
		adj := 0
		if x != 0 && !(x == w-1 && x%2 != 0) {
			if x%2 != 0 {
				adj = 1
			} else {
				adj = -1
			}
		}
		s.uvIdx[x] = int32(uvX)
		s.uvAdj[x] = int32(uvX + adj)
	}
}

func (s *colorState) fillIdx(data []byte, stride, y, n int, dst []uint16) {
	if s.depth == 8 {
		row := data[y*stride : y*stride+n]
		dst = dst[:n]
		for i, v := range row {
			dst[i] = uint16(v)
		}

		return
	}

	row := data[y*stride : y*stride+2*n]
	maxCh := uint16(s.maxChannel)
	for i := range n {
		dst[i] = min(binary.NativeEndian.Uint16(row[2*i:]), maxCh)
	}
}

func (s *colorState) chromaRow(pic *av1.Picture, y, w int) {
	uvY := y >> s.ssVer
	uw := (w + s.ssHor) >> s.ssHor

	s.fillIdx(pic.Data[1], pic.Stride[1], uvY, uw, s.uRow[0])
	s.fillIdx(pic.Data[2], pic.Stride[2], uvY, uw, s.vRow[0])

	tab := s.tableUV
	cbRow, crRow := s.cbRow[:w], s.crRow[:w]

	if s.ssHor == 0 {
		u, v := s.uRow[0][:w], s.vRow[0][:w]
		for x := range cbRow {
			cbRow[x] = tab[u[x]]
			crRow[x] = tab[v[x]]
		}

		return
	}

	adjRow := 0
	if s.ssVer != 0 && y != 0 && !(y == pic.Height-1 && y%2 != 0) {
		if y%2 != 0 {
			adjRow = 1
		} else {
			adjRow = -1
		}
	}

	u0, v0 := s.uRow[0], s.vRow[0]
	u1, v1 := u0, v0
	if adjRow != 0 {
		s.fillIdx(pic.Data[1], pic.Stride[1], uvY+adjRow, uw, s.uRow[1])
		s.fillIdx(pic.Data[2], pic.Stride[2], uvY+adjRow, uw, s.vRow[1])
		u1, v1 = s.uRow[1], s.vRow[1]
	}

	tw0, tw1, tw2 := s.tw0, s.tw1, s.tw2
	uvIdx, uvAdj := s.uvIdx[:w], s.uvAdj[:w]
	for x := range cbRow {
		i, j := uvIdx[x], uvAdj[x]
		cbRow[x] = tw0[u0[i]] + tw1[u0[j]] + tw1[u1[i]] + tw2[u1[j]]
		crRow[x] = tw0[v0[i]] + tw1[v0[j]] + tw1[v1[i]] + tw2[v1[j]]
	}
}

func (s *colorState) rgbRow(pic *av1.Picture, y int, dst []uint16) {
	w := pic.Width
	s.fillIdx(pic.Data[0], pic.Stride[0], y, w, s.yRow)

	if !s.hasColor {
		for x, v := range s.yRow {
			g := s.pixLUT[v]
			dst[3*x], dst[3*x+1], dst[3*x+2] = g, g, g
		}

		return
	}

	s.chromaRow(pic, y, w)

	tabY := s.tableY
	yRow := s.yRow[:w]
	cbRow, crRow := s.cbRow[:w], s.crRow[:w]
	dst = dst[:3*w]
	outMax := s.outMax

	for x, iy := range yRow {
		yy := tabY[iy]
		cb, cr := cbRow[x], crRow[x]

		var r, g, b float32
		switch s.matrix {
		case mcIdentity:
			g, b, r = yy, cb, cr
		case mcYCgCo:
			t := yy - cb
			g, b, r = yy+cb, t-cr, t+cr
		default:
			r = yy + s.crCoef*cr
			b = yy + s.cbCoef*cb
			g = yy - (2*((s.gcr*cr)+(s.gcb*cb)))/s.kg
		}

		o := dst[3*x : 3*x+3 : 3*x+3]
		o[0] = uint16(0.5 + clampF(r)*outMax)
		o[1] = uint16(0.5 + clampF(g)*outMax)
		o[2] = uint16(0.5 + clampF(b)*outMax)
	}
}

func (s *colorState) alphaRow(pic *av1.Picture, y, w int, dst []uint16) {
	s.fillIdx(pic.Data[0], pic.Stride[0], y, w, s.yRow)
	for x, v := range s.yRow {
		dst[x] = s.pixLUT[v]
	}
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
