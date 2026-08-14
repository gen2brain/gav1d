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

// ABSV replaces the absolute value RVV does not have: with mask = v >> 31,
// |v| is (v ^ mask) - mask.
#define ABSV(v, m) \
	VSRAVI $31, v, m; \
	VXORVV m, v, v;   \
	VSUBVV m, v, v

#define HAD4R(i0, i1, i2, i3, t0, t1, t2, t3) \
	VADDVV i2, i0, t0; \
	VADDVV i3, i1, t1; \
	VSUBVV i2, i0, t2; \
	VSUBVV i3, i1, t3; \
	VADDVV t1, t0, i0; \
	VSUBVV t1, t0, i1; \
	VADDVV t3, t2, i2; \
	VSUBVV t3, t2, i3

// satd4RVV is the sum of absolute 4x4 Hadamard coefficients. RVV has no lane
// shuffle, so the transpose between the two passes is a contiguous store
// followed by four strided loads.
TEXT ·satd4RVV(SB), NOSPLIT, $64-20
	MOV res+0(FP), X10
	MOV stride+8(FP), X11
	SLLI $2, X11, X11
	MOV $4, X12
	VSETVLI X12, E32, M1, TA, MA, X5

	VLE32V (X10), V0
	ADD    X11, X10, X13
	VLE32V (X13), V1
	ADD    X11, X13, X13
	VLE32V (X13), V2
	ADD    X11, X13, X13
	VLE32V (X13), V3

	HAD4R(V0, V1, V2, V3, V4, V5, V6, V7)

	MOV    $scratch-64(SP), X14
	VSE32V V0, (X14)
	ADD    $16, X14, X15
	VSE32V V1, (X15)
	ADD    $16, X15, X15
	VSE32V V2, (X15)
	ADD    $16, X15, X15
	VSE32V V3, (X15)

	MOV     $16, X16
	VLSE32V (X14), X16, V0
	ADD     $4, X14, X15
	VLSE32V (X15), X16, V1
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V2
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V3

	HAD4R(V0, V1, V2, V3, V4, V5, V6, V7)

	ABSV(V0, V8)
	ABSV(V1, V8)
	ABSV(V2, V8)
	ABSV(V3, V8)
	VADDVV    V1, V0, V0
	VADDVV    V3, V2, V2
	VADDVV    V2, V0, V0
	VMVVI     $0, V4
	VREDSUMVS V4, V0, V4
	VMVXS     V4, X6
	ADDW      $2, X6, X6
	SRAW      $2, X6, X6
	MOVW      X6, ret+16(FP)
	RET
