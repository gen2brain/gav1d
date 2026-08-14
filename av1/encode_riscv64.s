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

#define HAD8R(i0, i1, i2, i3, i4, i5, i6, i7) \
	VADDVV i4, i0, V20; \
	VADDVV i5, i1, V21; \
	VADDVV i6, i2, V22; \
	VADDVV i7, i3, V23; \
	VSUBVV i4, i0, V24; \
	VSUBVV i5, i1, V25; \
	VSUBVV i6, i2, V26; \
	VSUBVV i7, i3, V27; \
	VADDVV V22, V20, i0; \
	VADDVV V23, V21, i1; \
	VSUBVV V22, V20, i2; \
	VSUBVV V23, V21, i3; \
	VADDVV V26, V24, i4; \
	VADDVV V27, V25, i5; \
	VSUBVV V26, V24, i6; \
	VSUBVV V27, V25, i7; \
	VADDVV i1, i0, V20; \
	VSUBVV i1, i0, V21; \
	VADDVV i3, i2, V22; \
	VSUBVV i3, i2, V23; \
	VADDVV i5, i4, V24; \
	VSUBVV i5, i4, V25; \
	VADDVV i7, i6, V26; \
	VSUBVV i7, i6, V27; \
	VMVVV V20, i0; \
	VMVVV V21, i1; \
	VMVVV V22, i2; \
	VMVVV V23, i3; \
	VMVVV V24, i4; \
	VMVVV V25, i5; \
	VMVVV V26, i6; \
	VMVVV V27, i7

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

// satd8RVV is the sum of absolute 8x8 Hadamard coefficients, transposed the
// same way satd4RVV is.
TEXT ·satd8RVV(SB), NOSPLIT, $256-20
	MOV  res+0(FP), X10
	MOV  stride+8(FP), X11
	SLLI $2, X11, X11
	MOV  $8, X12
	VSETVLI X12, E32, M1, TA, MA, X5

	VLE32V (X10), V0
	ADD    X11, X10, X13
	VLE32V (X13), V1
	ADD    X11, X13, X13
	VLE32V (X13), V2
	ADD    X11, X13, X13
	VLE32V (X13), V3
	ADD    X11, X13, X13
	VLE32V (X13), V4
	ADD    X11, X13, X13
	VLE32V (X13), V5
	ADD    X11, X13, X13
	VLE32V (X13), V6
	ADD    X11, X13, X13
	VLE32V (X13), V7

	HAD8R(V0, V1, V2, V3, V4, V5, V6, V7)

	MOV    $scratch-256(SP), X14
	MOV    X14, X15
	VSE32V V0, (X15)
	ADD    $32, X15, X15
	VSE32V V1, (X15)
	ADD    $32, X15, X15
	VSE32V V2, (X15)
	ADD    $32, X15, X15
	VSE32V V3, (X15)
	ADD    $32, X15, X15
	VSE32V V4, (X15)
	ADD    $32, X15, X15
	VSE32V V5, (X15)
	ADD    $32, X15, X15
	VSE32V V6, (X15)
	ADD    $32, X15, X15
	VSE32V V7, (X15)

	MOV     $32, X16
	MOV     X14, X15
	VLSE32V (X15), X16, V0
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V1
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V2
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V3
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V4
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V5
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V6
	ADD     $4, X15, X15
	VLSE32V (X15), X16, V7

	HAD8R(V0, V1, V2, V3, V4, V5, V6, V7)

	ABSV(V0, V16)
	ABSV(V1, V16)
	ABSV(V2, V16)
	ABSV(V3, V16)
	ABSV(V4, V16)
	ABSV(V5, V16)
	ABSV(V6, V16)
	ABSV(V7, V16)
	VADDVV    V1, V0, V0
	VADDVV    V3, V2, V2
	VADDVV    V5, V4, V4
	VADDVV    V7, V6, V6
	VADDVV    V2, V0, V0
	VADDVV    V6, V4, V4
	VADDVV    V4, V0, V0
	VMVVI     $0, V8
	VREDSUMVS V8, V0, V8
	VMVXS     V8, X6
	ADDW      $4, X6, X6
	SRAW      $3, X6, X6
	MOVW      X6, ret+16(FP)
	RET

#define RSR(v) \
	VADDVX X21, v, v; \
	VSRAVI $13, v, v

#define FDCT4R(i0, i1, i2, i3) \
	VADDVV i3, i0, V10; \
	VADDVV i2, i1, V11; \
	VSUBVV i2, i1, V12; \
	VSUBVV i3, i0, V13; \
	VADDVV V11, V10, V14; \
	VSUBVV V11, V10, V15; \
	VMULVX X17, V14, i0; \
	RSR(i0); \
	VMULVX X17, V15, i2; \
	RSR(i2); \
	VMULVX X18, V12, V16; \
	VMULVX X19, V13, V17; \
	VADDVV V17, V16, i1; \
	RSR(i1); \
	VMULVX X18, V13, V16; \
	VMULVX X19, V12, V17; \
	VSUBVV V17, V16, i3; \
	RSR(i3)

#define FADST4R(i0, i1, i2, i3) \
	VMULVX X17, i0, V10; \
	VMULVX X19, i1, V11; \
	VADDVV V11, V10, V10; \
	VMULVX X20, i3, V11; \
	VADDVV V11, V10, V10; \
	VMULVX X20, i0, V11; \
	VMULVX X17, i1, V12; \
	VSUBVV V12, V11, V11; \
	VMULVX X19, i3, V12; \
	VADDVV V12, V11, V11; \
	VMULVX X18, i2, V12; \
	VADDVV i1, i0, V13; \
	VSUBVV i3, V13, V13; \
	VMULVX X18, V13, V13; \
	VADDVV V12, V10, V14; \
	RSR(V14); \
	RSR(V13); \
	VSUBVV V12, V11, V15; \
	RSR(V15); \
	VSUBVV V10, V11, V16; \
	VADDVV V12, V16, V16; \
	RSR(V16); \
	VMVVV V14, i0; \
	VMVVV V13, i1; \
	VMVVV V15, i2; \
	VMVVV V16, i3

// fwdTxfm4x4RVV is the 4x4 forward transform for the four non-flipping
// DCT/ADST combinations. cos_bit is 13 at this size, so the constants are
// scalars and every intermediate fits int32 for 8-bit input.
TEXT ·fwdTxfm4x4RVV(SB), NOSPLIT, $64-40
	MOV  res+0(FP), X10
	MOV  src+8(FP), X11
	MOV  srcStride+16(FP), X12
	SLLI $2, X12, X12
	MOV  $4, X13
	VSETVLI X13, E32, M1, TA, MA, X5
	MOV  $4096, X21

	VLE32V (X11), V0
	ADD    X12, X11, X14
	VLE32V (X14), V1
	ADD    X12, X14, X14
	VLE32V (X14), V2
	ADD    X12, X14, X14
	VLE32V (X14), V3
	VSLLVI $2, V0, V0
	VSLLVI $2, V1, V1
	VSLLVI $2, V2, V2
	VSLLVI $2, V3, V3

	MOV colAdst+24(FP), X22
	BNEZ X22, coladstr
	MOV $5793, X17
	MOV $3135, X18
	MOV $7568, X19
	FDCT4R(V0, V1, V2, V3)
	JMP colfinr

coladstr:
	MOV $2642, X17
	MOV $6689, X18
	MOV $4964, X19
	MOV $7606, X20
	FADST4R(V0, V1, V2, V3)

colfinr:
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

	MOV rowAdst+32(FP), X22
	BNEZ X22, rowadstr
	MOV $5793, X17
	MOV $3135, X18
	MOV $7568, X19
	FDCT4R(V0, V1, V2, V3)
	JMP storer

rowadstr:
	MOV $2642, X17
	MOV $6689, X18
	MOV $4964, X19
	MOV $7606, X20
	FADST4R(V0, V1, V2, V3)

storer:
	VSE32V V0, (X10)
	ADD    $16, X10, X15
	VSE32V V1, (X15)
	ADD    $16, X15, X15
	VSE32V V2, (X15)
	ADD    $16, X15, X15
	VSE32V V3, (X15)
	RET
