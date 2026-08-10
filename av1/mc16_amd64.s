//go:build amd64 && !noasm

#include "textflag.h"

// At high bit depth the samples no longer fit the byte multiply the 8-bit
// kernels use, so every tap goes through a 32-bit multiply. That makes the tap
// stride free, which is what lets one kernel serve both directions.

// FSETUP broadcasts the eight taps, which the caller has widened to dwords.
#define FSETUP                    \
	MOVQ         f+40(FP), AX; \
	VPBROADCASTD (AX), Y8;    \
	VPBROADCASTD 4(AX), Y9;   \
	VPBROADCASTD 8(AX), Y10;  \
	VPBROADCASTD 12(AX), Y11; \
	VPBROADCASTD 16(AX), Y12; \
	VPBROADCASTD 20(AX), Y13; \
	VPBROADCASTD 24(AX), Y14; \
	VPBROADCASTD 28(AX), Y15; \
	MOVQ         dst+0(FP), DI; \
	MOVQ         src+16(FP), SI; \
	MOVQ         tapStride+32(FP), R10; \
	SHLQ         $1, R10;     \
	MOVQ         rnd+64(FP), AX; \
	VMOVD        AX, X7;      \
	VPBROADCASTD X7, Y7;      \
	MOVQ         shift+72(FP), AX; \
	VMOVD        AX, X6;      \
	MOVQ         maxv+80(FP), AX; \
	VMOVD        AX, X5;      \
	VPBROADCASTD X5, Y5;      \
	VPXOR        Y4, Y4, Y4;  \
	MOVQ         h+56(FP), R12

// TAPS8 accumulates the eight taps of one output group into Y0.
#define TAPS8(WIDEN)     \
	WIDEN (CX), Y1;  \
	VPMULLD Y8, Y1, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y9, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y10, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y11, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y12, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y13, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y14, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	ADDQ  R10, CX;   \
	WIDEN (CX), Y1;  \
	VPMULLD Y15, Y1, Y1; \
	VPADDD Y1, Y0, Y0; \
	VPADDD Y7, Y0, Y0; \
	VPSRAD X6, Y0, Y0

#define ROWSTEP(NAME)              \
	MOVQ srcStride+24(FP), AX; \
	SHLQ $1, AX;               \
	ADDQ AX, SI;               \
	MOVQ dstStride+8(FP), AX;  \
	SHLQ $1, AX;               \
	ADDQ AX, DI;               \
	DECQ R12;                  \
	JNZ  NAME

// func mcTap16AVX2(dst *uint16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTap16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows1:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13

cols1:

	MOVQ R11, CX
	TAPS8(VPMOVZXWD)
	VPMAXSD   Y4, Y0, Y0
	VPMINSD   Y5, Y0, Y0
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (R13)
	ADDQ      $16, R11
	ADDQ      $16, R13
	SUBQ      $8, DX
	JNZ       cols1

	ROWSTEP(rows1)
	VZEROUPPER
	RET

// func mcTapMid16AVX2(dst *int16, dstStride int, src *uint16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows2:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13

cols2:
	MOVQ R11, CX
	TAPS8(VPMOVZXWD)
	VPACKSSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (R13)
	ADDQ      $16, R11
	ADDQ      $16, R13
	SUBQ      $8, DX
	JNZ       cols2

	ROWSTEP(rows2)
	VZEROUPPER
	RET

// func mcTapFromMid16AVX2(dst *uint16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapFromMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows3:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13

cols3:
	MOVQ R11, CX
	TAPS8(VPMOVSXWD)
	VPMAXSD   Y4, Y0, Y0
	VPMINSD   Y5, Y0, Y0
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (R13)
	ADDQ      $16, R11
	ADDQ      $16, R13
	SUBQ      $8, DX
	JNZ       cols3

	ROWSTEP(rows3)
	VZEROUPPER
	RET

// func mcTapPrepMid16AVX2(dst *int16, dstStride int, src *int16, srcStride, tapStride int, f *[8]int32, w, h int, rnd int32, shift int, maxv int32)
TEXT ·mcTapPrepMid16AVX2(SB), NOSPLIT, $0-84
	FSETUP

rows4:
	MOVQ w+48(FP), DX
	MOVQ SI, R11
	MOVQ DI, R13

cols4:
	MOVQ R11, CX
	TAPS8(VPMOVSXWD)
	VPACKSSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (R13)
	ADDQ      $16, R11
	ADDQ      $16, R13
	SUBQ      $8, DX
	JNZ       cols4

	ROWSTEP(rows4)
	VZEROUPPER
	RET
