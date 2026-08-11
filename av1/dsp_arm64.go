//go:build arm64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them.
// NEON is part of the architecture, so there is nothing to detect.
func dspInit[P pixel](d *dspContext[P]) {
	d.itxClip = itxClipVec
	d.itxCol = itxColVec
	d.itxTranspose = itxTransposeVec
	d.itxLanes = 4

	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16NEON
		d16.itxAdd = itxAdd16Vec
		d16.itxDC = itxDC16Vec
		d16.cdefDir = cdefFindDir16NEON
		d16.loopFilter = loopFilter16NEON
		d16.put8tap = put8tap16NEON
		d16.prep8tap = prep8tap16NEON
		d16.putBilin = putBilin16NEON
		d16.prepBilin = prepBilin16NEON
		d16.fgApplyRow = fgApplyRow16NEONRow
		d16.fguvApplyRow = fguvApplyRow16NEONRow
		d16.cdefCopy = cdefCopy16NEONRows
		d16.ipred[smoothPred] = ipredSmooth16NEON
		d16.ipred[smoothVPred] = ipredSmoothV16NEON
		d16.ipred[smoothHPred] = ipredSmoothH16NEON
		d16.ipred[paethPred] = ipredPaeth16NEON
		d16.ipred[vertPred] = ipredV16NEON
		d16.ipred[horPred] = ipredH16NEON
		d16.ipred[dcPred] = ipredDc16NEON
		d16.ipred[topDcPred] = ipredDcTop16NEON
		d16.ipred[leftDcPred] = ipredDcLeft16NEON
		d16.ipred[dc128Pred] = ipredDc12816NEON
		d16.ipred[filterPred] = ipredFilter16NEON
		d16.ipred[z1Pred] = ipredZ116NEON
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockNEON
		d8.fgApplyRow = fgApplyRow8NEON
		d8.fguvApplyRow = fguvApplyRow8NEON
		d8.put8tap = put8tap8NEON
		d8.prep8tap = prep8tap8NEON
		d8.putBilin = putBilin8NEON
		d8.prepBilin = prepBilin8NEON
		d8.itxAdd = itxAdd8Vec
		d8.itxDC = itxDC8Vec
		cdefDirCosts = cdefDirCostsNEON
		d8.cdefDir = cdefFindDir8NEON
		d8.cdefCopy = cdefCopy8NEONRows
		d8.loopFilter = loopFilter8NEON
		d8.ipred[smoothPred] = ipredSmooth8NEON
		d8.ipred[smoothVPred] = ipredSmoothV8NEON
		d8.ipred[smoothHPred] = ipredSmoothH8NEON
		d8.ipred[paethPred] = ipredPaeth8NEON
		d8.ipred[vertPred] = ipredV8NEON
		d8.ipred[horPred] = ipredH8NEON
		d8.ipred[dcPred] = ipredDc8NEON
		d8.ipred[topDcPred] = ipredDcTop8NEON
		d8.ipred[leftDcPred] = ipredDcLeft8NEON
		d8.ipred[dc128Pred] = ipredDc1288NEON
		d8.ipred[filterPred] = ipredFilter8NEON
		d8.ipred[z1Pred] = ipredZ18NEON
	}
}

func init() {
	widenCoefs16 = widenCoefs16NEON
	widenCoefs16Rect2 = widenCoefs16Rect2NEON
}
