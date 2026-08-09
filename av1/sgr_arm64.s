//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of these.
#define MULS(Vd, Vn, Vm)   WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS(Vd, Vn, Vm)  WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm)  WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHR11(Vd, Vn)     WORD $(0x4F000400 | ((64 - 11) << 16) | ((Vn) << 5) | (Vd))
#define XTNB(Vd, Vn)       WORD $(0x0E212800 | ((Vn) << 5) | (Vd))
#define SSHLL(Vd, Vn)      WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHR9(Vd, Vn)      WORD $(0x4F000400 | ((64 - 9) << 16) | ((Vn) << 5) | (Vd))
#define SSHR8(Vd, Vn)      WORD $(0x4F000400 | ((64 - 8) << 16) | ((Vn) << 5) | (Vd))
#define XTNS(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))

// The a rows are int32 and the b rows int16, three taps apart in each. A pair
// of registers holds enough of a row that the taps come from VEXT rather than
// three loads.

// AROW loads one int32 row into V0:V1 and leaves its three taps in V0, V6, V7.
#define AROW(REG)                    \
	VLD1 (REG), [V0.S4, V1.S4];  \
	VEXT $4, V1.B16, V0.B16, V6.B16; \
	VEXT $8, V1.B16, V0.B16, V7.B16

// BROW loads one int16 row and widens its three taps into V20, V21, V22.
#define BROW(REG)                \
	VLD1 (REG), [V10.H8];    \
	SSHLL(20, 10);           \
	VEXT $2, V10.B16, V10.B16, V21.B16; \
	SSHLL(21, 21);           \
	VEXT $4, V10.B16, V10.B16, V22.B16; \
	SSHLL(22, 22)

// PIXEL widens four source samples into V30.
#define PIXEL                       \
	MOVWU  (R1), R11;           \
	VMOV   R11, V29.S[0];       \
	VUSHLL $0, V29.B8, V29.H8;  \
	VUSHLL $0, V29.H4, V30.S4

// SCALE43 turns the four and three weighted sums in ACC and T into ACC.
#define SCALE43(ACC, T)              \
	VSHL $2, ACC.S4, ACC.S4;     \
	VSHL $1, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

// SCALE65 does the same for the six and five weighted sums.
#define SCALE65(ACC, T)              \
	VSHL $2, ACC.S4, V19.S4;     \
	VSHL $1, ACC.S4, ACC.S4;     \
	VADD V19.S4, ACC.S4, ACC.S4; \
	VSHL $2, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

// STORE finishes one group of four: subtract, round, shift and narrow.
#define STORE(SHIFT)                 \
	MULS(31, 31, 30);            \
	VSUB V31.S4, V28.S4, V28.S4; \
	VADD V27.S4, V28.S4, V28.S4; \
	SHIFT(28, 28);               \
	XTNS(28, 28);                \
	VST1 [V28.D1], (R0);         \
	ADD  $8, R0, R0;             \
	ADD  $4, R1, R1;             \
	SUBS $4, R10, R10

// func sgrFinish1NEON(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)
TEXT ·sgrFinish1NEON(SB), NOSPLIT, $0-72
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a0+16(FP), R2
	MOVD a1+24(FP), R3
	MOVD a2+32(FP), R4
	MOVD b0+40(FP), R5
	MOVD b1+48(FP), R6
	MOVD b2+56(FP), R7
	MOVD n+64(FP), R10
	MOVD $256, R11
	VDUP R11, V27.S4

loop1:
	AROW(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW(R3)
	VADD V0.S4, V28.S4, V28.S4
	VADD V6.S4, V28.S4, V28.S4
	VADD V7.S4, V28.S4, V28.S4
	AROW(R4)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE43(V28, V18)

	BROW(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW(R6)
	VADD V20.S4, V31.S4, V31.S4
	VADD V21.S4, V31.S4, V31.S4
	VADD V22.S4, V31.S4, V31.S4
	BROW(R7)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE43(V31, V26)

	PIXEL
	STORE(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $16, R4, R4
	ADD $8, R5, R5
	ADD $8, R6, R6
	ADD $8, R7, R7
	BNE loop1

	RET

// func sgrFinish2RowNEON(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)
TEXT ·sgrFinish2RowNEON(SB), NOSPLIT, $0-56
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a0+16(FP), R2
	MOVD a1+24(FP), R3
	MOVD b0+32(FP), R5
	MOVD b1+40(FP), R6
	MOVD n+48(FP), R10
	MOVD $256, R11
	VDUP R11, V27.S4

loop2:
	AROW(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW(R3)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE65(V28, V18)

	BROW(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW(R6)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL
	STORE(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $8, R5, R5
	ADD $8, R6, R6
	BNE loop2

	RET

// func sgrFinish2MidNEON(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)
TEXT ·sgrFinish2MidNEON(SB), NOSPLIT, $0-40
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a1+16(FP), R3
	MOVD b1+24(FP), R6
	MOVD n+32(FP), R10
	MOVD $128, R11
	VDUP R11, V27.S4

loop3:
	AROW(R3)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	SCALE65(V28, V18)

	BROW(R6)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL
	STORE(SSHR8)

	ADD $16, R3, R3
	ADD $8, R6, R6
	BNE loop3

	RET

// BOXV folds one more row into the running sums.
#define BOXV(RSQ, RSUM)              \
	VLD1 (RSQ), [V4.S4, V5.S4];  \
	VADD V4.S4, V0.S4, V0.S4;    \
	VADD V5.S4, V1.S4, V1.S4;    \
	VLD1 (RSUM), [V6.H8];        \
	VADD V6.H8, V2.H8, V2.H8;    \
	ADD  $32, RSQ, RSQ;          \
	ADD  $16, RSUM, RSUM

// func sgrBoxV3NEON(sq0, sq1, sq2 *int32, s0, s1, s2 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV3NEON(SB), NOSPLIT, $0-72
	MOVD sq0+0(FP), R2
	MOVD sq1+8(FP), R3
	MOVD sq2+16(FP), R4
	MOVD s0+24(FP), R5
	MOVD s1+32(FP), R6
	MOVD s2+40(FP), R7
	MOVD sqOut+48(FP), R0
	MOVD sOut+56(FP), R1
	MOVD n+64(FP), R10

boxv3:
	VLD1 (R2), [V0.S4, V1.S4]
	VLD1 (R5), [V2.H8]
	ADD  $32, R2, R2
	ADD  $16, R5, R5
	BOXV(R3, R6)
	BOXV(R4, R7)
	VST1 [V0.S4, V1.S4], (R0)
	VST1 [V2.H8], (R1)
	ADD  $32, R0, R0
	ADD  $16, R1, R1
	SUBS $8, R10, R10
	BNE  boxv3

	RET

// func sgrBoxV5NEON(sq0, sq1, sq2, sq3, sq4 *int32, s0, s1, s2, s3, s4 *int16, sqOut *int32, sOut *int16, n int)
TEXT ·sgrBoxV5NEON(SB), NOSPLIT, $0-104
	MOVD sq0+0(FP), R2
	MOVD sq1+8(FP), R3
	MOVD sq2+16(FP), R4
	MOVD sq3+24(FP), R11
	MOVD sq4+32(FP), R12
	MOVD s0+40(FP), R5
	MOVD s1+48(FP), R6
	MOVD s2+56(FP), R7
	MOVD s3+64(FP), R13
	MOVD s4+72(FP), R14
	MOVD sqOut+80(FP), R0
	MOVD sOut+88(FP), R1
	MOVD n+96(FP), R10

boxv5:
	VLD1 (R2), [V0.S4, V1.S4]
	VLD1 (R5), [V2.H8]
	ADD  $32, R2, R2
	ADD  $16, R5, R5
	BOXV(R3, R6)
	BOXV(R4, R7)
	BOXV(R11, R13)
	BOXV(R12, R14)
	VST1 [V0.S4, V1.S4], (R0)
	VST1 [V2.H8], (R1)
	ADD  $32, R0, R0
	ADD  $16, R1, R1
	SUBS $8, R10, R10
	BNE  boxv5

	RET

// BOXH folds one more sample of the horizontal window in.
#define BOXH(RS)                  \
	VLD1 (RS), [V3.B8];       \
	VUSHLL $0, V3.B8, V3.H8;  \
	VUSHLL $0, V3.H4, V3.S4;  \
	VADD V3.S4, V0.S4, V0.S4; \
	MULS(3, 3, 3);            \
	VADD V3.S4, V2.S4, V2.S4

// func sgrBoxH3NEON(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH3NEON(SB), NOSPLIT, $0-32
	MOVD sumsq+0(FP), R0
	MOVD sum+8(FP), R1
	MOVD src+16(FP), R2
	MOVD n+24(FP), R10

boxh3:
	SUB    $1, R2, R3
	VLD1   (R3), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VUSHLL $0, V0.H4, V0.S4
	MULS(2, 0, 0)
	BOXH(R2)
	ADD    $1, R2, R3
	BOXH(R3)
	XTNS(1, 0)
	VST1   [V1.D1], (R1)
	VST1   [V2.S4], (R0)
	ADD    $4, R2, R2
	ADD    $8, R1, R1
	ADD    $16, R0, R0
	SUBS   $4, R10, R10
	BNE    boxh3

	RET

// func sgrBoxH5NEON(sumsq *int32, sum *int16, src *uint8, n int)
TEXT ·sgrBoxH5NEON(SB), NOSPLIT, $0-32
	MOVD sumsq+0(FP), R0
	MOVD sum+8(FP), R1
	MOVD src+16(FP), R2
	MOVD n+24(FP), R10

boxh5:
	SUB    $2, R2, R3
	VLD1   (R3), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VUSHLL $0, V0.H4, V0.S4
	MULS(2, 0, 0)
	SUB    $1, R2, R3
	BOXH(R3)
	BOXH(R2)
	ADD    $1, R2, R3
	BOXH(R3)
	ADD    $2, R2, R3
	BOXH(R3)
	XTNS(1, 0)
	VST1   [V1.D1], (R1)
	VST1   [V2.S4], (R0)
	ADD    $4, R2, R2
	ADD    $8, R1, R1
	ADD    $16, R0, R0
	SUBS   $4, R10, R10
	BNE    boxh5

	RET

#define UMINS(Vd, Vn, Vm) WORD $(0x6EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define USHR20(Vd, Vn)    WORD $(0x6F000400 | ((64 - 20) << 16) | ((Vn) << 5) | (Vd))
#define USHR12(Vd, Vn)    WORD $(0x6F000400 | ((64 - 12) << 16) | ((Vn) << 5) | (Vd))
#define SMAXZ(Vd, Vn)     WORD $(0x4EA06400 | (28 << 16) | ((Vn) << 5) | (Vd))

// func sgrCalcABNEON(aa *int32, bb *int16, tab *[256]uint32, n int, s, mul, oneByX uint32)
TEXT ·sgrCalcABNEON(SB), NOSPLIT, $0-44
	MOVD  aa+0(FP), R0
	MOVD  bb+8(FP), R1
	MOVD  tab+16(FP), R2
	MOVD  n+24(FP), R10
	MOVWU s+32(FP), R3
	VDUP  R3, V16.S4
	MOVWU mul+36(FP), R3
	VDUP  R3, V17.S4
	MOVWU oneByX+40(FP), R3
	VDUP  R3, V18.S4
	MOVD  $(1 << 19), R3
	VDUP  R3, V19.S4
	MOVD  $255, R3
	VDUP  R3, V20.S4
	MOVD  $(1 << 11), R3
	VDUP  R3, V21.S4
	VMOVI $0, V28.B16

calcab:
	VLD1   (R0), [V0.S4]
	VLD1   (R1), [V5.H8]
	SSHLL(1, 5)
	MULS(2, 0, 17)
	MULS(3, 1, 1)
	VSUB   V3.S4, V2.S4, V2.S4
	SMAXZ(2, 2)
	MULS(2, 2, 16)
	VADD   V19.S4, V2.S4, V2.S4
	USHR20(2, 2)
	UMINS(2, 2, 20)

	VMOV   V2.S[0], R3
	MOVWU  (R2)(R3<<2), R4
	VMOV   R4, V4.S[0]
	VMOV   V2.S[1], R3
	MOVWU  (R2)(R3<<2), R4
	VMOV   R4, V4.S[1]
	VMOV   V2.S[2], R3
	MOVWU  (R2)(R3<<2), R4
	VMOV   R4, V4.S[2]
	VMOV   V2.S[3], R3
	MOVWU  (R2)(R3<<2), R4
	VMOV   R4, V4.S[3]

	MULS(6, 4, 1)
	MULS(6, 6, 18)
	VADD   V21.S4, V6.S4, V6.S4
	USHR12(6, 6)
	VST1   [V6.S4], (R0)
	XTNS(7, 4)
	VST1   [V7.D1], (R1)

	ADD  $16, R0, R0
	ADD  $8, R1, R1
	SUBS $4, R10, R10
	BNE  calcab

	RET

// WEND rounds the weighted sum, folds it into the destination and stores it.
#define WEND                     \
	VADD V24.S4, V0.S4, V0.S4; \
	SSHR11(0, 0);            \
	MOVWU (R0), R4;          \
	VMOV R4, V1.S[0];        \
	VUSHLL $0, V1.B8, V1.H8; \
	VUSHLL $0, V1.H4, V1.S4; \
	VADD V1.S4, V0.S4, V0.S4; \
	SMAXS(0, 0, 25);         \
	SMINS(0, 0, 26);         \
	XTNS(0, 0);              \
	XTNB(0, 0);              \
	VMOV V0.S[0], R4;        \
	MOVW R4, (R0);           \
	ADD  $4, R0, R0;         \
	ADD  $8, R1, R1;         \
	SUBS $4, R10, R10

// func sgrWeight1NEON(dst *uint8, t1 *int16, n int, w1 int32)
TEXT ·sgrWeight1NEON(SB), NOSPLIT, $0-28
	MOVD  dst+0(FP), R0
	MOVD  t1+8(FP), R1
	MOVD  n+16(FP), R10
	MOVWU w1+24(FP), R3
	VDUP  R3, V22.S4
	MOVD  $1024, R3
	VDUP  R3, V24.S4
	VMOVI $0, V25.B16
	MOVD  $255, R3
	VDUP  R3, V26.S4

weight1:
	VLD1 (R1), [V5.H8]
	SSHLL(0, 5)
	MULS(0, 0, 22)
	WEND
	BNE  weight1

	RET

// func sgrWeight2NEON(dst *uint8, t1, t2 *int16, n int, w0, w1 int32)
TEXT ·sgrWeight2NEON(SB), NOSPLIT, $0-40
	MOVD  dst+0(FP), R0
	MOVD  t1+8(FP), R1
	MOVD  t2+16(FP), R2
	MOVD  n+24(FP), R10
	MOVWU w0+32(FP), R3
	VDUP  R3, V23.S4
	MOVWU w1+36(FP), R3
	VDUP  R3, V22.S4
	MOVD  $1024, R3
	VDUP  R3, V24.S4
	VMOVI $0, V25.B16
	MOVD  $255, R3
	VDUP  R3, V26.S4

weight2:
	VLD1 (R1), [V5.H8]
	SSHLL(0, 5)
	MULS(0, 0, 23)
	VLD1 (R2), [V6.H8]
	SSHLL(2, 6)
	MULS(2, 2, 22)
	VADD V2.S4, V0.S4, V0.S4
	ADD  $8, R2, R2
	WEND
	BNE  weight2

	RET
