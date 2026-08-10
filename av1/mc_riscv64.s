//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// LOADTAPS sign-extends the eight coefficients into scalar registers; the
// vector multiplies take them from there.
#define LOADTAPS(REG)     \
	MOVB 0(REG), X17; \
	MOVB 1(REG), X18; \
	MOVB 2(REG), X19; \
	MOVB 3(REG), X20; \
	MOVB 4(REG), X21; \
	MOVB 5(REG), X22; \
	MOVB 6(REG), X23; \
	MOVB 7(REG), X24

// TAPS8 filters vl pixels from X30, stepping between taps by STEP. Products and
// their sum stay inside a halfword, so the loads widen once and stop there.
#define TAPS8(STEP)                      \
	VSETVLI  X28, E8, MF2, TA, MA, X29; \
	VLE8V    (X30), V1;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V2;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V3;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V4;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V5;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V6;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V7;              \
	ADD      STEP, X30;              \
	VLE8V    (X30), V8;              \
	VSETVLI  X28, E16, M1, TA, MA, X29; \
	VZEXTVF2 V1, V9;                 \
	VMULVX   X17, V9, V10;           \
	VZEXTVF2 V2, V9;                 \
	VMULVX   X18, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V3, V9;                 \
	VMULVX   X19, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V4, V9;                 \
	VMULVX   X20, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V5, V9;                 \
	VMULVX   X21, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V6, V9;                 \
	VMULVX   X22, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V7, V9;                 \
	VMULVX   X23, V9, V9;            \
	VADDVV   V9, V10, V10;           \
	VZEXTVF2 V8, V9;                 \
	VMULVX   X24, V9, V9;            \
	VADDVV   V9, V10, V10

// STOREPX clips to the pixel range and narrows back to bytes.
#define STOREPX(DST)                     \
	VMAXVX   X0, V10, V10;           \
	VMINVX   X26, V10, V10;          \
	VSETVLI  X28, E8, MF2, TA, MA, X29; \
	VNSRLWI  $0, V10, V11;           \
	VSE8V    V11, (DST);             \
	VSETVLI  X28, E16, M1, TA, MA, X29

// func put8tapHRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapHRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV f+32(FP), X14
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	LOADTAPS(X14)

	MOV $512, X25
	MOV $255, X26
	MOV $1, X6

hrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

hcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X6)
	VADDVI  $2, V10, V10
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, hcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, hrow

	RET

// func put8tapVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·put8tapVRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV f+32(FP), X14
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	LOADTAPS(X14)

	MOV $512, X25
	MOV $255, X26

vrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

vcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X13)
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, vcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, vrow

	RET

// func put8tapHVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, fh, fv *[8]int8, w, h int)
TEXT ·put8tapHVRVV(SB), NOSPLIT, $0-72
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV mid+32(FP), X5
	MOV w+56(FP), X15
	MOV h+64(FP), X16

	MOV $255, X26
	MOV $1, X6

hvstrip:
	// A strip is capped at one vector and at what the scratch holds, so
	// neither pass needs a column loop.
	MOV     $16, X29
	MIN     X15, X29, X29
	VSETVLI X29, E16, M1, TA, MA, X28
	SLLI    $1, X28, X14

	MOV  fh+40(FP), X25
	LOADTAPS(X25)
	MOV  $8192, X25
	MOV  X12, X8
	MOV  X5, X9
	MOV  X16, X7
	ADDI $7, X7

hvh:
	MOV     X8, X30
	TAPS8(X6)
	VSMULVX X25, V10, V10
	VSE16V  V10, (X9)

	ADD  X13, X8
	ADD  X14, X9
	ADDI $-1, X7
	BNEZ X7, hvh

	MOV fv+48(FP), X25
	LOADTAPS(X25)
	MOV X5, X9
	MOV X10, X8
	MOV X16, X7

hvv:
	MOV      X9, X30
	VLE16V   (X30), V1
	VWMULVX  X17, V1, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X18, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X19, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X20, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X21, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X22, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X23, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X24, V16

	// vnclip rounds the way rnd8 does, so the shift and the bias are one step.
	VNCLIPWI $10, V16, V10
	STOREPX(X8)

	ADD  X14, X9
	ADD  X11, X8
	ADDI $-1, X7
	BNEZ X7, hvv

	ADD  X28, X12
	ADD  X28, X10
	SUB  X28, X15
	BNEZ X15, hvstrip

	RET

// func prep8tapHRVV(tmp *int16, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·prep8tapHRVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV f+24(FP), X14
	MOV w+32(FP), X15
	MOV h+40(FP), X16

	LOADTAPS(X14)

	MOV $8192, X25
	MOV $1, X6

prph:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

prphcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X6)
	VSMULVX X25, V10, V10
	VSE16V  V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prphcol

	ADD  X13, X12
	SLLI $1, X15, X29
	ADD  X29, X10
	ADDI $-1, X16
	BNEZ X16, prph

	RET

// func prep8tapVRVV(tmp *int16, src *uint8, srcStride int, f *[8]int8, w, h int)
TEXT ·prep8tapVRVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV f+24(FP), X14
	MOV w+32(FP), X15
	MOV h+40(FP), X16

	LOADTAPS(X14)

	MOV $8192, X25

prpv:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

prpvcol:
	VSETVLI X7, E16, M1, TA, MA, X28
	MOV     X8, X30
	TAPS8(X13)
	VSMULVX X25, V10, V10
	VSE16V  V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prpvcol

	ADD  X13, X12
	SLLI $1, X15, X29
	ADD  X29, X10
	ADDI $-1, X16
	BNEZ X16, prpv

	RET

// func prep8tapHVRVV(tmp *int16, src *uint8, srcStride int, mid *int16, fh, fv *[8]int8, w, h int)
TEXT ·prep8tapHVRVV(SB), NOSPLIT, $0-64
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV mid+24(FP), X5
	MOV w+48(FP), X15
	MOV h+56(FP), X16
	MOV X15, X31

	MOV $1, X6

prphvstrip:
	MOV     $16, X29
	MIN     X15, X29, X29
	VSETVLI X29, E16, M1, TA, MA, X28
	SLLI    $1, X28, X14

	MOV  fh+32(FP), X25
	LOADTAPS(X25)
	MOV  $8192, X25
	MOV  X12, X8
	MOV  X5, X9
	MOV  X16, X7
	ADDI $7, X7

prphvh:
	MOV     X8, X30
	TAPS8(X6)
	VSMULVX X25, V10, V10
	VSE16V  V10, (X9)

	ADD  X13, X8
	ADD  X14, X9
	ADDI $-1, X7
	BNEZ X7, prphvh

	MOV fv+40(FP), X25
	LOADTAPS(X25)
	MOV X5, X9
	MOV X10, X8
	MOV X16, X7

prphvv:
	MOV      X9, X30
	VLE16V   (X30), V1
	VWMULVX  X17, V1, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X18, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X19, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X20, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X21, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X22, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X23, V16
	ADD      X14, X30
	VLE16V   (X30), V1
	VWMACCVX V1, X24, V16

	VNCLIPWI $6, V16, V10
	VSE16V   V10, (X8)

	ADD  X14, X9
	SLLI $1, X31, X29
	ADD  X29, X8
	ADDI $-1, X7
	BNEZ X7, prphvv

	ADD  X28, X12
	SLLI $1, X28, X29
	ADD  X29, X10
	SUB  X28, X15
	BNEZ X15, prphvstrip

	RET

#define BILHR                             \
	VSETVLI  X7, E8, MF2, TA, MA, X28; \
	VLE8V    (X8), V1;                \
	ADDI     $1, X8, X6;              \
	VLE8V    (X6), V2;                \
	VSETVLI  X7, E16, M1, TA, MA, X28; \
	VZEXTVF2 V1, V3;                  \
	VZEXTVF2 V2, V4;                  \
	VMULVX   X17, V3, V10;            \
	VMACCVX  V4, X18, V10

#define BILVR                             \
	VSETVLI  X7, E8, MF2, TA, MA, X28; \
	VLE8V    (X8), V1;                \
	ADD      X13, X8, X6;             \
	VLE8V    (X6), V2;                \
	VSETVLI  X7, E16, M1, TA, MA, X28; \
	VZEXTVF2 V1, V3;                  \
	VZEXTVF2 V2, V4;                  \
	VMULVX   X17, V3, V10;            \
	VMACCVX  V4, X18, V10

#define BILCOEFR(OFF)   \
	MOVW OFF, X18;  \
	MOV  $16, X17;  \
	SUB  X18, X17

// func putBilinHRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mx int32, w, h int)
TEXT ·putBilinHRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	BILCOEFR(mx+32(FP))
	MOV $2048, X25
	MOV $255, X26

blhrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

blhcol:
	BILHR
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, blhcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, blhrow

	RET

// func putBilinVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, my int32, w, h int)
TEXT ·putBilinVRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	BILCOEFR(my+32(FP))
	MOV $2048, X25
	MOV $255, X26

blvrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

blvcol:
	BILVR
	VSMULVX X25, V10, V10
	STOREPX(X9)

	ADD  X28, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, blvcol

	ADD  X13, X12
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, blvrow

	RET

// func prepBilinHRVV(tmp *int16, src *uint8, srcStride int, mx int32, w, h int)
TEXT ·prepBilinHRVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV w+32(FP), X15
	MOV h+40(FP), X16

	BILCOEFR(mx+24(FP))

prblhrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

prblhcol:
	BILHR
	VSE16V V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prblhcol

	ADD  X13, X12
	SLLI $1, X15, X29
	ADD  X29, X10
	ADDI $-1, X16
	BNEZ X16, prblhrow

	RET

// func prepBilinVRVV(tmp *int16, src *uint8, srcStride int, my int32, w, h int)
TEXT ·prepBilinVRVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV w+32(FP), X15
	MOV h+40(FP), X16

	BILCOEFR(my+24(FP))

prblvrow:
	MOV X12, X8
	MOV X10, X9
	MOV X15, X7

prblvcol:
	BILVR
	VSE16V V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prblvcol

	ADD  X13, X12
	SLLI $1, X15, X29
	ADD  X29, X10
	ADDI $-1, X16
	BNEZ X16, prblvrow

	RET

#define BILVMIDR(SHIFT)                    \
	VSETVLI  X7, E16, M1, TA, MA, X28; \
	VLE16V   (X8), V1;                 \
	ADD      X14, X8, X6;              \
	VLE16V   (X6), V2;                 \
	VWMULVX  X17, V1, V16;             \
	VWMACCVX V2, X18, V16;             \
	VNCLIPWI SHIFT, V16, V10

// func putBilinHVRVV(dst *uint8, dstStride int, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)
TEXT ·putBilinHVRVV(SB), NOSPLIT, $0-64
	MOV dst+0(FP), X10
	MOV dstStride+8(FP), X11
	MOV src+16(FP), X12
	MOV srcStride+24(FP), X13
	MOV mid+32(FP), X5
	MOV w+48(FP), X15
	MOV h+56(FP), X16

	BILCOEFR(mx+40(FP))
	MOV  $255, X26
	SLLI $1, X15, X14
	MOV  X5, X31
	MOV  X16, X30
	ADDI $1, X30

blhvh:
	MOV X12, X8
	MOV X31, X9
	MOV X15, X7

blhvhcol:
	BILHR
	VSE16V V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, blhvhcol

	ADD  X13, X12
	ADD  X14, X31
	ADDI $-1, X30
	BNEZ X30, blhvh

	BILCOEFR(my+44(FP))
	MOV X5, X31

blhvv:
	MOV X31, X8
	MOV X10, X9
	MOV X15, X7

blhvvcol:
	BILVMIDR($8)
	STOREPX(X9)

	SLLI $1, X28, X29
	ADD  X29, X8
	ADD  X28, X9
	SUB  X28, X7
	BNEZ X7, blhvvcol

	ADD  X14, X31
	ADD  X11, X10
	ADDI $-1, X16
	BNEZ X16, blhvv

	RET

// func prepBilinHVRVV(tmp *int16, src *uint8, srcStride int, mid *int16, mx, my int32, w, h int)
TEXT ·prepBilinHVRVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X10
	MOV src+8(FP), X12
	MOV srcStride+16(FP), X13
	MOV mid+24(FP), X5
	MOV w+40(FP), X15
	MOV h+48(FP), X16

	BILCOEFR(mx+32(FP))
	SLLI $1, X15, X14
	MOV  X5, X31
	MOV  X16, X30
	ADDI $1, X30

prblhvh:
	MOV X12, X8
	MOV X31, X9
	MOV X15, X7

prblhvhcol:
	BILHR
	VSE16V V10, (X9)

	ADD  X28, X8
	SLLI $1, X28, X29
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prblhvhcol

	ADD  X13, X12
	ADD  X14, X31
	ADDI $-1, X30
	BNEZ X30, prblhvh

	BILCOEFR(my+36(FP))
	MOV X5, X31

prblhvv:
	MOV X31, X8
	MOV X10, X9
	MOV X15, X7

prblhvvcol:
	BILVMIDR($4)
	VSE16V V10, (X9)

	SLLI $1, X28, X29
	ADD  X29, X8
	ADD  X29, X9
	SUB  X28, X7
	BNEZ X7, prblhvvcol

	ADD  X14, X31
	ADD  X14, X10
	ADDI $-1, X16
	BNEZ X16, prblhvv

	RET
