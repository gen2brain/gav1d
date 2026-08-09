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

	if d16, ok := any(d).(*dspContext[uint16]); ok {
		d16.cdefFilter = cdefFilterBlock16AVX2
		d16.itxAdd = itxAdd16Vec
		d16.cdefDir = cdefFindDir16AVX2
		d16.cdefCopy = cdefCopy16AVX2Rows
	}

	if d8, ok := any(d).(*dspContext[uint8]); ok {
		d8.cdefFilter = cdefFilterBlockAVX2
		d8.fgApplyRow = fgApplyRow8AVX2
		d8.fguvApplyRow = fguvApplyRow8AVX2
		d8.put8tap = put8tap8AVX2
		d8.itxAdd = itxAdd8Vec
		d8.cdefDir = cdefFindDir8AVX2
		d8.cdefCopy = cdefCopy8AVX2Rows
		d8.loopFilter = loopFilter8AVX2
	}
}
