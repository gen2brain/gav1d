package av1

import (
	"math/bits"
	"unsafe"
)

func splatDc[P pixel](dst []P, dstOff, stride, width, height int, dc int32) {
	for y := range height {
		row := dstOff + y*stride
		for x := range width {
			dst[row+x] = P(dc)
		}
	}
}

// cflPredict8Asm and cflPredict16Asm carry the whole block, so the dispatch is
// a package variable rather than a dsp entry: the four CFL modes differ only in
// the DC they resolve first.
var (
	cflPredict8Asm  func(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)
	cflPredict16Asm func(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)
)

func cflPredict[P pixel](dst []P, dstOff, stride, width, height int, dc int32,
	ac []int16, alpha int32, bitdepthMax int32,
) {
	var z P
	if unsafe.Sizeof(z) == 1 && cflPredict8Asm != nil {
		cflPredict8Asm((*uint8)(unsafe.Pointer(&dst[dstOff])), stride, width, height,
			dc, &ac[0], alpha, bitdepthMax)

		return
	}
	if unsafe.Sizeof(z) == 2 && cflPredict16Asm != nil {
		cflPredict16Asm((*uint16)(unsafe.Pointer(&dst[dstOff])), stride, width, height,
			dc, &ac[0], alpha, bitdepthMax)

		return
	}

	acOff := 0
	for y := range height {
		row := dstOff + y*stride
		for x := range width {
			diff := alpha * int32(ac[acOff+x])
			d := diff
			if d < 0 {
				d = -d
			}
			dst[row+x] = P(clip(dc+applySign((d+32)>>6, diff), 0, bitdepthMax))
		}
		acOff += width
	}
}

func dcGenTop[P pixel](tl []P, tlOff, width int) int32 {
	dc := uint32(width >> 1)
	for i := range width {
		dc += uint32(tl[tlOff+1+i])
	}

	return int32(dc >> bits.TrailingZeros32(uint32(width)))
}

func dcGenLeft[P pixel](tl []P, tlOff, height int) int32 {
	dc := uint32(height >> 1)
	for i := range height {
		dc += uint32(tl[tlOff-(1+i)])
	}

	return int32(dc >> bits.TrailingZeros32(uint32(height)))
}

func sumPixels[P pixel](a []P) uint32 {
	var s uint32
	for _, v := range a {
		s += uint32(v)
	}

	return s
}

func dcGen[P pixel](tl []P, tlOff, width, height int, bitdepthMax int32) int32 {
	dc := uint32((width+height)>>1) + sumPixels(tl[tlOff+1:tlOff+1+width]) +
		sumPixels(tl[tlOff-height:tlOff])
	dc >>= bits.TrailingZeros32(uint32(width + height))

	if width != height {
		var mul1x2, mul1x4, baseShift uint32
		if bitdepthMax == 0xff {
			mul1x2, mul1x4, baseShift = 0x5556, 0x3334, 16
		} else {
			mul1x2, mul1x4, baseShift = 0xAAAB, 0x6667, 17
		}
		if width > height*2 || height > width*2 {
			dc *= mul1x4
		} else {
			dc *= mul1x2
		}
		dc >>= baseShift
	}

	return int32(dc)
}

func ipredDcTop[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc(dst, dstOff, stride, width, height, dcGenTop(tl, tlOff, width))
}

func ipredDcLeft[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc(dst, dstOff, stride, width, height, dcGenLeft(tl, tlOff, height))
}

func ipredDc[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc(dst, dstOff, stride, width, height, dcGen(tl, tlOff, width, height, bitdepthMax))
}

func ipredDc128[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	splatDc(dst, dstOff, stride, width, height, (bitdepthMax+1)>>1)
}

func ipredCflTop[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height int, ac []int16, alpha int32, bitdepthMax int32,
) {
	cflPredict(dst, dstOff, stride, width, height, dcGenTop(tl, tlOff, width), ac, alpha, bitdepthMax)
}

func ipredCflLeft[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height int, ac []int16, alpha int32, bitdepthMax int32,
) {
	cflPredict(dst, dstOff, stride, width, height, dcGenLeft(tl, tlOff, height), ac, alpha, bitdepthMax)
}

func ipredCfl[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height int, ac []int16, alpha int32, bitdepthMax int32,
) {
	dc := dcGen(tl, tlOff, width, height, bitdepthMax)
	cflPredict(dst, dstOff, stride, width, height, dc, ac, alpha, bitdepthMax)
}

func ipredCfl128[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height int, ac []int16, alpha int32, bitdepthMax int32,
) {
	cflPredict(dst, dstOff, stride, width, height, (bitdepthMax+1)>>1, ac, alpha, bitdepthMax)
}

func ipredV[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	for y := range height {
		copy(dst[dstOff+y*stride:][:width], tl[tlOff+1:][:width])
	}
}

func ipredH[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	for y := range height {
		row := dstOff + y*stride
		v := tl[tlOff-(1+y)]
		for x := range width {
			dst[row+x] = v
		}
	}
}

func ipredPaeth[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	topleft := int32(tl[tlOff])
	for y := range height {
		row := dstOff + y*stride
		left := int32(tl[tlOff-(y+1)])
		for x := range width {
			top := int32(tl[tlOff+1+x])
			base := left + top - topleft
			ldiff := abs32(left - base)
			tdiff := abs32(top - base)
			tldiff := abs32(topleft - base)

			switch {
			case ldiff <= tdiff && ldiff <= tldiff:
				dst[row+x] = P(left)
			case tdiff <= tldiff:
				dst[row+x] = P(top)
			default:
				dst[row+x] = P(topleft)
			}
		}
	}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}

	return v
}

func ipredSmooth[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	weightsHor := smWeights[width:]
	weightsVer := smWeights[height:]
	right, bottom := int32(tl[tlOff+width]), int32(tl[tlOff-height])

	for y := range height {
		row := dstOff + y*stride
		for x := range width {
			pred := int32(weightsVer[y])*int32(tl[tlOff+1+x]) +
				(256-int32(weightsVer[y]))*bottom +
				int32(weightsHor[x])*int32(tl[tlOff-(1+y)]) +
				(256-int32(weightsHor[x]))*right
			dst[row+x] = P((pred + 256) >> 9)
		}
	}
}

func ipredSmoothV[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	weightsVer := smWeights[height:]
	bottom := int32(tl[tlOff-height])

	for y := range height {
		row := dstOff + y*stride
		for x := range width {
			pred := int32(weightsVer[y])*int32(tl[tlOff+1+x]) +
				(256-int32(weightsVer[y]))*bottom
			dst[row+x] = P((pred + 128) >> 8)
		}
	}
}

func ipredSmoothH[P pixel](dst []P, dstOff, stride int, tl []P, tlOff,
	width, height, a, maxWidth, maxHeight int, bitdepthMax int32,
) {
	weightsHor := smWeights[width:]
	right := int32(tl[tlOff+width])

	for y := range height {
		row := dstOff + y*stride
		for x := range width {
			pred := int32(weightsHor[x])*int32(tl[tlOff-(y+1)]) +
				(256-int32(weightsHor[x]))*right
			dst[row+x] = P((pred + 128) >> 8)
		}
	}
}

func getFilterStrength(wh, angle, isSm int) int {
	if isSm != 0 {
		switch {
		case wh <= 8:
			if angle >= 64 {
				return 2
			}
			if angle >= 40 {
				return 1
			}
		case wh <= 16:
			if angle >= 48 {
				return 2
			}
			if angle >= 20 {
				return 1
			}
		case wh <= 24:
			if angle >= 4 {
				return 3
			}
		default:
			return 3
		}

		return 0
	}

	switch {
	case wh <= 8:
		if angle >= 56 {
			return 1
		}
	case wh <= 16:
		if angle >= 40 {
			return 1
		}
	case wh <= 24:
		if angle >= 32 {
			return 3
		}
		if angle >= 16 {
			return 2
		}
		if angle >= 8 {
			return 1
		}
	case wh <= 32:
		if angle >= 32 {
			return 3
		}
		if angle >= 4 {
			return 2
		}

		return 1
	default:
		return 3
	}

	return 0
}

var filterEdgeKernel = [3][5]uint8{
	{0, 4, 8, 4, 0},
	{0, 5, 6, 5, 0},
	{2, 4, 4, 4, 2},
}

func filterEdgeRun[P pixel](out []P, outOff int, in []P, inOff, n int, k *[5]uint8) {
	for i := range n {
		s := int(in[inOff+i])*int(k[0]) + int(in[inOff+i+1])*int(k[1]) +
			int(in[inOff+i+2])*int(k[2]) + int(in[inOff+i+3])*int(k[3]) +
			int(in[inOff+i+4])*int(k[4])
		out[outOff+i] = P((s + 8) >> 4)
	}
}

func filterEdge[P pixel](out []P, outOff, sz, limFrom, limTo int,
	in []P, inOff, from, to, strength int,
) {
	k := &filterEdgeKernel[strength-1]

	i := 0
	for ; i < min(sz, limFrom); i++ {
		out[outOff+i] = in[inOff+clip(i, from, to-1)]
	}

	end := min(limTo, sz)
	lo := min(max(i, from+2), end)
	hi := max(min(end, to-2), lo)
	if hi-lo < 8 {
		lo, hi = end, end
	}

	for ; i < lo; i++ {
		s := 0
		for j := range 5 {
			s += int(in[inOff+clip(i-2+j, from, to-1)]) * int(k[j])
		}
		out[outOff+i] = P((s + 8) >> 4)
	}
	if hi > i {
		filterEdgeRun(out, outOff+i, in, inOff+i-2, hi-i, k)
		i = hi
	}
	for ; i < end; i++ {
		s := 0
		for j := range 5 {
			s += int(in[inOff+clip(i-2+j, from, to-1)]) * int(k[j])
		}
		out[outOff+i] = P((s + 8) >> 4)
	}

	for ; i < sz; i++ {
		out[outOff+i] = in[inOff+clip(i, from, to-1)]
	}
}

func getUpsample(wh, angle, isSm int) int {
	if angle < 40 && wh <= 16>>isSm {
		return 1
	}

	return 0
}

var upsampleEdgeKernel = [4]int8{-1, 9, 9, -1}

func upsampleEdge[P pixel](out []P, outOff, hsz int, in []P, inOff, from, to int,
	bitdepthMax int32,
) {
	i := 0
	for ; i < hsz-1; i++ {
		out[outOff+i*2] = in[inOff+clip(i, from, to-1)]

		s := int32(0)
		for j := range 4 {
			s += int32(in[inOff+clip(i+j-1, from, to-1)]) * int32(upsampleEdgeKernel[j])
		}
		out[outOff+i*2+1] = P(clip((s+8)>>4, 0, bitdepthMax))
	}
	out[outOff+i*2] = in[inOff+clip(i, from, to-1)]
}

func z1Edge[P pixel](tlIn []P, tlInOff, width, height, angle int,
	bitdepthMax int32, out []P,
) (top []P, topOff, maxBaseX, dx, baseInc int) {
	isSm := (angle >> 9) & 1
	enableEdgeFilter := angle >> 10
	angle &= 511
	dx = int(drIntraDerivative[angle>>1])

	upsampleAbove := 0
	if enableEdgeFilter != 0 {
		upsampleAbove = getUpsample(width+height, 90-angle, isSm)
	}
	if upsampleAbove != 0 {
		upsampleEdge(out, 0, width+height, tlIn, tlInOff+1, -1,
			width+min(width, height), bitdepthMax)
		top, topOff = out, 0
		maxBaseX = 2*(width+height) - 2
		dx <<= 1
	} else {
		filterStrength := 0
		if enableEdgeFilter != 0 {
			filterStrength = getFilterStrength(width+height, 90-angle, isSm)
		}
		if filterStrength != 0 {
			filterEdge(out, 0, width+height, 0, width+height,
				tlIn, tlInOff+1, -1, width+min(width, height), filterStrength)
			top, topOff = out, 0
			maxBaseX = width + height - 1
		} else {
			top, topOff = tlIn, tlInOff+1
			maxBaseX = width + min(width, height) - 1
		}
	}

	return top, topOff, maxBaseX, dx, 1 + upsampleAbove
}

func z1Rows(t, dx, height int) int {
	if t < 0 {
		return 0
	}

	return min(height, ((t+1)*64-1)/dx)
}

func z1Sample[P pixel](dst []P, dstOff, stride int, top []P, topOff,
	width, height, dx, baseInc, maxBaseX, xpos int,
) {
	for y := range height {
		row := dstOff + y*stride
		frac := int32(xpos & 0x3E)

		base := xpos >> 6
		for x := 0; x < width; x, base = x+1, base+baseInc {
			if base < maxBaseX {
				v := int32(top[topOff+base])*(64-frac) + int32(top[topOff+base+1])*frac
				dst[row+x] = P((v + 32) >> 6)
			} else {
				fill := top[topOff+maxBaseX]
				for ; x < width; x++ {
					dst[row+x] = fill
				}

				break
			}
		}
		xpos += dx
	}
}

func ipredZ1[P pixel](dst []P, dstOff, stride int, tlIn []P, tlInOff,
	width, height, angle, maxWidth, maxHeight int, bitdepthMax int32,
) {
	var topOut [64 + 64]P
	top, topOff, maxBaseX, dx, baseInc := z1Edge(tlIn, tlInOff, width, height,
		angle, bitdepthMax, topOut[:])

	z1Sample(dst, dstOff, stride, top, topOff, width, height, dx, baseInc, maxBaseX, dx)
}

func ipredZ2[P pixel](dst []P, dstOff, stride int, tlIn []P, tlInOff,
	width, height, angle, maxWidth, maxHeight int, bitdepthMax int32,
) {
	isSm := (angle >> 9) & 1
	enableEdgeFilter := angle >> 10
	angle &= 511
	dy := int(drIntraDerivative[(angle-90)>>1])
	dx := int(drIntraDerivative[(180-angle)>>1])

	upsampleLeft, upsampleAbove := 0, 0
	if enableEdgeFilter != 0 {
		upsampleLeft = getUpsample(width+height, 180-angle, isSm)
		upsampleAbove = getUpsample(width+height, angle-90, isSm)
	}

	var edge [64 + 64 + 1]P
	const tlOff = 64

	if upsampleAbove != 0 {
		upsampleEdge(edge[:], tlOff, width+1, tlIn, tlInOff, 0, width+1, bitdepthMax)
		dx <<= 1
	} else {
		filterStrength := 0
		if enableEdgeFilter != 0 {
			filterStrength = getFilterStrength(width+height, angle-90, isSm)
		}
		if filterStrength != 0 {
			filterEdge(edge[:], tlOff+1, width, 0, maxWidth,
				tlIn, tlInOff+1, -1, width, filterStrength)
		} else {
			copy(edge[tlOff+1:][:width], tlIn[tlInOff+1:][:width])
		}
	}
	if upsampleLeft != 0 {
		upsampleEdge(edge[:], tlOff-height*2, height+1, tlIn, tlInOff-height,
			0, height+1, bitdepthMax)
		dy <<= 1
	} else {
		filterStrength := 0
		if enableEdgeFilter != 0 {
			filterStrength = getFilterStrength(width+height, 180-angle, isSm)
		}
		if filterStrength != 0 {
			filterEdge(edge[:], tlOff-height, height, height-maxHeight, height,
				tlIn, tlInOff-height, 0, height+1, filterStrength)
		} else {
			copy(edge[tlOff-height:][:height], tlIn[tlInOff-height:][:height])
		}
	}
	edge[tlOff] = tlIn[tlInOff]

	baseIncX := 1 + upsampleAbove
	leftOff := tlOff - (1 + upsampleLeft)
	xpos := ((1 + upsampleAbove) << 6) - dx
	for y := range height {
		row := dstOff + y*stride
		baseX := xpos >> 6
		fracX := int32(xpos & 0x3E)

		ypos := (y << (6 + upsampleLeft)) - dy
		x := 0

		// Without the upsample the top edge is read straight through, so the
		// run past the last negative column is one contiguous kernel call.
		if baseIncX == 1 {
			if lrun := min(width, -baseX); lrun > 0 {
				if n := z2LeftRun(dst, row+x, edge[:], leftOff-1, lrun, ypos, dy); n > 0 {
					x, baseX, ypos = x+n, baseX+n, ypos-n*dy
				}
			}
			for ; x < width && baseX < 0; x, baseX, ypos = x+1, baseX+1, ypos-dy {
				baseY := ypos >> 6
				fracY := int32(ypos & 0x3E)
				v := int32(edge[leftOff-baseY])*(64-fracY) +
					int32(edge[leftOff-(baseY+1)])*fracY
				dst[row+x] = P((v + 32) >> 6)
			}
			if n := z2TopRun(dst, row+x, edge[:], tlOff+baseX, width-x, fracX); n > 0 {
				x, baseX, ypos = x+n, baseX+n, ypos-n*dy
			}
		}

		for ; x < width; x, baseX, ypos = x+1, baseX+baseIncX, ypos-dy {
			var v int32
			if baseX >= 0 {
				v = int32(edge[tlOff+baseX])*(64-fracX) +
					int32(edge[tlOff+baseX+1])*fracX
			} else {
				baseY := ypos >> 6
				fracY := int32(ypos & 0x3E)
				v = int32(edge[leftOff-baseY])*(64-fracY) +
					int32(edge[leftOff-(baseY+1)])*fracY
			}
			dst[row+x] = P((v + 32) >> 6)
		}
		xpos -= dx
	}
}

// z2LeftRun fills the columns that read the left edge, where the position
// steps by dy so both the sample and the fraction move with the column.
func ipredZ3[P pixel](dst []P, dstOff, stride int, tlIn []P, tlInOff,
	width, height, angle, maxWidth, maxHeight int, bitdepthMax int32,
) {
	isSm := (angle >> 9) & 1
	enableEdgeFilter := angle >> 10
	angle &= 511
	dy := int(drIntraDerivative[(270-angle)>>1])

	var leftOut [64 + 64]P
	var left []P
	var leftOff, maxBaseY int

	upsampleLeft := 0
	if enableEdgeFilter != 0 {
		upsampleLeft = getUpsample(width+height, angle-180, isSm)
	}
	if upsampleLeft != 0 {
		upsampleEdge(leftOut[:], 0, width+height, tlIn, tlInOff-(width+height),
			max(width-height, 0), width+height+1, bitdepthMax)
		left, leftOff = leftOut[:], 2*(width+height)-2
		maxBaseY = 2*(width+height) - 2
		dy <<= 1
	} else {
		filterStrength := 0
		if enableEdgeFilter != 0 {
			filterStrength = getFilterStrength(width+height, angle-180, isSm)
		}
		if filterStrength != 0 {
			filterEdge(leftOut[:], 0, width+height, 0, width+height,
				tlIn, tlInOff-(width+height), max(width-height, 0),
				width+height+1, filterStrength)
			left, leftOff = leftOut[:], width+height-1
			maxBaseY = width + height - 1
		} else {
			left, leftOff = tlIn, tlInOff-1
			maxBaseY = height + min(width, height) - 1
		}
	}

	baseInc := 1 + upsampleLeft
	ypos := dy
	for x := range width {
		frac := int32(ypos & 0x3E)

		base := ypos >> 6
		y := 0
		for ; y < height; y, base = y+1, base+baseInc {
			if base < maxBaseY {
				v := int32(left[leftOff-base])*(64-frac) +
					int32(left[leftOff-(base+1)])*frac
				dst[dstOff+y*stride+x] = P((v + 32) >> 6)
			} else {
				fill := left[leftOff-maxBaseY]
				for ; y < height; y++ {
					dst[dstOff+y*stride+x] = fill
				}

				break
			}
		}
		ypos += dy
	}
}

func ipredFilter[P pixel](dst []P, dstOff, stride int, tlIn []P, tlInOff,
	width, height, filtIdx, maxWidth, maxHeight int, bitdepthMax int32,
) {
	filtIdx &= 511
	filter := &filterIntraTaps[filtIdx]

	topS, topO := tlIn, tlInOff+1
	for y := 0; y < height; y += 2 {
		tlS, tlO := tlIn, tlInOff-y
		leftS, leftO := tlIn, tlInOff-y-1
		leftStride := -1
		for x := 0; x < width; x += 4 {
			p0 := int32(tlS[tlO])
			p1, p2 := int32(topS[topO+0]), int32(topS[topO+1])
			p3, p4 := int32(topS[topO+2]), int32(topS[topO+3])
			p5, p6 := int32(leftS[leftO]), int32(leftS[leftO+leftStride])

			ptr := dstOff + x
			fp := 0
			for range 2 {
				for xx := range 4 {
					acc := int32(filter[fp])*p0 + int32(filter[fp+8])*p1 +
						int32(filter[fp+16])*p2 + int32(filter[fp+24])*p3 +
						int32(filter[fp+32])*p4 + int32(filter[fp+40])*p5 +
						int32(filter[fp+48])*p6
					dst[ptr+xx] = P(clip((acc+8)>>4, 0, bitdepthMax))
					fp++
				}
				ptr += stride
			}

			leftS, leftO = dst, dstOff+x+4-1
			leftStride = stride
			topO += 4
			tlS, tlO = topS, topO-1
		}
		topS, topO = dst, dstOff+stride
		dstOff += stride * 2
	}
}

func cflAc[P pixel](ac []int16, ypx []P, ypxOff, stride, wPad, hPad,
	width, height, ssHor, ssVer int,
) {
	acOff := 0
	y, x := 0, 0

	// The kernel takes the whole main rectangle; the replicated right columns
	// and bottom rows stay here, where they are a copy rather than a sum.
	n8 := 0
	var z P
	if unsafe.Sizeof(z) == 1 && cflAcMain8Asm != nil {
		if rows := height - 4*hPad; rows > 0 {
			if n8 = (width - 4*wPad) &^ 7; n8 > 0 {
				cflAcMain8Asm(&ac[0], width,
					(*uint8)(unsafe.Pointer(&ypx[ypxOff])), stride, n8, rows, ssHor, ssVer)
			}
		}
	}

	for ; y < height-4*hPad; y++ {
		for x = n8; x < width-4*wPad; x++ {
			acSum := int32(ypx[ypxOff+(x<<ssHor)])
			if ssHor != 0 {
				acSum += int32(ypx[ypxOff+x*2+1])
			}
			if ssVer != 0 {
				acSum += int32(ypx[ypxOff+(x<<ssHor)+stride])
				if ssHor != 0 {
					acSum += int32(ypx[ypxOff+x*2+1+stride])
				}
			}
			shift := 1
			if ssVer == 0 {
				shift++
			}
			if ssHor == 0 {
				shift++
			}
			ac[acOff+x] = int16(acSum << shift)
		}
		for ; x < width; x++ {
			ac[acOff+x] = ac[acOff+x-1]
		}
		acOff += width
		ypxOff += stride << ssVer
	}
	for ; y < height; y++ {
		copy(ac[acOff:][:width], ac[acOff-width:][:width])
		acOff += width
	}

	log2sz := bits.TrailingZeros32(uint32(width)) + bits.TrailingZeros32(uint32(height))
	if cflAcNorm != nil {
		cflAcNorm(&ac[0], width*height, log2sz)

		return
	}

	sum := int32((1 << log2sz) >> 1)
	for i := range width * height {
		sum += int32(ac[i])
	}
	sum >>= log2sz

	for i := range width * height {
		ac[i] -= int16(sum)
	}
}

// cflAcMain8Asm fills the subsampled luma sums for the main rectangle.
var cflAcMain8Asm func(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)

// cflAcNorm takes the mean out of the AC block. It is neither typed on the
// pixel nor on the chroma layout, so one kernel covers every case.
var cflAcNorm func(ac *int16, n int, log2sz int)

func palPred[P pixel](dst []P, dstOff, stride int, pal []P, idx []uint8, w, h int) {
	n := 0
	for y := range h {
		row := dstOff + y*stride
		for x := 0; x < w; x += 2 {
			i := idx[n]
			n++
			dst[row+x+0] = pal[i&7]
			dst[row+x+1] = pal[i>>4]
		}
	}
}

func intraPredFn[P pixel](mode int) func(dst []P, dstOff, stride int, tl []P, tlOff,
	w, h, angle, maxW, maxH int, bitdepthMax int32,
) {
	switch mode {
	case dcPred:
		return ipredDc[P]
	case vertPred:
		return ipredV[P]
	case horPred:
		return ipredH[P]
	case leftDcPred:
		return ipredDcLeft[P]
	case topDcPred:
		return ipredDcTop[P]
	case dc128Pred:
		return ipredDc128[P]
	case z1Pred:
		return ipredZ1[P]
	case z2Pred:
		return ipredZ2[P]
	case z3Pred:
		return ipredZ3[P]
	case smoothPred:
		return ipredSmooth[P]
	case smoothVPred:
		return ipredSmoothV[P]
	case smoothHPred:
		return ipredSmoothH[P]
	case paethPred:
		return ipredPaeth[P]
	case filterPred:
		return ipredFilter[P]
	}

	return nil
}

func cflPredFn[P pixel](mode int) func(dst []P, dstOff, stride int, tl []P, tlOff,
	w, h int, ac []int16, alpha int32, bitdepthMax int32,
) {
	switch mode {
	case dcPred:
		return ipredCfl[P]
	case leftDcPred:
		return ipredCflLeft[P]
	case topDcPred:
		return ipredCflTop[P]
	case dc128Pred:
		return ipredCfl128[P]
	}

	return nil
}
