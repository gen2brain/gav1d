package avif

import (
	"errors"
	"image"
	"image/color"

	"github.com/gen2brain/gav1d/av1"
)

// ErrUnsupported is returned for a file this package cannot render but which
// is otherwise well formed: an essential property it does not implement, or a
// sample format it has no conversion for. A caller that has another decoder
// to fall back on should test for this one rather than [ErrInvalid].
var ErrUnsupported = errors.New("avif: unsupported image")

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
// aliasesPicture reports whether img shares memory with the decoded planes, in
// which case the picture has to outlive it and must not be released.
func aliasesPicture(img image.Image) bool {
	switch img.(type) {
	case *image.YCbCr, *image.NYCbCrA, *image.Gray:
		return true
	}

	return false
}

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

	cs := newColorState(pic, f.colorInfo(it, pic), outDepth)
	if cs.unsupportedSpace {
		return nil, ErrUnsupported
	}

	var as *colorState
	if alpha != nil {
		if alpha.Layout != av1.LayoutI400 {
			return nil, ErrUnsupported
		}
		as = alphaState(alpha, outDepth, alpha.FullRange)
	}

	rect := image.Rect(0, 0, pic.Width, pic.Height)
	w := pic.Width

	cs.prepare(w)
	if cs.fastRow(outDepth) {
		cs.consts = cs.rowConsts()
		if outDepth == 8 {
			cs.row = rowFn(cs.ssHor)
		} else {
			cs.row16 = rowFn16(cs.ssHor)
		}
	}
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
		if cs.row != nil {
			ab := make([]uint8, w)
			for i := range ab {
				ab[i] = 0xff
			}
			for y := range pic.Height {
				if as != nil {
					as.alphaRow(alpha, y, w, aRow)
					for x, v := range aRow {
						ab[x] = uint8(v)
					}
				}
				u0, u1, v0, v1 := cs.rowPlanes(pic, y, w)
				cs.row(dst.Pix[y*dst.Stride:], cs.lumaRow(pic, y, w),
					u0, u1, v0, v1, ab, w, &cs.consts)
			}

			return dst, nil
		}

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
	if cs.row16 != nil {
		for y := range pic.Height {
			if as != nil {
				as.alphaRow(alpha, y, w, aRow)
			}
			u0, u1, v0, v1 := cs.rowPlanes(pic, y, w)
			cs.row16(dst.Pix[y*dst.Stride:], cs.lumaRow(pic, y, w),
				u0, u1, v0, v1, aRow, w, &cs.consts)
		}

		return dst, nil
	}

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
