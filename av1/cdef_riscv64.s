//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// TAP accumulates one neighbour. The out-of-frame fill saturates the threshold
// subtract to zero, so it contributes nothing without needing a mask.
#define TAP(OP, OFFR, THRV, SHR, TAPR) \
	OP       OFFR, X12, X6;  \
	VLE16V   (X6), V5;       \
	VSUBVV   V1, V5, V6;     \
	VRSUBVX  X0, V6, V7;     \
	VMAXVV   V6, V7, V7;     \
	VSRLVX   SHR, V7, V8;    \
	VSSUBUVV V8, THRV, V9;   \
	VMINUVV  V7, V9, V10;    \
	VSRAVI   $15, V6, V11;   \
	VXORVV   V11, V10, V10;  \
	VSUBVV   V11, V10, V10;  \
	VMULVX   TAPR, V10, V10; \
	VADDVV   V10, V2, V2

// TAPC also tracks the range the result is clipped to; the fill loses the
// signed maximum, and read as unsigned it loses the unsigned minimum.
#define TAPC(OP, OFFR, THRV, SHR, TAPR) \
	TAP(OP, OFFR, THRV, SHR, TAPR); \
	VMAXVV  V5, V3, V3;      \
	VMINUVV V5, V4, V4

#define SETUP                     \
	MOV     dst+0(FP), X10;   \
	MOV     dstStride+8(FP), X11; \
	MOV     tmp+16(FP), X12;  \
	MOV     p+24(FP), X13;    \
	MOVW    0(X13), X18;      \
	MOVW    4(X13), X19;      \
	MOVW    8(X13), X20;      \
	MOVW    12(X13), X21;     \
	MOVW    16(X13), X22;     \
	MOVW    20(X13), X23;     \
	SLLI    $1, X18, X18;     \
	SLLI    $1, X19, X19;     \
	SLLI    $1, X20, X20;     \
	SLLI    $1, X21, X21;     \
	SLLI    $1, X22, X22;     \
	SLLI    $1, X23, X23;     \
	MOVW    32(X13), X8;      \
	MOVW    36(X13), X9;      \
	MOVW    56(X13), X14;     \
	MOVW    60(X13), X7;      \
	VSETVLI X14, E16, M1, TA, MA, X5; \
	MOVW    24(X13), X6;      \
	VMVVX   X6, V12;          \
	MOVW    28(X13), X6;      \
	VMVVX   X6, V13

#define LOADPX                    \
	VSETVLI  X5, E8, MF2, TA, MA, X6; \
	VLE8V    (X10), V14;      \
	VSETVLI  X5, E16, M1, TA, MA, X6; \
	VZEXTVF2 V14, V1;         \
	VMVVX    X0, V2

// LOADPX16 needs no widening, and SETUP16 adds the pixel maximum the narrowing
// store used to clamp to, plus the stride in bytes.
#define LOADPX16                  \
	VLE16V (X10), V1;         \
	VMVVX  X0, V2

#define SETUP16                   \
	SETUP;                    \
	SLLI $1, X11, X11;        \
	MOVW 64(X13), X24

#define FINISH16                  \
	VSRAVI $15, V2, V11;      \
	VADDVV V11, V2, V2;       \
	VADDVI $8, V2, V2;        \
	VSRAVI $4, V2, V2;        \
	VADDVV V1, V2, V2

#define STORE16                   \
	VSE16V V2, (X10);         \
	ADD    X11, X10;          \
	ADDI   $24, X12;          \
	ADDI   $-1, X7

#define FINISH                    \
	VSRAVI   $15, V2, V11;    \
	VADDVV   V11, V2, V2;     \
	VADDVI   $8, V2, V2;      \
	VSRAVI   $4, V2, V2;      \
	VADDVV   V1, V2, V2;      \
	VSETVLI  X5, E8, MF2, TA, MA, X6; \
	VNSRLWI  $0, V2, V15;     \
	VSE8V    V15, (X10);      \
	VSETVLI  X5, E16, M1, TA, MA, X6; \
	ADD      X11, X10;        \
	ADDI     $24, X12;        \
	ADDI     $-1, X7

// TAPP stacks the samples of two rows, so a block only four wide still fills
// eight lanes. The second load starts four elements early, which lands that
// row's samples in the upper lanes the mask selects.
#define TAPP(OP, OFFR, THRV, SHR, TAPR) \
	OP        OFFR, X12, X6;   \
	VLE16V    (X6), V5;        \
	ADDI      $16, X6;         \
	VLE16V    (X6), V16;       \
	VMERGEVVM V16, V5, V0, V5; \
	VSUBVV    V1, V5, V6;      \
	VRSUBVX   X0, V6, V7;      \
	VMAXVV    V6, V7, V7;      \
	VSRLVX    SHR, V7, V8;     \
	VSSUBUVV  V8, THRV, V9;    \
	VMINUVV   V7, V9, V10;     \
	VSRAVI    $15, V6, V11;    \
	VXORVV    V11, V10, V10;   \
	VSUBVV    V11, V10, V10;   \
	VMULVX    TAPR, V10, V10;  \
	VADDVV    V10, V2, V2

#define TAPPC(OP, OFFR, THRV, SHR, TAPR) \
	TAPP(OP, OFFR, THRV, SHR, TAPR); \
	VMAXVV  V5, V3, V3;        \
	VMINUVV V5, V4, V4

#define SETUP4                            \
	SETUP;                            \
	MOV      $4, X16;                 \
	SLLI     $1, X14, X14;            \
	VSETVLI  X14, E16, M1, TA, MA, X5; \
	MOVW     24(X13), X6;             \
	VMVVX    X6, V12;                 \
	MOVW     28(X13), X6;             \
	VMVVX    X6, V13;                 \
	VIDV     V17;                     \
	VMSGTUVI $3, V17, V0

#define SETUP416   \
	SETUP4;    \
	SLLI $1, X11, X11; \
	MOVW 64(X13), X24

#define LOADPX4                            \
	VSETVLI    X16, E8, MF2, TA, MA, X6; \
	VLE8V      (X10), V14;             \
	ADD        X11, X10, X6;           \
	VLE8V      (X6), V15;              \
	VSETVLI    X5, E8, MF2, TA, MA, X6; \
	VSLIDEUPVX X16, V15, V14;          \
	VSETVLI    X5, E16, M1, TA, MA, X6; \
	VZEXTVF2   V14, V1;                \
	VMVVX      X0, V2

#define LOADPX416                          \
	VSETVLI    X16, E16, M1, TA, MA, X6; \
	VLE16V     (X10), V1;              \
	ADD        X11, X10, X6;           \
	VLE16V     (X6), V15;              \
	VSETVLI    X5, E16, M1, TA, MA, X6; \
	VSLIDEUPVX X16, V15, V1;           \
	VMVVX      X0, V2

#define FINISH4            \
	VSRAVI $15, V2, V11; \
	VADDVV V11, V2, V2;  \
	VADDVI $8, V2, V2;   \
	VSRAVI $4, V2, V2;   \
	VADDVV V1, V2, V2

#define ADVANCE4      \
	ADD  X11, X10; \
	ADD  X11, X10; \
	ADDI $48, X12; \
	ADDI $-2, X7

#define STORE4                               \
	VSETVLI      X5, E8, MF2, TA, MA, X6; \
	VNSRLWI      $0, V2, V15;            \
	VSETVLI      X16, E8, MF2, TA, MA, X6; \
	VSE8V        V15, (X10);             \
	VSLIDEDOWNVX X16, V15, V18;          \
	ADD          X11, X10, X6;           \
	VSE8V        V18, (X6);              \
	VSETVLI      X5, E16, M1, TA, MA, X6; \
	ADVANCE4

#define STORE416                             \
	VSETVLI      X16, E16, M1, TA, MA, X6; \
	VSE16V       V2, (X10);              \
	VSLIDEDOWNVX X16, V2, V18;           \
	ADD          X11, X10, X6;           \
	VSE16V       V18, (X6);              \
	VSETVLI      X5, E16, M1, TA, MA, X6; \
	ADVANCE4

// func cdefFilterRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterRVV(SB), NOSPLIT, $0-32
	SETUP

loop:
	LOADPX

	MOVW 40(X13), X15
	TAP(ADD, X18, V12, X8, X15)
	TAP(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAP(ADD, X19, V12, X8, X15)
	TAP(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAP(ADD, X20, V13, X9, X15)
	TAP(SUB, X20, V13, X9, X15)
	TAP(ADD, X22, V13, X9, X15)
	TAP(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAP(ADD, X21, V13, X9, X15)
	TAP(SUB, X21, V13, X9, X15)
	TAP(ADD, X23, V13, X9, X15)
	TAP(SUB, X23, V13, X9, X15)

	FINISH
	BNEZ X7, loop

	RET

// func cdefFilterClipRVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClipRVV(SB), NOSPLIT, $0-32
	SETUP

loopc:
	LOADPX
	VMVVV V1, V3
	VMVVV V1, V4

	MOVW 40(X13), X15
	TAPC(ADD, X18, V12, X8, X15)
	TAPC(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPC(ADD, X19, V12, X8, X15)
	TAPC(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPC(ADD, X20, V13, X9, X15)
	TAPC(SUB, X20, V13, X9, X15)
	TAPC(ADD, X22, V13, X9, X15)
	TAPC(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPC(ADD, X21, V13, X9, X15)
	TAPC(SUB, X21, V13, X9, X15)
	TAPC(ADD, X23, V13, X9, X15)
	TAPC(SUB, X23, V13, X9, X15)

	VSRAVI  $15, V2, V11
	VADDVV  V11, V2, V2
	VADDVI  $8, V2, V2
	VSRAVI  $4, V2, V2
	VADDVV  V1, V2, V2
	VMAXVV  V4, V2, V2
	VMINVV  V3, V2, V2
	VSETVLI X5, E8, MF2, TA, MA, X6
	VNSRLWI $0, V2, V15
	VSE8V   V15, (X10)
	VSETVLI X5, E16, M1, TA, MA, X6
	ADD     X11, X10
	ADDI    $24, X12
	ADDI    $-1, X7
	BNEZ    X7, loopc

	RET

// func cdefFilter16RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter16RVV(SB), NOSPLIT, $0-32
	SETUP16

loop16:
	LOADPX16

	MOVW 40(X13), X15
	TAP(ADD, X18, V12, X8, X15)
	TAP(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAP(ADD, X19, V12, X8, X15)
	TAP(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAP(ADD, X20, V13, X9, X15)
	TAP(SUB, X20, V13, X9, X15)
	TAP(ADD, X22, V13, X9, X15)
	TAP(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAP(ADD, X21, V13, X9, X15)
	TAP(SUB, X21, V13, X9, X15)
	TAP(ADD, X23, V13, X9, X15)
	TAP(SUB, X23, V13, X9, X15)

	FINISH16
	VMAXVX X0, V2, V2
	VMINVX X24, V2, V2
	STORE16
	BNEZ X7, loop16

	RET

// func cdefFilterClip16RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip16RVV(SB), NOSPLIT, $0-32
	SETUP16

loopc16:
	LOADPX16
	VMVVV V1, V3
	VMVVV V1, V4

	MOVW 40(X13), X15
	TAPC(ADD, X18, V12, X8, X15)
	TAPC(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPC(ADD, X19, V12, X8, X15)
	TAPC(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPC(ADD, X20, V13, X9, X15)
	TAPC(SUB, X20, V13, X9, X15)
	TAPC(ADD, X22, V13, X9, X15)
	TAPC(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPC(ADD, X21, V13, X9, X15)
	TAPC(SUB, X21, V13, X9, X15)
	TAPC(ADD, X23, V13, X9, X15)
	TAPC(SUB, X23, V13, X9, X15)

	FINISH16
	VMAXVV V4, V2, V2
	VMINVV V3, V2, V2
	STORE16
	BNEZ X7, loopc16

	RET

// func cdefFilter4RVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter4RVV(SB), NOSPLIT, $0-32
	SETUP4

loop4:
	LOADPX4

	MOVW 40(X13), X15
	TAPP(ADD, X18, V12, X8, X15)
	TAPP(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPP(ADD, X19, V12, X8, X15)
	TAPP(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPP(ADD, X20, V13, X9, X15)
	TAPP(SUB, X20, V13, X9, X15)
	TAPP(ADD, X22, V13, X9, X15)
	TAPP(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPP(ADD, X21, V13, X9, X15)
	TAPP(SUB, X21, V13, X9, X15)
	TAPP(ADD, X23, V13, X9, X15)
	TAPP(SUB, X23, V13, X9, X15)

	FINISH4
	STORE4
	BNEZ X7, loop4

	RET

// func cdefFilterClip4RVV(dst *uint8, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip4RVV(SB), NOSPLIT, $0-32
	SETUP4

loopc4:
	LOADPX4
	VMVVV V1, V3
	VMVVV V1, V4

	MOVW 40(X13), X15
	TAPPC(ADD, X18, V12, X8, X15)
	TAPPC(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPPC(ADD, X19, V12, X8, X15)
	TAPPC(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPPC(ADD, X20, V13, X9, X15)
	TAPPC(SUB, X20, V13, X9, X15)
	TAPPC(ADD, X22, V13, X9, X15)
	TAPPC(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPPC(ADD, X21, V13, X9, X15)
	TAPPC(SUB, X21, V13, X9, X15)
	TAPPC(ADD, X23, V13, X9, X15)
	TAPPC(SUB, X23, V13, X9, X15)

	FINISH4
	VMAXVV V4, V2, V2
	VMINVV V3, V2, V2
	STORE4
	BNEZ X7, loopc4

	RET

// func cdefFilter416RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilter416RVV(SB), NOSPLIT, $0-32
	SETUP416

loop416:
	LOADPX416

	MOVW 40(X13), X15
	TAPP(ADD, X18, V12, X8, X15)
	TAPP(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPP(ADD, X19, V12, X8, X15)
	TAPP(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPP(ADD, X20, V13, X9, X15)
	TAPP(SUB, X20, V13, X9, X15)
	TAPP(ADD, X22, V13, X9, X15)
	TAPP(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPP(ADD, X21, V13, X9, X15)
	TAPP(SUB, X21, V13, X9, X15)
	TAPP(ADD, X23, V13, X9, X15)
	TAPP(SUB, X23, V13, X9, X15)

	FINISH4
	VMAXVX X0, V2, V2
	VMINVX X24, V2, V2
	STORE416
	BNEZ X7, loop416

	RET

// func cdefFilterClip416RVV(dst *uint16, dstStride int, tmp *int16, p *cdefParams)
TEXT ·cdefFilterClip416RVV(SB), NOSPLIT, $0-32
	SETUP416

loopc416:
	LOADPX416
	VMVVV V1, V3
	VMVVV V1, V4

	MOVW 40(X13), X15
	TAPPC(ADD, X18, V12, X8, X15)
	TAPPC(SUB, X18, V12, X8, X15)
	MOVW 44(X13), X15
	TAPPC(ADD, X19, V12, X8, X15)
	TAPPC(SUB, X19, V12, X8, X15)

	MOVW 48(X13), X15
	TAPPC(ADD, X20, V13, X9, X15)
	TAPPC(SUB, X20, V13, X9, X15)
	TAPPC(ADD, X22, V13, X9, X15)
	TAPPC(SUB, X22, V13, X9, X15)
	MOVW 52(X13), X15
	TAPPC(ADD, X21, V13, X9, X15)
	TAPPC(SUB, X21, V13, X9, X15)
	TAPPC(ADD, X23, V13, X9, X15)
	TAPPC(SUB, X23, V13, X9, X15)

	FINISH4
	VMAXVV V4, V2, V2
	VMINVV V3, V2, V2
	STORE416
	BNEZ X7, loopc416

	RET
