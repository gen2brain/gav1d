package avif

import (
	"image"
	"io"

	"github.com/gen2brain/gav1d/av1"
)

const alphaURN = "urn:mpeg:mpegB:cicp:systems:auxiliary:alpha"

type file struct {
	data  []byte
	meta  *metaBox
	movie *movie
}

func parse(data []byte) (*file, error) {
	f := &file{data: data}
	var brandOK bool

	err := eachBox(data, func(typ string, b []byte) error {
		switch typ {
		case "ftyp":
			r := &reader{b: b}
			major := r.str4()
			r.u32()
			brands := []string{major}
			for r.remaining() >= 4 {
				brands = append(brands, r.str4())
			}
			for _, br := range brands {
				switch br {
				case "avif", "avis", "mif1", "msf1", "miaf":
					brandOK = true
				}
			}

		case "meta":
			m, err := parseMeta(b)
			if err != nil {
				return err
			}
			f.meta = m

		case "moov":
			mv, err := parseMoov(b)
			if err != nil {
				return err
			}
			f.movie = mv
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if !brandOK || f.meta == nil {
		return nil, errInvalid
	}

	return f, nil
}

// primary returns the primary item, falling back to the first av01 or grid item.
func (f *file) primary() *item {
	m := f.meta
	if it := m.items[m.primary]; it != nil && (it.typ == "av01" || it.typ == "grid") {
		return it
	}
	for _, id := range m.order {
		if it := m.items[id]; it.typ == "av01" || it.typ == "grid" {
			return it
		}
	}

	return nil
}

// alphaFor finds the auxiliary item carrying alpha for id.
func (f *file) alphaFor(id uint32) *item {
	m := f.meta
	for _, r := range m.refs {
		if r.typ != "auxl" || len(r.to) == 0 || r.to[0] != id {
			continue
		}
		it := m.items[r.from]
		if it == nil {
			continue
		}
		if p := m.prop(it, "auxC"); p != nil && p.auxC == alphaURN {
			return it
		}
	}

	return nil
}

func (f *file) dimensions(it *item) (int, int, error) {
	p := f.meta.prop(it, "ispe")
	if p == nil {
		return 0, 0, errInvalid
	}
	if p.w == 0 || p.h == 0 || p.w > 1<<20 || p.h > 1<<20 {
		return 0, 0, errInvalid
	}

	return int(p.w), int(p.h), nil
}

// decodeItem decodes a single av01 item to a picture.
func (f *file) decodeItem(it *item) (*av1.Picture, error) {
	data, err := f.meta.data(it, f.data)
	if err != nil {
		return nil, err
	}

	var d av1.Decoder
	if p := f.meta.prop(it, "a1op"); p != nil {
		d.OperatingPoint = int(p.opIdx)
	}
	pics, err := d.DecodeOBUs(data)
	if err != nil {
		return nil, err
	}
	if len(pics) == 0 {
		return nil, errInvalid
	}

	return pics[len(pics)-1], nil
}

type gridInfo struct {
	rows, cols int
	w, h       int
}

func parseGrid(b []byte) (gridInfo, error) {
	r := &reader{b: b}
	r.u8()
	flags := r.u8()
	var g gridInfo
	g.rows = int(r.u8()) + 1
	g.cols = int(r.u8()) + 1
	if flags&1 != 0 {
		g.w, g.h = int(r.u32()), int(r.u32())
	} else {
		g.w, g.h = int(r.u16()), int(r.u16())
	}
	if r.err || g.w == 0 || g.h == 0 {
		return g, errInvalid
	}

	return g, nil
}

// decodeGrid decodes a grid item's dimg tiles and stitches them row-major.
func (f *file) gridOf(it *item) (gridInfo, []uint32, error) {
	data, err := f.meta.data(it, f.data)
	if err != nil {
		return gridInfo{}, nil, err
	}
	g, err := parseGrid(data)
	if err != nil {
		return gridInfo{}, nil, err
	}
	tiles := f.meta.refsTo("dimg", it.id)
	if len(tiles) != g.rows*g.cols {
		return gridInfo{}, nil, errInvalid
	}

	return g, tiles, nil
}

func (f *file) decodeGrid(it *item) (*av1.Picture, error) {
	g, tiles, err := f.gridOf(it)
	if err != nil {
		return nil, err
	}

	return f.decodeTiles(g, tiles)
}

func (f *file) decodeTiles(g gridInfo, tiles []uint32) (*av1.Picture, error) {
	var err error
	pics := make([]*av1.Picture, len(tiles))
	for i, id := range tiles {
		t := f.meta.items[id]
		if t == nil || t.typ != "av01" {
			return nil, errInvalid
		}
		if pics[i], err = f.decodeItem(t); err != nil {
			return nil, err
		}
	}

	first := pics[0]
	tw, th := first.Width, first.Height
	for _, p := range pics {
		if p.Width != tw || p.Height != th || p.Layout != first.Layout ||
			p.BitDepth != first.BitDepth {
			return nil, errInvalid
		}
	}
	if tw*g.cols < g.w || th*g.rows < g.h {
		return nil, errInvalid
	}

	return stitch(pics, g, tw, th), nil
}

func stitch(pics []*av1.Picture, g gridInfo, tw, th int) *av1.Picture {
	first := pics[0]
	ssHor, ssVer := subsampling(first.Layout)
	bpp := 1
	if first.BitDepth > 8 {
		bpp = 2
	}

	out := &av1.Picture{
		Width:          g.w,
		Height:         g.h,
		Layout:         first.Layout,
		BitDepth:       first.BitDepth,
		ColorPrimaries: first.ColorPrimaries,
		Transfer:       first.Transfer,
		MatrixCoeffs:   first.MatrixCoeffs,
		FullRange:      first.FullRange,
	}
	for pl := range 3 {
		if first.Data[pl] == nil {
			continue
		}
		pw, ph := g.w, g.h
		if pl != 0 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		out.Stride[pl] = pw * bpp
		out.Data[pl] = make([]byte, out.Stride[pl]*ph)
	}

	for i, p := range pics {
		row, col := i/g.cols, i%g.cols
		for pl := range 3 {
			if out.Data[pl] == nil {
				continue
			}
			sx, sy := col*tw, row*th
			cw, ch := tw, th
			ow, oh := g.w, g.h
			if pl != 0 {
				sx, sy = sx>>ssHor, sy>>ssVer
				cw, ch = cw>>ssHor, ch>>ssVer
				ow, oh = (ow+ssHor)>>ssHor, (oh+ssVer)>>ssVer
			}
			cw = min(cw, ow-sx)
			ch = min(ch, oh-sy)
			if cw <= 0 || ch <= 0 {
				continue
			}
			for y := range ch {
				dst := (sy+y)*out.Stride[pl] + sx*bpp
				src := y * p.Stride[pl]
				copy(out.Data[pl][dst:dst+cw*bpp], p.Data[pl][src:src+cw*bpp])
			}
		}
	}

	return out
}

func subsampling(layout int) (int, int) {
	switch layout {
	case av1.LayoutI420:
		return 1, 1
	case av1.LayoutI422:
		return 1, 0
	default:
		return 0, 0
	}
}

// gridAlpha assembles alpha from items attached to a grid's tiles.
func (f *file) gridAlpha(it *item) (*av1.Picture, error) {
	g, tiles, err := f.gridOf(it)
	if err != nil {
		return nil, err
	}

	ids := make([]uint32, len(tiles))
	for i, id := range tiles {
		a := f.alphaFor(id)
		if a == nil {
			return nil, nil
		}
		ids[i] = a.id
	}

	return f.decodeTiles(g, ids)
}

func (f *file) decodePrimary() (*av1.Picture, *av1.Picture, error) {
	it := f.primary()
	if it == nil {
		return nil, nil, errInvalid
	}

	var pic *av1.Picture
	var err error
	if it.typ == "grid" {
		pic, err = f.decodeGrid(it)
	} else {
		pic, err = f.decodeItem(it)
	}
	if err != nil {
		return nil, nil, err
	}

	w, h, err := f.dimensions(it)
	if err != nil {
		return nil, nil, err
	}
	if w > pic.Width || h > pic.Height {
		return nil, nil, errInvalid
	}
	pic.Width, pic.Height = w, h

	var alpha *av1.Picture
	if a := f.alphaFor(it.id); a != nil {
		if a.typ == "grid" {
			alpha, err = f.decodeGrid(a)
		} else {
			alpha, err = f.decodeItem(a)
		}
		if err != nil {
			return nil, nil, err
		}
	} else if it.typ == "grid" {
		alpha, err = f.gridAlpha(it)
		if err != nil {
			return nil, nil, err
		}
	}
	if alpha != nil {
		if alpha.Width < w || alpha.Height < h {
			return nil, nil, errInvalid
		}
		alpha.Width, alpha.Height = w, h
	}

	return pic, alpha, nil
}

// AVIF holds the images of an AVIF file, which may be an image sequence.
type AVIF struct {
	// Image holds the decoded frames, *image.NRGBA or *image.NRGBA64.
	Image []image.Image
	// Delay holds each frame's duration in seconds.
	Delay []float64
	// LoopCount is how many times the animation repeats, zero for forever.
	LoopCount int
}

// Options are the decoding parameters.
type Options struct {
	// AutoRotate applies the clap/irot/imir transforms to the decoded image.
	AutoRotate bool
}

// Decode reads an AVIF image as *image.NRGBA, or *image.NRGBA64 above 8 bits.
func Decode(r io.Reader, opts ...Options) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f, err := parse(data)
	if err != nil {
		return nil, err
	}

	return f.decodeStill(options(opts))
}

// DecodeAll reads every frame of an AVIF file. A still image gives one frame.
func DecodeAll(r io.Reader, opts ...Options) (*AVIF, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f, err := parse(data)
	if err != nil {
		return nil, err
	}
	o := options(opts)

	if f.movie != nil {
		if a, err := f.decodeSequence(o); err == nil && a != nil {
			return a, nil
		}
	}

	img, err := f.decodeStill(o)
	if err != nil {
		return nil, err
	}

	return &AVIF{Image: []image.Image{img}, Delay: []float64{0}}, nil
}

// DecodeConfig reads the dimensions and color model without decoding pixels.
func DecodeConfig(r io.Reader) (image.Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	f, err := parse(data)
	if err != nil {
		return image.Config{}, err
	}
	it := f.primary()
	if it == nil {
		return image.Config{}, errInvalid
	}
	w, h, err := f.dimensions(it)
	if err != nil {
		return image.Config{}, err
	}

	return image.Config{Width: w, Height: h, ColorModel: colorModelFor(f, it)}, nil
}

func options(opts []Options) Options {
	if len(opts) > 0 {
		return opts[0]
	}

	return Options{}
}

func (f *file) decodeStill(o Options) (image.Image, error) {
	pic, alpha, err := f.decodePrimary()
	if err != nil {
		return nil, err
	}
	it := f.primary()
	img, err := toImage(f, it, pic, alpha)
	if err != nil {
		return nil, err
	}
	if !o.AutoRotate {
		return img, nil
	}

	return f.applyTransforms(img, it)
}

func decodeWrapper(r io.Reader) (image.Image, error) {
	return Decode(r)
}

func init() {
	image.RegisterFormat("avif", "????ftypavif", decodeWrapper, DecodeConfig)
	image.RegisterFormat("avif", "????ftypavis", decodeWrapper, DecodeConfig)
	image.RegisterFormat("avif", "????ftypmif1", decodeWrapper, DecodeConfig)
}
