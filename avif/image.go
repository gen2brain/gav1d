package avif

import (
	"errors"
	"image"
	"image/color"

	"github.com/gen2brain/gav1d/av1"
)

var errUnsupported = errors.New("avif: unsupported image")

func colorModelFor(f *file, it *item) color.Model {
	if p := f.meta.prop(it, "av1C"); p != nil && p.av1C.highBitdepth {
		return color.NRGBA64Model
	}

	return color.NRGBAModel
}

func alphaState(alpha *av1.Picture, outDepth int, full bool) *colorState {
	s := &colorState{
		depth:      alpha.BitDepth,
		maxChannel: int(1)<<alpha.BitDepth - 1,
		outMax:     float32(int(1)<<outDepth - 1),
		tableY:     lumaTable(alpha.BitDepth, full),
	}
	s.tableUV = s.tableY
	s.derive()

	return s
}

// planar wraps the decoded planes without converting them, which is what the
// bitstream already holds. image.YCbCr reads them as full-range BT.601, so a
// file signalling anything else is exact as data but not as color.
func planar(pic, alpha *av1.Picture) (image.Image, bool) {
	if pic.BitDepth != 8 {
		return nil, false
	}

	rect := image.Rect(0, 0, pic.Width, pic.Height)

	if pic.Layout == av1.LayoutI400 {
		if alpha != nil {
			return nil, false
		}

		return &image.Gray{Pix: pic.Data[0], Stride: pic.Stride[0], Rect: rect}, true
	}

	var ratio image.YCbCrSubsampleRatio
	switch pic.Layout {
	case av1.LayoutI420:
		ratio = image.YCbCrSubsampleRatio420
	case av1.LayoutI422:
		ratio = image.YCbCrSubsampleRatio422
	case av1.LayoutI444:
		ratio = image.YCbCrSubsampleRatio444
	default:
		return nil, false
	}

	y := image.YCbCr{
		Y: pic.Data[0], Cb: pic.Data[1], Cr: pic.Data[2],
		YStride: pic.Stride[0], CStride: pic.Stride[1],
		SubsampleRatio: ratio, Rect: rect,
	}
	if alpha == nil {
		return &y, true
	}
	if alpha.BitDepth != 8 || alpha.Layout != av1.LayoutI400 {
		return nil, false
	}

	return &image.NYCbCrA{YCbCr: y, A: alpha.Data[0], AStride: alpha.Stride[0]}, true
}

func toImage(f *file, it *item, pic, alpha *av1.Picture, ycbcr bool) (image.Image, error) {
	if ycbcr {
		if img, ok := planar(pic, alpha); ok {
			return img, nil
		}
	}

	outDepth := 8
	if pic.BitDepth > 8 {
		outDepth = 16
	}

	var nclx *colorInfo
	if p := f.meta.prop(it, "colr"); p != nil {
		nclx = p.colr
	}
	cs := newColorState(pic, nclx, outDepth)
	if cs.unsupportedSpace {
		return nil, errUnsupported
	}

	var as *colorState
	if alpha != nil {
		if alpha.Layout != av1.LayoutI400 {
			return nil, errUnsupported
		}
		as = alphaState(alpha, outDepth, alpha.FullRange)
	}

	rect := image.Rect(0, 0, pic.Width, pic.Height)
	w := pic.Width

	cs.prepare(w)
	rgb := make([]uint16, 3*w)

	aRow := make([]uint16, w)
	if as != nil {
		as.prepare(w)
	} else {
		for x := range aRow {
			aRow[x] = 0xffff
		}
	}

	if outDepth == 8 {
		dst := image.NewNRGBA(rect)
		for y := range pic.Height {
			cs.rgbRow(pic, y, rgb)
			if as != nil {
				as.alphaRow(alpha, y, w, aRow)
			}
			o := y * dst.Stride
			for x := range w {
				dst.Pix[o] = uint8(rgb[3*x])
				dst.Pix[o+1] = uint8(rgb[3*x+1])
				dst.Pix[o+2] = uint8(rgb[3*x+2])
				dst.Pix[o+3] = uint8(aRow[x])
				o += 4
			}
		}

		return dst, nil
	}

	dst := image.NewNRGBA64(rect)
	for y := range pic.Height {
		cs.rgbRow(pic, y, rgb)
		if as != nil {
			as.alphaRow(alpha, y, w, aRow)
		}
		o := y * dst.Stride
		for x := range w {
			r, g, b := rgb[3*x], rgb[3*x+1], rgb[3*x+2]
			a := aRow[x]
			dst.Pix[o] = uint8(r >> 8)
			dst.Pix[o+1] = uint8(r)
			dst.Pix[o+2] = uint8(g >> 8)
			dst.Pix[o+3] = uint8(g)
			dst.Pix[o+4] = uint8(b >> 8)
			dst.Pix[o+5] = uint8(b)
			dst.Pix[o+6] = uint8(a >> 8)
			dst.Pix[o+7] = uint8(a)
			o += 8
		}
	}

	return dst, nil
}
