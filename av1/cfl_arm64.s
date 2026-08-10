//go:build arm64 && !noasm

#include "textflag.h"

// Go's assembler has no SSHLL, so the widening of a signed half into a word is
// hand encoded, low half and high half.
#define SSHLLL(Vd, Vn) WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHLLH(Vd, Vn) WORD $(0x4F10A400 | ((Vn) << 5) | (Vd))

// func cflAcNormNEON(ac *int16, n int, log2sz int)
TEXT ·cflAcNormNEON(SB), NOSPLIT, $0-24
	MOVD ac+0(FP), R0
	MOVD n+8(FP), R1
	MOVD log2sz+16(FP), R2

	VMOVI $0, V0.B16
	MOVD  R0, R3
	MOVD  R1, R4

acsum:
	VLD1.P 16(R3), [V1.H8]
	SSHLLL(2, 1)
	SSHLLH(3, 1)
	VADD   V2.S4, V0.S4, V0.S4
	VADD   V3.S4, V0.S4, V0.S4
	SUBS   $8, R4
	BNE    acsum

	VADDV V0.S4, V4
	VMOV  V4.S[0], R5

	MOVD $1, R6
	LSL  R2, R6, R6
	LSR  $1, R6, R6
	ADDW R6, R5, R5
	ASRW R2, R5, R5

	VDUP R5, V5.H8
	MOVD R0, R3
	MOVD R1, R4

acsub:
	VLD1   (R3), [V1.H8]
	VSUB   V5.H8, V1.H8, V1.H8
	VST1.P [V1.H8], 16(R3)
	SUBS   $8, R4
	BNE    acsub

	RET
