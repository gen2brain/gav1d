package av1

import "math"

const (
	cdefHaveLeft = 1 << iota
	cdefHaveRight
	cdefHaveTop
	cdefHaveBottom
)

func cdefConstrain(diff, threshold, shift int32) int32 {
	adiff := abs32(diff)

	return applySign(min(adiff, max(0, threshold-(adiff>>shift))), diff)
}

func cdefFill(tmp []int16, off, stride, w, h int) {
	for range h {
		for x := range w {
			tmp[off+x] = math.MinInt16
		}
		off += stride
	}
}

func cdefPadding[P pixel](tmp []int16, tmpOff, tmpStride int,
	src []P, srcOff, srcStride int, left []P, leftOff int,
	top []P, topOff int, bottom []P, bottomOff int, w, h, edges int,
) {
	xStart, xEnd, yStart, yEnd := -2, w+2, -2, h+2

	if edges&cdefHaveTop == 0 {
		cdefFill(tmp, tmpOff-2-2*tmpStride, tmpStride, w+4, 2)
		yStart = 0
	}
	if edges&cdefHaveBottom == 0 {
		cdefFill(tmp, tmpOff+h*tmpStride-2, tmpStride, w+4, 2)
		yEnd -= 2
	}
	if edges&cdefHaveLeft == 0 {
		cdefFill(tmp, tmpOff+yStart*tmpStride-2, tmpStride, 2, yEnd-yStart)
		xStart = 0
	}
	if edges&cdefHaveRight == 0 {
		cdefFill(tmp, tmpOff+yStart*tmpStride+w, tmpStride, 2, yEnd-yStart)
		xEnd -= 2
	}

	for y := yStart; y < 0; y++ {
		for x := xStart; x < xEnd; x++ {
			tmp[tmpOff+x+y*tmpStride] = int16(top[topOff+x])
		}
		topOff += srcStride
	}
	for y := range h {
		for x := xStart; x < 0; x++ {
			tmp[tmpOff+x+y*tmpStride] = int16(left[leftOff+y*2+2+x])
		}
	}
	for range h {
		for x := 0; x < xEnd; x++ {
			tmp[tmpOff+x] = int16(src[srcOff+x])
		}
		srcOff += srcStride
		tmpOff += tmpStride
	}
	for y := h; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			tmp[tmpOff+x] = int16(bottom[bottomOff+x])
		}
		bottomOff += srcStride
		tmpOff += tmpStride
	}
}

func cdefFilterBlock[P pixel](dst []P, dstOff, dstStride int,
	left []P, leftOff int, top []P, topOff int, bottom []P, bottomOff int,
	priStrength, secStrength, dir, damping, w, h, edges int, bitdepthMax int32,
) {
	const tmpStride = 12
	var tmpBuf [144]int16
	tmp := tmpBuf[:]
	tmpOff := 2*tmpStride + 2

	cdefPadding(tmp, tmpOff, tmpStride, dst, dstOff, dstStride,
		left, leftOff, top, topOff, bottom, bottomOff, w, h, edges)

	dirs := &cdefDirections

	switch {
	case priStrength != 0:
		bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
		priTap := int32(4 - ((priStrength >> bitdepthMin8) & 1))
		priShift := int32(max(0, damping-ulog2(uint32(priStrength))))

		if secStrength != 0 {
			secShift := int32(damping - ulog2(uint32(secStrength)))
			for range h {
				for x := range w {
					px := int32(dst[dstOff+x])
					sum := int32(0)
					mx, mn := px, uint32(px)
					priTapK := priTap
					for k := range 2 {
						off1 := int(dirs[dir+2][k])
						p0 := int32(tmp[tmpOff+x+off1])
						p1 := int32(tmp[tmpOff+x-off1])
						sum += priTapK * cdefConstrain(p0-px, int32(priStrength), priShift)
						sum += priTapK * cdefConstrain(p1-px, int32(priStrength), priShift)
						priTapK = (priTapK & 3) | 2
						mn = min(uint32(p0), mn)
						mx = max(p0, mx)
						mn = min(uint32(p1), mn)
						mx = max(p1, mx)

						off2 := int(dirs[dir+4][k])
						off3 := int(dirs[dir+0][k])
						s0 := int32(tmp[tmpOff+x+off2])
						s1 := int32(tmp[tmpOff+x-off2])
						s2 := int32(tmp[tmpOff+x+off3])
						s3 := int32(tmp[tmpOff+x-off3])
						secTap := int32(2 - k)
						sum += secTap * cdefConstrain(s0-px, int32(secStrength), secShift)
						sum += secTap * cdefConstrain(s1-px, int32(secStrength), secShift)
						sum += secTap * cdefConstrain(s2-px, int32(secStrength), secShift)
						sum += secTap * cdefConstrain(s3-px, int32(secStrength), secShift)
						mn = min(uint32(s0), mn)
						mx = max(s0, mx)
						mn = min(uint32(s1), mn)
						mx = max(s1, mx)
						mn = min(uint32(s2), mn)
						mx = max(s2, mx)
						mn = min(uint32(s3), mn)
						mx = max(s3, mx)
					}
					dst[dstOff+x] = P(clip(px+((sum-b2i32(sum < 0)+8)>>4), int32(mn), mx))
				}
				dstOff += dstStride
				tmpOff += tmpStride
			}

			return
		}

		for range h {
			for x := range w {
				px := int32(dst[dstOff+x])
				sum := int32(0)
				priTapK := priTap
				for k := range 2 {
					off := int(dirs[dir+2][k])
					p0 := int32(tmp[tmpOff+x+off])
					p1 := int32(tmp[tmpOff+x-off])
					sum += priTapK * cdefConstrain(p0-px, int32(priStrength), priShift)
					sum += priTapK * cdefConstrain(p1-px, int32(priStrength), priShift)
					priTapK = (priTapK & 3) | 2
				}
				dst[dstOff+x] = P(px + ((sum - b2i32(sum < 0) + 8) >> 4))
			}
			dstOff += dstStride
			tmpOff += tmpStride
		}

		return
	}

	secShift := int32(damping - ulog2(uint32(secStrength)))
	for range h {
		for x := range w {
			px := int32(dst[dstOff+x])
			sum := int32(0)
			for k := range 2 {
				off1 := int(dirs[dir+4][k])
				off2 := int(dirs[dir+0][k])
				s0 := int32(tmp[tmpOff+x+off1])
				s1 := int32(tmp[tmpOff+x-off1])
				s2 := int32(tmp[tmpOff+x+off2])
				s3 := int32(tmp[tmpOff+x-off2])
				secTap := int32(2 - k)
				sum += secTap * cdefConstrain(s0-px, int32(secStrength), secShift)
				sum += secTap * cdefConstrain(s1-px, int32(secStrength), secShift)
				sum += secTap * cdefConstrain(s2-px, int32(secStrength), secShift)
				sum += secTap * cdefConstrain(s3-px, int32(secStrength), secShift)
			}
			dst[dstOff+x] = P(px + ((sum - b2i32(sum < 0) + 8) >> 4))
		}
		dstOff += dstStride
		tmpOff += tmpStride
	}
}

var cdefDivTable = [7]uint32{840, 420, 280, 210, 168, 140, 120}

func cdefFindDir[P pixel](img []P, imgOff, stride int, bitdepthMax int32) (int, uint32) {
	bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
	var partialSumHv [2][8]int32
	var partialSumDiag [2][15]int32
	var partialSumAlt [4][11]int32

	for y := range 8 {
		for x := range 8 {
			px := int32(img[imgOff+x]>>bitdepthMin8) - 128

			partialSumDiag[0][y+x] += px
			partialSumAlt[0][y+(x>>1)] += px
			partialSumHv[0][y] += px
			partialSumAlt[1][3+y-(x>>1)] += px
			partialSumDiag[1][7+y-x] += px
			partialSumAlt[2][3-(y>>1)+x] += px
			partialSumHv[1][x] += px
			partialSumAlt[3][(y>>1)+x] += px
		}
		imgOff += stride
	}

	var cost [8]uint32
	for n := range 8 {
		cost[2] += uint32(partialSumHv[0][n] * partialSumHv[0][n])
		cost[6] += uint32(partialSumHv[1][n] * partialSumHv[1][n])
	}
	cost[2] *= 105
	cost[6] *= 105

	for n := range 7 {
		d := cdefDivTable[n]
		cost[0] += uint32(partialSumDiag[0][n]*partialSumDiag[0][n]+
			partialSumDiag[0][14-n]*partialSumDiag[0][14-n]) * d
		cost[4] += uint32(partialSumDiag[1][n]*partialSumDiag[1][n]+
			partialSumDiag[1][14-n]*partialSumDiag[1][14-n]) * d
	}
	cost[0] += uint32(partialSumDiag[0][7]*partialSumDiag[0][7]) * 105
	cost[4] += uint32(partialSumDiag[1][7]*partialSumDiag[1][7]) * 105

	for n := range 4 {
		c := &cost[n*2+1]
		for m := range 5 {
			*c += uint32(partialSumAlt[n][3+m] * partialSumAlt[n][3+m])
		}
		*c *= 105
		for m := range 3 {
			d := cdefDivTable[2*m+1]
			*c += uint32(partialSumAlt[n][m]*partialSumAlt[n][m]+
				partialSumAlt[n][10-m]*partialSumAlt[n][10-m]) * d
		}
	}

	bestDir := 0
	bestCost := cost[0]
	for n := 1; n < 8; n++ {
		if cost[n] > bestCost {
			bestCost = cost[n]
			bestDir = n
		}
	}

	return bestDir, (bestCost - cost[bestDir^4]) >> 10
}

func b2i32(v bool) int32 {
	if v {
		return 1
	}

	return 0
}
