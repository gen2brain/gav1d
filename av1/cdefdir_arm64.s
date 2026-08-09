//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no variable vector shift.
#define USHL(Vd, Vn, Vm) WORD $(0x6EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

#define HV0  0
#define HV1  32
#define DIA0 64
#define DIA1 128
#define ALT0 192
#define ALT1 256
#define ALT2 320
#define ALT3 384

// ADDAT accumulates a whole row into one of the padded arrays at an offset
// that only the row index decides.
#define ADDAT(OFF, SA, SB)              \
	ADD  $OFF, R1, R9;              \
	VLD1 (R9), [V4.S4, V5.S4];      \
	VADD SA.S4, V4.S4, V4.S4;       \
	VADD SB.S4, V5.S4, V5.S4;       \
	VST1 [V4.S4, V5.S4], (R9)

#define ADDAT4(OFF, SA)                 \
	ADD  $OFF, R1, R9;              \
	VLD1 (R9), [V4.S4];             \
	VADD SA.S4, V4.S4, V4.S4;       \
	VST1 [V4.S4], (R9)

// ROW folds one row of eight samples, held as V2:V3, into all eight partial
// sums. HY is Y>>1, where the alternating sums step half as fast.
#define ROW(Y, HY)                      \
	VADD  V2.S4, V0.S4, V0.S4;      \
	VADD  V3.S4, V1.S4, V1.S4;      \
	VADD  V3.S4, V2.S4, V6.S4;      \
	VADDV V6.S4, V7;                \
	VMOV  V7.S[0], R9;              \
	MOVW  R9, (HV0+4*(Y))(R1);      \
	ADDAT(DIA0+4*(Y), V2, V3);      \
	VREV64 V3.S4, V6.S4;            \
	VEXT  $8, V6.B16, V6.B16, V6.B16; \
	VREV64 V2.S4, V7.S4;            \
	VEXT  $8, V7.B16, V7.B16, V7.B16; \
	ADDAT(DIA1+4*(Y), V6, V7);      \
	ADDAT(ALT2+4*(3-(HY)), V2, V3); \
	ADDAT(ALT3+4*(HY), V2, V3);     \
	VUZP1 V3.S4, V2.S4, V16.S4;     \
	VUZP2 V3.S4, V2.S4, V17.S4;     \
	VADD  V17.S4, V16.S4, V16.S4;   \
	ADDAT4(ALT0+4*(Y), V16);        \
	VREV64 V16.S4, V17.S4;          \
	VEXT  $8, V17.B16, V17.B16, V17.B16; \
	ADDAT4(ALT1+4*(Y), V17)

#define LOAD8                           \
	VLD1  (R0), [V8.D1];            \
	VUSHLL $0, V8.B8, V9.H8;        \
	VUSHLL $0, V9.H4, V2.S4;        \
	VUSHLL2 $0, V9.H8, V3.S4;       \
	VSUB  V10.S4, V2.S4, V2.S4;     \
	VSUB  V10.S4, V3.S4, V3.S4

#define LOAD16                          \
	VLD1  (R0), [V8.H8];            \
	VUSHLL $0, V8.H4, V2.S4;        \
	VUSHLL2 $0, V8.H8, V3.S4;       \
	USHL(2, 2, 11);                 \
	USHL(3, 3, 11);                 \
	VSUB  V10.S4, V2.S4, V2.S4;     \
	VSUB  V10.S4, V3.S4, V3.S4

// func cdefSums8NEON(img *uint8, stride int, s *cdefSums)
TEXT ·cdefSums8NEON(SB), NOSPLIT, $0-24
	MOVD img+0(FP), R0
	MOVD stride+8(FP), R2
	MOVD s+16(FP), R1
	VMOVI $0, V0.B16
	VMOVI $0, V1.B16
	MOVD  $128, R9
	VDUP  R9, V10.S4

	LOAD8
	ROW(0, 0)
	ADD R2, R0, R0
	LOAD8
	ROW(1, 0)
	ADD R2, R0, R0
	LOAD8
	ROW(2, 1)
	ADD R2, R0, R0
	LOAD8
	ROW(3, 1)
	ADD R2, R0, R0
	LOAD8
	ROW(4, 2)
	ADD R2, R0, R0
	LOAD8
	ROW(5, 2)
	ADD R2, R0, R0
	LOAD8
	ROW(6, 3)
	ADD R2, R0, R0
	LOAD8
	ROW(7, 3)

	ADD  $HV1, R1, R9
	VST1 [V0.S4, V1.S4], (R9)
	RET

// func cdefSums16NEON(img *uint16, stride int, s *cdefSums, shift int)
TEXT ·cdefSums16NEON(SB), NOSPLIT, $0-32
	MOVD img+0(FP), R0
	MOVD stride+8(FP), R2
	LSL  $1, R2, R2
	MOVD s+16(FP), R1
	VMOVI $0, V0.B16
	VMOVI $0, V1.B16
	MOVD  $128, R9
	VDUP  R9, V10.S4
	MOVD  shift+24(FP), R9
	NEG   R9, R9
	VDUP  R9, V11.S4

	LOAD16
	ROW(0, 0)
	ADD R2, R0, R0
	LOAD16
	ROW(1, 0)
	ADD R2, R0, R0
	LOAD16
	ROW(2, 1)
	ADD R2, R0, R0
	LOAD16
	ROW(3, 1)
	ADD R2, R0, R0
	LOAD16
	ROW(4, 2)
	ADD R2, R0, R0
	LOAD16
	ROW(5, 2)
	ADD R2, R0, R0
	LOAD16
	ROW(6, 3)
	ADD R2, R0, R0
	LOAD16
	ROW(7, 3)

	ADD  $HV1, R1, R9
	VST1 [V0.S4, V1.S4], (R9)
	RET
