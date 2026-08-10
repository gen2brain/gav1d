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
		d16.cdefDir = cdefFindDir16RVV
		d16.loopFilter = loopFilter16RVV
		d16.put8tap = put8tap16RVV
		d16.fgApplyRow = fgApplyRow16RVVRow
		d16.fguvApplyRow = fguvApplyRow16RVVRow
		d16.cdefCopy = cdefCopy16RVVRows
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
		d8.cdefDir = cdefFindDir8RVV
		d8.cdefCopy = cdefCopy8RVVRows
		d8.loopFilter = loopFilter8RVV
	}
}
