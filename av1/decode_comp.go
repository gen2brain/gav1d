package av1

// readCompoundType parses jnt_comp vs. segment vs. wedge for a compound block.
func (t *taskContext[P, C]) readCompoundType(b *av1Block, bs, by4, bx4 int,
	ref0poc, ref1poc int,
) {
	f := t.f
	ts := t.ts
	mi := &ts.cdf.mi

	isSegwedge := 0
	if f.seqHdr.maskedCompound != 0 {
		maskCtx := getMaskCompCtx(t.a, &t.l, by4, bx4)
		isSegwedge = int(ts.msac.boolAdapt(mi.maskComp[maskCtx][:]))
	}

	if isSegwedge == 0 {
		if f.seqHdr.jntComp != 0 {
			jntCtx := getJntCompCtx(int(f.seqHdr.orderHintNBits),
				int(f.frameHdr.frameOffset), ref0poc, ref1poc,
				t.a, &t.l, by4, bx4)
			b.compType = uint8(compInterWeightedAvg +
				int(ts.msac.boolAdapt(mi.jntComp[jntCtx][:])))
		} else {
			b.compType = compInterAvg
		}

		return
	}

	if wedgeAllowedMask&(1<<bs) != 0 {
		ctx := int(wedgeCtxLut[bs])
		b.compType = uint8(compInterWedge - int(ts.msac.boolAdapt(mi.wedgeComp[ctx][:])))
		if b.compType == compInterWedge {
			b.wedgeIdx = uint8(ts.msac.symbolAdapt(ts.cdf.mi.wedgeIdx[ctx][:], 15))
		}
	} else {
		b.compType = compInterSeg
	}
	b.maskSign = uint8(ts.msac.boolEqui())
}

// readInterIntra parses the interintra flags for an inter block.
func (t *taskContext[P, C]) readInterIntra(b *av1Block, bs int) {
	f := t.f
	ts := t.ts
	m := &ts.cdf.mi

	iiSzGrp := int(ymodeSizeContext[bs])
	if f.seqHdr.interIntra == 0 || interintraAllowedMask&(1<<bs) == 0 ||
		ts.msac.boolAdapt(m.interintra[iiSzGrp][:]) == 0 {
		b.interintraType = interIntraNone

		return
	}

	b.interintraMode = uint8(ts.msac.symbolAdapt(m.interintraMode[iiSzGrp][:],
		nInterIntraPredModes-1))
	wedgeCtx := int(wedgeCtxLut[bs])
	b.interintraType = uint8(interIntraBlend +
		int(ts.msac.boolAdapt(m.interintraWedge[wedgeCtx][:])))
	if b.interintraType == interIntraWedge {
		b.wedgeIdx = uint8(ts.msac.symbolAdapt(m.wedgeIdx[wedgeCtx][:], 15))
	}
}
