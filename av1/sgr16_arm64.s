//go:build arm64 && !noasm

#include "textflag.h"

#define MULS(Vd, Vn, Vm)   WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS(Vd, Vn, Vm)  WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm)  WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLS(Vd, Vn, Vm)  WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHR11(Vd, Vn)     WORD $(0x4F000400 | ((64 - 11) << 16) | ((Vn) << 5) | (Vd))
#define SSHR9(Vd, Vn)      WORD $(0x4F000400 | ((64 - 9) << 16) | ((Vn) << 5) | (Vd))
#define SSHR8(Vd, Vn)      WORD $(0x4F000400 | ((64 - 8) << 16) | ((Vn) << 5) | (Vd))
#define XTNS(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))
#define UMINS(Vd, Vn, Vm)  WORD $(0x6EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define USHR20(Vd, Vn)     WORD $(0x6F000400 | ((64 - 20) << 16) | ((Vn) << 5) | (Vd))
#define USHR12(Vd, Vn)     WORD $(0x6F000400 | ((64 - 12) << 16) | ((Vn) << 5) | (Vd))
#define SMAXZ(Vd, Vn)      WORD $(0x4EA06400 | (28 << 16) | ((Vn) << 5) | (Vd))

// At high bit depth both rows are int32, so the b row loads the same way the a
// row does and the widening the eight bit kernels need disappears.

#define AROW16(REG)                  \
	VLD1 (REG), [V0.S4, V1.S4];  \
	VEXT $4, V1.B16, V0.B16, V6.B16; \
	VEXT $8, V1.B16, V0.B16, V7.B16

#define BROW16(REG)                    \
	VLD1 (REG), [V10.S4, V11.S4];  \
	VMOV V10.B16, V20.B16;         \
	VEXT $4, V11.B16, V10.B16, V21.B16; \
	VEXT $8, V11.B16, V10.B16, V22.B16

#define PIXEL16                    \
	VLD1   (R1), [V29.H4];     \
	VUSHLL $0, V29.H4, V30.S4

#define SCALE43(ACC, T)              \
	VSHL $2, ACC.S4, ACC.S4;     \
	VSHL $1, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

#define SCALE65(ACC, T)              \
	VSHL $2, ACC.S4, V19.S4;     \
	VSHL $1, ACC.S4, ACC.S4;     \
	VADD V19.S4, ACC.S4, ACC.S4; \
	VSHL $2, T.S4, V19.S4;       \
	VADD V19.S4, T.S4, T.S4;     \
	VADD T.S4, ACC.S4, ACC.S4

#define STORE16(SHIFT)               \
	MULS(31, 31, 30);            \
	VSUB V31.S4, V28.S4, V28.S4; \
	VADD V27.S4, V28.S4, V28.S4; \
	SHIFT(28, 28);               \
	VST1 [V28.S4], (R0);         \
	ADD  $16, R0, R0;            \
	ADD  $8, R1, R1;             \
	SUBS $4, R10, R10

// func sgrFinish1_16NEON(tmp *int32, src *uint16, a0, a1, a2, b0, b1, b2 *int32, n int)
TEXT ·sgrFinish1_16NEON(SB), NOSPLIT, $0-72
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

loop1_16:
	AROW16(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW16(R3)
	VADD V0.S4, V28.S4, V28.S4
	VADD V6.S4, V28.S4, V28.S4
	VADD V7.S4, V28.S4, V28.S4
	AROW16(R4)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE43(V28, V18)

	BROW16(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW16(R6)
	VADD V20.S4, V31.S4, V31.S4
	VADD V21.S4, V31.S4, V31.S4
	VADD V22.S4, V31.S4, V31.S4
	BROW16(R7)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE43(V31, V26)

	PIXEL16
	STORE16(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $16, R4, R4
	ADD $16, R5, R5
	ADD $16, R6, R6
	ADD $16, R7, R7
	BNE loop1_16

	RET

// func sgrFinish2Row_16NEON(tmp *int32, src *uint16, a0, a1, b0, b1 *int32, n int)
TEXT ·sgrFinish2Row_16NEON(SB), NOSPLIT, $0-56
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a0+16(FP), R2
	MOVD a1+24(FP), R3
	MOVD b0+32(FP), R5
	MOVD b1+40(FP), R6
	MOVD n+48(FP), R10
	MOVD $256, R11
	VDUP R11, V27.S4

loop2_16:
	AROW16(R2)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	AROW16(R3)
	VADD V6.S4, V28.S4, V28.S4
	VADD V0.S4, V18.S4, V18.S4
	VADD V7.S4, V18.S4, V18.S4
	SCALE65(V28, V18)

	BROW16(R5)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	BROW16(R6)
	VADD V21.S4, V31.S4, V31.S4
	VADD V20.S4, V26.S4, V26.S4
	VADD V22.S4, V26.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL16
	STORE16(SSHR9)

	ADD $16, R2, R2
	ADD $16, R3, R3
	ADD $16, R5, R5
	ADD $16, R6, R6
	BNE loop2_16

	RET

// func sgrFinish2Mid_16NEON(tmp *int32, src *uint16, a1, b1 *int32, n int)
TEXT ·sgrFinish2Mid_16NEON(SB), NOSPLIT, $0-40
	MOVD tmp+0(FP), R0
	MOVD src+8(FP), R1
	MOVD a1+16(FP), R3
	MOVD b1+24(FP), R6
	MOVD n+32(FP), R10
	MOVD $128, R11
	VDUP R11, V27.S4

loop3_16:
	AROW16(R3)
	VMOV V6.B16, V28.B16
	VADD V7.S4, V0.S4, V18.S4
	SCALE65(V28, V18)

	BROW16(R6)
	VMOV V21.B16, V31.B16
	VADD V22.S4, V20.S4, V26.S4
	SCALE65(V31, V26)

	PIXEL16
	STORE16(SSHR8)

	ADD $16, R3, R3
	ADD $16, R6, R6
	BNE loop3_16

	RET

#define BOXV16(RSQ, RSUM)            \
	VLD1 (RSQ), [V4.S4, V5.S4];  \
	VADD V4.S4, V0.S4, V0.S4;    \
	VADD V5.S4, V1.S4, V1.S4;    \
	VLD1 (RSUM), [V6.S4, V7.S4]; \
	VADD V6.S4, V2.S4, V2.S4;    \
	VADD V7.S4, V3.S4, V3.S4;    \
	ADD  $32, RSQ, RSQ;          \
	ADD  $32, RSUM, RSUM

// func sgrBoxV3_16NEON(sq0, sq1, sq2, s0, s1, s2, sqOut, sOut *int32, n int)
TEXT ·sgrBoxV3_16NEON(SB), NOSPLIT, $0-72
	MOVD sq0+0(FP), R2
	MOVD sq1+8(FP), R3
	MOVD sq2+16(FP), R4
	MOVD s0+24(FP), R5
	MOVD s1+32(FP), R6
	MOVD s2+40(FP), R7
	MOVD sqOut+48(FP), R0
	MOVD sOut+56(FP), R1
	MOVD n+64(FP), R10

boxv3_16:
	VLD1 (R2), [V0.S4, V1.S4]
	VLD1 (R5), [V2.S4, V3.S4]
	ADD  $32, R2, R2
	ADD  $32, R5, R5
	BOXV16(R3, R6)
	BOXV16(R4, R7)
	VST1 [V0.S4, V1.S4], (R0)
	VST1 [V2.S4, V3.S4], (R1)
	ADD  $32, R0, R0
	ADD  $32, R1, R1
	SUBS $8, R10, R10
	BNE  boxv3_16

	RET

// func sgrBoxV5_16NEON(sq0, sq1, sq2, sq3, sq4, s0, s1, s2, s3, s4, sqOut, sOut *int32, n int)
TEXT ·sgrBoxV5_16NEON(SB), NOSPLIT, $0-104
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

boxv5_16:
	VLD1 (R2), [V0.S4, V1.S4]
	VLD1 (R5), [V2.S4, V3.S4]
	ADD  $32, R2, R2
	ADD  $32, R5, R5
	BOXV16(R3, R6)
	BOXV16(R4, R7)
	BOXV16(R11, R13)
	BOXV16(R12, R14)
	VST1 [V0.S4, V1.S4], (R0)
	VST1 [V2.S4, V3.S4], (R1)
	ADD  $32, R0, R0
	ADD  $32, R1, R1
	SUBS $8, R10, R10
	BNE  boxv5_16

	RET

#define BOXH16(RS)                \
	VLD1 (RS), [V3.H4];       \
	VUSHLL $0, V3.H4, V3.S4;  \
	VADD V3.S4, V0.S4, V0.S4; \
	MULS(3, 3, 3);            \
	VADD V3.S4, V2.S4, V2.S4

// func sgrBoxH3_16NEON(sumsq, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH3_16NEON(SB), NOSPLIT, $0-32
	MOVD sumsq+0(FP), R0
	MOVD sum+8(FP), R1
	MOVD src+16(FP), R2
	MOVD n+24(FP), R10

boxh3_16:
	SUB    $2, R2, R3
	VLD1   (R3), [V0.H4]
	VUSHLL $0, V0.H4, V0.S4
	MULS(2, 0, 0)
	BOXH16(R2)
	ADD    $2, R2, R3
	BOXH16(R3)
	VST1   [V0.S4], (R1)
	VST1   [V2.S4], (R0)
	ADD    $8, R2, R2
	ADD    $16, R1, R1
	ADD    $16, R0, R0
	SUBS   $4, R10, R10
	BNE    boxh3_16

	RET

// func sgrBoxH5_16NEON(sumsq, sum *int32, src *uint16, n int)
TEXT ·sgrBoxH5_16NEON(SB), NOSPLIT, $0-32
	MOVD sumsq+0(FP), R0
	MOVD sum+8(FP), R1
	MOVD src+16(FP), R2
	MOVD n+24(FP), R10

boxh5_16:
	SUB    $4, R2, R3
	VLD1   (R3), [V0.H4]
	VUSHLL $0, V0.H4, V0.S4
	MULS(2, 0, 0)
	SUB    $2, R2, R3
	BOXH16(R3)
	BOXH16(R2)
	ADD    $2, R2, R3
	BOXH16(R3)
	ADD    $4, R2, R3
	BOXH16(R3)
	VST1   [V0.S4], (R1)
	VST1   [V2.S4], (R0)
	ADD    $8, R2, R2
	ADD    $16, R1, R1
	ADD    $16, R0, R0
	SUBS   $4, R10, R10
	BNE    boxh5_16

	RET

// func sgrCalcAB_16NEON(aa, bb *int32, tab *[256]uint32, n int, s, mul, oneByX uint32, rndA, shA, rndB, shB int32)
TEXT ·sgrCalcAB_16NEON(SB), NOSPLIT, $0-60
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
	MOVW  rndA+44(FP), R3
	VDUP  R3, V23.S4
	MOVW  shA+48(FP), R3
	NEG   R3, R3
	VDUP  R3, V24.S4
	MOVW  rndB+52(FP), R3
	VDUP  R3, V25.S4
	MOVW  shB+56(FP), R3
	NEG   R3, R3
	VDUP  R3, V26.S4
	VMOVI $0, V28.B16

calcab_16:
	VLD1 (R0), [V0.S4]
	VLD1 (R1), [V1.S4]
	VADD V23.S4, V0.S4, V2.S4
	SSHLS(2, 2, 24)
	VADD V25.S4, V1.S4, V3.S4
	SSHLS(3, 3, 26)

	MULS(2, 2, 17)
	MULS(3, 3, 3)
	VSUB V3.S4, V2.S4, V2.S4
	SMAXZ(2, 2)
	MULS(2, 2, 16)
	VADD V19.S4, V2.S4, V2.S4
	USHR20(2, 2)
	UMINS(2, 2, 20)

	VMOV  V2.S[0], R3
	MOVWU (R2)(R3<<2), R4
	VMOV  R4, V4.S[0]
	VMOV  V2.S[1], R3
	MOVWU (R2)(R3<<2), R4
	VMOV  R4, V4.S[1]
	VMOV  V2.S[2], R3
	MOVWU (R2)(R3<<2), R4
	VMOV  R4, V4.S[2]
	VMOV  V2.S[3], R3
	MOVWU (R2)(R3<<2), R4
	VMOV  R4, V4.S[3]

	MULS(6, 4, 1)
	MULS(6, 6, 18)
	VADD   V21.S4, V6.S4, V6.S4
	USHR12(6, 6)
	VST1   [V6.S4], (R0)
	VST1   [V4.S4], (R1)

	ADD  $16, R0, R0
	ADD  $16, R1, R1
	SUBS $4, R10, R10
	BNE  calcab_16

	RET

#define WEND16                     \
	VADD V24.S4, V0.S4, V0.S4; \
	SSHR11(0, 0);              \
	VLD1 (R0), [V1.H4];        \
	VUSHLL $0, V1.H4, V1.S4;   \
	VADD V1.S4, V0.S4, V0.S4;  \
	SMAXS(0, 0, 25);           \
	SMINS(0, 0, 26);           \
	XTNS(0, 0);                \
	VST1 [V0.D1], (R0);        \
	ADD  $8, R0, R0;           \
	ADD  $16, R1, R1;          \
	SUBS $4, R10, R10

// func sgrWeight1_16NEON(dst *uint16, t1 *int32, n int, w1, bitdepthMax int32)
TEXT ·sgrWeight1_16NEON(SB), NOSPLIT, $0-32
	MOVD  dst+0(FP), R0
	MOVD  t1+8(FP), R1
	MOVD  n+16(FP), R10
	MOVWU w1+24(FP), R3
	VDUP  R3, V22.S4
	MOVD  $1024, R3
	VDUP  R3, V24.S4
	VMOVI $0, V25.B16
	MOVW  bitdepthMax+28(FP), R3
	VDUP  R3, V26.S4

weight1_16:
	VLD1 (R1), [V0.S4]
	MULS(0, 0, 22)
	WEND16
	BNE  weight1_16

	RET

// func sgrWeight2_16NEON(dst *uint16, t1, t2 *int32, n int, w0, w1, bitdepthMax int32)
TEXT ·sgrWeight2_16NEON(SB), NOSPLIT, $0-44
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
	MOVW  bitdepthMax+40(FP), R3
	VDUP  R3, V26.S4

weight2_16:
	VLD1 (R1), [V0.S4]
	MULS(0, 0, 23)
	VLD1 (R2), [V2.S4]
	MULS(2, 2, 22)
	VADD V2.S4, V0.S4, V0.S4
	ADD  $16, R2, R2
	WEND16
	BNE  weight2_16

	RET
