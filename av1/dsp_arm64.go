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
		d16.cdefDir = cdefFindDir16NEON
		d16.loopFilter = loopFilter16NEON
		d16.put8tap = put8tap16NEON
		d16.fgApplyRow = fgApplyRow16NEONRow
		d16.fguvApplyRow = fguvApplyRow16NEONRow
		d16.cdefCopy = cdefCopy16NEONRows
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
		d8.cdefDir = cdefFindDir8NEON
		d8.cdefCopy = cdefCopy8NEONRows
		d8.loopFilter = loopFilter8NEON
	}
}
