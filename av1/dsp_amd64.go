//go:build amd64 && !noasm

package av1

var hasAVX2 = cpuidAVX2()

func cpuidAVX2() bool

// dspInit swaps in the assembly kernels for the bit depths that have them. The
// assertion is on the pointer, so it costs an interface compare and no
// allocation, and the fields it writes are already concrete.
func dspInit[P pixel](d *dspContext[P]) {
	if !hasAVX2 {
		return
	}

	d.itxClip = itxClipVec
	d.itxCol = itxColVec
	d.itxTranspose = itxTransposeVec
	d.itxLanes = 8

	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16AVX2
		d16.itxAdd = itxAdd16Vec
		d16.cdefDir = cdefFindDir16AVX2
		d16.loopFilter = loopFilter16AVX2
		d16.put8tap = put8tap16AVX2
		d16.prep8tap = prep8tap16AVX2
		d16.putBilin = putBilin16AVX2
		d16.prepBilin = prepBilin16AVX2
		d16.fgApplyRow = fgApplyRow16AVX2Row
		d16.fguvApplyRow = fguvApplyRow16AVX2Row
		d16.cdefCopy = cdefCopy16AVX2Rows
		d16.ipred[smoothPred] = ipredSmooth16AVX2
		d16.ipred[smoothVPred] = ipredSmoothV16AVX2
		d16.ipred[smoothHPred] = ipredSmoothH16AVX2
		d16.ipred[paethPred] = ipredPaeth16AVX2
		d16.ipred[vertPred] = ipredV16AVX2
		d16.ipred[horPred] = ipredH16AVX2
		d16.ipred[dcPred] = ipredDc16AVX2
		d16.ipred[topDcPred] = ipredDcTop16AVX2
		d16.ipred[leftDcPred] = ipredDcLeft16AVX2
		d16.ipred[dc128Pred] = ipredDc12816AVX2
		d16.ipred[filterPred] = ipredFilter16AVX2
		d16.ipred[z1Pred] = ipredZ116AVX2
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockAVX2
		d8.fgApplyRow = fgApplyRow8AVX2
		d8.fguvApplyRow = fguvApplyRow8AVX2
		d8.put8tap = put8tap8AVX2
		d8.prep8tap = prep8tap8AVX2
		d8.putBilin = putBilin8AVX2
		d8.prepBilin = prepBilin8AVX2
		d8.itxAdd = itxAdd8Vec
		d8.cdefDir = cdefFindDir8AVX2
		d8.cdefCopy = cdefCopy8AVX2Rows
		d8.loopFilter = loopFilter8AVX2
		d8.ipred[smoothPred] = ipredSmooth8AVX2
		d8.ipred[smoothVPred] = ipredSmoothV8AVX2
		d8.ipred[smoothHPred] = ipredSmoothH8AVX2
		d8.ipred[paethPred] = ipredPaeth8AVX2
		d8.ipred[vertPred] = ipredV8AVX2
		d8.ipred[horPred] = ipredH8AVX2
		d8.ipred[dcPred] = ipredDc8AVX2
		d8.ipred[topDcPred] = ipredDcTop8AVX2
		d8.ipred[leftDcPred] = ipredDcLeft8AVX2
		d8.ipred[dc128Pred] = ipredDc1288AVX2
		d8.ipred[filterPred] = ipredFilter8AVX2
		d8.ipred[z1Pred] = ipredZ18AVX2
	}
}
