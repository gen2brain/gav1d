//go:build arm64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them.
// NEON is part of the architecture, so there is nothing to detect.
func dspInit[P pixel](d *dspContext[P]) {
	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockNEON
		d8.fgApplyRow = fgApplyRow8NEON
		d8.fguvApplyRow = fguvApplyRow8NEON
	}
}
