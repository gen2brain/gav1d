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
