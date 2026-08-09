//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has none of the word-wide vector arithmetic, so the
// whole butterfly is hand encoded. VSRSHR carries the rounding term, which is
// why no rounding constant is loaded.
#define ADDS(Vd, Vn, Vm)  WORD $(0x4EA08400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SUBS(Vd, Vn, Vm)  WORD $(0x6EA08400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define MULS(Vd, Vn, Vm)  WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS(Vd, Vn, Vm) WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm) WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SRSHR12(Vd, Vn)   WORD $(0x4F342400 | ((Vn) << 5) | (Vd))
#define SRSHR11(Vd, Vn)   WORD $(0x4F352400 | ((Vn) << 5) | (Vd))
#define SRSHR8(Vd, Vn)    WORD $(0x4F382400 | ((Vn) << 5) | (Vd))

#define MACA(D, A, CA, B, CB, SH, T) \
	MULS(D, A, CA); \
	MULS(T, B, CB); \
	ADDS(D, D, T);  \
	SH(D, D)

#define MACS(D, A, CA, B, CB, SH, T) \
	MULS(D, A, CA); \
	MULS(T, B, CB); \
	SUBS(D, D, T);  \
	SH(D, D)

#define R181(D, A, B, OP) \
	OP(D, A, B);      \
	MULS(D, D, 16);   \
	SRSHR8(D, D)

#define CLIP(V)       \
	SMAXS(V, V, 14); \
	SMINS(V, V, 15)

#define DCT4(A0, A1, A2, A3, T0, T1, T2, T3) \
	R181(T0, A0, A2, ADDS);   \
	R181(T1, A0, A2, SUBS);   \
	MACA(T2, A1, 17, A3, 18, SRSHR12, T3); \
	SUBS(T2, T2, A3);         \
	MACS(T3, A3, 17, A1, 18, SRSHR12, A0); \
	ADDS(T3, T3, A1);         \
	ADDS(A0, T0, T3);         \
	CLIP(A0);                 \
	ADDS(A1, T1, T2);         \
	CLIP(A1);                 \
	SUBS(A2, T1, T2);         \
	CLIP(A2);                 \
	SUBS(A3, T0, T3);         \
	CLIP(A3)

// KONST loads the butterfly coefficients, which stay in registers throughout.
#define KONST                \
	MOVD $181, R8;       \
	VDUP R8, V16.S4;     \
	MOVD $1567, R8;      \
	VDUP R8, V17.S4;     \
	MOVD $312, R8;       \
	VDUP R8, V18.S4;     \
	MOVD $799, R8;       \
	VDUP R8, V19.S4;     \
	MOVD $79, R8;        \
	VDUP R8, V20.S4;     \
	MOVD $1703, R8;      \
	VDUP R8, V21.S4;     \
	MOVD $1138, R8;      \
	VDUP R8, V22.S4;     \
	MOVD $401, R8;       \
	VDUP R8, V23.S4;     \
	MOVD $20, R8;        \
	VDUP R8, V24.S4;     \
	MOVD $1583, R8;      \
	VDUP R8, V25.S4;     \
	MOVD $1299, R8;      \
	VDUP R8, V26.S4;     \
	MOVD $1931, R8;      \
	VDUP R8, V27.S4;     \
	MOVD $484, R8;       \
	VDUP R8, V28.S4;     \
	MOVD $176, R8;       \
	VDUP R8, V29.S4;     \
	MOVD $1189, R8;      \
	VDUP R8, V30.S4

#define SETUP              \
	MOVD tmp+0(FP), R0; \
	MOVD w+8(FP), R2;   \
	MOVW lo+16(FP), R8; \
	VDUP R8, V14.S4;    \
	MOVW hi+20(FP), R8; \
	VDUP R8, V15.S4;    \
	MOVD R2, R1;        \
	LSL  $2, R1, R1;    \
	ADD  R1, R1, R9;    \
	ADD  R1, R9, R3;    \
	ADD  R9, R9, R11;   \
	ADD  R11, R11, R12; \
	ADD  R11, R12, R13; \
	KONST

#define LD(BASE, OFF, V)  \
	ADD OFF, BASE, R10; \
	VLD1 (R10), [V]

#define ST(BASE, OFF, V)  \
	ADD OFF, BASE, R10; \
	VST1 [V], (R10)

// func itxColDct4NEON(tmp *int32, w int, lo, hi int32)
TEXT ·itxColDct4NEON(SB), NOSPLIT, $0-24
	SETUP

col4:
	MOVD R0, R4
	VLD1 (R4), [V0.S4]
	LD(R4, R1, V1.S4)
	LD(R4, R9, V2.S4)
	LD(R4, R3, V3.S4)
	DCT4(0, 1, 2, 3, 4, 5, 6, 7)
	VST1 [V0.S4], (R4)
	ST(R4, R1, V1.S4)
	ST(R4, R9, V2.S4)
	ST(R4, R3, V3.S4)

	ADD  $16, R0
	SUB  $4, R2
	CBNZ R2, col4

	RET

// ODD8 is the four-point odd half of an eight-point transform, reading the
// rows given and leaving t4, t5, t6 and t7 in V4, V8, V9 and V7.
#define ODD8 \
	MACA(8, 4, 19, 7, 20, SRSHR12, 12); \
	SUBS(8, 8, 7);                      \
	MACS(9, 6, 21, 5, 22, SRSHR11, 12); \
	MACA(10, 6, 22, 5, 21, SRSHR11, 12); \
	MACS(11, 7, 19, 4, 20, SRSHR12, 12); \
	ADDS(11, 11, 4);                    \
	ADDS(4, 8, 9);                      \
	CLIP(4);                            \
	SUBS(5, 8, 9);                      \
	CLIP(5);                            \
	ADDS(7, 11, 10);                    \
	CLIP(7);                            \
	SUBS(6, 11, 10);                    \
	CLIP(6);                            \
	R181(8, 6, 5, SUBS);                \
	R181(9, 6, 5, ADDS)

// func itxColDct8NEON(tmp *int32, w int, lo, hi int32)
TEXT ·itxColDct8NEON(SB), NOSPLIT, $0-24
	SETUP

col8:
	MOVD R0, R4
	ADD  R11, R0, R5

	VLD1 (R4), [V0.S4]
	LD(R4, R9, V1.S4)
	VLD1 (R5), [V2.S4]
	LD(R5, R9, V3.S4)
	DCT4(0, 1, 2, 3, 4, 5, 6, 7)

	LD(R4, R1, V4.S4)
	LD(R4, R3, V5.S4)
	LD(R5, R1, V6.S4)
	LD(R5, R3, V7.S4)
	ODD8

	ADDS(10, 0, 7)
	CLIP(10)
	VST1 [V10.S4], (R4)
	SUBS(10, 0, 7)
	CLIP(10)
	ST(R5, R3, V10.S4)
	ADDS(10, 1, 9)
	CLIP(10)
	ST(R4, R1, V10.S4)
	SUBS(10, 1, 9)
	CLIP(10)
	ST(R5, R9, V10.S4)
	ADDS(10, 2, 8)
	CLIP(10)
	ST(R4, R9, V10.S4)
	SUBS(10, 2, 8)
	CLIP(10)
	ST(R5, R1, V10.S4)
	ADDS(10, 3, 4)
	CLIP(10)
	ST(R4, R3, V10.S4)
	SUBS(10, 3, 4)
	CLIP(10)
	VST1 [V10.S4], (R5)

	ADD  $16, R0
	SUB  $4, R2
	CBNZ R2, col8

	RET

// func itxColDct16NEON(tmp *int32, w int, lo, hi int32)
TEXT ·itxColDct16NEON(SB), NOSPLIT, $0-24
	SETUP
	VEOR V31.B16, V31.B16, V31.B16

col16:
	MOVD R0, R4
	ADD  R11, R0, R5
	ADD  R12, R0, R6
	ADD  R13, R0, R7

	VLD1 (R4), [V0.S4]
	VLD1 (R5), [V1.S4]
	VLD1 (R6), [V2.S4]
	VLD1 (R7), [V3.S4]
	DCT4(0, 1, 2, 3, 4, 5, 6, 7)

	LD(R4, R9, V4.S4)
	LD(R5, R9, V5.S4)
	LD(R6, R9, V6.S4)
	LD(R7, R9, V7.S4)
	ODD8

	ADDS(10, 0, 7)
	CLIP(10)
	VST1 [V10.S4], (R4)
	SUBS(10, 0, 7)
	CLIP(10)
	ST(R7, R9, V10.S4)
	ADDS(10, 1, 9)
	CLIP(10)
	ST(R4, R9, V10.S4)
	SUBS(10, 1, 9)
	CLIP(10)
	VST1 [V10.S4], (R7)
	ADDS(10, 2, 8)
	CLIP(10)
	VST1 [V10.S4], (R5)
	SUBS(10, 2, 8)
	CLIP(10)
	ST(R6, R9, V10.S4)
	ADDS(10, 3, 4)
	CLIP(10)
	ST(R5, R9, V10.S4)
	SUBS(10, 3, 4)
	CLIP(10)
	VST1 [V10.S4], (R6)

	LD(R4, R1, V8.S4)
	LD(R7, R3, V9.S4)
	MACA(0, 8, 23, 9, 24, SRSHR12, 10)
	SUBS(0, 0, 9)
	MACS(7, 9, 23, 8, 24, SRSHR12, 10)
	ADDS(7, 7, 8)

	LD(R6, R1, V8.S4)
	LD(R5, R3, V9.S4)
	MACS(1, 8, 25, 9, 26, SRSHR11, 10)
	MACA(6, 8, 26, 9, 25, SRSHR11, 10)

	LD(R5, R1, V8.S4)
	LD(R6, R3, V9.S4)
	MACA(2, 8, 27, 9, 28, SRSHR12, 10)
	SUBS(2, 2, 9)
	MACS(5, 9, 27, 8, 28, SRSHR12, 10)
	ADDS(5, 5, 8)

	LD(R7, R1, V8.S4)
	LD(R4, R3, V9.S4)
	MULS(3, 8, 29)
	MULS(10, 9, 30)
	ADDS(3, 3, 10)
	SUBS(3, 31, 3)
	SRSHR12(3, 3)
	ADDS(3, 3, 8)
	MACS(4, 8, 30, 9, 29, SRSHR12, 10)
	ADDS(4, 4, 9)

	ADDS(8, 0, 1)
	CLIP(8)
	SUBS(9, 0, 1)
	CLIP(9)
	SUBS(10, 3, 2)
	CLIP(10)
	ADDS(0, 3, 2)
	CLIP(0)
	ADDS(1, 4, 5)
	CLIP(1)
	SUBS(2, 4, 5)
	CLIP(2)
	SUBS(3, 7, 6)
	CLIP(3)
	ADDS(4, 7, 6)
	CLIP(4)

	MACA(5, 3, 17, 9, 18, SRSHR12, 13)
	SUBS(5, 5, 9)
	MACS(6, 9, 17, 3, 18, SRSHR12, 13)
	ADDS(6, 6, 3)
	MACS(7, 2, 18, 10, 17, SRSHR12, 13)
	SUBS(7, 7, 2)
	MACA(12, 2, 17, 10, 18, SRSHR12, 13)
	SUBS(12, 12, 10)

	ADDS(2, 8, 0)
	CLIP(2)
	SUBS(3, 8, 0)
	CLIP(3)
	ADDS(9, 5, 7)
	CLIP(9)
	SUBS(10, 5, 7)
	CLIP(10)
	SUBS(13, 4, 1)
	CLIP(13)
	ADDS(0, 4, 1)
	CLIP(0)
	SUBS(1, 6, 12)
	CLIP(1)
	ADDS(4, 6, 12)
	CLIP(4)

	R181(5, 1, 10, SUBS)
	R181(6, 1, 10, ADDS)
	R181(7, 13, 3, SUBS)
	R181(8, 13, 3, ADDS)

	// The coefficients are reloaded each iteration anyway, so their registers
	// hold the odd half while the even half comes back from the scratch.
	VMOV V0.B16, V16.B16
	VMOV V4.B16, V17.B16
	VMOV V6.B16, V18.B16
	VMOV V8.B16, V19.B16
	VMOV V7.B16, V20.B16
	VMOV V5.B16, V21.B16
	VMOV V9.B16, V22.B16
	VMOV V2.B16, V23.B16

	VLD1 (R4), [V0.S4]
	LD(R4, R9, V1.S4)
	VLD1 (R5), [V2.S4]
	LD(R5, R9, V3.S4)
	VLD1 (R6), [V4.S4]
	LD(R6, R9, V5.S4)
	VLD1 (R7), [V6.S4]
	LD(R7, R9, V7.S4)

	ADDS(30, 0, 16)
	CLIP(30)
	VST1 [V30.S4], (R4)
	SUBS(30, 0, 16)
	CLIP(30)
	ST(R7, R3, V30.S4)
	ADDS(30, 1, 17)
	CLIP(30)
	ST(R4, R1, V30.S4)
	SUBS(30, 1, 17)
	CLIP(30)
	ST(R7, R9, V30.S4)
	ADDS(30, 2, 18)
	CLIP(30)
	ST(R4, R9, V30.S4)
	SUBS(30, 2, 18)
	CLIP(30)
	ST(R7, R1, V30.S4)
	ADDS(30, 3, 19)
	CLIP(30)
	ST(R4, R3, V30.S4)
	SUBS(30, 3, 19)
	CLIP(30)
	VST1 [V30.S4], (R7)
	ADDS(30, 4, 20)
	CLIP(30)
	VST1 [V30.S4], (R5)
	SUBS(30, 4, 20)
	CLIP(30)
	ST(R6, R3, V30.S4)
	ADDS(30, 5, 21)
	CLIP(30)
	ST(R5, R1, V30.S4)
	SUBS(30, 5, 21)
	CLIP(30)
	ST(R6, R9, V30.S4)
	ADDS(30, 6, 22)
	CLIP(30)
	ST(R5, R9, V30.S4)
	SUBS(30, 6, 22)
	CLIP(30)
	ST(R6, R1, V30.S4)
	ADDS(30, 7, 23)
	CLIP(30)
	ST(R5, R3, V30.S4)
	SUBS(30, 7, 23)
	CLIP(30)
	VST1 [V30.S4], (R6)

	KONST
	ADD  $16, R0
	SUB  $4, R2
	CBNZ R2, col16

	RET
