//go:build amd64 && !noasm

#include "textflag.h"

// func itxClipAVX2(tmp *int32, n int, rnd int32, shift int, lo, hi int32)
TEXT ·itxClipAVX2(SB), NOSPLIT, $0-40
	MOVQ         tmp+0(FP), DI
	MOVQ         n+8(FP), CX
	MOVL         rnd+16(FP), AX
	VMOVD        AX, X1
	VPBROADCASTD X1, Y1
	MOVQ         shift+24(FP), AX
	VMOVD        AX, X2
	MOVL         lo+32(FP), AX
	VMOVD        AX, X3
	VPBROADCASTD X3, Y3
	MOVL         hi+36(FP), AX
	VMOVD        AX, X4
	VPBROADCASTD X4, Y4

loop:
	VMOVDQU  (DI), Y0
	VPADDD   Y1, Y0, Y0
	VPSRAD   X2, Y0, Y0
	VPMAXSD  Y3, Y0, Y0
	VPMINSD  Y4, Y0, Y0
	VMOVDQU  Y0, (DI)
	ADDQ     $32, DI
	SUBQ     $8, CX
	JNZ      loop

	VZEROUPPER
	RET

// ADDROW folds one vector of residual into the destination.
#define ADDROW(PX)                 \
	VMOVDQU (SI), Y0;          \
	VPADDD  Y5, Y0, Y0;        \
	VPSRAD  $4, Y0, Y0;        \
	VPADDD  PX, Y0, Y0;        \
	VPMAXSD Y6, Y0, Y0;        \
	VPMINSD Y7, Y0, Y0

// func itxAdd8AVX2(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd8AVX2(SB), NOSPLIT, $0-44
	MOVQ         dst+0(FP), DI
	MOVQ         stride+8(FP), DX
	MOVQ         tmp+16(FP), SI
	MOVQ         w+24(FP), BX
	MOVQ         h+32(FP), CX
	MOVL         bitdepthMax+40(FP), AX
	VMOVD        AX, X7
	VPBROADCASTD X7, Y7
	VPXOR        Y6, Y6, Y6
	MOVL         $8, AX
	VMOVD        AX, X5
	VPBROADCASTD X5, Y5

rows8:
	XORQ R8, R8

cols8:
	VPMOVZXBD (DI)(R8*1), Y1
	ADDROW(Y1)
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VPACKUSWB X0, X0, X0
	VMOVQ     X0, (DI)(R8*1)
	ADDQ      $32, SI
	ADDQ      $8, R8
	CMPQ      R8, BX
	JLT       cols8

	ADDQ DX, DI
	DECQ CX
	JNZ  rows8

	VZEROUPPER
	RET

// func itxAdd16AVX2(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd16AVX2(SB), NOSPLIT, $0-44
	MOVQ         dst+0(FP), DI
	MOVQ         stride+8(FP), DX
	MOVQ         tmp+16(FP), SI
	MOVQ         w+24(FP), BX
	MOVQ         h+32(FP), CX
	MOVL         bitdepthMax+40(FP), AX
	VMOVD        AX, X7
	VPBROADCASTD X7, Y7
	VPXOR        Y6, Y6, Y6
	MOVL         $8, AX
	VMOVD        AX, X5
	VPBROADCASTD X5, Y5
	SHLQ         $1, DX

rows16:
	XORQ R8, R8

cols16:
	VPMOVZXWD (DI)(R8*2), Y1
	ADDROW(Y1)
	VPACKUSDW Y0, Y0, Y0
	VPERMQ    $0x08, Y0, Y0
	VMOVDQU   X0, (DI)(R8*2)
	ADDQ      $32, SI
	ADDQ      $8, R8
	CMPQ      R8, BX
	JLT       cols16

	ADDQ DX, DI
	DECQ CX
	JNZ  rows16

	VZEROUPPER
	RET

#define LOAD8(P, S)     \
	VMOVDQU (P), Y0; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y1; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y2; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y3; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y4; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y5; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y6; \
	ADDQ    S, P;    \
	VMOVDQU (P), Y7

#define STORE8(P, S)     \
	VMOVDQU Y8, (P);  \
	ADDQ    S, P;     \
	VMOVDQU Y9, (P);  \
	ADDQ    S, P;     \
	VMOVDQU Y10, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y11, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y12, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y13, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y14, (P); \
	ADDQ    S, P;     \
	VMOVDQU Y15, (P)

// func itxTransposeAVX2(wide, out *int32, w, sh, n int)
TEXT ·itxTransposeAVX2(SB), NOSPLIT, $0-40
	MOVQ wide+0(FP), SI
	MOVQ out+8(FP), DI
	MOVQ w+16(FP), BX
	MOVQ sh+24(FP), DX
	MOVQ n+32(FP), CX
	SHLQ $2, DX
	MOVQ BX, R9
	SHLQ $2, R9

trlane:
	MOVQ SI, R10
	MOVQ DI, R11
	MOVQ BX, R12

trtap:
	MOVQ R10, R13
	LOAD8(R13, DX)

	VPUNPCKLDQ Y1, Y0, Y8
	VPUNPCKHDQ Y1, Y0, Y9
	VPUNPCKLDQ Y3, Y2, Y10
	VPUNPCKHDQ Y3, Y2, Y11
	VPUNPCKLDQ Y5, Y4, Y12
	VPUNPCKHDQ Y5, Y4, Y13
	VPUNPCKLDQ Y7, Y6, Y14
	VPUNPCKHDQ Y7, Y6, Y15

	VPUNPCKLQDQ Y10, Y8, Y0
	VPUNPCKHQDQ Y10, Y8, Y1
	VPUNPCKLQDQ Y11, Y9, Y2
	VPUNPCKHQDQ Y11, Y9, Y3
	VPUNPCKLQDQ Y14, Y12, Y4
	VPUNPCKHQDQ Y14, Y12, Y5
	VPUNPCKLQDQ Y15, Y13, Y6
	VPUNPCKHQDQ Y15, Y13, Y7

	VPERM2I128 $0x20, Y4, Y0, Y8
	VPERM2I128 $0x20, Y5, Y1, Y9
	VPERM2I128 $0x20, Y6, Y2, Y10
	VPERM2I128 $0x20, Y7, Y3, Y11
	VPERM2I128 $0x31, Y4, Y0, Y12
	VPERM2I128 $0x31, Y5, Y1, Y13
	VPERM2I128 $0x31, Y6, Y2, Y14
	VPERM2I128 $0x31, Y7, Y3, Y15

	MOVQ R11, R13
	STORE8(R13, R9)

	LEAQ (R10)(DX*8), R10
	ADDQ $32, R11
	SUBQ $8, R12
	JNZ  trtap

	ADDQ $32, SI
	LEAQ (DI)(R9*8), DI
	SUBQ $8, CX
	JNZ  trlane

	VZEROUPPER
	RET
