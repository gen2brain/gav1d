package av1

func getFilterCtx(a, l *blockContext, comp, dir, ref, yb4, xb4 int) int {
	aFilter := nSwitchableFilters
	if int(a.ref[0][xb4]) == ref || int(a.ref[1][xb4]) == ref {
		aFilter = int(a.filter[dir][xb4])
	}
	lFilter := nSwitchableFilters
	if int(l.ref[0][yb4]) == ref || int(l.ref[1][yb4]) == ref {
		lFilter = int(l.filter[dir][yb4])
	}

	switch {
	case aFilter == lFilter:
		return comp*4 + aFilter
	case aFilter == nSwitchableFilters:
		return comp*4 + lFilter
	case lFilter == nSwitchableFilters:
		return comp*4 + aFilter
	default:
		return comp*4 + nSwitchableFilters
	}
}

func getCompCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	switch {
	case haveTop && haveLeft:
		switch {
		case a.compType[xb4] != 0:
			if l.compType[yb4] != 0 {
				return 4
			}

			return 2 + b2i(uint8(l.ref[0][yb4]) >= 4)
		case l.compType[yb4] != 0:
			return 2 + b2i(uint8(a.ref[0][xb4]) >= 4)
		default:
			return b2i(l.ref[0][yb4] >= 4) ^ b2i(a.ref[0][xb4] >= 4)
		}
	case haveTop:
		if a.compType[xb4] != 0 {
			return 3
		}

		return b2i(a.ref[0][xb4] >= 4)
	case haveLeft:
		if l.compType[yb4] != 0 {
			return 3
		}

		return b2i(l.ref[0][yb4] >= 4)
	default:
		return 1
	}
}

func hasUniComp(edge *blockContext, off int) bool {
	return (edge.ref[0][off] < 4) == (edge.ref[1][off] < 4)
}

func getCompDirCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	if haveTop && haveLeft {
		aIntra, lIntra := a.intra[xb4] != 0, l.intra[yb4] != 0

		if aIntra && lIntra {
			return 2
		}
		if aIntra || lIntra {
			edge, off := a, xb4
			if aIntra {
				edge, off = l, yb4
			}
			if edge.compType[off] == compInterNone {
				return 2
			}

			return 1 + 2*b2i(hasUniComp(edge, off))
		}

		aComp := a.compType[xb4] != compInterNone
		lComp := l.compType[yb4] != compInterNone
		aRef0, lRef0 := a.ref[0][xb4], l.ref[0][yb4]

		switch {
		case !aComp && !lComp:
			return 1 + 2*b2i((aRef0 >= 4) == (lRef0 >= 4))
		case !aComp || !lComp:
			edge, off := l, yb4
			if aComp {
				edge, off = a, xb4
			}
			if !hasUniComp(edge, off) {
				return 1
			}

			return 3 + b2i((aRef0 >= 4) == (lRef0 >= 4))
		default:
			aUni, lUni := hasUniComp(a, xb4), hasUniComp(l, yb4)
			if !aUni && !lUni {
				return 0
			}
			if !aUni || !lUni {
				return 2
			}

			return 3 + b2i((aRef0 == 4) == (lRef0 == 4))
		}
	}

	if haveTop || haveLeft {
		edge, off := a, xb4
		if haveLeft {
			edge, off = l, yb4
		}
		if edge.intra[off] != 0 {
			return 2
		}
		if edge.compType[off] == compInterNone {
			return 2
		}

		return 4 * b2i(hasUniComp(edge, off))
	}

	return 2
}

func getJntCompCtx(orderHintNBits, poc, ref0poc, ref1poc int,
	a, l *blockContext, yb4, xb4 int,
) int {
	d0 := abs32(int32(getPocDiff(orderHintNBits, ref0poc, poc)))
	d1 := abs32(int32(getPocDiff(orderHintNBits, poc, ref1poc)))
	offset := b2i(d0 == d1)
	aCtx := b2i(a.compType[xb4] >= compInterAvg || a.ref[0][xb4] == 6)
	lCtx := b2i(l.compType[yb4] >= compInterAvg || l.ref[0][yb4] == 6)

	return 3*offset + aCtx + lCtx
}

func getMaskCompCtx(a, l *blockContext, yb4, xb4 int) int {
	edgeCtx := func(e *blockContext, off int) int {
		switch {
		case e.compType[off] >= compInterSeg:
			return 1
		case e.ref[0][off] == 6:
			return 3
		default:
			return 0
		}
	}

	return min(edgeCtx(a, xb4)+edgeCtx(l, yb4), 5)
}

func refCtxCount(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool,
	cnt []int, hit func(ref int8) (int, bool),
) {
	if haveTop && a.intra[xb4] == 0 {
		if i, ok := hit(a.ref[0][xb4]); ok {
			cnt[i]++
		}
		if a.compType[xb4] != 0 {
			if i, ok := hit(a.ref[1][xb4]); ok {
				cnt[i]++
			}
		}
	}
	if haveLeft && l.intra[yb4] == 0 {
		if i, ok := hit(l.ref[0][yb4]); ok {
			cnt[i]++
		}
		if l.compType[yb4] != 0 {
			if i, ok := hit(l.ref[1][yb4]); ok {
				cnt[i]++
			}
		}
	}
}

func cmpCtx(a, b int) int {
	switch {
	case a == b:
		return 1
	case a < b:
		return 0
	default:
		return 2
	}
}

func av1GetRefCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 2)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return b2i(r >= 4), true
	})

	return cmpCtx(cnt[0], cnt[1])
}

func av1GetFwdRefCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 4)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r), r < 4
	})
	cnt[0] += cnt[1]
	cnt[2] += cnt[3]

	return cmpCtx(cnt[0], cnt[2])
}

func av1GetFwdRef1Ctx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 2)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r), r < 2
	})

	return cmpCtx(cnt[0], cnt[1])
}

func av1GetFwdRef2Ctx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 2)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r) - 2, uint8(r)^2 < 2
	})

	return cmpCtx(cnt[0], cnt[1])
}

func av1GetBwdRefCtx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 3)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r) - 4, r >= 4
	})
	cnt[1] += cnt[0]

	return cmpCtx(cnt[1], cnt[2])
}

func av1GetBwdRef1Ctx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 3)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r) - 4, r >= 4
	})

	return cmpCtx(cnt[0], cnt[1])
}

func av1GetUniP1Ctx(a, l *blockContext, yb4, xb4 int, haveTop, haveLeft bool) int {
	cnt := make([]int, 3)
	refCtxCount(a, l, yb4, xb4, haveTop, haveLeft, cnt, func(r int8) (int, bool) {
		return int(r) - 1, uint8(r)-1 < 3
	})
	cnt[1] += cnt[2]

	return cmpCtx(cnt[0], cnt[1])
}

func getDrlContext(stack []refmvsCandidate, refIdx int) int {
	if stack[refIdx].weight >= 640 {
		return b2i(stack[refIdx+1].weight < 640)
	}
	if stack[refIdx+1].weight < 640 {
		return 2
	}

	return 0
}
