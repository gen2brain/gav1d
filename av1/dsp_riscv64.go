//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them. The
// build tag already required the vector extension, so there is nothing to
// detect.
func dspInit[P pixel](d *dspContext[P]) {
	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16RVV
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockRVV
		d8.fgApplyRow = fgApplyRow8RVV
		d8.fguvApplyRow = fguvApplyRow8RVV
		d8.put8tap = put8tap8RVV
	}
}
