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
	return &colorState{
		depth:      alpha.BitDepth,
		maxChannel: int(1)<<alpha.BitDepth - 1,
		outMax:     float32(int(1)<<outDepth - 1),
		tableY:     lumaTable(alpha.BitDepth, full),
	}
}

func toImage(f *file, it *item, pic, alpha *av1.Picture) (image.Image, error) {
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

	if outDepth == 8 {
		dst := image.NewNRGBA(rect)
		for y := range pic.Height {
			o := y * dst.Stride
			for x := range pic.Width {
				r, g, b := cs.rgb(pic, x, y)
				a := uint16(0xff)
				if as != nil {
					a = as.alpha(alpha, x, y)
				}
				dst.Pix[o] = uint8(r)
				dst.Pix[o+1] = uint8(g)
				dst.Pix[o+2] = uint8(b)
				dst.Pix[o+3] = uint8(a)
				o += 4
			}
		}

		return dst, nil
	}

	dst := image.NewNRGBA64(rect)
	for y := range pic.Height {
		o := y * dst.Stride
		for x := range pic.Width {
			r, g, b := cs.rgb(pic, x, y)
			a := uint16(0xffff)
			if as != nil {
				a = as.alpha(alpha, x, y)
			}
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
