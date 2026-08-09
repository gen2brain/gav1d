//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// The a rows are int32 and the b rows int16, three taps apart in each, so a
// tap is just the row pointer plus a constant.

// ATAP loads one int32 tap.
#define ATAP(OFF, REG, VD)   \
	ADD    $OFF, REG, X6; \
	VLE32V (X6), VD

// BTAP loads one int16 tap and widens it.
#define BTAP(OFF, REG, VD)      \
	ADD     $OFF, REG, X6;  \
	VSETVLI X10, E16, MF2, TA, MA, X7; \
	VLE16V  (X6), V15;      \
	VSETVLI X10, E32, M1, TA, MA, X7; \
	VSEXTVF2 V15, VD

// STORE finishes one group: subtract, round, shift and narrow.
#define STORE(SHIFT)              \
	VSETVLI  X10, E8, MF4, TA, MA, X7; \
	VLE8V    (X11), V16;      \
	VSETVLI  X10, E32, M1, TA, MA, X7; \
	VZEXTVF4 V16, V17;        \
	VMULVV   V17, V20, V20;   \
	VSUBVV   V20, V19, V19;   \
	VADDVX   X14, V19, V19;   \
	VSRAVI   SHIFT, V19, V19; \
	VSETVLI  X10, E16, MF2, TA, MA, X7; \
	VNSRLWI  $0, V19, V18;    \
	VSE16V   V18, (X12);      \
	VSETVLI  X10, E32, M1, TA, MA, X7

// func sgrFinish1RVV(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)
TEXT ·sgrFinish1RVV(SB), NOSPLIT, $0-72
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

loop1:
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

	BTAP(2, X22, V20)
	BTAP(0, X22, V1)
	VADDVV V1, V20, V20
	BTAP(4, X22, V1)
	VADDVV V1, V20, V20
	BTAP(2, X21, V1)
	VADDVV V1, V20, V20
	BTAP(2, X23, V1)
	VADDVV V1, V20, V20
	BTAP(0, X21, V2)
	BTAP(0, X23, V1)
	VADDVV V1, V2, V2
	BTAP(4, X21, V1)
	VADDVV V1, V2, V2
	BTAP(4, X23, V1)
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
	SLLI $1, X10, X6
	ADD  X6, X21, X21
	ADD  X6, X22, X22
	ADD  X6, X23, X23
	ADD  X6, X12, X12
	ADD  X10, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop1

	RET

// func sgrFinish2RowRVV(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)
TEXT ·sgrFinish2RowRVV(SB), NOSPLIT, $0-56
	MOV tmp+0(FP), X12
	MOV src+8(FP), X11
	MOV a0+16(FP), X18
	MOV a1+24(FP), X19
	MOV b0+32(FP), X21
	MOV b1+40(FP), X22
	MOV n+48(FP), X13
	MOV $256, X14

loop2:
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

	BTAP(2, X21, V20)
	BTAP(2, X22, V1)
	VADDVV V1, V20, V20
	BTAP(0, X21, V2)
	BTAP(0, X22, V1)
	VADDVV V1, V2, V2
	BTAP(4, X21, V1)
	VADDVV V1, V2, V2
	BTAP(4, X22, V1)
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
	SLLI $1, X10, X6
	ADD  X6, X21, X21
	ADD  X6, X22, X22
	ADD  X6, X12, X12
	ADD  X10, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop2

	RET

// func sgrFinish2MidRVV(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)
TEXT ·sgrFinish2MidRVV(SB), NOSPLIT, $0-40
	MOV tmp+0(FP), X12
	MOV src+8(FP), X11
	MOV a1+16(FP), X19
	MOV b1+24(FP), X22
	MOV n+32(FP), X13
	MOV $128, X14

loop3:
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

	BTAP(2, X22, V20)
	BTAP(0, X22, V2)
	BTAP(4, X22, V1)
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
	SLLI $1, X10, X6
	ADD  X6, X22, X22
	ADD  X6, X12, X12
	ADD  X10, X11, X11
	SUB  X10, X13, X13
	BNE  X0, X13, loop3

	RET

// BOXV folds one more row into the running sums.
#define BOXV(RSQ, RSUM)              \
	VSETVLI X13, E32, M2, TA, MA, X10; \
	VLE32V  (RSQ), V4;           \
	VADDVV  V4, V0, V0;          \
	VSETVLI X13, E16, M1, TA, MA, X10; \
	VLE16V  (RSUM), V6;          \
	VADDVV  V6, V2, V2;          \
	SLLI    $2, X10, X6;         \
	ADD     X6, RSQ, RSQ;        \
	SLLI    $1, X10, X6;         \
	ADD     X6, RSUM, RSUM

// func sgrBoxV3RVV(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV3RVV(SB), NOSPLIT, $0-72
	MOV sq0+0(FP), X18
	MOV sq1+8(FP), X19
	MOV sq2+16(FP), X20
	MOV s0+24(FP), X21
	MOV s1+32(FP), X22
	MOV s2+40(FP), X23
	MOV sqOut+48(FP), X11
	MOV sOut+56(FP), X12
	MOV n+64(FP), X13

boxv3:
	VSETVLI X13, E32, M2, TA, MA, X10
	VLE32V  (X18), V0
	VSETVLI X13, E16, M1, TA, MA, X10
	VLE16V  (X21), V2
	SLLI    $2, X10, X6
	ADD     X6, X18, X18
	SLLI    $1, X10, X6
	ADD     X6, X21, X21
	BOXV(X19, X22)
	BOXV(X20, X23)
	VSETVLI X13, E32, M2, TA, MA, X10
	VSE32V  V0, (X11)
	VSETVLI X13, E16, M1, TA, MA, X10
	VSE16V  V2, (X12)
	SLLI    $2, X10, X6
	ADD     X6, X11, X11
	SLLI    $1, X10, X6
	ADD     X6, X12, X12
	SUB     X10, X13, X13
	BNE     X0, X13, boxv3

	RET

// func sgrBoxV5RVV(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV5RVV(SB), NOSPLIT, $0-104
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

boxv5:
	VSETVLI X13, E32, M2, TA, MA, X10
	VLE32V  (X18), V0
	VSETVLI X13, E16, M1, TA, MA, X10
	VLE16V  (X21), V2
	SLLI    $2, X10, X6
	ADD     X6, X18, X18
	SLLI    $1, X10, X6
	ADD     X6, X21, X21
	BOXV(X19, X22)
	BOXV(X20, X23)
	BOXV(X24, X26)
	BOXV(X25, X28)
	VSETVLI X13, E32, M2, TA, MA, X10
	VSE32V  V0, (X11)
	VSETVLI X13, E16, M1, TA, MA, X10
	VSE16V  V2, (X12)
	SLLI    $2, X10, X6
	ADD     X6, X11, X11
	SLLI    $1, X10, X6
	ADD     X6, X12, X12
	SUB     X10, X13, X13
	BNE     X0, X13, boxv5

	RET

// BOXH folds one more sample of the horizontal window in.
#define BOXH(OFF)                \
	ADD      $OFF, X11, X6;  \
	VSETVLI  X13, E8, MF4, TA, MA, X10; \
	VLE8V    (X6), V5;       \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	VZEXTVF4 V5, V3;         \
	VADDVV   V3, V0, V0;     \
	VMULVV   V3, V3, V3;     \
	VADDVV   V3, V2, V2

// BOXHSTORE narrows the running sum and stores both rows.
#define BOXHSTORE                \
	VSE32V   V2, (X12);      \
	VSETVLI  X13, E16, MF2, TA, MA, X10; \
	VNSRLWI  $0, V0, V1;     \
	VSE16V   V1, (X14);      \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	ADD      X10, X11, X11;  \
	SLLI     $1, X10, X6;    \
	ADD      X6, X14, X14;   \
	SLLI     $2, X10, X6;    \
	ADD      X6, X12, X12;   \
	SUB      X10, X13, X13

// func sgrBoxH3RVV(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH3RVV(SB), NOSPLIT, $0-32
	MOV sumsq+0(FP), X12
	MOV sum+8(FP), X14
	MOV src+16(FP), X11
	MOV n+24(FP), X13

boxh3:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VMVVX    X0, V0
	VMVVX    X0, V2
	BOXH(-1)
	BOXH(0)
	BOXH(1)
	BOXHSTORE
	BNE      X0, X13, boxh3

	RET

// func sgrBoxH5RVV(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH5RVV(SB), NOSPLIT, $0-32
	MOV sumsq+0(FP), X12
	MOV sum+8(FP), X14
	MOV src+16(FP), X11
	MOV n+24(FP), X13

boxh5:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VMVVX    X0, V0
	VMVVX    X0, V2
	BOXH(-2)
	BOXH(-1)
	BOXH(0)
	BOXH(1)
	BOXH(2)
	BOXHSTORE
	BNE      X0, X13, boxh5

	RET

// func sgrCalcABRVV(aa *int32, bb *int16, tab *[256]uint32, n int, s, mul, oneByX uint32)
TEXT ·sgrCalcABRVV(SB), NOSPLIT, $0-44
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

calcab:
	VSETVLI  X13, E32, M1, TA, MA, X10
	VLE32V   (X12), V0
	VSETVLI  X13, E16, MF2, TA, MA, X10
	VLE16V   (X14), V5
	VSETVLI  X13, E32, M1, TA, MA, X10
	VSEXTVF2 V5, V1

	VMULVX   X17, V0, V2
	VMULVV   V1, V1, V3
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
	VSETVLI  X13, E16, MF2, TA, MA, X10
	VNSRLWI  $0, V4, V7
	VSE16V   V7, (X14)
	VSETVLI  X13, E32, M1, TA, MA, X10

	SLLI $2, X10, X6
	ADD  X6, X12, X12
	SLLI $1, X10, X6
	ADD  X6, X14, X14
	SUB  X10, X13, X13
	BNE  X0, X13, calcab

	RET

// WEND rounds the weighted sum, folds it into the destination and stores it.
#define WEND                     \
	VADDVX   X19, V0, V0;    \
	VSRAVI   $11, V0, V0;    \
	VSETVLI  X13, E8, MF4, TA, MA, X10; \
	VLE8V    (X12), V5;      \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	VZEXTVF4 V5, V6;         \
	VADDVV   V6, V0, V0;     \
	VMAXVX   X0, V0, V0;     \
	VMINVX   X20, V0, V0;    \
	VSETVLI  X13, E16, MF2, TA, MA, X10; \
	VNSRLWI  $0, V0, V1;     \
	VSETVLI  X13, E8, MF4, TA, MA, X10; \
	VNSRLWI  $0, V1, V2;     \
	VSE8V    V2, (X12);      \
	VSETVLI  X13, E32, M1, TA, MA, X10; \
	ADD      X10, X12, X12;  \
	SLLI     $1, X10, X6;    \
	ADD      X6, X14, X14;   \
	SUB      X10, X13, X13

// func sgrWeight1RVV(dst *uint8, t1 *int16, n int, w1 int32)
TEXT ·sgrWeight1RVV(SB), NOSPLIT, $0-28
	MOV  dst+0(FP), X12
	MOV  t1+8(FP), X14
	MOV  n+16(FP), X13
	MOVW w1+24(FP), X17
	MOV  $1024, X19
	MOV  $255, X20

weight1:
	VSETVLI  X13, E16, MF2, TA, MA, X10
	VLE16V   (X14), V5
	VSETVLI  X13, E32, M1, TA, MA, X10
	VSEXTVF2 V5, V0
	VMULVX   X17, V0, V0
	WEND
	BNE      X0, X13, weight1

	RET

// func sgrWeight2RVV(dst *uint8, t1, t2 *int16, n int, w0, w1 int32)
TEXT ·sgrWeight2RVV(SB), NOSPLIT, $0-40
	MOV  dst+0(FP), X12
	MOV  t1+8(FP), X14
	MOV  t2+16(FP), X15
	MOV  n+24(FP), X13
	MOVW w0+32(FP), X18
	MOVW w1+36(FP), X17
	MOV  $1024, X19
	MOV  $255, X20

weight2:
	VSETVLI  X13, E16, MF2, TA, MA, X10
	VLE16V   (X14), V5
	VLE16V   (X15), V7
	VSETVLI  X13, E32, M1, TA, MA, X10
	VSEXTVF2 V5, V0
	VMULVX   X18, V0, V0
	VSEXTVF2 V7, V3
	VMULVX   X17, V3, V3
	VADDVV   V3, V0, V0
	SLLI     $1, X10, X6
	ADD      X6, X15, X15
	WEND
	BNE      X0, X13, weight2

	RET
