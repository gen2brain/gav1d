//go:build arm64 && !noasm

package av1

// dspInit swaps in the assembly kernels for the bit depths that have them.
// NEON is part of the architecture, so there is nothing to detect.
func dspInit[P pixel](d *dspContext[P]) {
	d.itxClip = itxClipVec
	d.itxCol = itxColVec

	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16NEON
		d16.itxAdd = itxAdd16Vec
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockNEON
		d8.fgApplyRow = fgApplyRow8NEON
		d8.fguvApplyRow = fguvApplyRow8NEON
		d8.put8tap = put8tap8NEON
		d8.itxAdd = itxAdd8Vec
	}
}
