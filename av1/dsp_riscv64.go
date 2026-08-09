//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them. The
// build tag already required the vector extension, so there is nothing to
// detect.
func dspInit[P pixel](d *dspContext[P]) {
	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.fgApplyRow = fgApplyRow8RVV
		d8.fguvApplyRow = fguvApplyRow8RVV
	}
}
