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
