#include "textflag.h"

// residualAVX2 subtracts the prediction from the source, stores the difference
// and returns its sum of squares. The narrow shapes pack whole rows into one
// register: four rows at 4 px, two at 8 px.
TEXT ·residualAVX2(SB), NOSPLIT, $0-64
	MOVQ res+0(FP), DI
	MOVQ dst+8(FP), SI
	MOVQ stride+16(FP), R8
	MOVQ src+24(FP), DX
	MOVQ sstride+32(FP), R9
	MOVQ w+40(FP), CX
	MOVQ h+48(FP), BX
	VPXOR Y7, Y7, Y7

	CMPQ CX, $4
	JE   w4loop
	CMPQ CX, $8
	JE   w8loop
	JMP  rowloop

w4loop:
	LEAQ         (DX)(R9*2), R10
	VMOVD        (DX), X0
	VPINSRD      $1, (DX)(R9*1), X0, X0
	VPINSRD      $2, (R10), X0, X0
	VPINSRD      $3, (R10)(R9*1), X0, X0
	LEAQ         (SI)(R8*2), R11
	VMOVD        (SI), X1
	VPINSRD      $1, (SI)(R8*1), X1, X1
	VPINSRD      $2, (R11), X1, X1
	VPINSRD      $3, (R11)(R8*1), X1, X1
	VPMOVZXBW    X0, Y0
	VPMOVZXBW    X1, Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	LEAQ         (DX)(R9*4), DX
	LEAQ         (SI)(R8*4), SI
	SUBQ         $4, BX
	JNZ          w4loop
	JMP          done

w8loop:
	VMOVQ        (DX), X0
	VPINSRQ      $1, (DX)(R9*1), X0, X0
	VMOVQ        (SI), X1
	VPINSRQ      $1, (SI)(R8*1), X1, X1
	VPMOVZXBW    X0, Y0
	VPMOVZXBW    X1, Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	LEAQ         (DX)(R9*2), DX
	LEAQ         (SI)(R8*2), SI
	SUBQ         $2, BX
	JNZ          w8loop
	JMP          done

rowloop:
	MOVQ CX, R11
	MOVQ DX, R12
	MOVQ SI, R13

colloop:
	VPMOVZXBW    (R12), Y0
	VPMOVZXBW    (R13), Y1
	VPSUBW       Y1, Y0, Y2
	VPMOVSXWD    X2, Y3
	VMOVDQU      Y3, (DI)
	VEXTRACTI128 $1, Y2, X4
	VPMOVSXWD    X4, Y5
	VMOVDQU      Y5, 32(DI)
	VPMADDWD     Y2, Y2, Y6
	VPADDD       Y6, Y7, Y7
	ADDQ         $64, DI
	ADDQ         $16, R12
	ADDQ         $16, R13
	SUBQ         $16, R11
	JNZ          colloop

	ADDQ R9, DX
	ADDQ R8, SI
	DECQ BX
	JNZ  rowloop

done:
	VEXTRACTI128 $1, Y7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x0e, X7, X0
	VPADDD       X0, X7, X7
	VPSHUFD      $0x01, X7, X0
	VPADDD       X0, X7, X7
	VMOVD        X7, AX
	MOVLQZX      AX, AX
	MOVQ         AX, ret+56(FP)
	VZEROUPPER
	RET

#define HAD8(i0, i1, i2, i3, i4, i5, i6, i7, o0, o1, o2, o3, o4, o5, o6, o7) \
	VPADDD i4, i0, o0 \
	VPADDD i5, i1, o1 \
	VPADDD i6, i2, o2 \
	VPADDD i7, i3, o3 \
	VPSUBD i4, i0, o4 \
	VPSUBD i5, i1, o5 \
	VPSUBD i6, i2, o6 \
	VPSUBD i7, i3, o7 \
	VPADDD o2, o0, i0 \
	VPADDD o3, o1, i1 \
	VPSUBD o2, o0, i2 \
	VPSUBD o3, o1, i3 \
	VPADDD o6, o4, i4 \
	VPADDD o7, o5, i5 \
	VPSUBD o6, o4, i6 \
	VPSUBD o7, o5, i7 \
	VPADDD i1, i0, o0 \
	VPSUBD i1, i0, o1 \
	VPADDD i3, i2, o2 \
	VPSUBD i3, i2, o3 \
	VPADDD i5, i4, o4 \
	VPSUBD i5, i4, o5 \
	VPADDD i7, i6, o6 \
	VPSUBD i7, i6, o7

// satd8AVX2 is the sum of absolute 8x8 Hadamard coefficients. The transform is
// separable and the sum does not care about transposition, so both passes run
// down the columns with a transpose between them.
TEXT ·satd8AVX2(SB), NOSPLIT, $0-24
	MOVQ res+0(FP), SI
	MOVQ stride+8(FP), DX
	SHLQ $2, DX

	VMOVDQU (SI), Y0
	VMOVDQU (SI)(DX*1), Y1
	LEAQ    (SI)(DX*2), AX
	VMOVDQU (AX), Y2
	VMOVDQU (AX)(DX*1), Y3
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y4
	VMOVDQU (AX)(DX*1), Y5
	LEAQ    (AX)(DX*2), AX
	VMOVDQU (AX), Y6
	VMOVDQU (AX)(DX*1), Y7

	HAD8(Y0, Y1, Y2, Y3, Y4, Y5, Y6, Y7, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15)

	VPUNPCKLDQ  Y9, Y8, Y0
	VPUNPCKHDQ  Y9, Y8, Y1
	VPUNPCKLDQ  Y11, Y10, Y2
	VPUNPCKHDQ  Y11, Y10, Y3
	VPUNPCKLDQ  Y13, Y12, Y4
	VPUNPCKHDQ  Y13, Y12, Y5
	VPUNPCKLDQ  Y15, Y14, Y6
	VPUNPCKHDQ  Y15, Y14, Y7
	VPUNPCKLQDQ Y2, Y0, Y8
	VPUNPCKHQDQ Y2, Y0, Y9
	VPUNPCKLQDQ Y3, Y1, Y10
	VPUNPCKHQDQ Y3, Y1, Y11
	VPUNPCKLQDQ Y6, Y4, Y12
	VPUNPCKHQDQ Y6, Y4, Y13
	VPUNPCKLQDQ Y7, Y5, Y14
	VPUNPCKHQDQ Y7, Y5, Y15
	VPERM2I128  $0x20, Y12, Y8, Y0
	VPERM2I128  $0x20, Y13, Y9, Y1
	VPERM2I128  $0x20, Y14, Y10, Y2
	VPERM2I128  $0x20, Y15, Y11, Y3
	VPERM2I128  $0x31, Y12, Y8, Y4
	VPERM2I128  $0x31, Y13, Y9, Y5
	VPERM2I128  $0x31, Y14, Y10, Y6
	VPERM2I128  $0x31, Y15, Y11, Y7

	HAD8(Y0, Y1, Y2, Y3, Y4, Y5, Y6, Y7, Y8, Y9, Y10, Y11, Y12, Y13, Y14, Y15)

	VPABSD Y8, Y8
	VPABSD Y9, Y9
	VPABSD Y10, Y10
	VPABSD Y11, Y11
	VPABSD Y12, Y12
	VPABSD Y13, Y13
	VPABSD Y14, Y14
	VPABSD Y15, Y15
	VPADDD Y9, Y8, Y8
	VPADDD Y11, Y10, Y10
	VPADDD Y13, Y12, Y12
	VPADDD Y15, Y14, Y14
	VPADDD Y10, Y8, Y8
	VPADDD Y14, Y12, Y12
	VPADDD Y12, Y8, Y8

	VEXTRACTI128 $1, Y8, X0
	VPADDD       X0, X8, X8
	VPSHUFD      $0x0e, X8, X0
	VPADDD       X0, X8, X8
	VPSHUFD      $0x01, X8, X0
	VPADDD       X0, X8, X8
	VMOVD        X8, AX
	ADDL         $4, AX
	SARL         $3, AX
	MOVL         AX, ret+16(FP)
	VZEROUPPER
	RET
