package av1

import "unsafe"

const (
	grainWidth     = 82
	grainHeight    = 73
	subGrainWidth  = 44
	subGrainHeight = 38
	fgBlockSize    = 32
)

type grainLut [grainHeight][grainWidth]int16

func fgRandom(bits int, state *uint32) int {
	r := *state
	bit := (r ^ (r >> 1) ^ (r >> 3) ^ (r >> 12)) & 1
	*state = (r >> 1) | (bit << 15)

	return int((*state >> (16 - bits)) & (1<<bits - 1))
}

func fgRound2(x, shift int) int {
	return (x + ((1 << shift) >> 1)) >> shift
}

func generateGrainY(buf *grainLut, data *filmGrainData, bpc int) {
	bitdepthMin8 := bpc - 8
	seed := data.seed
	shift := 4 - bitdepthMin8 + data.grainScaleShift
	grainCtr := 128 << bitdepthMin8
	grainMin, grainMax := -grainCtr, grainCtr-1

	for y := range grainHeight {
		for x := range grainWidth {
			v := fgRandom(11, &seed)
			buf[y][x] = int16(fgRound2(int(gaussianSequence[v]), shift))
		}
	}

	const arPad = 3
	arLag := data.arCoeffLag
	if arLag == 0 {
		return
	}

	var pre [grainWidth]int32
	x0, x1 := arPad, grainWidth-arPad
	nprev := arLag * (2*arLag + 1)
	arShift := int(data.arCoeffShift)

	for y := arPad; y < grainHeight; y++ {
		grainAR(pre[:], buf, y, arLag, data.arCoeffsY[:], x0, x1)

		row := &buf[y]
		for x := x0; x < x1; x++ {
			sum := pre[x]
			for dx := range arLag {
				sum += int32(data.arCoeffsY[nprev+dx]) * int32(row[x-arLag+dx])
			}
			grain := int(row[x]) + fgRound2(int(sum), arShift)
			row[x] = int16(clip(grain, grainMin, grainMax))
		}
	}
}

var grainAR = grainARGo

func grainARGo(pre []int32, buf *grainLut, y, lag int, coeffs []int8, x0, x1 int) {
	clear(pre[x0:x1])
	grainARAcc(pre, buf, y, lag, coeffs, x0, x1)
}

func grainARAcc(pre []int32, buf *grainLut, y, lag int, coeffs []int8, x0, x1 int) {
	c := 0
	for dy := -lag; dy < 0; dy++ {
		row := &buf[y+dy]
		for dx := -lag; dx <= lag; dx++ {
			co := int32(coeffs[c])
			c++
			if co == 0 {
				continue
			}
			for x := x0; x < x1; x++ {
				pre[x] += co * int32(row[x+dx])
			}
		}
	}
}

func generateGrainUV(buf, bufY *grainLut, data *filmGrainData, uv, subx, suby, bpc int) {
	bitdepthMin8 := bpc - 8
	seed := data.seed
	if uv != 0 {
		seed ^= 0x49d8
	} else {
		seed ^= 0xb524
	}
	shift := 4 - bitdepthMin8 + data.grainScaleShift
	grainCtr := 128 << bitdepthMin8
	grainMin, grainMax := -grainCtr, grainCtr-1

	chromaW, chromaH := grainWidth, grainHeight
	if subx != 0 {
		chromaW = subGrainWidth
	}
	if suby != 0 {
		chromaH = subGrainHeight
	}

	for y := range chromaH {
		for x := range chromaW {
			v := fgRandom(11, &seed)
			buf[y][x] = int16(fgRound2(int(gaussianSequence[v]), shift))
		}
	}

	const arPad = 3
	arLag := data.arCoeffLag

	var pre [grainWidth]int32
	x0, x1 := arPad, chromaW-arPad
	nprev := arLag * (2*arLag + 1)
	arShift := int(data.arCoeffShift)
	coeffs := data.arCoeffsUv[uv][:]

	for y := arPad; y < chromaH; y++ {
		if arLag > 0 {
			grainAR(pre[:], buf, y, arLag, coeffs, x0, x1)
		} else {
			clear(pre[x0:x1])
		}

		row := &buf[y]
		for x := x0; x < x1; x++ {
			sum := pre[x]
			for dx := range arLag {
				sum += int32(coeffs[nprev+dx]) * int32(row[x-arLag+dx])
			}
			if data.numYPoints != 0 {
				luma := 0
				lumaX := ((x - arPad) << subx) + arPad
				lumaY := ((y - arPad) << suby) + arPad
				for i := 0; i <= suby; i++ {
					for j := 0; j <= subx; j++ {
						luma += int(bufY[lumaY+i][lumaX+j])
					}
				}
				sum += int32(fgRound2(luma, subx+suby)) * int32(coeffs[nprev+arLag])
			}

			grain := int(row[x]) + fgRound2(int(sum), arShift)
			row[x] = int16(clip(grain, grainMin, grainMax))
		}
	}
}

func fgSampleLut(lut *grainLut, offsets *[2][2]int, subx, suby, bx, by, x, y int) int {
	randval := offsets[bx][by]
	offx := 3 + (2>>subx)*(3+(randval>>4))
	offy := 3 + (2>>suby)*(3+(randval&0xf))

	return int(lut[offy+y+(fgBlockSize>>suby)*by][offx+x+(fgBlockSize>>subx)*bx])
}

var fgOverlapWeights = [2][2][2]int{
	{{27, 17}, {17, 27}},
	{{23, 22}, {0, 0}},
}

func fgSeeds(data *filmGrainData, rowNum, rows int) (seed [2]uint32) {
	for i := range rows {
		seed[i] = data.seed
		seed[i] ^= uint32(((rowNum-i)*37+178)&0xff) << 8
		seed[i] ^= uint32(((rowNum-i)*173 + 105) & 0xff)
	}

	return seed
}

// fgApplyRow adds noise to a run of pixels whose grain samples are contiguous,
// which is every row of a block except its overlapped columns.
func fgApplyRow[P pixel](dst []P, dstOff int, src []P, srcOff int,
	scaling []uint8, grain []int16, n, shift, minValue, maxValue int,
) {
	for i := range n {
		v := int(src[srcOff+i])
		noise := fgRound2(int(scaling[v])*int(grain[i]), shift)
		dst[dstOff+i] = P(clip(v+noise, minValue, maxValue))
	}
}

func fgy32x32xn[P pixel](dsp *dspContext[P], dstRow []P, dstOff int, srcRow []P, srcOff, stride int,
	data *filmGrainData, pw int, scaling []uint8, lut *grainLut, bh, rowNum, bpc int,
) {
	rows := 1
	if data.overlapFlag != 0 && rowNum > 0 {
		rows = 2
	}
	bitdepthMin8 := bpc - 8
	grainCtr := 128 << bitdepthMin8
	grainMin, grainMax := -grainCtr, grainCtr-1

	minValue, maxValue := 0, 1<<bpc-1
	if data.clipToRestrictedRange != 0 {
		minValue = 16 << bitdepthMin8
		maxValue = 235 << bitdepthMin8
	}

	seed := fgSeeds(data, rowNum, rows)
	shift := data.scalingShift

	addNoise := func(x, y, grain, bx int) {
		s := srcOff + y*stride + x + bx
		noise := fgRound2(int(scaling[srcRow[s]])*grain, shift)
		dstRow[dstOff+y*stride+x+bx] = P(clip(int(srcRow[s])+noise, minValue, maxValue))
	}

	var offsets [2][2]int

	for bx := 0; bx < pw; bx += fgBlockSize {
		bw := min(fgBlockSize, pw-bx)

		if data.overlapFlag != 0 && bx != 0 {
			for i := range rows {
				offsets[1][i] = offsets[0][i]
			}
		}
		for i := range rows {
			offsets[0][i] = fgRandom(8, &seed[i])
		}

		ystart, xstart := 0, 0
		if data.overlapFlag != 0 && rowNum != 0 {
			ystart = min(2, bh)
		}
		if data.overlapFlag != 0 && bx != 0 {
			xstart = min(2, bw)
		}

		w := &fgOverlapWeights[0]

		randval := offsets[0][0]
		offx := 3 + 2*(3+(randval>>4))
		offy := 3 + 2*(3+(randval&0xf))

		for y := ystart; y < bh; y++ {
			off := y*stride + bx + xstart
			dsp.fgApplyRow(dstRow, dstOff+off, srcRow, srcOff+off, scaling,
				lut[offy+y][offx+xstart:], bw-xstart, shift, minValue, maxValue)
			for x := range xstart {
				grain := fgSampleLut(lut, &offsets, 0, 0, 0, 0, x, y)
				old := fgSampleLut(lut, &offsets, 0, 0, 1, 0, x, y)
				grain = fgRound2(old*w[x][0]+grain*w[x][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax), bx)
			}
		}

		for y := range ystart {
			for x := xstart; x < bw; x++ {
				grain := fgSampleLut(lut, &offsets, 0, 0, 0, 0, x, y)
				old := fgSampleLut(lut, &offsets, 0, 0, 0, 1, x, y)
				grain = fgRound2(old*w[y][0]+grain*w[y][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax), bx)
			}
			for x := range xstart {
				top := fgSampleLut(lut, &offsets, 0, 0, 0, 1, x, y)
				old := fgSampleLut(lut, &offsets, 0, 0, 1, 1, x, y)
				top = clip(fgRound2(old*w[x][0]+top*w[x][1], 5), grainMin, grainMax)

				grain := fgSampleLut(lut, &offsets, 0, 0, 0, 0, x, y)
				old = fgSampleLut(lut, &offsets, 0, 0, 1, 0, x, y)
				grain = clip(fgRound2(old*w[x][0]+grain*w[x][1], 5), grainMin, grainMax)

				grain = fgRound2(top*w[y][0]+grain*w[y][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax), bx)
			}
		}
	}
}

// fguvApplyRow is fgApplyRow for chroma, where the scaling index comes from the
// co-located luma rather than the pixel itself.
func fguvApplyRow[P pixel](dst []P, dstOff int, src []P, srcOff int,
	luma []P, lumaOff int, scaling []uint8, grain []int16,
	n, shift, minValue, maxValue int, sx, lumaMult, mult, offset, pixelMax int, csfl bool,
) {
	for i := range n {
		l := lumaOff + i<<sx
		avg := int(luma[l])
		if sx != 0 {
			avg = (avg + int(luma[l+1]) + 1) >> 1
		}

		val := avg
		if !csfl {
			val = clip((avg*lumaMult+int(src[srcOff+i])*mult)>>6+offset, 0, pixelMax)
		}

		noise := fgRound2(int(scaling[val])*int(grain[i]), shift)
		dst[dstOff+i] = P(clip(int(src[srcOff+i])+noise, minValue, maxValue))
	}
}

func fguv32x32xn[P pixel](dsp *dspContext[P], dstRow []P, dstOff int, srcRow []P, srcOff, stride int,
	data *filmGrainData, pw int, scaling []uint8, lut *grainLut, bh, rowNum int,
	lumaRow []P, lumaOff, lumaStride, uv int, isID bool, sx, sy, bpc int,
) {
	rows := 1
	if data.overlapFlag != 0 && rowNum > 0 {
		rows = 2
	}
	bitdepthMin8 := bpc - 8
	grainCtr := 128 << bitdepthMin8
	grainMin, grainMax := -grainCtr, grainCtr-1

	minValue, maxValue := 0, 1<<bpc-1
	if data.clipToRestrictedRange != 0 {
		minValue = 16 << bitdepthMin8
		maxValue = 240 << bitdepthMin8
		if isID {
			maxValue = 235 << bitdepthMin8
		}
	}

	seed := fgSeeds(data, rowNum, rows)
	shift := data.scalingShift

	var bx int
	addNoise := func(x, y, grain int) {
		lx := (bx + x) << sx
		ly := y << sy
		l := lumaOff + ly*lumaStride + lx
		avg := int(lumaRow[l])
		if sx != 0 {
			avg = (avg + int(lumaRow[l+1]) + 1) >> 1
		}
		s := srcOff + y*stride + bx + x
		val := avg
		if data.chromaScalingFromLuma == 0 {
			combined := avg*data.uvLumaMult[uv] + int(srcRow[s])*data.uvMult[uv]
			val = clip((combined>>6)+data.uvOffset[uv]*(1<<bitdepthMin8), 0, 1<<bpc-1)
		}
		noise := fgRound2(int(scaling[val])*grain, shift)
		dstRow[dstOff+y*stride+bx+x] = P(clip(int(srcRow[s])+noise, minValue, maxValue))
	}

	var offsets [2][2]int

	for bx = 0; bx < pw; bx += fgBlockSize >> sx {
		bw := min(fgBlockSize>>sx, pw-bx)

		if data.overlapFlag != 0 && bx != 0 {
			for i := range rows {
				offsets[1][i] = offsets[0][i]
			}
		}
		for i := range rows {
			offsets[0][i] = fgRandom(8, &seed[i])
		}

		ystart, xstart := 0, 0
		if data.overlapFlag != 0 && rowNum != 0 {
			ystart = min(2>>sy, bh)
		}
		if data.overlapFlag != 0 && bx != 0 {
			xstart = min(2>>sx, bw)
		}

		wx, wy := &fgOverlapWeights[sx], &fgOverlapWeights[sy]

		randval := offsets[0][0]
		offx := 3 + (2>>sx)*(3+(randval>>4))
		offy := 3 + (2>>sy)*(3+(randval&0xf))

		for y := ystart; y < bh; y++ {
			off := y*stride + bx + xstart
			dsp.fguvApplyRow(dstRow, dstOff+off, srcRow, srcOff+off,
				lumaRow, lumaOff+(y<<sy)*lumaStride+(bx+xstart)<<sx,
				scaling, lut[offy+y][offx+xstart:], bw-xstart, shift,
				minValue, maxValue, sx, data.uvLumaMult[uv], data.uvMult[uv],
				data.uvOffset[uv]*(1<<bitdepthMin8), 1<<bpc-1,
				data.chromaScalingFromLuma != 0)
			for x := range xstart {
				grain := fgSampleLut(lut, &offsets, sx, sy, 0, 0, x, y)
				old := fgSampleLut(lut, &offsets, sx, sy, 1, 0, x, y)
				grain = fgRound2(old*wx[x][0]+grain*wx[x][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax))
			}
		}

		for y := range ystart {
			for x := xstart; x < bw; x++ {
				grain := fgSampleLut(lut, &offsets, sx, sy, 0, 0, x, y)
				old := fgSampleLut(lut, &offsets, sx, sy, 0, 1, x, y)
				grain = fgRound2(old*wy[y][0]+grain*wy[y][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax))
			}
			for x := range xstart {
				top := fgSampleLut(lut, &offsets, sx, sy, 0, 1, x, y)
				old := fgSampleLut(lut, &offsets, sx, sy, 1, 1, x, y)
				top = clip(fgRound2(old*wx[x][0]+top*wx[x][1], 5), grainMin, grainMax)

				grain := fgSampleLut(lut, &offsets, sx, sy, 0, 0, x, y)
				old = fgSampleLut(lut, &offsets, sx, sy, 1, 0, x, y)
				grain = clip(fgRound2(old*wx[x][0]+grain*wx[x][1], 5), grainMin, grainMax)

				grain = fgRound2(top*wy[y][0]+grain*wy[y][1], 5)
				addNoise(x, y, clip(grain, grainMin, grainMax))
			}
		}
	}
}

func generateScaling(bpc int, points [][2]uint8, num int, scaling []uint8) {
	shiftX, scalingSize := 0, 256
	if bpc > 8 {
		shiftX, scalingSize = bpc-8, 1<<bpc
	}

	if num == 0 {
		clear(scaling[:scalingSize])

		return
	}

	for i := range int(points[0][0]) << shiftX {
		scaling[i] = points[0][1]
	}

	for i := range num - 1 {
		bx, by := int(points[i][0]), int(points[i][1])
		ex, ey := int(points[i+1][0]), int(points[i+1][1])
		dx, dy := ex-bx, ey-by
		delta := dy * ((0x10000 + (dx >> 1)) / dx)
		for x, d := 0, 0x8000; x < dx; x++ {
			scaling[(bx+x)<<shiftX] = uint8(by + (d >> 16))
			d += delta
		}
	}

	n := int(points[num-1][0]) << shiftX
	for i := n; i < scalingSize; i++ {
		scaling[i] = points[num-1][1]
	}

	if bpc == 8 {
		return
	}

	pad, rnd := 1<<shiftX, (1<<shiftX)>>1
	for i := range num - 1 {
		bx := int(points[i][0]) << shiftX
		ex := int(points[i+1][0]) << shiftX
		for x := 0; x < ex-bx; x += pad {
			rng := int(scaling[bx+x+pad]) - int(scaling[bx+x])
			for k, r := 1, rnd; k < pad; k++ {
				r += rng
				scaling[bx+x+k] = uint8(int(scaling[bx+x]) + (r >> shiftX))
			}
		}
	}
}

type filmGrainCtx struct {
	scaling [3][]uint8
	lut     [3]grainLut
}

func hasFilmGrain(hdr *frameHeader) bool {
	d := &hdr.filmGrain.data

	return d.numYPoints != 0 || d.numUvPoints[0] != 0 || d.numUvPoints[1] != 0 ||
		(d.clipToRestrictedRange != 0 && d.chromaScalingFromLuma != 0)
}

func prepGrain[P pixel](fg *filmGrainCtx, data *filmGrainData, layout, bpc, w, h int,
	dst *[3][]P, src *[3][]P, stride *[3]int,
) {
	ssVer := b2i(layout == pixelLayoutI420)
	ssHor := b2i(layout != pixelLayoutI444)

	generateGrainY(&fg.lut[0], data, bpc)
	for pl := range 2 {
		if data.numUvPoints[pl] != 0 || data.chromaScalingFromLuma != 0 {
			generateGrainUV(&fg.lut[1+pl], &fg.lut[0], data, pl, ssHor, ssVer, bpc)
		}
	}

	scalingSize := 256
	if bpc > 8 {
		scalingSize = 1 << bpc
	}
	for pl := range 3 {
		fg.scaling[pl] = make([]uint8, scalingSize+4)[:scalingSize]
	}
	if data.numYPoints != 0 || data.chromaScalingFromLuma != 0 {
		generateScaling(bpc, data.yPoints[:], data.numYPoints, fg.scaling[0])
	}
	for pl := range 2 {
		if data.numUvPoints[pl] != 0 {
			generateScaling(bpc, data.uvPoints[pl][:], data.numUvPoints[pl], fg.scaling[1+pl])
		}
	}

	if data.numYPoints == 0 {
		copy(dst[0][:h*stride[0]], src[0][:h*stride[0]])
	}

	if layout == pixelLayoutI400 || data.chromaScalingFromLuma != 0 {
		return
	}
	n := ((h + ssVer) >> ssVer) * stride[1]
	for pl := range 2 {
		if data.numUvPoints[pl] == 0 {
			copy(dst[1+pl][:n], src[1+pl][:n])
		}
	}
}

func applyGrainRow[P pixel](dsp *dspContext[P], fg *filmGrainCtx, data *filmGrainData,
	layout, bpc, w, h int, isID bool, dst *[3][]P, src *[3][]P, stride *[3]int, row int,
) {
	ssVer := b2i(layout == pixelLayoutI420)
	ssHor := b2i(layout != pixelLayoutI444)
	cpw := (w + ssHor) >> ssHor
	lumaSrc := row * fgBlockSize * stride[0]

	if data.numYPoints != 0 {
		bh := min(h-row*fgBlockSize, fgBlockSize)
		fgy32x32xn(dsp, dst[0], lumaSrc, src[0], lumaSrc, stride[0], data, w,
			fg.scaling[0], &fg.lut[0], bh, row, bpc)
	}

	if data.numUvPoints[0] == 0 && data.numUvPoints[1] == 0 &&
		data.chromaScalingFromLuma == 0 {
		return
	}

	bh := (min(h-row*fgBlockSize, fgBlockSize) + ssVer) >> ssVer

	if w&ssHor != 0 {
		ptr := lumaSrc
		for range bh {
			src[0][ptr+w] = src[0][ptr+w-1]
			ptr += stride[0] << ssVer
		}
	}

	uvOff := row * fgBlockSize * stride[1] >> ssVer
	for pl := range 2 {
		scaling := fg.scaling[1+pl]
		if data.chromaScalingFromLuma != 0 {
			scaling = fg.scaling[0]
		} else if data.numUvPoints[pl] == 0 {
			continue
		}
		fguv32x32xn(dsp, dst[1+pl], uvOff, src[1+pl], uvOff, stride[1], data, cpw,
			scaling, &fg.lut[1+pl], bh, row, src[0], lumaSrc, stride[0], pl,
			isID, ssHor, ssVer, bpc)
	}
}

func applyGrain[P pixel](dsp *dspContext[P], data *filmGrainData, layout, bpc, w, h int,
	isID bool, dst *[3][]P, src *[3][]P, stride *[3]int,
) {
	var fg filmGrainCtx

	prepGrain(&fg, data, layout, bpc, w, h, dst, src, stride)
	for row := range (h + fgBlockSize - 1) / fgBlockSize {
		applyGrainRow(dsp, &fg, data, layout, bpc, w, h, isID, dst, src, stride, row)
	}
}

func (f *frameContext[P, C]) grainPicture() *Picture {
	w, h := f.srWidth, f.frameHdr.height
	stride := [3]int{f.srStride[0], f.srStride[1], f.srStride[1]}

	m := &picMem{pool: f.pool}
	m.refs.Store(1)

	var dst [3][]P
	for pl := range 3 {
		if f.srCur[pl] == nil {
			continue
		}
		ph := h
		if pl != 0 {
			ph = (h + f.ssVer) >> f.ssVer
		}
		dst[pl] = picAlloc[P](f.pool, m, stride[pl]*ph)
	}

	applyGrain(f.dsp, &f.frameHdr.filmGrain.data, f.layout, f.bpc, w, h,
		f.seqHdr.mtrx == mcIdentity, &dst, &f.srCur, &stride)

	p := &Picture{
		Width:    w,
		Height:   h,
		Layout:   f.layout,
		BitDepth: f.bpc,
		mem:      m,
	}
	f.setColor(p)
	for pl := range 3 {
		if dst[pl] == nil {
			continue
		}
		var z P
		p.Stride[pl] = stride[pl] * int(unsafe.Sizeof(z))
		p.Data[pl] = pixelsToBytes(dst[pl])
	}

	return p
}
