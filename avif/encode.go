package avif

import (
	"image"
	"image/color"
	"io"

	"github.com/gen2brain/gav1d/av1"
)

// DefaultQuality is used when EncodeOptions.Quality is zero.
const DefaultQuality = 60

// DefaultSpeed is used when EncodeOptions.Speed is negative.
const DefaultSpeed = 6

// EncodeOptions holds the encoding parameters.
type EncodeOptions struct {
	// Quality runs from 1 to 100, where 100 is lossless. Zero selects
	// DefaultQuality.
	Quality int
	// QualityAlpha runs from 1 to 100 for the alpha plane. Zero follows
	// Quality.
	QualityAlpha int
	// Speed runs from 0 to 10, where 0 searches the most and 10 the least.
	// A negative value selects DefaultSpeed.
	Speed int
	// Lossless encodes at the lowest quantizer. Color conversion and chroma
	// subsampling still apply, so only a YCbCr or grayscale source is exact.
	Lossless bool
}

func qindex(quality int) int {
	if quality <= 0 {
		quality = DefaultQuality
	}

	q := (100 - min(quality, 100)) * 255 / 100

	return max(q, 0)
}

func encodeOptions(opts []EncodeOptions) EncodeOptions {
	var o EncodeOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	if o.Quality == 0 {
		o.Quality = DefaultQuality
	}

	if o.QualityAlpha == 0 {
		o.QualityAlpha = o.Quality
	}

	if o.Lossless {
		o.Quality, o.QualityAlpha = 100, 100
	}

	if o.Speed < 0 {
		o.Speed = DefaultSpeed
	}

	return o
}

type planes struct {
	y, u, v, a       []byte
	yStride, cStride int
	width, height    int
	mono             bool
}

func newPlanes(w, h int, mono bool) *planes {
	p := &planes{
		yStride: w,
		cStride: (w + 1) / 2,
		width:   w,
		height:  h,
		mono:    mono,
	}

	p.y = make([]byte, p.yStride*h)
	if !mono {
		n := p.cStride * ((h + 1) / 2)
		p.u = make([]byte, n)
		p.v = make([]byte, n)
	}

	return p
}

// BT.601, which is what the decoder assumes for an unspecified matrix.
const (
	kr = 0.299
	kg = 0.587
	kb = 0.114
)

func clamp8(v float32) byte {
	if v <= 0 {
		return 0
	}

	if v >= 255 {
		return 255
	}

	return byte(v + 0.5)
}

func (p *planes) setRGB(x, y int, r, g, b byte, cb, cr []float32) {
	fr, fg, fb := float32(r), float32(g), float32(b)
	yf := kr*fr + kg*fg + kb*fb

	p.y[y*p.yStride+x] = clamp8(yf)

	if p.mono {
		return
	}

	cb[x] = (fb - yf) / (2 * (1 - kb))
	cr[x] = (fr - yf) / (2 * (1 - kr))
}

func (p *planes) flushChroma(y int, cb0, cr0, cb1, cr1 []float32, rows int) {
	if p.mono {
		return
	}

	row := (y / 2) * p.cStride
	for x := 0; x < p.cStride; x++ {
		x0, x1 := 2*x, min(2*x+1, p.width-1)

		n := float32(rows * 2)
		sb := cb0[x0] + cb0[x1]
		sr := cr0[x0] + cr0[x1]
		if rows == 2 {
			sb += cb1[x0] + cb1[x1]
			sr += cr1[x0] + cr1[x1]
		}

		p.u[row+x] = clamp8(sb/n + 128)
		p.v[row+x] = clamp8(sr/n + 128)
	}
}

func fromImage(m image.Image) (*planes, error) {
	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil, ErrUnsupported
	}

	_, mono := m.(*image.Gray)
	p := newPlanes(w, h, mono)

	cb0 := make([]float32, w)
	cr0 := make([]float32, w)
	cb1 := make([]float32, w)
	cr1 := make([]float32, w)

	var alpha []byte
	opaque := true

	for y := 0; y < h; y++ {
		cb, cr := cb0, cr0
		if y%2 == 1 {
			cb, cr = cb1, cr1
		}

		for x := 0; x < w; x++ {
			r, g, bl, a := nrgbaAt(m, b.Min.X+x, b.Min.Y+y)
			p.setRGB(x, y, r, g, bl, cb, cr)

			if a != 0xff {
				if opaque {
					alpha = make([]byte, w*h)
					for i := range alpha {
						alpha[i] = 0xff
					}
					opaque = false
				}
			}

			if alpha != nil {
				alpha[y*w+x] = a
			}
		}

		if y%2 == 1 {
			p.flushChroma(y-1, cb0, cr0, cb1, cr1, 2)
		} else if y == h-1 {
			p.flushChroma(y, cb0, cr0, nil, nil, 1)
		}
	}

	p.a = alpha

	return p, nil
}

func nrgbaAt(m image.Image, x, y int) (r, g, b, a byte) {
	switch src := m.(type) {
	case *image.NRGBA:
		i := src.PixOffset(x, y)
		s := src.Pix[i : i+4 : i+4]

		return s[0], s[1], s[2], s[3]

	case *image.RGBA:
		i := src.PixOffset(x, y)
		s := src.Pix[i : i+4 : i+4]
		if s[3] == 0 {
			return 0, 0, 0, 0
		}
		if s[3] == 0xff {
			return s[0], s[1], s[2], 0xff
		}

		a := uint32(s[3])

		return byte(uint32(s[0]) * 0xff / a), byte(uint32(s[1]) * 0xff / a),
			byte(uint32(s[2]) * 0xff / a), s[3]

	case *image.Gray:
		v := src.GrayAt(x, y).Y

		return v, v, v, 0xff
	}

	c := color.NRGBAModel.Convert(m.At(x, y)).(color.NRGBA)

	return c.R, c.G, c.B, c.A
}

func fromYCbCr(m *image.YCbCr, a *image.NYCbCrA) (*planes, bool) {
	if m.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		return nil, false
	}

	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	p := &planes{
		y: m.Y, u: m.Cb, v: m.Cr,
		yStride: m.YStride, cStride: m.CStride,
		width: w, height: h,
	}

	if m.YOffset(b.Min.X, b.Min.Y) != 0 || m.COffset(b.Min.X, b.Min.Y) != 0 {
		return nil, false
	}

	if a != nil {
		if a.AStride != m.YStride {
			return nil, false
		}
		p.a = a.A
	}

	return p, true
}

// Encode writes m to w as a still AVIF image, all-intra and 8-bit, at 4:2:0
// or grayscale for an *[image.Gray] source. An *[image.YCbCr] at 4:2:0 and an
// *[image.NYCbCrA] are taken from their planes without resampling, anything
// else is converted. Alpha becomes a second, monochrome item.
func Encode(w io.Writer, m image.Image, opts ...EncodeOptions) error {
	o := encodeOptions(opts)

	var p *planes
	var err error

	switch src := m.(type) {
	case *image.NYCbCrA:
		var ok bool
		if p, ok = fromYCbCr(&src.YCbCr, src); !ok {
			p, err = fromImage(m)
		}
	case *image.YCbCr:
		var ok bool
		if p, ok = fromYCbCr(src, nil); !ok {
			p, err = fromImage(m)
		}
	default:
		p, err = fromImage(m)
	}

	if err != nil {
		return err
	}

	data, err := encodePlanes(p, o)
	if err != nil {
		return err
	}

	_, err = w.Write(data)

	return err
}

func encodePlanes(p *planes, o EncodeOptions) ([]byte, error) {
	color := av1.Encode(av1.EncodeConfig{
		Width:       p.width,
		Height:      p.height,
		BitDepth:    8,
		QIndex:      qindex(o.Quality),
		Speed:       o.Speed,
		Monochrome:  p.mono,
		FullRange:   true,
		Src:         p.y,
		SrcStride:   p.yStride,
		SrcU:        p.u,
		SrcV:        p.v,
		SrcUVStride: p.cStride,
	})
	if color == nil {
		return nil, ErrUnsupported
	}

	props := []encProp{
		ispeProp(p.width, p.height),
		pixiProp(channels(p.mono)),
		av1CProp(p.mono, seqHdrOBU(color)),
		colrProp(ColorInfo{Primaries: 1, Transfer: 13, Matrix: 6, FullRange: true}),
	}

	items := []encItem{{id: 1, name: "Color", data: color, props: []int{0, 1, 2, 3}}}
	alphaOf := uint16(0)

	if p.a != nil {
		alpha := av1.Encode(av1.EncodeConfig{
			Width:      p.width,
			Height:     p.height,
			BitDepth:   8,
			QIndex:     qindex(o.QualityAlpha),
			Speed:      o.Speed,
			Monochrome: true,
			FullRange:  true,
			Src:        p.a,
			SrcStride:  p.yStride,
		})
		if alpha == nil {
			return nil, ErrUnsupported
		}

		props = append(props,
			pixiProp(1),
			av1CProp(true, seqHdrOBU(alpha)),
			auxCProp(alphaURN),
		)

		items = append(items, encItem{
			id: 2, name: "Alpha", data: alpha,
			props: []int{0, 4, 5, 6},
		})
		alphaOf = 1
	}

	return buildAVIF(items, props, alphaOf), nil
}

func channels(mono bool) int {
	if mono {
		return 1
	}

	return 3
}

func seqHdrOBU(tu []byte) []byte {
	for i := 0; i < len(tu); {
		start := i

		if tu[i]&0x80 != 0 {
			return nil
		}
		typ := int(tu[i] >> 3 & 0xf)
		ext := tu[i]>>2&1 != 0
		hasSize := tu[i]>>1&1 != 0
		i++

		if ext {
			i++
		}

		if !hasSize || i > len(tu) {
			return nil
		}

		size, n := uleb128(tu[i:])
		if n == 0 {
			return nil
		}
		i += n

		if i+size > len(tu) {
			return nil
		}
		i += size

		if typ == 1 {
			return tu[start:i]
		}
	}

	return nil
}

func uleb128(b []byte) (int, int) {
	var v uint64
	for i := 0; i < len(b) && i < 8; i++ {
		v |= uint64(b[i]&0x7f) << (7 * i)
		if b[i]&0x80 == 0 {
			return int(v), i + 1
		}
	}

	return 0, 0
}
