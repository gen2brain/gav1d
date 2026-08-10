//go:build arm64 && !noasm

#include "textflag.h"

// At high bit depth the samples no longer fit the byte multiply the 8-bit
// kernels use, so every tap goes through a 32-bit multiply. That makes the tap
// stride free, which is what lets one kernel serve both directions.

#define MULS4(Vd, Vn, Vm)  WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS4(Vd, Vn, Vm) WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS4(Vd, Vn, Vm) WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLS4(Vd, Vn, Vm) WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLL4(Vd, Vn)     WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define XTN4(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))
#define SQXTUN4(Vd, Vn)    WORD $(0x2E612800 | ((Vn) << 5) | (Vd))

#define FSETUP                     \
	MOVD  f+40(FP), R7;        \
	MOVW  0(R7), R6;           \
	VDUP  R6, V8.S4;           \
	MOVW  4(R7), R6;           \
	VDUP  R6, V9.S4;           \
	MOVW  8(R7), R6;           \
	VDUP  R6, V10.S4;          \
	MOVW  12(R7), R6;          \
	VDUP  R6, V11.S4;          \
	MOVW  16(R7), R6;          \
	VDUP  R6, V12.S4;          \
	MOVW  20(R7), R6;          \
	VDUP  R6, V13.S4;          \
	MOVW  24(R7), R6;          \
	VDUP  R6, V14.S4;          \
	MOVW  28(R7), R6;          \
	VDUP  R6, V15.S4;          \
	MOVD  dst+0(FP), R0;       \
	MOVD  src+16(FP), R1;      \
	MOVD  tapStride+32(FP), R10; \
	LSL   $1, R10, R10;        \
	MOVD  rnd+64(FP), R6;      \
	VDUP  R6, V7.S4;           \
	MOVD  shift+72(FP), R6;    \
	NEG   R6, R6;              \
	VDUP  R6, V6.S4;           \
	MOVD  maxv+80(FP), R6;     \
	VDUP  R6, V5.S4;           \
	VMOVI $0, V4.B16;          \
	MOVD  h+56(FP), R12

// TAPU and TAPS load one tap, widen it unsigned or signed, and fold it in.
#define TAPU(IDX)                  \
	VLD1   (R2), [V1.H4];      \
	VUSHLL $0, V1.H4, V1.S4;   \
	MULS4(1, 1, IDX);          \
	VADD   V1.S4, V0.S4, V0.S4; \
	ADD    R10, R2, R2

#define TAPS(IDX)                  \
	VLD1  (R2), [V1.H4];       \
	SSHLL4(1, 1);              \
	MULS4(1, 1, IDX);          \
	VADD  V1.S4, V0.S4, V0.S4; \
	ADD   R10, R2, R2

#define TAPS8U                     \
	VLD1   (R2), [V1.H4];      \
	VUSHLL $0, V1.H4, V1.S4;   \
	MULS4(0, 1, 8);            \
	ADD    R10, R2, R2;        \
	TAPU(9);                   \
	TAPU(10);                  \
	TAPU(11);                  \
	TAPU(12);                  \
	TAPU(13);                  \
	TAPU(14);                  \
	TAPU(15);                  \
	VADD   V7.S4, V0.S4, V0.S4; \
	SSHLS4(0, 0, 6)

#define TAPS8S                     \
	VLD1  (R2), [V1.H4];       \
	SSHLL4(1, 1);              \
	MULS4(0, 1, 8);            \
	ADD   R10, R2, R2;         \
	TAPS(9);                   \
	TAPS(10);                  \
	TAPS(11);                  \
	TAPS(12);                  \
	TAPS(13);                  \
	TAPS(14);                  \
	TAPS(15);                  \
	VADD  V7.S4, V0.S4, V0.S4; \
	SSHLS4(0, 0, 6)

#define ROWSTEP(NAME)              \
	MOVD srcStride+24(FP), R6; \
	LSL  $1, R6, R6;           \
	ADD  R6, R1, R1;           \
	MOVD dstStride+8(FP), R6;  \
	LSL  $1, R6, R6;           \
	ADD  R6, R0, R0;           \
	SUBS $1, R12, R12;         \
	BNE  NAME

// The bilinear filter is two taps, and its horizontal put rounds twice at
// twelve bits, so it carries a second shift the eight tap kernels do not.
#define BSETUP                     \
	MOVW  mxy+40(FP), R6;      \
	VDUP  R6, V9.S4;           \
	MOVD  $16, R7;             \
	SUB   R6, R7, R7;          \
	VDUP  R7, V8.S4;           \
	MOVD  dst+0(FP), R0;       \
	MOVD  src+16(FP), R1;      \
	MOVD  tapStride+32(FP), R10; \
	LSL   $1, R10, R10;        \
	MOVD  rnd+64(FP), R6;      \
	VDUP  R6, V7.S4;           \
	MOVD  shift+72(FP), R6;    \
	NEG   R6, R6;              \
	VDUP  R6, V6.S4;           \
	MOVD  rnd2+80(FP), R6;     \
	VDUP  R6, V3.S4;           \
	MOVD  shift2+88(FP), R6;   \
	NEG   R6, R6;              \
	VDUP  R6, V2.S4;           \
	MOVD  maxv+96(FP), R6;     \
	VDUP  R6, V5.S4;           \
	VMOVI $0, V4.B16;          \
	MOVD  h+56(FP), R12

#define TAPS2U                      \
	VLD1   (R2), [V1.H4];       \
	VUSHLL $0, V1.H4, V1.S4;    \
	MULS4(0, 1, 8);             \
	ADD    R10, R2, R2;         \
	TAPU(9);                    \
	VADD   V7.S4, V0.S4, V0.S4; \
	SSHLS4(0, 0, 6);            \
	VADD   V3.S4, V0.S4, V0.S4; \
	SSHLS4(0, 0, 2)

#define TAPS2S                      \
	VLD1  (R2), [V1.H4];        \
	SSHLL4(1, 1);               \
	MULS4(0, 1, 8);             \
	ADD   R10, R2, R2;          \
	TAPS(9);                    \
	VADD  V7.S4, V0.S4, V0.S4;  \
	SSHLS4(0, 0, 6);            \
	VADD  V3.S4, V0.S4, V0.S4;  \
	SSHLS4(0, 0, 2)

#define BROWSTEP(NAME)             \
	MOVD srcStride+24(FP), R6; \
	LSL  $1, R6, R6;           \
	ADD  R6, R1, R1;           \
	MOVD dstStride+8(FP), R6;  \
	LSL  $1, R6, R6;           \
	ADD  R6, R0, R0;           \
	SUBS $1, R12, R12;         \
	BNE  NAME

// func mcTap16NEON(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTap16NEON(SB), NOSPLIT, $0-84
	FSETUP

rows1:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

cols1:
	MOVD R11, R2
	TAPS8U
	SMAXS4(0, 0, 4)
	SMINS4(0, 0, 5)
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  cols1

	ROWSTEP(rows1)
	RET

// func mcTapMid16NEON(dst *int16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapMid16NEON(SB), NOSPLIT, $0-84
	FSETUP

rows2:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

cols2:
	MOVD R11, R2
	TAPS8U
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  cols2

	ROWSTEP(rows2)
	RET

// func mcTapFromMid16NEON(dst *uint16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapFromMid16NEON(SB), NOSPLIT, $0-84
	FSETUP

rows3:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

cols3:
	MOVD R11, R2
	TAPS8S
	SMAXS4(0, 0, 4)
	SMINS4(0, 0, 5)
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  cols3

	ROWSTEP(rows3)
	RET

// func mcTapPrepMid16NEON(dst *int16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapPrepMid16NEON(SB), NOSPLIT, $0-84
	FSETUP

rows4:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

cols4:
	MOVD R11, R2
	TAPS8S
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  cols4

	ROWSTEP(rows4)
	RET

TEXT ·mcBilin16NEON(SB), NOSPLIT, $0-100
	BSETUP

b1rows:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

b1cols:
	MOVD R11, R2
	TAPS2U
	SMAXS4(0, 0, 4)
	SMINS4(0, 0, 5)
	SQXTUN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  b1cols

	BROWSTEP(b1rows)
	RET

TEXT ·mcBilinMid16NEON(SB), NOSPLIT, $0-100
	BSETUP

b2rows:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

b2cols:
	MOVD R11, R2
	TAPS2U
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  b2cols

	BROWSTEP(b2rows)
	RET

TEXT ·mcBilinFromMid16NEON(SB), NOSPLIT, $0-100
	BSETUP

b3rows:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

b3cols:
	MOVD R11, R2
	TAPS2S
	SMAXS4(0, 0, 4)
	SMINS4(0, 0, 5)
	SQXTUN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  b3cols

	BROWSTEP(b3rows)
	RET

TEXT ·mcBilinPrepMid16NEON(SB), NOSPLIT, $0-100
	BSETUP

b4rows:
	MOVD w+48(FP), R13
	MOVD R1, R11
	MOVD R0, R14

b4cols:
	MOVD R11, R2
	TAPS2S
	XTN4(0, 0)
	VST1 [V0.D1], (R14)
	ADD  $8, R11, R11
	ADD  $8, R14, R14
	SUBS $4, R13, R13
	BNE  b4cols

	BROWSTEP(b4rows)
	RET
