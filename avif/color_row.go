package avif

import "github.com/gen2brain/gav1d/av1"

// rowConsts carries everything the row kernel needs. The scalar path builds the
// same values into tables, so the kernels reproduce its arithmetic exactly
// rather than approximating it.
type rowConsts struct {
	w0, w1, w2     float32
	crCoef, cbCoef float32
	gcr, gcb, kg   float32
	outMax         float32
	one, half      float32
}

// convertRow is a whole output row. The samples arrive already normalized,
// which is one table lookup per sample rather than one per tap, and leaves the
// kernels pure arithmetic: no gather and no divide.
type convertRow func(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts)

// fastRow reports whether the row kernel covers this image. It handles 8-bit
// input and output, subsampled chroma and the three-coefficient matrices; the
// scalar path keeps everything else.
func (s *colorState) fastRow(outDepth int) bool {
	if s.depth != 8 || outDepth != 8 || !s.hasColor || s.ssHor != 1 {
		return false
	}
	switch s.matrix {
	case mcIdentity, mcYCgCo:
		return false
	}

	return true
}

func (s *colorState) rowConsts() rowConsts {
	const w0, w1, w2 = 9.0 / 16.0, 3.0 / 16.0, 1.0 / 16.0

	return rowConsts{
		w0: w0, w1: w1, w2: w2,
		crCoef: s.crCoef, cbCoef: s.cbCoef,
		gcr: s.gcr, gcb: s.gcb, kg: s.kg,
		outMax: s.outMax, one: 1, half: 0.5,
	}
}

// padUV normalizes one chroma row with its end samples replicated, which turns
// the column clamp at the two edges into the same neighbour arithmetic as the
// rest of the row.
func padUV(dst []float32, tab []float32, src []byte, stride, y, uw int) []float32 {
	row := src[y*stride : y*stride+uw]
	for i, v := range row {
		dst[i+1] = tab[v]
	}
	dst[0] = dst[1]
	dst[uw+1] = dst[uw]

	return dst
}

// convertRowGo is the reference the kernels are tested against.
func convertRowGo(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts,
) {
	for x := range n {
		k := x >> 1
		j := k
		if x&1 != 0 {
			j = k + 2
		}
		k++

		yy := yf[x]
		cb := u0[k]*c.w0 + u0[j]*c.w1 + u1[k]*c.w1 + u1[j]*c.w2
		cr := v0[k]*c.w0 + v0[j]*c.w1 + v1[k]*c.w1 + v1[j]*c.w2

		r := yy + c.crCoef*cr
		b := yy + c.cbCoef*cb
		g := yy - (2*((c.gcr*cr)+(c.gcb*cb)))/c.kg

		dst[4*x] = uint8(0.5 + clampF(r)*c.outMax)
		dst[4*x+1] = uint8(0.5 + clampF(g)*c.outMax)
		dst[4*x+2] = uint8(0.5 + clampF(b)*c.outMax)
		dst[4*x+3] = a[x]
	}
}

// rowPlanes picks the two chroma rows an output row reads, replicating the
// vertical neighbour the way the scalar path clamps it.
func (s *colorState) rowPlanes(pic *av1.Picture, y, w int) (u0, u1, v0, v1 []float32) {
	uvY := y >> s.ssVer
	uw := (w + s.ssHor) >> s.ssHor

	adjRow := 0
	if s.ssVer != 0 && y != 0 && !(y == pic.Height-1 && y%2 != 0) {
		if y%2 != 0 {
			adjRow = 1
		} else {
			adjRow = -1
		}
	}

	u0 = padUV(s.uPad[0], s.tableUV, pic.Data[1], pic.Stride[1], uvY, uw)
	v0 = padUV(s.vPad[0], s.tableUV, pic.Data[2], pic.Stride[2], uvY, uw)
	if adjRow == 0 {
		return u0, u0, v0, v0
	}

	u1 = padUV(s.uPad[1], s.tableUV, pic.Data[1], pic.Stride[1], uvY+adjRow, uw)
	v1 = padUV(s.vPad[1], s.tableUV, pic.Data[2], pic.Stride[2], uvY+adjRow, uw)

	return u0, u1, v0, v1
}

// lumaRow normalizes one luma row.
func (s *colorState) lumaRow(pic *av1.Picture, y, w int) []float32 {
	row := pic.Data[0][y*pic.Stride[0] : y*pic.Stride[0]+w]
	for i, v := range row {
		s.yf[i] = s.tableY[v]
	}

	return s.yf
}
