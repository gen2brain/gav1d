package avif

import (
	"errors"
	"image"
	"image/color"
	"math"
)

var errColorSpace = errors.New("avif: unsupported color space")

const (
	priBT709        = 1
	cicpUnspecified = 2
	priBT470M       = 4
	priBT470BG      = 5
	priBT601        = 6
	priSMPTE240     = 7
	priGenericFilm  = 8
	priBT2020       = 9
	priXYZ          = 10
	priSMPTE431     = 11
	priSMPTE432     = 12
	priEBU3213      = 22
)

const (
	trcBT709        = 1
	trcBT470M       = 4
	trcBT470BG      = 5
	trcBT601        = 6
	trcSMPTE240     = 7
	trcLinear       = 8
	trcLog100       = 9
	trcLog100Sqrt10 = 10
	trcIEC61966     = 11
	trcBT1361       = 12
	trcSRGB         = 13
	trcBT2020_10    = 14
	trcBT2020_12    = 15
	trcPQ           = 16
	trcSMPTE428     = 17
	trcHLG          = 18
)

const (
	sdrWhiteNits = 203.0
	pqMaxNits    = 10000.0
	hlgPeakNits  = 1000.0
)

func clamp01(v float64) float64 {
	return min(max(v, 0), 1)
}

func toLinear709(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 4.5*0.018053968510807:
		return v / 4.5
	case v < 1:
		return math.Pow((v+0.09929682680944)/1.09929682680944, 1/0.45)
	default:
		return 1
	}
}

func toGamma709(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 0.018053968510807:
		return v * 4.5
	case v < 1:
		return 1.09929682680944*math.Pow(v, 0.45) - 0.09929682680944
	default:
		return 1
	}
}

func toLinearSMPTE240(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 4*0.022821585529445:
		return v / 4
	case v < 1:
		return math.Pow((v+0.111572195921731)/1.111572195921731, 1/0.45)
	default:
		return 1
	}
}

func toGammaSMPTE240(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 0.022821585529445:
		return v * 4
	case v < 1:
		return 1.111572195921731*math.Pow(v, 0.45) - 0.111572195921731
	default:
		return 1
	}
}

func toLinearLog100(v float64) float64 {
	if v <= 0 {
		return 0.01 / 2
	}

	return math.Pow(10, 2*(min(v, 1)-1))
}

func toGammaLog100(v float64) float64 {
	if v <= 0.01 {
		return 0
	}

	return 1 + math.Log10(min(v, 1))/2
}

func toLinearLog100Sqrt10(v float64) float64 {
	if v <= 0 {
		return 0.00316227766 / 2
	}

	return math.Pow(10, 2.5*(min(v, 1)-1))
}

func toGammaLog100Sqrt10(v float64) float64 {
	if v <= 0.00316227766 {
		return 0
	}

	return 1 + math.Log10(min(v, 1))/2.5
}

func toLinearIEC61966(v float64) float64 {
	switch {
	case v < -4.5*0.018053968510807:
		return math.Pow((-v+0.09929682680944)/-1.09929682680944, 1/0.45)
	case v < 4.5*0.018053968510807:
		return v / 4.5
	default:
		return math.Pow((v+0.09929682680944)/1.09929682680944, 1/0.45)
	}
}

func toGammaIEC61966(v float64) float64 {
	switch {
	case v < -0.018053968510807:
		return -1.09929682680944*math.Pow(-v, 0.45) + 0.09929682680944
	case v < 0.018053968510807:
		return v * 4.5
	default:
		return 1.09929682680944*math.Pow(v, 0.45) - 0.09929682680944
	}
}

func toLinearBT1361(v float64) float64 {
	switch {
	case v < -0.25:
		return -0.25
	case v < 0:
		return math.Pow((v-0.02482420670236)/-0.27482420670236, 1/0.45) / -4
	case v < 4.5*0.018053968510807:
		return v / 4.5
	case v < 1:
		return math.Pow((v+0.09929682680944)/1.09929682680944, 1/0.45)
	default:
		return 1
	}
}

func toGammaBT1361(v float64) float64 {
	switch {
	case v < -0.25:
		return -0.25
	case v < 0:
		return -0.27482420670236*math.Pow(-4*v, 0.45) + 0.02482420670236
	case v < 0.018053968510807:
		return v * 4.5
	case v < 1:
		return 1.09929682680944*math.Pow(v, 0.45) - 0.09929682680944
	default:
		return 1
	}
}

func toLinearSRGB(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 12.92*0.0030412825601275209:
		return v / 12.92
	case v < 1:
		return math.Pow((v+0.0550107189475866)/1.0550107189475866, 2.4)
	default:
		return 1
	}
}

func toGammaSRGB(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v < 0.0030412825601275209:
		return v * 12.92
	case v < 1:
		return 1.0550107189475866*math.Pow(v, 1/2.4) - 0.0550107189475866
	default:
		return 1
	}
}

// toLinearPQ and toLinearHLG scale their result so that SDR white is 1, which
// is the extended SDR range the gain map math works in.
func toLinearPQ(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(v, 1/78.84375)
	num := max(p-0.8359375, 0)
	den := max(18.8515625-18.6875*p, math.SmallestNonzeroFloat32)

	return math.Pow(num/den, 1/0.1593017578125) * pqMaxNits / sdrWhiteNits
}

func toGammaPQ(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(clamp01(v*sdrWhiteNits/pqMaxNits), 0.1593017578125)

	return math.Pow(1+(0.1640625*p-0.1640625)/(1+18.6875*p), 78.84375)
}

func toLinearSMPTE428(v float64) float64 {
	return math.Pow(max(v, 0), 2.6) / 0.91655527974030934
}

func toGammaSMPTE428(v float64) float64 {
	return math.Pow(0.91655527974030934*max(v, 0), 1/2.6)
}

func toLinearHLG(v float64) float64 {
	if v < 0 {
		return 0
	}
	var linear float64
	if v <= 0.5 {
		linear = math.Pow(v*v/3, 1.2)
	} else {
		linear = math.Pow((math.Exp((v-0.55991073)/0.17883277)+0.28466892)/12, 1.2)
	}

	return linear * hlgPeakNits / sdrWhiteNits
}

func toGammaHLG(v float64) float64 {
	v = math.Pow(clamp01(v*sdrWhiteNits/hlgPeakNits), 1/1.2)
	switch {
	case v < 0:
		return 0
	case v <= 1.0/12:
		return math.Sqrt(3 * v)
	default:
		return 0.17883277*math.Log(12*v-0.28466892) + 0.55991073
	}
}

func gamma(e float64) func(float64) float64 {
	return func(v float64) float64 { return math.Pow(clamp01(v), e) }
}

// transferFuncs returns the pair converting between the transfer's encoding
// and linear light, defaulting to BT.709 as libavif does.
func transferFuncs(trc uint16) (toLinear, toGamma func(float64) float64) {
	switch trc {
	case trcBT470M:
		return gamma(2.2), gamma(1 / 2.2)
	case trcBT470BG:
		return gamma(2.8), gamma(1 / 2.8)
	case trcSMPTE240:
		return toLinearSMPTE240, toGammaSMPTE240
	case trcLinear:
		return clamp01, clamp01
	case trcLog100:
		return toLinearLog100, toGammaLog100
	case trcLog100Sqrt10:
		return toLinearLog100Sqrt10, toGammaLog100Sqrt10
	case trcIEC61966:
		return toLinearIEC61966, toGammaIEC61966
	case trcBT1361:
		return toLinearBT1361, toGammaBT1361
	case trcSRGB:
		return toLinearSRGB, toGammaSRGB
	case trcPQ:
		return toLinearPQ, toGammaPQ
	case trcSMPTE428:
		return toLinearSMPTE428, toGammaSMPTE428
	case trcHLG:
		return toLinearHLG, toGammaHLG
	}

	return toLinear709, toGamma709
}

// primariesValues returns rX, rY, gX, gY, bX, bY, wX, wY.
func primariesValues(cp uint16) [8]float64 {
	switch cp {
	case priBT470M:
		return [8]float64{0.67, 0.33, 0.21, 0.71, 0.14, 0.08, 0.310, 0.316}
	case priBT470BG:
		return [8]float64{0.64, 0.33, 0.29, 0.60, 0.15, 0.06, 0.3127, 0.3290}
	case priBT601, priSMPTE240:
		return [8]float64{0.630, 0.340, 0.310, 0.595, 0.155, 0.070, 0.3127, 0.3290}
	case priGenericFilm:
		return [8]float64{0.681, 0.319, 0.243, 0.692, 0.145, 0.049, 0.310, 0.316}
	case priBT2020:
		return [8]float64{0.708, 0.292, 0.170, 0.797, 0.131, 0.046, 0.3127, 0.3290}
	case priXYZ:
		return [8]float64{1, 0, 0, 1, 0, 0, 0.3333, 0.3333}
	case priSMPTE431:
		return [8]float64{0.680, 0.320, 0.265, 0.690, 0.150, 0.060, 0.314, 0.351}
	case priSMPTE432:
		return [8]float64{0.680, 0.320, 0.265, 0.690, 0.150, 0.060, 0.3127, 0.3290}
	case priEBU3213:
		return [8]float64{0.630, 0.340, 0.295, 0.605, 0.155, 0.077, 0.3127, 0.3290}
	}

	return [8]float64{0.64, 0.33, 0.3, 0.6, 0.15, 0.06, 0.3127, 0.329}
}

type mat3 [3][3]float64

func (m mat3) mul(n mat3) mat3 {
	var out mat3
	for i := range 3 {
		for j := range 3 {
			out[i][j] = m[i][0]*n[0][j] + m[i][1]*n[1][j] + m[i][2]*n[2][j]
		}
	}

	return out
}

func (m mat3) apply(v [3]float64) [3]float64 {
	return [3]float64{
		m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2],
		m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2],
		m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2],
	}
}

const matEpsilon = 1e-12

func (m mat3) inv() (mat3, bool) {
	det := m[0][0]*(m[1][1]*m[2][2]-m[2][1]*m[1][2]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	if math.Abs(det) < matEpsilon {
		return mat3{}, false
	}
	d := 1 / det

	return mat3{
		{
			(m[1][1]*m[2][2] - m[2][1]*m[1][2]) * d,
			(m[0][2]*m[2][1] - m[0][1]*m[2][2]) * d,
			(m[0][1]*m[1][2] - m[0][2]*m[1][1]) * d,
		},
		{
			(m[1][2]*m[2][0] - m[1][0]*m[2][2]) * d,
			(m[0][0]*m[2][2] - m[0][2]*m[2][0]) * d,
			(m[1][0]*m[0][2] - m[0][0]*m[1][2]) * d,
		},
		{
			(m[1][0]*m[2][1] - m[2][0]*m[1][1]) * d,
			(m[2][0]*m[0][1] - m[0][0]*m[2][1]) * d,
			(m[0][0]*m[1][1] - m[1][0]*m[0][1]) * d,
		},
	}, true
}

func diag(d [3]float64) mat3 {
	return mat3{{d[0], 0, 0}, {0, d[1], 0}, {0, 0, d[2]}}
}

// bradford adapts between whitepoints, and lmsD50 is the D50 whitepoint the
// primaries are adapted to.
var bradford = mat3{
	{0.8951, 0.2664, -0.1614},
	{-0.7502, 1.7135, 0.0367},
	{0.0389, -0.0685, 1.0296},
}

var lmsD50 = [3]float64{0.996284, 1.02043, 0.818644}

func rgbToXYZD50(cp uint16) (mat3, bool) {
	p := primariesValues(cp)
	if math.Abs(p[7]) < matEpsilon {
		return mat3{}, false
	}

	white := [3]float64{p[6] / p[7], 1, (1 - p[6] - p[7]) / p[7]}
	rgb := mat3{
		{p[0], p[2], p[4]},
		{p[1], p[3], p[5]},
		{1 - p[0] - p[1], 1 - p[2] - p[3], 1 - p[4] - p[5]},
	}
	rgbInv, ok := rgb.inv()
	if !ok {
		return mat3{}, false
	}
	rgbXYZ := rgb.mul(diag(rgbInv.apply(white)))

	lms := bradford.apply(white)
	for i, v := range lms {
		if math.Abs(v) < matEpsilon {
			return mat3{}, false
		}
		lms[i] = lmsD50[i] / v
	}
	bradfordInv, ok := bradford.inv()
	if !ok {
		return mat3{}, false
	}

	return bradfordInv.mul(diag(lms).mul(bradford)).mul(rgbXYZ), true
}

func rgbToRGBMatrix(src, dst uint16) (mat3, bool) {
	srcToXYZ, ok := rgbToXYZD50(src)
	if !ok {
		return mat3{}, false
	}
	dstToXYZ, ok := rgbToXYZD50(dst)
	if !ok {
		return mat3{}, false
	}
	xyzToDst, ok := dstToXYZ.inv()
	if !ok {
		return mat3{}, false
	}

	return xyzToDst.mul(srcToXYZ), true
}

// weight is how much of the gain map to apply for a display with the given
// headroom, in [-1, 1]. It is zero when the two renditions claim the same
// headroom, which the format does not define.
func (g *GainMap) weight(headroom float64) float64 {
	base := g.BaseHDRHeadroom.Float()
	alt := g.AlternateHDRHeadroom.Float()
	if base == alt {
		return 0
	}

	w := min(max((headroom-base)/(alt-base), 0), 1)
	if alt < base {
		return -w
	}

	return w
}

type pixelReader func(x, y int) [4]float64

func newPixelReader(img image.Image) pixelReader {
	b := img.Bounds()

	switch src := img.(type) {
	case *image.NRGBA:
		return func(x, y int) [4]float64 {
			i := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			p := src.Pix[i : i+4 : i+4]

			return [4]float64{float64(p[0]) / 255, float64(p[1]) / 255, float64(p[2]) / 255, float64(p[3]) / 255}
		}

	case *image.NRGBA64:
		return func(x, y int) [4]float64 {
			i := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			p := src.Pix[i : i+8 : i+8]

			return [4]float64{
				float64(uint16(p[0])<<8|uint16(p[1])) / 65535,
				float64(uint16(p[2])<<8|uint16(p[3])) / 65535,
				float64(uint16(p[4])<<8|uint16(p[5])) / 65535,
				float64(uint16(p[6])<<8|uint16(p[7])) / 65535,
			}
		}
	}

	return func(x, y int) [4]float64 {
		c := color.NRGBA64Model.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA64)

		return [4]float64{
			float64(c.R) / 65535, float64(c.G) / 65535,
			float64(c.B) / 65535, float64(c.A) / 65535,
		}
	}
}

// sampleGain reads the gain map at the base image's resolution, interpolating
// when it was stored smaller.
func sampleGain(read pixelReader, gw, gh, w, h, x, y int) [4]float64 {
	if gw == w && gh == h {
		return read(x, y)
	}

	fx := (float64(x)+0.5)*float64(gw)/float64(w) - 0.5
	fy := (float64(y)+0.5)*float64(gh)/float64(h) - 0.5
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	dx, dy := fx-float64(x0), fy-float64(y0)

	clampX := func(v int) int { return min(max(v, 0), gw-1) }
	clampY := func(v int) int { return min(max(v, 0), gh-1) }

	p00 := read(clampX(x0), clampY(y0))
	p10 := read(clampX(x0+1), clampY(y0))
	p01 := read(clampX(x0), clampY(y0+1))
	p11 := read(clampX(x0+1), clampY(y0+1))

	var out [4]float64
	for c := range out {
		top := p00[c] + (p10[c]-p00[c])*dx
		bot := p01[c] + (p11[c]-p01[c])*dx
		out[c] = top + (bot-top)*dy
	}

	return out
}

// ApplyGainMap tone maps base toward the HDR rendition g describes, for a
// display whose headroom is the given number of stops above SDR white: zero
// leaves the base image alone and log2(peak/white) reaches the alternate
// rendition. baseColor describes base and out the image to produce. The
// result keeps the base image's pixel format, and is base itself when there
// is nothing to do.
//
// The gain applies in the color space the metadata names, converting
// primaries either side when they differ, so out.Transfer decides what the
// result can hold: an SDR curve clips the highlights a gain map opens up
// where PQ or HLG keeps them. A gain map stored smaller than the base image
// is interpolated.
func ApplyGainMap(base image.Image, baseColor ColorInfo, g *GainMap, headroom float64, out ColorInfo) (image.Image, error) {
	if base == nil || g == nil || g.Image == nil {
		return nil, ErrInvalid
	}
	if headroom < 0 || math.IsNaN(headroom) {
		return nil, ErrInvalid
	}

	b := base.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, ErrInvalid
	}

	weight := g.weight(headroom)
	if weight == 0 && out.Transfer == baseColor.Transfer && out.Primaries == baseColor.Primaries {
		return base, nil
	}

	mathPrimaries := baseColor.Primaries
	if !g.UseBaseColorSpace && g.Alt.Primaries != cicpUnspecified {
		mathPrimaries = g.Alt.Primaries
	}

	var inMat, outMat mat3
	var needIn, needOut bool
	var ok bool
	if weight == 0 {
		mathPrimaries = baseColor.Primaries
	}
	if mathPrimaries != baseColor.Primaries {
		needIn = true
		if inMat, ok = rgbToRGBMatrix(baseColor.Primaries, mathPrimaries); !ok {
			return nil, errColorSpace
		}
	}
	if mathPrimaries != out.Primaries {
		needOut = true
		if outMat, ok = rgbToRGBMatrix(mathPrimaries, out.Primaries); !ok {
			return nil, errColorSpace
		}
	}

	toLinear, _ := transferFuncs(baseColor.Transfer)
	_, toGamma := transferFuncs(out.Transfer)

	var gammaInv, gmMin, gmMax, baseOff, altOff [3]float64
	for c := range 3 {
		gammaInv[c] = 1 / g.Gamma[c].Float()
		gmMin[c] = g.Min[c].Float()
		gmMax[c] = g.Max[c].Float()
		baseOff[c] = g.BaseOffset[c].Float()
		altOff[c] = g.AlternateOffset[c].Float()
	}

	readBase := newPixelReader(base)
	gb := g.Image.Bounds()
	readGain := newPixelReader(g.Image)

	dst, write := newToneMapped(base, w, h)

	for y := range h {
		for x := range w {
			px := readBase(x, y)
			alpha := px[3]

			var lin [3]float64
			for c := range 3 {
				lin[c] = toLinear(px[c])
			}
			if needIn {
				lin = inMat.apply(lin)
			}

			if weight != 0 {
				gm := sampleGain(readGain, gb.Dx(), gb.Dy(), w, h, x, y)
				for c := range 3 {
					log2Gain := gmMin[c] + (gmMax[c]-gmMin[c])*math.Pow(gm[c], gammaInv[c])
					lin[c] = (lin[c]+baseOff[c])*math.Exp2(log2Gain*weight) - altOff[c]
				}
			}
			if needOut {
				lin = outMat.apply(lin)
			}

			var px2 [4]float64
			for c := range 3 {
				px2[c] = clamp01(toGamma(lin[c]))
			}
			px2[3] = alpha
			write(x, y, px2)
		}
	}

	return dst, nil
}

// newToneMapped allocates the output, which keeps the base image's depth.
func newToneMapped(base image.Image, w, h int) (image.Image, func(x, y int, px [4]float64)) {
	if _, ok := base.(*image.NRGBA); ok {
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))

		return dst, func(x, y int, px [4]float64) {
			i := dst.PixOffset(x, y)
			p := dst.Pix[i : i+4 : i+4]
			for c := range 4 {
				p[c] = uint8(math.Round(px[c] * 255))
			}
		}
	}

	dst := image.NewNRGBA64(image.Rect(0, 0, w, h))

	return dst, func(x, y int, px [4]float64) {
		i := dst.PixOffset(x, y)
		p := dst.Pix[i : i+8 : i+8]
		for c := range 4 {
			v := uint16(math.Round(px[c] * 65535))
			p[c*2], p[c*2+1] = uint8(v>>8), uint8(v)
		}
	}
}
