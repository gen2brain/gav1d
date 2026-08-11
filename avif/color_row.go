package avif

import (
	"encoding/binary"

	"github.com/gen2brain/gav1d/av1"
)

type rowConsts struct {
	w0, w1, w2     float32
	crCoef, cbCoef float32
	gcr, gcb, kg   float32
	outMax         float32
	one, half      float32
	zero           float32
}

type convertRow func(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts)

type convertRow16 func(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint16,
	n int, c *rowConsts)

func (s *colorState) fastRow(outDepth int) bool {
	if !s.hasColor {
		return false
	}
	if s.depth == 8 != (outDepth == 8) {
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

func normRowGo(dst []float32, src []uint8, bias, rng float32) {
	src = src[:len(dst)]
	for i, v := range src {
		dst[i] = (float32(v) - bias) / rng
	}
}

func normRow16Go(dst []float32, src []uint8, maxCh uint16, bias, rng float32) {
	src = src[: 2*len(dst) : 2*len(dst)]
	for i := range dst {
		v := min(binary.NativeEndian.Uint16(src[2*i:]), maxCh)
		dst[i] = (float32(v) - bias) / rng
	}
}

func convertRow444Go(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint8,
	n int, c *rowConsts,
) {
	for x := range n {
		yy := yf[x]
		cb, cr := u0[x+1], v0[x+1]

		r := yy + c.crCoef*cr
		b := yy + c.cbCoef*cb
		g := yy - (2*((c.gcr*cr)+(c.gcb*cb)))/c.kg

		dst[4*x] = uint8(0.5 + clampF(r)*c.outMax)
		dst[4*x+1] = uint8(0.5 + clampF(g)*c.outMax)
		dst[4*x+2] = uint8(0.5 + clampF(b)*c.outMax)
		dst[4*x+3] = a[x]
	}
}

func convertRow16x444Go(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint16,
	n int, c *rowConsts,
) {
	for x := range n {
		yy := yf[x]
		cb, cr := u0[x+1], v0[x+1]

		r := yy + c.crCoef*cr
		b := yy + c.cbCoef*cb
		g := yy - (2*((c.gcr*cr)+(c.gcb*cb)))/c.kg

		o := dst[8*x : 8*x+8 : 8*x+8]
		binary.BigEndian.PutUint16(o[0:], uint16(0.5+clampF(r)*c.outMax))
		binary.BigEndian.PutUint16(o[2:], uint16(0.5+clampF(g)*c.outMax))
		binary.BigEndian.PutUint16(o[4:], uint16(0.5+clampF(b)*c.outMax))
		binary.BigEndian.PutUint16(o[6:], a[x])
	}
}

func convertRow16Go(dst []uint8, yf, u0, u1, v0, v1 []float32, a []uint16,
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

		o := dst[8*x : 8*x+8 : 8*x+8]
		binary.BigEndian.PutUint16(o[0:], uint16(0.5+clampF(r)*c.outMax))
		binary.BigEndian.PutUint16(o[2:], uint16(0.5+clampF(g)*c.outMax))
		binary.BigEndian.PutUint16(o[4:], uint16(0.5+clampF(b)*c.outMax))
		binary.BigEndian.PutUint16(o[6:], a[x])
	}
}

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

	u0 = s.padRow(s.uPad[0], pic.Data[1], pic.Stride[1], uvY, uw)
	v0 = s.padRow(s.vPad[0], pic.Data[2], pic.Stride[2], uvY, uw)
	if adjRow == 0 {
		return u0, u0, v0, v0
	}

	u1 = s.padRow(s.uPad[1], pic.Data[1], pic.Stride[1], uvY+adjRow, uw)
	v1 = s.padRow(s.vPad[1], pic.Data[2], pic.Stride[2], uvY+adjRow, uw)

	return u0, u1, v0, v1
}

func (s *colorState) padRow(dst []float32, src []byte, stride, y, uw int) []float32 {
	if s.depth == 8 {
		normRow(dst[1:1+uw], src[y*stride:y*stride+uw], s.biasUV, s.rangeUV)
	} else {
		normRow16(dst[1:1+uw], src[y*stride:y*stride+2*uw],
			uint16(s.maxChannel), s.biasUV, s.rangeUV)
	}
	dst[0] = dst[1]
	dst[uw+1] = dst[uw]

	return dst
}

func (s *colorState) lumaRow(pic *av1.Picture, y, w int) []float32 {
	off := y * pic.Stride[0]
	if s.depth == 8 {
		normRow(s.yf[:w], pic.Data[0][off:off+w], s.biasY, s.rangeY)
	} else {
		normRow16(s.yf[:w], pic.Data[0][off:off+2*w],
			uint16(s.maxChannel), s.biasY, s.rangeY)
	}

	return s.yf
}
