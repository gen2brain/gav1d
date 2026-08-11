//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them. The
// build tag already required the vector extension, so there is nothing to
// detect.
func dspInit[P pixel](d *dspContext[P]) {
	d.itxClip = itxClipVec
	d.itxCol = itxColVec
	d.itxTranspose = itxTransposeVec
	d.itxLanes = 4

	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16RVV
		d16.itxAdd = itxAdd16Vec
		d16.itxDC = itxDC16Vec
		d16.cdefDir = cdefFindDir16RVV
		d16.loopFilter = loopFilter16RVV
		d16.put8tap = put8tap16RVV
		d16.prep8tap = prep8tap16RVV
		d16.putBilin = putBilin16RVV
		d16.prepBilin = prepBilin16RVV
		d16.fgApplyRow = fgApplyRow16RVVRow
		d16.fguvApplyRow = fguvApplyRow16RVVRow
		d16.cdefCopy = cdefCopy16RVVRows
		d16.ipred[smoothPred] = ipredSmooth16RVV
		d16.ipred[smoothVPred] = ipredSmoothV16RVV
		d16.ipred[smoothHPred] = ipredSmoothH16RVV
		d16.ipred[paethPred] = ipredPaeth16RVV
		d16.ipred[vertPred] = ipredV16RVV
		d16.ipred[horPred] = ipredH16RVV
		d16.ipred[dcPred] = ipredDc16RVV
		d16.ipred[topDcPred] = ipredDcTop16RVV
		d16.ipred[leftDcPred] = ipredDcLeft16RVV
		d16.ipred[dc128Pred] = ipredDc12816RVV
		d16.ipred[filterPred] = ipredFilter16RVV
		d16.ipred[z1Pred] = ipredZ116RVV
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockRVV
		d8.fgApplyRow = fgApplyRow8RVV
		d8.fguvApplyRow = fguvApplyRow8RVV
		d8.put8tap = put8tap8RVV
		d8.prep8tap = prep8tap8RVV
		d8.putBilin = putBilin8RVV
		d8.prepBilin = prepBilin8RVV
		d8.itxAdd = itxAdd8Vec
		d8.itxDC = itxDC8Vec
		cdefDirCosts = cdefDirCostsRVV
		d8.cdefDir = cdefFindDir8RVV
		d8.cdefCopy = cdefCopy8RVVRows
		d8.loopFilter = loopFilter8RVV
		d8.ipred[smoothPred] = ipredSmooth8RVV
		d8.ipred[smoothVPred] = ipredSmoothV8RVV
		d8.ipred[smoothHPred] = ipredSmoothH8RVV
		d8.ipred[paethPred] = ipredPaeth8RVV
		d8.ipred[vertPred] = ipredV8RVV
		d8.ipred[horPred] = ipredH8RVV
		d8.ipred[dcPred] = ipredDc8RVV
		d8.ipred[topDcPred] = ipredDcTop8RVV
		d8.ipred[leftDcPred] = ipredDcLeft8RVV
		d8.ipred[dc128Pred] = ipredDc1288RVV
		d8.ipred[filterPred] = ipredFilter8RVV
		d8.ipred[z1Pred] = ipredZ18RVV
	}
}

func init() {
	widenCoefs16 = widenCoefs16RVV
	widenCoefs16Rect2 = widenCoefs16Rect2RVV
}
