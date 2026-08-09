//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// func fgApplyRowRVV(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)
//
// Vector length agnostic: vsetvli clamps the last iteration, so there is no
// tail to fix up. Pre-shifting the scaling value by 15-lshift turns the
// rounding shift into vsmul, which rounds and narrows in one instruction.
TEXT ·fgApplyRowRVV(SB), NOSPLIT, $0-64
	MOV dst+0(FP), X10
	MOV src+8(FP), X11
	MOV scaling+16(FP), X12
	MOV grain+24(FP), X13
	MOV n+32(FP), X14
	MOV lshift+40(FP), X15
	MOV minValue+48(FP), X16
	MOV maxValue+56(FP), X17

loop:
	VSETVLI X14, E16, M1, TA, MA, X5

	// src bytes, widened to the index and to the addend
	VSETVLI   X5, E8, MF2, TA, MA, X6
	VLE8V     (X11), V1
	VSETVLI   X5, E16, M1, TA, MA, X6
	VZEXTVF2  V1, V2

	// scaling[src], gathered by the widened index
	VSETVLI   X5, E8, MF2, TA, MA, X6
	VLUXEI16V (X12), V2, V3
	VSETVLI   X5, E16, M1, TA, MA, X6
	VZEXTVF2  V3, V4
	VSLLVX    X15, V4, V4

	VLE16V  (X13), V5
	VSMULVV V4, V5, V6
	VADDVV  V2, V6, V6

	VMAXVX X16, V6, V6
	VMINVX X17, V6, V6

	VSETVLI  X5, E8, MF2, TA, MA, X6
	VNSRLWI  $0, V6, V7
	VSE8V    V7, (X10)

	ADD  X5, X10
	ADD  X5, X11
	SLLI $1, X5, X6
	ADD  X6, X13
	SUB  X5, X14
	BNEZ X14, loop

	RET
