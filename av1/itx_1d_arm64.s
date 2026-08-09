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

#define SETUP                   \
	MOVD tmp+0(FP), R0;     \
	MOVD lanes+8(FP), R2;   \
	MOVD stride+16(FP), R1; \
	MOVW lo+24(FP), R8;     \
	VDUP R8, V14.S4;        \
	MOVW hi+28(FP), R8;     \
	VDUP R8, V15.S4;        \
	LSL  $2, R1, R1;        \
	ADD  R1, R1, R9;        \
	ADD  R1, R9, R3;        \
	ADD  R9, R9, R11;       \
	ADD  R11, R11, R12;     \
	ADD  R11, R12, R13;     \
	KONST

#define LD(BASE, OFF, V)  \
	ADD OFF, BASE, R10; \
	VLD1 (R10), [V]

#define ST(BASE, OFF, V)  \
	ADD OFF, BASE, R10; \
	VST1 [V], (R10)

#define NEGS(Vd, Vn)  WORD $(0x6EA0B800 | ((Vn) << 5) | (Vd))

#define MACAN(D, A, CA, B, CB, SH, T) \
	MULS(D, A, CA); \
	MULS(T, B, CB); \
	ADDS(D, D, T);  \
	NEGS(D, D);     \
	SH(D, D)

#define KONST32A       \
	MOVD $201, R8;  \
	VDUP R8, V16.S4; \
	MOVD $5, R8;  \
	VDUP R8, V17.S4; \
	MOVD $1061, R8;  \
	VDUP R8, V18.S4; \
	MOVD $2751, R8;  \
	VDUP R8, V19.S4; \
	MOVD $1751, R8;  \
	VDUP R8, V20.S4; \
	MOVD $393, R8;  \
	VDUP R8, V21.S4; \
	MOVD $239, R8;  \
	VDUP R8, V22.S4; \
	MOVD $1380, R8;  \
	VDUP R8, V23.S4; \
	MOVD $995, R8;  \
	VDUP R8, V24.S4; \
	MOVD $123, R8;  \
	VDUP R8, V25.S4; \
	MOVD $583, R8;  \
	VDUP R8, V26.S4; \
	MOVD $2106, R8;  \
	VDUP R8, V27.S4; \
	MOVD $1220, R8;  \
	VDUP R8, V28.S4; \
	MOVD $1645, R8;  \
	VDUP R8, V29.S4; \
	MOVD $44, R8;  \
	VDUP R8, V30.S4; \
	MOVD $601, R8;  \
	VDUP R8, V31.S4

#define KONST32B       \
	MOVD $181, R8;  \
	VDUP R8, V16.S4; \
	MOVD $799, R8;  \
	VDUP R8, V17.S4; \
	MOVD $79, R8;  \
	VDUP R8, V18.S4; \
	MOVD $1703, R8;  \
	VDUP R8, V19.S4; \
	MOVD $1138, R8;  \
	VDUP R8, V20.S4; \
	MOVD $1567, R8;  \
	VDUP R8, V21.S4; \
	MOVD $312, R8;  \
	VDUP R8, V22.S4

#define SETUP32                 \
	MOVD tmp+0(FP), R0;     \
	MOVD lanes+8(FP), R2;   \
	MOVD stride+16(FP), R1; \
	MOVW lo+24(FP), R8;     \
	VDUP R8, V14.S4;        \
	MOVW hi+28(FP), R8;     \
	VDUP R8, V15.S4;        \
	LSL  $2, R1, R1;        \
	ADD  R1, R1, R9;        \
	ADD  R1, R9, R3;        \
	ADD  R9, R9, R11;       \
	ADD  R1, R11, R4;       \
	ADD  R9, R11, R5;       \
	ADD  R3, R11, R6;       \
	ADD  $16, RSP, R21

// func itxColDct4NEON(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct4NEON(SB), NOSPLIT, $0-32
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

// func itxColDct8NEON(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct8NEON(SB), NOSPLIT, $0-32
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

// func itxColDct16NEON(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct16NEON(SB), NOSPLIT, $0-32
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

#define SSHLS(Vd, Vn, Vm) WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

// func itxIdentityNEON(tmp *int32, count, stride, n int, p *[4]int32)
TEXT ·itxIdentityNEON(SB), NOSPLIT, $0-40
	MOVD tmp+0(FP), R0
	MOVD count+8(FP), R1
	MOVD stride+16(FP), R2
	MOVD n+24(FP), R3
	MOVD p+32(FP), R4

	MOVW (R4), R5
	VDUP R5, V1.S4
	MOVW 4(R4), R5
	VDUP R5, V2.S4
	MOVW 8(R4), R5
	VDUP R5, V3.S4
	MOVW 12(R4), R5
	NEG  R5, R5
	VDUP R5, V4.S4
	LSL  $2, R2, R2

idtap:
	MOVD R0, R6
	MOVD R1, R7

idlane:
	VLD1 (R6), [V0.S4]
	MULS(5, 0, 1)
	MULS(0, 0, 2)
	ADDS(0, 0, 3)
	SSHLS(0, 0, 4)
	ADDS(0, 0, 5)
	VST1 [V0.S4], (R6)

	ADD  $16, R6
	SUB  $4, R7
	CBNZ R7, idlane

	ADD  R2, R0
	SUB  $1, R3
	CBNZ R3, idtap

	RET

// func itxColDct32NEON(tmp *int32, lanes, stride int, lo, hi int32)
TEXT ·itxColDct32NEON(SB), NOSPLIT, $288-32
	SETUP32

col32:
	ADD R11<<1, R0, R7
	ADD R11<<1, R7, R19
	ADD R11<<1, R19, R20
	KONST32A

	LD(R0, R1, V0.S4)
	LD(R20, R6, V1.S4)
	LD(R19, R1, V2.S4)
	LD(R7, R6, V3.S4)
	MACA(4, 0, 16, 1, 17, SRSHR12, 13)
	SUBS(4, 4, 1)
	MACS(5, 1, 16, 0, 17, SRSHR12, 13)
	ADDS(5, 5, 0)
	MACAN(6, 2, 18, 3, 19, SRSHR12, 13)
	ADDS(6, 6, 2)
	MACS(7, 2, 19, 3, 18, SRSHR12, 13)
	ADDS(7, 7, 3)
	ADDS(12, 4, 6)
	CLIP(12)
	ADD $0, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 4, 6)
	CLIP(12)
	ADD $16, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 5, 7)
	CLIP(12)
	ADD $224, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 5, 7)
	CLIP(12)
	ADD $240, R21, R10
	VST1 [V12.S4], (R10)

	LD(R7, R1, V0.S4)
	LD(R19, R6, V1.S4)
	LD(R20, R1, V2.S4)
	LD(R0, R6, V3.S4)
	MACA(4, 0, 20, 1, 21, SRSHR12, 13)
	SUBS(4, 4, 1)
	MACS(5, 1, 20, 0, 21, SRSHR12, 13)
	ADDS(5, 5, 0)
	MACAN(6, 2, 22, 3, 23, SRSHR12, 13)
	ADDS(6, 6, 2)
	MACS(7, 2, 23, 3, 22, SRSHR12, 13)
	ADDS(7, 7, 3)
	SUBS(12, 6, 4)
	CLIP(12)
	ADD $32, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 6, 4)
	CLIP(12)
	ADD $48, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 7, 5)
	CLIP(12)
	ADD $192, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 7, 5)
	CLIP(12)
	ADD $208, R21, R10
	VST1 [V12.S4], (R10)

	LD(R0, R4, V0.S4)
	LD(R20, R3, V1.S4)
	LD(R19, R4, V2.S4)
	LD(R7, R3, V3.S4)
	MACA(4, 0, 24, 1, 25, SRSHR12, 13)
	SUBS(4, 4, 1)
	MACS(5, 1, 24, 0, 25, SRSHR12, 13)
	ADDS(5, 5, 0)
	MACAN(6, 2, 26, 3, 27, SRSHR12, 13)
	ADDS(6, 6, 2)
	MACS(7, 2, 27, 3, 26, SRSHR12, 13)
	ADDS(7, 7, 3)
	ADDS(12, 4, 6)
	CLIP(12)
	ADD $64, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 4, 6)
	CLIP(12)
	ADD $80, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 5, 7)
	CLIP(12)
	ADD $160, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 5, 7)
	CLIP(12)
	ADD $176, R21, R10
	VST1 [V12.S4], (R10)

	LD(R7, R4, V0.S4)
	LD(R19, R3, V1.S4)
	LD(R20, R4, V2.S4)
	LD(R0, R3, V3.S4)
	MACS(4, 0, 28, 1, 29, SRSHR11, 13)
	MACA(5, 0, 29, 1, 28, SRSHR11, 13)
	MACAN(6, 2, 30, 3, 31, SRSHR12, 13)
	ADDS(6, 6, 2)
	MACS(7, 2, 31, 3, 30, SRSHR12, 13)
	ADDS(7, 7, 3)
	SUBS(12, 6, 4)
	CLIP(12)
	ADD $96, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 6, 4)
	CLIP(12)
	ADD $112, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 7, 5)
	CLIP(12)
	ADD $128, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 7, 5)
	CLIP(12)
	ADD $144, R21, R10
	VST1 [V12.S4], (R10)

	KONST32B

	ADD $0, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $16, R21, R10
	VLD1 (R10), [V1.S4]
	ADD $32, R21, R10
	VLD1 (R10), [V2.S4]
	ADD $48, R21, R10
	VLD1 (R10), [V3.S4]
	ADD $192, R21, R10
	VLD1 (R10), [V4.S4]
	ADD $208, R21, R10
	VLD1 (R10), [V5.S4]
	ADD $224, R21, R10
	VLD1 (R10), [V6.S4]
	ADD $240, R21, R10
	VLD1 (R10), [V7.S4]
	MACA(8, 6, 17, 1, 18, SRSHR12, 13)
	SUBS(8, 8, 1)
	MACS(9, 1, 17, 6, 18, SRSHR12, 13)
	ADDS(9, 9, 6)
	MACS(10, 5, 18, 2, 17, SRSHR12, 13)
	SUBS(10, 10, 5)
	MACA(11, 5, 17, 2, 18, SRSHR12, 13)
	SUBS(11, 11, 2)
	ADDS(12, 0, 3)
	CLIP(12)
	ADD $0, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 0, 3)
	CLIP(12)
	ADD $48, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 8, 10)
	CLIP(12)
	ADD $16, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 8, 10)
	CLIP(12)
	ADD $32, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 7, 4)
	CLIP(12)
	ADD $192, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 7, 4)
	CLIP(12)
	ADD $240, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 9, 11)
	CLIP(12)
	ADD $208, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 9, 11)
	CLIP(12)
	ADD $224, R21, R10
	VST1 [V12.S4], (R10)

	ADD $64, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $80, R21, R10
	VLD1 (R10), [V1.S4]
	ADD $96, R21, R10
	VLD1 (R10), [V2.S4]
	ADD $112, R21, R10
	VLD1 (R10), [V3.S4]
	ADD $128, R21, R10
	VLD1 (R10), [V4.S4]
	ADD $144, R21, R10
	VLD1 (R10), [V5.S4]
	ADD $160, R21, R10
	VLD1 (R10), [V6.S4]
	ADD $176, R21, R10
	VLD1 (R10), [V7.S4]
	MACS(8, 6, 19, 1, 20, SRSHR11, 13)
	MACA(9, 6, 20, 1, 19, SRSHR11, 13)
	MACAN(10, 5, 20, 2, 19, SRSHR11, 13)
	MACS(11, 5, 19, 2, 20, SRSHR11, 13)
	SUBS(12, 3, 0)
	CLIP(12)
	ADD $64, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 3, 0)
	CLIP(12)
	ADD $112, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 10, 8)
	CLIP(12)
	ADD $80, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 10, 8)
	CLIP(12)
	ADD $96, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 4, 7)
	CLIP(12)
	ADD $128, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 4, 7)
	CLIP(12)
	ADD $176, R21, R10
	VST1 [V12.S4], (R10)
	ADDS(12, 11, 9)
	CLIP(12)
	ADD $144, R21, R10
	VST1 [V12.S4], (R10)
	SUBS(12, 11, 9)
	CLIP(12)
	ADD $160, R21, R10
	VST1 [V12.S4], (R10)

	ADD $208, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $32, R21, R10
	VLD1 (R10), [V1.S4]
	MACA(2, 0, 21, 1, 22, SRSHR12, 13)
	SUBS(2, 2, 1)
	MACS(3, 1, 21, 0, 22, SRSHR12, 13)
	ADDS(3, 3, 0)
	ADD $32, R21, R10
	VST1 [V2.S4], (R10)
	ADD $208, R21, R10
	VST1 [V3.S4], (R10)
	ADD $192, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $48, R21, R10
	VLD1 (R10), [V1.S4]
	MACA(2, 0, 21, 1, 22, SRSHR12, 13)
	SUBS(2, 2, 1)
	MACS(3, 1, 21, 0, 22, SRSHR12, 13)
	ADDS(3, 3, 0)
	ADD $48, R21, R10
	VST1 [V2.S4], (R10)
	ADD $192, R21, R10
	VST1 [V3.S4], (R10)
	ADD $176, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $64, R21, R10
	VLD1 (R10), [V1.S4]
	MACS(2, 0, 22, 1, 21, SRSHR12, 13)
	SUBS(2, 2, 0)
	MACA(3, 0, 21, 1, 22, SRSHR12, 13)
	SUBS(3, 3, 1)
	ADD $64, R21, R10
	VST1 [V2.S4], (R10)
	ADD $176, R21, R10
	VST1 [V3.S4], (R10)
	ADD $160, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $80, R21, R10
	VLD1 (R10), [V1.S4]
	MACS(2, 0, 22, 1, 21, SRSHR12, 13)
	SUBS(2, 2, 0)
	MACA(3, 0, 21, 1, 22, SRSHR12, 13)
	SUBS(3, 3, 1)
	ADD $80, R21, R10
	VST1 [V2.S4], (R10)
	ADD $160, R21, R10
	VST1 [V3.S4], (R10)

	ADD $0, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $112, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $0, R21, R10
	VST1 [V2.S4], (R10)
	ADD $112, R21, R10
	VST1 [V3.S4], (R10)
	ADD $16, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $96, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $16, R21, R10
	VST1 [V2.S4], (R10)
	ADD $96, R21, R10
	VST1 [V3.S4], (R10)
	ADD $32, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $80, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $32, R21, R10
	VST1 [V2.S4], (R10)
	ADD $80, R21, R10
	VST1 [V3.S4], (R10)
	ADD $48, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $64, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $48, R21, R10
	VST1 [V2.S4], (R10)
	ADD $64, R21, R10
	VST1 [V3.S4], (R10)
	ADD $240, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $128, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $240, R21, R10
	VST1 [V2.S4], (R10)
	ADD $128, R21, R10
	VST1 [V3.S4], (R10)
	ADD $224, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $144, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $224, R21, R10
	VST1 [V2.S4], (R10)
	ADD $144, R21, R10
	VST1 [V3.S4], (R10)
	ADD $208, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $160, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $208, R21, R10
	VST1 [V2.S4], (R10)
	ADD $160, R21, R10
	VST1 [V3.S4], (R10)
	ADD $192, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $176, R21, R10
	VLD1 (R10), [V1.S4]
	ADDS(2, 0, 1)
	CLIP(2)
	SUBS(3, 0, 1)
	CLIP(3)
	ADD $192, R21, R10
	VST1 [V2.S4], (R10)
	ADD $176, R21, R10
	VST1 [V3.S4], (R10)

	ADD $176, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $64, R21, R10
	VLD1 (R10), [V1.S4]
	R181(2, 0, 1, SUBS)
	R181(3, 0, 1, ADDS)
	ADD $64, R21, R10
	VST1 [V2.S4], (R10)
	ADD $176, R21, R10
	VST1 [V3.S4], (R10)
	ADD $160, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $80, R21, R10
	VLD1 (R10), [V1.S4]
	R181(2, 0, 1, SUBS)
	R181(3, 0, 1, ADDS)
	ADD $80, R21, R10
	VST1 [V2.S4], (R10)
	ADD $160, R21, R10
	VST1 [V3.S4], (R10)
	ADD $144, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $96, R21, R10
	VLD1 (R10), [V1.S4]
	R181(2, 0, 1, SUBS)
	R181(3, 0, 1, ADDS)
	ADD $96, R21, R10
	VST1 [V2.S4], (R10)
	ADD $144, R21, R10
	VST1 [V3.S4], (R10)
	ADD $128, R21, R10
	VLD1 (R10), [V0.S4]
	ADD $112, R21, R10
	VLD1 (R10), [V1.S4]
	R181(2, 0, 1, SUBS)
	R181(3, 0, 1, ADDS)
	ADD $112, R21, R10
	VST1 [V2.S4], (R10)
	ADD $128, R21, R10
	VST1 [V3.S4], (R10)

	VLD1 (R0), [V0.S4]
	LD(R0, R9, V1.S4)
	LD(R0, R11, V2.S4)
	LD(R0, R5, V3.S4)
	VLD1 (R7), [V4.S4]
	LD(R7, R9, V5.S4)
	LD(R7, R11, V6.S4)
	LD(R7, R5, V7.S4)
	VLD1 (R20), [V8.S4]
	LD(R20, R9, V9.S4)
	LD(R20, R11, V10.S4)
	LD(R20, R5, V11.S4)
	ADD $240, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 0, 12)
	CLIP(13)
	VST1 [V13.S4], (R0)
	SUBS(13, 0, 12)
	CLIP(13)
	ST(R20, R6, V13.S4)
	ADD $224, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 1, 12)
	CLIP(13)
	ST(R0, R1, V13.S4)
	SUBS(13, 1, 12)
	CLIP(13)
	ST(R20, R5, V13.S4)
	ADD $208, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 2, 12)
	CLIP(13)
	ST(R0, R9, V13.S4)
	SUBS(13, 2, 12)
	CLIP(13)
	ST(R20, R4, V13.S4)
	ADD $192, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 3, 12)
	CLIP(13)
	ST(R0, R3, V13.S4)
	SUBS(13, 3, 12)
	CLIP(13)
	ST(R20, R11, V13.S4)
	ADD $176, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 4, 12)
	CLIP(13)
	ST(R0, R11, V13.S4)
	SUBS(13, 4, 12)
	CLIP(13)
	ST(R20, R3, V13.S4)
	ADD $160, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 5, 12)
	CLIP(13)
	ST(R0, R4, V13.S4)
	SUBS(13, 5, 12)
	CLIP(13)
	ST(R20, R9, V13.S4)
	ADD $144, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 6, 12)
	CLIP(13)
	ST(R0, R5, V13.S4)
	SUBS(13, 6, 12)
	CLIP(13)
	ST(R20, R1, V13.S4)
	ADD $128, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 7, 12)
	CLIP(13)
	ST(R0, R6, V13.S4)
	SUBS(13, 7, 12)
	CLIP(13)
	VST1 [V13.S4], (R20)
	VLD1 (R19), [V0.S4]
	LD(R19, R9, V1.S4)
	LD(R19, R11, V2.S4)
	LD(R19, R5, V3.S4)
	ADD $112, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 0, 12)
	CLIP(13)
	VST1 [V13.S4], (R7)
	SUBS(13, 0, 12)
	CLIP(13)
	ST(R19, R6, V13.S4)
	ADD $96, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 1, 12)
	CLIP(13)
	ST(R7, R1, V13.S4)
	SUBS(13, 1, 12)
	CLIP(13)
	ST(R19, R5, V13.S4)
	ADD $80, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 2, 12)
	CLIP(13)
	ST(R7, R9, V13.S4)
	SUBS(13, 2, 12)
	CLIP(13)
	ST(R19, R4, V13.S4)
	ADD $64, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 3, 12)
	CLIP(13)
	ST(R7, R3, V13.S4)
	SUBS(13, 3, 12)
	CLIP(13)
	ST(R19, R11, V13.S4)
	ADD $48, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 8, 12)
	CLIP(13)
	ST(R7, R11, V13.S4)
	SUBS(13, 8, 12)
	CLIP(13)
	ST(R19, R3, V13.S4)
	ADD $32, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 9, 12)
	CLIP(13)
	ST(R7, R4, V13.S4)
	SUBS(13, 9, 12)
	CLIP(13)
	ST(R19, R9, V13.S4)
	ADD $16, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 10, 12)
	CLIP(13)
	ST(R7, R5, V13.S4)
	SUBS(13, 10, 12)
	CLIP(13)
	ST(R19, R1, V13.S4)
	ADD $0, R21, R10
	VLD1 (R10), [V12.S4]
	ADDS(13, 11, 12)
	CLIP(13)
	ST(R7, R6, V13.S4)
	SUBS(13, 11, 12)
	CLIP(13)
	VST1 [V13.S4], (R19)

	ADD  $16, R0
	SUB  $4, R2
	CBNZ R2, col32

	RET
