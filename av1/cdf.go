package av1

import "sync"

type cdfModeContext struct {
	uvMode            [2][nIntraPredModes][nUvIntraPredModes + 2]uint16
	partition         [nBlLevels][4][nPartitions + 6]uint16
	cflAlpha          [6][16]uint16
	txtpInter1        [2][16]uint16
	txtpInter2        [12 + 4]uint16
	txtpIntra1        [2][nIntraPredModes][7 + 1]uint16
	txtpIntra2        [3][nIntraPredModes][5 + 3]uint16
	cflSign           [8]uint16
	angleDelta        [8][8]uint16
	filterIntra       [5 + 3]uint16
	segID             [3][maxSegments]uint16
	palSz             [2][7][7 + 1]uint16
	colorMap          [2][7][5][8]uint16
	txsz              [nTxSizes - 1][3][4]uint16
	deltaQ            [4]uint16
	deltaLf           [5][4]uint16
	restoreSwitchable [3 + 1]uint16
	restoreWiener     [2]uint16
	restoreSgrproj    [2]uint16
	txtpInter3        [4][2]uint16
	useFilterIntra    [nBsSizes][2]uint16
	txpart            [7][3][2]uint16
	skip              [3][2]uint16
	palY              [7][3][2]uint16
	palUv             [2][2]uint16
}

type cdfModeInterContext struct {
	yMode           [4][nIntraPredModes + 3]uint16
	wedgeIdx        [9][16]uint16
	compInterMode   [8][nCompInterPredModes]uint16
	filter          [2][8][nSwitchableFilters + 1]uint16
	interintraMode  [4][4]uint16
	motionMode      [nBsSizes][3 + 1]uint16
	skipMode        [3][2]uint16
	newmvMode       [6][2]uint16
	globalmvMode    [2][2]uint16
	refmvMode       [6][2]uint16
	drlBit          [3][2]uint16
	intra           [4][2]uint16
	comp            [5][2]uint16
	compDir         [5][2]uint16
	jntComp         [6][2]uint16
	maskComp        [6][2]uint16
	wedgeComp       [9][2]uint16
	ref             [6][3][2]uint16
	compFwdRef      [3][3][2]uint16
	compBwdRef      [2][3][2]uint16
	compUniRef      [3][3][2]uint16
	segPred         [3][2]uint16
	interintra      [7][2]uint16
	interintraWedge [7][2]uint16
	obmc            [nBsSizes][2]uint16
}

type cdfCoefContext struct {
	eobBin16   [2][2][5 + 3]uint16
	eobBin32   [2][2][6 + 2]uint16
	eobBin64   [2][2][7 + 1]uint16
	eobBin128  [2][2][8 + 0]uint16
	eobBin256  [2][2][9 + 7]uint16
	eobBin512  [2][10 + 6]uint16
	eobBin1024 [2][11 + 5]uint16
	eobBaseTok [nTxSizes][2][4][4]uint16
	baseTok    [nTxSizes][2][41][4]uint16
	brTok      [4][2][21][4]uint16
	eobHiBit   [nTxSizes][2][9][2]uint16
	skip       [nTxSizes][13][2]uint16
	dcSign     [2][3][2]uint16
}

type cdfMvComponent struct {
	classes  [11 + 5]uint16
	sign     [2]uint16
	class0   [2]uint16
	class0Fp [2][4]uint16
	class0Hp [2]uint16
	classN   [10][2]uint16
	classNFp [4]uint16
	classNHp [2]uint16
}

type cdfMvContext struct {
	comp  [2]cdfMvComponent
	joint [nMvJoints]uint16
}

type cdfContext struct {
	coef    cdfCoefContext
	m       cdfModeContext
	intrabc [2]uint16
	mi      cdfModeInterContext
	mv      cdfMvContext
	kfym    [5][5][nIntraPredModes + 3]uint16
}

type cdfDefaultMv struct {
	comp  cdfMvComponent
	joint [nMvJoints]uint16
}

type cdfDefaultContext struct {
	m       cdfModeContext
	intrabc [2]uint16
	mi      cdfModeInterContext
	mv      cdfDefaultMv
	kfym    [5][5][nIntraPredModes + 3]uint16
}

func qcatFromQidx(qidx uint8) int {
	n := 0
	if qidx > 20 {
		n++
	}
	if qidx > 60 {
		n++
	}
	if qidx > 120 {
		n++
	}

	return n
}

func (c *cdfContext) initStatic(qidx uint8) {
	c.coef = defaultCoefCdf[qcatFromQidx(qidx)]
	c.m = defaultCdf.m
	c.intrabc = defaultCdf.intrabc
	c.mi = defaultCdf.mi
	c.mv.comp[0] = defaultCdf.mv.comp
	c.mv.comp[1] = defaultCdf.mv.comp
	c.mv.joint = defaultCdf.mv.joint
	c.kfym = defaultCdf.kfym
}

func (c *cdfContext) resetCountsIntra() {
	for i := range 2 {
		for j := range 2 {
			c.coef.eobBin16[i][j][4] = 0
			c.coef.eobBin32[i][j][5] = 0
			c.coef.eobBin64[i][j][6] = 0
			c.coef.eobBin128[i][j][7] = 0
			c.coef.eobBin256[i][j][8] = 0
		}
		c.coef.eobBin512[i][9] = 0
		c.coef.eobBin1024[i][10] = 0
	}
	for i := range nTxSizes {
		for j := range 2 {
			for k := range 4 {
				c.coef.eobBaseTok[i][j][k][2] = 0
			}
			for k := range 41 {
				c.coef.baseTok[i][j][k][3] = 0
			}
			for k := range 9 {
				c.coef.eobHiBit[i][j][k][1] = 0
			}
		}
		for k := range 13 {
			c.coef.skip[i][k][1] = 0
		}
	}
	for i := range 4 {
		for j := range 2 {
			for k := range 21 {
				c.coef.brTok[i][j][k][3] = 0
			}
		}
	}
	for i := range 2 {
		for j := range 3 {
			c.coef.dcSign[i][j][1] = 0
		}
	}

	m := &c.m
	for k := range 2 {
		for j := range nIntraPredModes {
			n := nUvIntraPredModes - 1
			if k == 0 {
				n--
			}
			m.uvMode[k][j][n] = 0
		}
	}
	for j := range 4 {
		m.partition[bl128x128][j][nPartitions-3] = 0
	}
	for k := bl64x64; k < bl8x8; k++ {
		for j := range 4 {
			m.partition[k][j][nPartitions-1] = 0
		}
	}
	for j := range 4 {
		m.partition[bl8x8][j][nSub8x8Partitions-1] = 0
	}
	for i := range m.cflAlpha {
		m.cflAlpha[i][15] = 0
	}
	for i := range m.txtpInter1 {
		m.txtpInter1[i][15] = 0
	}
	m.txtpInter2[11] = 0
	for k := range 2 {
		for i := range m.txtpIntra1[k] {
			m.txtpIntra1[k][i][6] = 0
		}
	}
	for k := range 3 {
		for i := range m.txtpIntra2[k] {
			m.txtpIntra2[k][i][4] = 0
		}
	}
	m.cflSign[7] = 0
	for i := range m.angleDelta {
		m.angleDelta[i][6] = 0
	}
	m.filterIntra[4] = 0
	for i := range m.segID {
		m.segID[i][7] = 0
	}
	for k := range 2 {
		for i := range m.palSz[k] {
			m.palSz[k][i][6] = 0
		}
	}
	for k := range 2 {
		for j := range 7 {
			for i := range m.colorMap[k][j] {
				m.colorMap[k][j][i][j+1] = 0
			}
		}
	}
	for k := range nTxSizes - 1 {
		for i := range m.txsz[k] {
			m.txsz[k][i][min(k+1, 2)] = 0
		}
	}
	m.deltaQ[3] = 0
	for i := range m.deltaLf {
		m.deltaLf[i][3] = 0
	}
	m.restoreSwitchable[2] = 0
	m.restoreWiener[1] = 0
	m.restoreSgrproj[1] = 0
	for i := range m.txtpInter3 {
		m.txtpInter3[i][1] = 0
	}
	for i := range m.useFilterIntra {
		m.useFilterIntra[i][1] = 0
	}
	for k := range 7 {
		for i := range m.txpart[k] {
			m.txpart[k][i][1] = 0
		}
	}
	for i := range m.skip {
		m.skip[i][1] = 0
	}
	for k := range 7 {
		for i := range m.palY[k] {
			m.palY[k][i][1] = 0
		}
	}
	for i := range m.palUv {
		m.palUv[i][1] = 0
	}
}

func (c *cdfContext) resetCountsInter() {
	mi := &c.mi
	for i := range mi.yMode {
		mi.yMode[i][nIntraPredModes-1] = 0
	}
	for i := range mi.wedgeIdx {
		mi.wedgeIdx[i][15] = 0
	}
	for i := range mi.compInterMode {
		mi.compInterMode[i][nCompInterPredModes-1] = 0
	}
	for k := range 2 {
		for i := range mi.filter[k] {
			mi.filter[k][i][nSwitchableFilters-1] = 0
		}
	}
	for i := range mi.interintraMode {
		mi.interintraMode[i][3] = 0
	}
	for i := range mi.motionMode {
		mi.motionMode[i][2] = 0
	}
	for i := range mi.skipMode {
		mi.skipMode[i][1] = 0
	}
	for i := range mi.newmvMode {
		mi.newmvMode[i][1] = 0
	}
	for i := range mi.globalmvMode {
		mi.globalmvMode[i][1] = 0
	}
	for i := range mi.refmvMode {
		mi.refmvMode[i][1] = 0
	}
	for i := range mi.drlBit {
		mi.drlBit[i][1] = 0
	}
	for i := range mi.intra {
		mi.intra[i][1] = 0
	}
	for i := range mi.comp {
		mi.comp[i][1] = 0
	}
	for i := range mi.compDir {
		mi.compDir[i][1] = 0
	}
	for i := range mi.jntComp {
		mi.jntComp[i][1] = 0
	}
	for i := range mi.maskComp {
		mi.maskComp[i][1] = 0
	}
	for i := range mi.wedgeComp {
		mi.wedgeComp[i][1] = 0
	}
	for k := range 6 {
		for i := range mi.ref[k] {
			mi.ref[k][i][1] = 0
		}
	}
	for k := range 3 {
		for i := range mi.compFwdRef[k] {
			mi.compFwdRef[k][i][1] = 0
		}
	}
	for k := range 2 {
		for i := range mi.compBwdRef[k] {
			mi.compBwdRef[k][i][1] = 0
		}
	}
	for k := range 3 {
		for i := range mi.compUniRef[k] {
			mi.compUniRef[k][i][1] = 0
		}
	}
	for i := range mi.segPred {
		mi.segPred[i][1] = 0
	}
	for i := range 4 {
		mi.interintra[i][1] = 0
	}
	for i := range mi.interintraWedge {
		mi.interintraWedge[i][1] = 0
	}
	for i := range mi.obmc {
		mi.obmc[i][1] = 0
	}

	for k := range 2 {
		mv := &c.mv.comp[k]
		mv.classes[10] = 0
		mv.sign[1] = 0
		mv.class0[1] = 0
		for i := range mv.class0Fp {
			mv.class0Fp[i][3] = 0
		}
		mv.class0Hp[1] = 0
		for i := range mv.classN {
			mv.classN[i][1] = 0
		}
		mv.classNFp[3] = 0
		mv.classNHp[1] = 0
	}
	c.mv.joint[nMvJoints-1] = 0
}

// cdfPromise is a reference slot's adapted CDFs, known once the frame that
// refreshes the slot finishes its context update tile.
type cdfPromise struct {
	cdf  cdfContext
	done chan struct{}
	once sync.Once
	err  error
}

func newCdfPromise() *cdfPromise {
	return &cdfPromise{done: make(chan struct{})}
}

func (p *cdfPromise) finish(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.done)
	})
}

func (p *cdfPromise) wait() (*cdfContext, error) {
	<-p.done
	if p.err != nil {
		return nil, p.err
	}

	return &p.cdf, nil
}

func resolveCdf(hdr *frameHeader, in *cdfPromise) (*cdfContext, error) {
	if hdr.primaryRefFrame == primaryRefNone {
		c := new(cdfContext)
		c.initStatic(hdr.quant.yac)

		return c, nil
	}
	if in == nil {
		return nil, errInvalid
	}

	return in.wait()
}

func cdfThreadUpdate(hdr *frameHeader, dst, src *cdfContext) {
	dst.coef = src.coef
	dst.m = src.m
	dst.resetCountsIntra()

	if isKeyOrIntra(hdr) {
		return
	}

	dst.mi = src.mi
	dst.mv = src.mv
	dst.resetCountsInter()
}
