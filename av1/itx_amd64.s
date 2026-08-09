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
