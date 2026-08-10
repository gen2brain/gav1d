//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// At high bit depth the samples no longer fit the byte multiply the 8-bit
// kernels use, so every tap goes through a 32-bit multiply. That makes the tap
// stride free, which is what lets one kernel serve both directions.

#define FSETUP                     \
	MOV  f+40(FP), X7;         \
	MOVW 0(X7), X18;           \
	MOVW 4(X7), X19;           \
	MOVW 8(X7), X20;           \
	MOVW 12(X7), X21;          \
	MOVW 16(X7), X22;          \
	MOVW 20(X7), X23;          \
	MOVW 24(X7), X24;          \
	MOVW 28(X7), X25;          \
	MOV  dst+0(FP), X10;       \
	MOV  src+16(FP), X11;      \
	MOV  tapStride+32(FP), X26; \
	SLLI $1, X26, X26;         \
	MOV  rnd+64(FP), X28;      \
	MOV  shift+72(FP), X29;    \
	MOV  maxv+80(FP), X30;     \
	MOV  h+56(FP), X31

#define TAP(EXT, REG, V)           \
	VSETVLI  X13, E16, MF2, TA, MA, X5; \
	VLE16V   (X6), V3;         \
	VSETVLI  X13, E32, M1, TA, MA, X5; \
	EXT      V3, V4;           \
	VMULVX   REG, V4, V4;      \
	VADDVV   V4, V, V;         \
	ADD      X26, X6, X6

#define TAPS8(EXT)                 \
	VSETVLI  X13, E16, MF2, TA, MA, X5; \
	VLE16V   (X6), V3;         \
	VSETVLI  X13, E32, M1, TA, MA, X5; \
	EXT      V3, V4;           \
	VMULVX   X18, V4, V0;      \
	ADD      X26, X6, X6;      \
	TAP(EXT, X19, V0);         \
	TAP(EXT, X20, V0);         \
	TAP(EXT, X21, V0);         \
	TAP(EXT, X22, V0);         \
	TAP(EXT, X23, V0);         \
	TAP(EXT, X24, V0);         \
	TAP(EXT, X25, V0);         \
	VADDVX   X28, V0, V0;      \
	VSRAVX   X29, V0, V0

#define ROWSTEP(NAME)              \
	MOV  srcStride+24(FP), X6; \
	SLLI $1, X6, X6;           \
	ADD  X6, X11, X11;         \
	MOV  dstStride+8(FP), X6;  \
	SLLI $1, X6, X6;           \
	ADD  X6, X10, X10;         \
	SUB  $1, X31, X31;         \
	BNE  X0, X31, NAME

// The bilinear filter is two taps, and its horizontal put rounds twice at
// twelve bits, so it carries a second shift the eight tap kernels do not.
#define BSETUP                      \
	MOVW mxy+40(FP), X19;       \
	MOV  $16, X18;              \
	SUB  X19, X18, X18;         \
	MOV  dst+0(FP), X10;        \
	MOV  src+16(FP), X11;       \
	MOV  tapStride+32(FP), X26; \
	SLLI $1, X26, X26;          \
	MOV  rnd+64(FP), X28;       \
	MOV  shift+72(FP), X29;     \
	MOV  rnd2+80(FP), X20;      \
	MOV  shift2+88(FP), X21;    \
	MOV  maxv+96(FP), X30;      \
	MOV  h+56(FP), X31

#define TAPS2(EXT)                          \
	VSETVLI  X13, E16, MF2, TA, MA, X5; \
	VLE16V   (X6), V3;                  \
	VSETVLI  X13, E32, M1, TA, MA, X5;  \
	EXT      V3, V4;                    \
	VMULVX   X18, V4, V0;               \
	ADD      X26, X6, X6;               \
	TAP(EXT, X19, V0);                  \
	VADDVX   X28, V0, V0;               \
	VSRAVX   X29, V0, V0;               \
	VADDVX   X20, V0, V0;               \
	VSRAVX   X21, V0, V0

#define BROWSTEP(NAME)             \
	MOV  srcStride+24(FP), X6; \
	SLLI $1, X6, X6;           \
	ADD  X6, X11, X11;         \
	MOV  dstStride+8(FP), X6;  \
	SLLI $1, X6, X6;           \
	ADD  X6, X10, X10;         \
	SUB  $1, X31, X31;         \
	BNE  X0, X31, NAME

// func mcTap16RVV(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTap16RVV(SB), NOSPLIT, $0-84
	FSETUP

rows1:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

cols1:
	MOV     X14, X6
	TAPS8(VZEXTVF2)
	VMAXVX  X0, V0, V0
	VMINVX  X30, V0, V0
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, cols1

	ROWSTEP(rows1)
	RET

// func mcTapMid16RVV(dst *int16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapMid16RVV(SB), NOSPLIT, $0-84
	FSETUP

rows2:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

cols2:
	MOV     X14, X6
	TAPS8(VZEXTVF2)
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, cols2

	ROWSTEP(rows2)
	RET

// func mcTapFromMid16RVV(dst *uint16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapFromMid16RVV(SB), NOSPLIT, $0-84
	FSETUP

rows3:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

cols3:
	MOV     X14, X6
	TAPS8(VSEXTVF2)
	VMAXVX  X0, V0, V0
	VMINVX  X30, V0, V0
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, cols3

	ROWSTEP(rows3)
	RET

// func mcTapPrepMid16RVV(dst *int16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapPrepMid16RVV(SB), NOSPLIT, $0-84
	FSETUP

rows4:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

cols4:
	MOV     X14, X6
	TAPS8(VSEXTVF2)
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, cols4

	ROWSTEP(rows4)
	RET

TEXT ·mcBilin16RVV(SB), NOSPLIT, $0-100
	BSETUP

b1rows:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

b1cols:
	MOV     X14, X6
	TAPS2(VZEXTVF2)
	VMAXVX  X0, V0, V0
	VMINVX  X30, V0, V0
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, b1cols

	BROWSTEP(b1rows)
	RET

TEXT ·mcBilinMid16RVV(SB), NOSPLIT, $0-100
	BSETUP

b2rows:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

b2cols:
	MOV     X14, X6
	TAPS2(VZEXTVF2)
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, b2cols

	BROWSTEP(b2rows)
	RET

TEXT ·mcBilinFromMid16RVV(SB), NOSPLIT, $0-100
	BSETUP

b3rows:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

b3cols:
	MOV     X14, X6
	TAPS2(VSEXTVF2)
	VMAXVX  X0, V0, V0
	VMINVX  X30, V0, V0
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, b3cols

	BROWSTEP(b3rows)
	RET

TEXT ·mcBilinPrepMid16RVV(SB), NOSPLIT, $0-100
	BSETUP

b4rows:
	MOV w+48(FP), X13
	MOV X11, X14
	MOV X10, X15

b4cols:
	MOV     X14, X6
	TAPS2(VSEXTVF2)
	VSETVLI X13, E16, MF2, TA, MA, X5
	VNSRLWI $0, V0, V5
	VSE16V  V5, (X15)
	VSETVLI X13, E32, M1, TA, MA, X5
	SLLI    $1, X5, X6
	ADD     X6, X14, X14
	ADD     X6, X15, X15
	SUB     X5, X13, X13
	BNE     X0, X13, b4cols

	BROWSTEP(b4rows)
	RET
