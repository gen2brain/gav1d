//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// At high bit depth both rows are int32, so a tap is the row pointer plus a
// constant in either of them.

// ATAP loads one int32 tap.
#define ATAP(OFF, REG, VD)   \
	ADD    $OFF, REG, X6; \
	VLE32V (X6), VD

// BTAP loads one int32 tap, which at high bit depth is what the b rows are.
#define BTAP(OFF, REG, VD)   \
	ADD    $OFF, REG, X6; \
	VLE32V (X6), VD

// STORE finishes one group: subtract, round, shift and narrow.
#define STORE(SHIFT)              \
	VSETVLI  X10, E16, MF2, TA, MA, X7; \
	VLE16V   (X11), V16;      \
	VSETVLI  X10, E32, M1, TA, MA, X7; \
	VZEXTVF2 V16, V17;        \
	VMULVV   V17, V20, V20;   \
	VSUBVV   V20, V19, V19;   \
	VADDVX   X14, V19, V19;   \
	VSRAVI   SHIFT, V19, V19; \
	VSE32V   V19, (X12)

// func sgrFinish1_16RVV(tmp *int32, src *uint16, a0, a1, a2 *int32, b0, b1, b2 *int32, n int)
TEXT ·sgrFinish1_16RVV(SB), NOSPLIT, $0-72
	MOV tmp+0(FP), X12
	MOV src+8(FP), X11
	MOV a0+16(FP), X18
	MOV a1+24(FP), X19
	MOV a2+32(FP), X20
	MOV b0+40(FP), X21
	MOV b1+48(FP), X22
	MOV b2+56(FP), X23
	MOV n+64(FP), X13
	MOV $256, X14

loop1_16:
	VSETVLI X13, E32, M1, TA, MA, X10

	ATAP(4, X19, V19)
	ATAP(0, X19, V1)
	VADDVV V1, V19, V19
	ATAP(8, X19, V1)
	VADDVV V1, V19, V19
	ATAP(4, X18, V1)
	VADDVV V1, V19, V19
	ATAP(4, X20, V1)
	VADDVV V1, V19, V19
	ATAP(0, X18, V2)
	ATAP(0, X20, V1)
	VADDVV V1, V2, V2
	ATAP(8, X18, V1)
	VADDVV V1, V2, V2
	ATAP(8, X20, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V19, V19
	VSLLVI $1, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V19, V19

	BTAP(4, X22, V20)
	BTAP(0, X22, V1)
	VADDVV V1, V20, V20
	BTAP(8, X22, V1)
	VADDVV V1, V20, V20
	BTAP(4, X21, V1)
	VADDVV V1, V20, V20
	BTAP(4, X23, V1)
	VADDVV V1, V20, V20
	BTAP(0, X21, V2)
	BTAP(0, X23, V1)
	VADDVV V1, V2, V2
	BTAP(8, X21, V1)
	VADDVV V1, V2, V2
	BTAP(8, X23, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V20, V20
	VSLLVI $1, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V20, V20

	STORE($9)

	SLLI $2, X10, X6
	ADD  X6, X18, X18
	ADD  X6, X19, X19
	ADD  X6, X20, X20
	ADD  X6, X21, X21
	ADD  X6, X22, X22
	ADD  X6, X23, X23
	ADD  X6, X12, X12
	SLLI $1, X10, X6
	ADD  X6, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop1_16

	RET

// func sgrFinish2Row_16RVV(tmp *int32, src *uint16, a0, a1 *int32, b0, b1 *int32, n int)
TEXT ·sgrFinish2Row_16RVV(SB), NOSPLIT, $0-56
	MOV tmp+0(FP), X12
	MOV src+8(FP), X11
	MOV a0+16(FP), X18
	MOV a1+24(FP), X19
	MOV b0+32(FP), X21
	MOV b1+40(FP), X22
	MOV n+48(FP), X13
	MOV $256, X14

loop2_16:
	VSETVLI X13, E32, M1, TA, MA, X10

	ATAP(4, X18, V19)
	ATAP(4, X19, V1)
	VADDVV V1, V19, V19
	ATAP(0, X18, V2)
	ATAP(0, X19, V1)
	VADDVV V1, V2, V2
	ATAP(8, X18, V1)
	VADDVV V1, V2, V2
	ATAP(8, X19, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V19, V3
	VSLLVI $1, V19, V19
	VADDVV V3, V19, V19
	VSLLVI $2, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V19, V19

	BTAP(4, X21, V20)
	BTAP(4, X22, V1)
	VADDVV V1, V20, V20
	BTAP(0, X21, V2)
	BTAP(0, X22, V1)
	VADDVV V1, V2, V2
	BTAP(8, X21, V1)
	VADDVV V1, V2, V2
	BTAP(8, X22, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V20, V3
	VSLLVI $1, V20, V20
	VADDVV V3, V20, V20
	VSLLVI $2, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V20, V20

	STORE($9)

	SLLI $2, X10, X6
	ADD  X6, X18, X18
	ADD  X6, X19, X19
	ADD  X6, X21, X21
	ADD  X6, X22, X22
	ADD  X6, X12, X12
	SLLI $1, X10, X6
	ADD  X6, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop2_16

	RET

// func sgrFinish2Mid_16RVV(tmp *int32, src *uint16, a1 *int32, b1 *int32, n int)
TEXT ·sgrFinish2Mid_16RVV(SB), NOSPLIT, $0-40
	MOV tmp+0(FP), X12
	MOV src+8(FP), X11
	MOV a1+16(FP), X19
	MOV b1+24(FP), X22
	MOV n+32(FP), X13
	MOV $128, X14

loop3_16:
	VSETVLI X13, E32, M1, TA, MA, X10

	ATAP(4, X19, V19)
	ATAP(0, X19, V2)
	ATAP(8, X19, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V19, V3
	VSLLVI $1, V19, V19
	VADDVV V3, V19, V19
	VSLLVI $2, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V19, V19

	BTAP(4, X22, V20)
	BTAP(0, X22, V2)
	BTAP(8, X22, V1)
	VADDVV V1, V2, V2
	VSLLVI $2, V20, V3
	VSLLVI $1, V20, V20
	VADDVV V3, V20, V20
	VSLLVI $2, V2, V3
	VADDVV V3, V2, V2
	VADDVV V2, V20, V20

	STORE($8)

	SLLI $2, X10, X6
	ADD  X6, X19, X19
	ADD  X6, X22, X22
	ADD  X6, X12, X12
	SLLI $1, X10, X6
	ADD  X6, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop3_16

	RET

// BOXV folds one more row into the running sums.
#define BOXV(RSQ, RSUM)              \
	VSETVLI X13, E32, M2, TA, MA, X10; \
	VLE32V  (RSQ), V4;           \
	VADDVV  V4, V0, V0;          \
	VLE32V  (RSUM), V6;          \
	VADDVV  V6, V2, V2;          \
	SLLI    $2, X10, X6;         \
	ADD     X6, RSQ, RSQ;        \
	ADD     X6, RSUM, RSUM

// func sgrBoxV3_16RVV(sq0, sq1, sq2 *int32, s0, s1, s2 *int32, sqOut *int32, sOut *int32, n int)
TEXT ·sgrBoxV3_16RVV(SB), NOSPLIT, $0-72
	MOV sq0+0(FP), X18
	MOV sq1+8(FP), X19
	MOV sq2+16(FP), X20
	MOV s0+24(FP), X21
	MOV s1+32(FP), X22
	MOV s2+40(FP), X23
	MOV sqOut+48(FP), X11
	MOV sOut+56(FP), X12
	MOV n+64(FP), X13

boxv3_16:
	VSETVLI X13, E32, M2, TA, MA, X10
	VLE32V  (X18), V0
	VLE32V  (X21), V2
	SLLI    $2, X10, X6
	ADD     X6, X18, X18
	ADD     X6, X21, X21
	BOXV(X19, X22)
	BOXV(X20, X23)
	VSETVLI X13, E32, M2, TA, MA, X10
	VSE32V  V0, (X11)
	VSE32V  V2, (X12)
	SLLI    $2, X10, X6
	ADD     X6, X11, X11
	ADD     X6, X12, X12
	SUB     X10, X13, X13
	BNE     X0, X13, boxv3_16

	RET

// func sgrBoxV5_16RVV(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int32, sqOut *int32, sOut *int32, n int)
TEXT ·sgrBoxV5_16RVV(SB), NOSPLIT, $0-104
	MOV sq0+0(FP), X18
	MOV sq1+8(FP), X19
	MOV sq2+16(FP), X20
	MOV sq3+24(FP), X24
	MOV sq4+32(FP), X25
	MOV s0+40(FP), X21
	MOV s1+48(FP), X22
	MOV s2+56(FP), X23
	MOV s3+64(FP), X26
	MOV s4+72(FP), X28
	MOV sqOut+80(FP), X11
	MOV sOut+88(FP), X12
	MOV n+96(FP), X13

boxv5_16:
	VSETVLI X13, E32, M2, TA, MA, X10
	VLE32V  (X18), V0
	VLE32V  (X21), V2
	SLLI    $2, X10, X6
	ADD     X6, X18, X18
	ADD     X6, X21, X21
	BOXV(X19, X22)
	BOXV(X20, X23)
	BOXV(X24, X26)
	BOXV(X25, X28)
	VSETVLI X13, E32, M2, TA, MA, X10
	VSE32V  V0, (X11)
	VSE32V  V2, (X12)
	SLLI    $2, X10, X6
	ADD     X6, X11, X11
	ADD     X6, X12, X12
	SUB     X10, X13, X13
	BNE     X0, X13, boxv5_16

	RET

// BOXH folds one more sample of the horizontal window in.
#define BOXH(OFF)                \
	ADD      $OFF, X11, X6;  \
	VSETVLI  X13, E16, MF2, TA, MA, X10; \
	VLE16V   (X6), V5;       \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	VZEXTVF2 V5, V3;         \
	VADDVV   V3, V0, V0;     \
	VMULVV   V3, V3, V3;     \
	VADDVV   V3, V2, V2

// BOXHSTORE narrows the running sum and stores both rows.
#define BOXHSTORE                \
	VSE32V   V2, (X12);      \
	VSE32V   V0, (X14);      \
	SLLI     $1, X10, X6;    \
	ADD      X6, X11, X11;   \
	SLLI     $2, X10, X6;    \
	ADD      X6, X14, X14;   \
	ADD      X6, X12, X12;   \
	SUB      X10, X13, X13

// func sgrBoxH3_16RVV(sumsq *int32, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH3_16RVV(SB), NOSPLIT, $0-32
	MOV sumsq+0(FP), X12
	MOV sum+8(FP), X14
	MOV src+16(FP), X11
	MOV n+24(FP), X13

boxh3_16:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VMVVX    X0, V0
	VMVVX    X0, V2
	BOXH(-2)
	BOXH(0)
	BOXH(2)
	BOXHSTORE
	BNE      X0, X13, boxh3_16

	RET

// func sgrBoxH5_16RVV(sumsq *int32, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH5_16RVV(SB), NOSPLIT, $0-32
	MOV sumsq+0(FP), X12
	MOV sum+8(FP), X14
	MOV src+16(FP), X11
	MOV n+24(FP), X13

boxh5_16:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VMVVX    X0, V0
	VMVVX    X0, V2
	BOXH(-4)
	BOXH(-2)
	BOXH(0)
	BOXH(2)
	BOXH(4)
	BOXHSTORE
	BNE      X0, X13, boxh5_16

	RET

// func sgrCalcAB_16RVV(aa *int32, bb *int32, tab *[256]uint32, n int, s, mul, oneByX uint32)
TEXT ·sgrCalcAB_16RVV(SB), NOSPLIT, $0-60
	MOV  aa+0(FP), X12
	MOV  bb+8(FP), X14
	MOV  tab+16(FP), X15
	MOV  n+24(FP), X13
	MOVW s+32(FP), X16
	MOVW mul+36(FP), X17
	MOVW oneByX+40(FP), X18
	MOV  $(1 << 19), X19
	MOV  $255, X20
	MOV  $(1 << 11), X21
	MOVW rndA+44(FP), X22
	MOVW shA+48(FP), X23
	MOVW rndB+52(FP), X24
	MOVW shB+56(FP), X25

calcab_16:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VLE32V   (X12), V0
	VLE32V   (X14), V1

	VADDVX   X22, V0, V2
	VSRAVX   X23, V2, V2
	VADDVX   X24, V1, V3
	VSRAVX   X25, V3, V3

	VMULVX   X17, V2, V2
	VMULVV   V3, V3, V3
	VSUBVV   V3, V2, V2
	VMAXVX   X0, V2, V2
	VMULVX   X16, V2, V2
	VADDVX   X19, V2, V2
	VSRLVI   $20, V2, V2
	VMINUVX  X20, V2, V2
	VSLLVI   $2, V2, V2
	VLUXEI32V (X15), V2, V4

	VMULVV   V4, V1, V6
	VMULVX   X18, V6, V6
	VADDVX   X21, V6, V6
	VSRLVI   $12, V6, V6
	VSE32V   V6, (X12)
	VSE32V   V4, (X14)

	SLLI $2, X10, X6
	ADD  X6, X12, X12
	ADD  X6, X14, X14
	SUB  X10, X13, X13
	BNE  X0, X13, calcab_16

	RET

// WEND rounds the weighted sum, folds it into the destination and stores it.
#define WEND                     \
	VADDVX   X19, V0, V0;    \
	VSRAVI   $11, V0, V0;    \
	VSETVLI  X13, E16, MF2, TA, MA, X10; \
	VLE16V   (X12), V5;      \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	VZEXTVF2 V5, V6;         \
	VADDVV   V6, V0, V0;     \
	VMAXVX   X0, V0, V0;     \
	VMINVX   X20, V0, V0;    \
	VSETVLI  X13, E16, MF2, TA, MA, X10; \
	VNSRLWI  $0, V0, V1;     \
	VSE16V   V1, (X12);      \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	SLLI     $1, X10, X6;    \
	ADD      X6, X12, X12;   \
	SLLI     $2, X10, X6;    \
	ADD      X6, X14, X14;   \
	SUB      X10, X13, X13

// func sgrWeight1_16RVV(dst *uint16, t1 *int32, n int, w1 int32)
TEXT ·sgrWeight1_16RVV(SB), NOSPLIT, $0-32
	MOV  dst+0(FP), X12
	MOV  t1+8(FP), X14
	MOV  n+16(FP), X13
	MOVW w1+24(FP), X17
	MOV  $1024, X19
	MOVW bitdepthMax+28(FP), X20

weight1_16:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VLE32V   (X14), V0
	VMULVX   X17, V0, V0
	WEND
	BNE      X0, X13, weight1_16

	RET

// func sgrWeight2_16RVV(dst *uint16, t1, t2 *int32, n int, w0, w1 int32)
TEXT ·sgrWeight2_16RVV(SB), NOSPLIT, $0-44
	MOV  dst+0(FP), X12
	MOV  t1+8(FP), X14
	MOV  t2+16(FP), X15
	MOV  n+24(FP), X13
	MOVW w0+32(FP), X18
	MOVW w1+36(FP), X17
	MOV  $1024, X19
	MOVW bitdepthMax+40(FP), X20

weight2_16:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VLE32V   (X14), V0
	VMULVX   X18, V0, V0
	VLE32V   (X15), V3
	VMULVX   X17, V3, V3
	VADDVV   V3, V0, V0
	SLLI     $2, X10, X6
	ADD      X6, X15, X15
	WEND
	BNE      X0, X13, weight2_16

	RET
