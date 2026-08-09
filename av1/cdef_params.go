//go:build !noasm && (amd64 || arm64 || (riscv64 && riscv64.rva23u64))

package av1

type cdefParams struct {
	off      [6]int32
	priStr   int32
	secStr   int32
	priShift int32
	secShift int32
	priTap   [2]int32
	secTap   [2]int32
	w        int32
	h        int32
	bdMax    int32
}

func cdefParamsFor(priStrength, secStrength, dir, damping, w, h int,
	bitdepthMax int32,
) cdefParams {
	dirs := &cdefDirections
	p := cdefParams{
		off: [6]int32{
			int32(dirs[dir+2][0]), int32(dirs[dir+2][1]),
			int32(dirs[dir+4][0]), int32(dirs[dir+4][1]),
			int32(dirs[dir+0][0]), int32(dirs[dir+0][1]),
		},
		priStr: int32(priStrength),
		secStr: int32(secStrength),
		secTap: [2]int32{2, 1},
		w:      int32(w),
		h:      int32(h),
		bdMax:  bitdepthMax,
	}

	if priStrength != 0 {
		bitdepthMin8 := bitdepthFromMax(bitdepthMax) - 8
		tap := int32(4 - ((priStrength >> bitdepthMin8) & 1))
		p.priTap = [2]int32{tap, (tap & 3) | 2}
		p.priShift = int32(max(0, damping-ulog2(uint32(priStrength))))
	}
	if secStrength != 0 {
		p.secShift = int32(damping - ulog2(uint32(secStrength)))
	}

	return p
}
