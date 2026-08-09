//go:build amd64 && !noasm

#include "textflag.h"

// The a rows are int32 and the b rows int16, three taps apart in each, so the
// taps are fixed displacements off one pointer per row.

// SUM3 folds the three taps of one int32 row, scaled, into ACC.
#define LOADB(OFF, REG, VD) \
	VPMOVSXWD OFF(REG), VD

// TAIL turns the two accumulators into eight samples and stores them.
#define TAIL(SHIFT)           \
	VPMOVZXBD (SI), Y4;   \
	VPMULLD   Y4, Y3, Y3; \
	VPSUBD    Y3, Y0, Y0; \
	VPADDD    Y6, Y0, Y0; \
	VPSRAD    SHIFT, Y0, Y0; \
	VPACKSSDW Y0, Y0, Y0; \
	VPERMQ    $0x08, Y0, Y0; \
	VMOVDQU   X0, (DI)

// func sgrFinish1AVX2(tmp *int16, src *uint8, a0, a1, a2 *int32, b0, b1, b2 *int16, n int)
TEXT ·sgrFinish1AVX2(SB), NOSPLIT, $0-72
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a0+16(FP), R8
	MOVQ         a1+24(FP), R9
	MOVQ         a2+32(FP), R10
	MOVQ         b0+40(FP), R11
	MOVQ         b1+48(FP), R12
	MOVQ         b2+56(FP), R13
	MOVQ         n+64(FP), CX
	MOVL         $256, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop1:
	VMOVDQU 4(R9), Y0
	VPADDD  (R9), Y0, Y0
	VPADDD  8(R9), Y0, Y0
	VPADDD  4(R8), Y0, Y0
	VPADDD  4(R10), Y0, Y0
	VMOVDQU (R8), Y1
	VPADDD  (R10), Y1, Y1
	VPADDD  8(R8), Y1, Y1
	VPADDD  8(R10), Y1, Y1
	VPSLLD  $2, Y0, Y0
	VPSLLD  $1, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R12, Y3)
	LOADB(0, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(4, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(2, R11, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(2, R13, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(0, R11, Y5)
	LOADB(0, R13, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R13, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y3
	VPSLLD $1, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $32, R10
	ADDQ $16, R11
	ADDQ $16, R12
	ADDQ $16, R13
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop1

	VZEROUPPER
	RET

// func sgrFinish2RowAVX2(tmp *int16, src *uint8, a0, a1 *int32, b0, b1 *int16, n int)
TEXT ·sgrFinish2RowAVX2(SB), NOSPLIT, $0-56
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a0+16(FP), R8
	MOVQ         a1+24(FP), R9
	MOVQ         b0+32(FP), R11
	MOVQ         b1+40(FP), R12
	MOVQ         n+48(FP), CX
	MOVL         $256, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop2:
	VMOVDQU 4(R8), Y0
	VPADDD  4(R9), Y0, Y0
	VMOVDQU (R8), Y1
	VPADDD  (R9), Y1, Y1
	VPADDD  8(R8), Y1, Y1
	VPADDD  8(R9), Y1, Y1
	VPSLLD  $2, Y0, Y2
	VPSLLD  $1, Y0, Y0
	VPADDD  Y2, Y0, Y0
	VPSLLD  $2, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R11, Y3)
	LOADB(2, R12, Y4)
	VPADDD Y4, Y3, Y3
	LOADB(0, R11, Y5)
	LOADB(0, R12, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R11, Y4)
	VPADDD Y4, Y5, Y5
	LOADB(4, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($9)

	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $16, R11
	ADDQ $16, R12
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop2

	VZEROUPPER
	RET

// func sgrFinish2MidAVX2(tmp *int16, src *uint8, a1 *int32, b1 *int16, n int)
TEXT ·sgrFinish2MidAVX2(SB), NOSPLIT, $0-40
	MOVQ         tmp+0(FP), DI
	MOVQ         src+8(FP), SI
	MOVQ         a1+16(FP), R9
	MOVQ         b1+24(FP), R12
	MOVQ         n+32(FP), CX
	MOVL         $128, AX
	VMOVD        AX, X6
	VPBROADCASTD X6, Y6

loop3:
	VMOVDQU 4(R9), Y0
	VMOVDQU (R9), Y1
	VPADDD  8(R9), Y1, Y1
	VPSLLD  $2, Y0, Y2
	VPSLLD  $1, Y0, Y0
	VPADDD  Y2, Y0, Y0
	VPSLLD  $2, Y1, Y2
	VPADDD  Y2, Y1, Y1
	VPADDD  Y1, Y0, Y0

	LOADB(2, R12, Y3)
	LOADB(0, R12, Y5)
	LOADB(4, R12, Y4)
	VPADDD Y4, Y5, Y5
	VPSLLD $2, Y3, Y4
	VPSLLD $1, Y3, Y3
	VPADDD Y4, Y3, Y3
	VPSLLD $2, Y5, Y4
	VPADDD Y4, Y5, Y5
	VPADDD Y5, Y3, Y3

	TAIL($8)

	ADDQ $32, R9
	ADDQ $16, R12
	ADDQ $8, SI
	ADDQ $16, DI
	SUBQ $8, CX
	JNZ  loop3

	VZEROUPPER
	RET
