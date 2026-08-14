//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// residualRVV subtracts the prediction from the source, stores the difference
// and returns its sum of squares. Vector length agnostic: one strip-mined pass
// per row, so a 4 pixel row costs one iteration and a 32 pixel row costs four.
TEXT ·residualRVV(SB), NOSPLIT, $0-64
	MOV res+0(FP), X10
	MOV dst+8(FP), X11
	MOV stride+16(FP), X12
	MOV src+24(FP), X13
	MOV sstride+32(FP), X14
	MOV w+40(FP), X15
	MOV h+48(FP), X16
	MOV $0, X17

rowloop:
	MOV X15, X18
	MOV X13, X19
	MOV X11, X20

strip:
	VSETVLI   X18, E8, MF4, TA, MA, X5
	VLE8V     (X19), V1
	VLE8V     (X20), V2
	VSETVLI   X18, E16, MF2, TA, MA, X5
	VZEXTVF2  V1, V4
	VZEXTVF2  V2, V5
	VSUBVV    V5, V4, V6
	VSETVLI   X18, E32, M1, TA, MA, X5
	VSEXTVF2  V6, V8
	VSE32V    V8, (X10)
	VMULVV    V8, V8, V10
	VMVVI     $0, V12
	VREDSUMVS V12, V10, V12
	VMVXS     V12, X6
	ADD       X6, X17, X17

	SLLI $2, X5, X6
	ADD  X6, X10, X10
	ADD  X5, X19, X19
	ADD  X5, X20, X20
	SUB  X5, X18, X18
	BNEZ X18, strip

	ADD  X14, X13, X13
	ADD  X12, X11, X11
	SUB  $1, X16, X16
	BNEZ X16, rowloop

	MOV X17, ret+56(FP)
	RET

// sseRVV is the sum of squared differences, strip-mined per row.
TEXT ·sseRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X11
	MOV stride+8(FP), X12
	MOV src+16(FP), X13
	MOV sstride+24(FP), X14
	MOV w+32(FP), X15
	MOV h+40(FP), X16
	MOV $0, X17

srow:
	MOV X15, X18
	MOV X13, X19
	MOV X11, X20

sstrip:
	VSETVLI   X18, E8, MF4, TA, MA, X5
	VLE8V     (X19), V1
	VLE8V     (X20), V2
	VSETVLI   X18, E16, MF2, TA, MA, X5
	VZEXTVF2  V1, V4
	VZEXTVF2  V2, V5
	VSUBVV    V5, V4, V6
	VSETVLI   X18, E32, M1, TA, MA, X5
	VSEXTVF2  V6, V8
	VMULVV    V8, V8, V10
	VMVVI     $0, V12
	VREDSUMVS V12, V10, V12
	VMVXS     V12, X6
	ADD       X6, X17, X17

	ADD  X5, X19, X19
	ADD  X5, X20, X20
	SUB  X5, X18, X18
	BNEZ X18, sstrip

	ADD  X14, X13, X13
	ADD  X12, X11, X11
	SUB  $1, X16, X16
	BNEZ X16, srow

	MOV X17, ret+48(FP)
	RET
