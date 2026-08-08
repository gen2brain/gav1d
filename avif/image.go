package avif

import (
	"errors"
	"image"
	"image/color"

	"github.com/gen2brain/gav1d/av1"
)

var errUnsupported = errors.New("avif: unsupported image")

func ratio(layout int) image.YCbCrSubsampleRatio {
	switch layout {
	case av1.LayoutI420:
		return image.YCbCrSubsampleRatio420
	case av1.LayoutI422:
		return image.YCbCrSubsampleRatio422
	default:
		return image.YCbCrSubsampleRatio444
	}
}

func colorModelFor(f *file, it *item) color.Model {
	p := f.meta.prop(it, "av1C")
	mono := p != nil && p.av1C.monochrome
	if f.alphaFor(it.id) != nil {
		return color.NYCbCrAModel
	}
	if mono {
		return color.GrayModel
	}

	return color.YCbCrModel
}

func toImage(pic, alpha *av1.Picture) (image.Image, error) {
	if pic.BitDepth != 8 {
		return nil, errUnsupported
	}

	rect := image.Rect(0, 0, pic.Width, pic.Height)

	if pic.Layout == av1.LayoutI400 {
		if alpha != nil {
			return nil, errUnsupported
		}

		return &image.Gray{
			Pix:    pic.Data[0],
			Stride: pic.Stride[0],
			Rect:   rect,
		}, nil
	}

	y := &image.YCbCr{
		Y:              pic.Data[0],
		Cb:             pic.Data[1],
		Cr:             pic.Data[2],
		YStride:        pic.Stride[0],
		CStride:        pic.Stride[1],
		SubsampleRatio: ratio(pic.Layout),
		Rect:           rect,
	}
	if alpha == nil {
		return y, nil
	}
	if alpha.Layout != av1.LayoutI400 || alpha.BitDepth != 8 {
		return nil, errUnsupported
	}

	return &image.NYCbCrA{
		YCbCr:   *y,
		A:       alpha.Data[0],
		AStride: alpha.Stride[0],
	}, nil
}
