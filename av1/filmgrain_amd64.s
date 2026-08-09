//go:build amd64 && !noasm

#include "textflag.h"

DATA fgOnes<>+0(SB)/8, $0x0101010101010101
DATA fgOnes<>+8(SB)/8, $0x0101010101010101
DATA fgOnes<>+16(SB)/8, $0x0101010101010101
DATA fgOnes<>+24(SB)/8, $0x0101010101010101
GLOBL fgOnes<>(SB), RODATA|NOPTR, $32

// NOISE applies the gathered scaling to the grain. The multiply-high replaces
// the rounding shift, which would overflow a word.
#define NOISE                      \
	VPCMPEQD   Y3, Y3, Y3;     \
	VPGATHERDD Y3, (BX)(Y1*1), Y4; \
	VPCMPEQD   Y3, Y3, Y3;     \
	VPGATHERDD Y3, (BX)(Y2*1), Y5; \
	VPAND      Y8, Y4, Y4;     \
	VPAND      Y8, Y5, Y5;     \
	VPACKUSDW  Y5, Y4, Y6;     \
	VPERMQ     $0xD8, Y6, Y6;  \
	VMOVDQU    (DX), Y7;       \
	VPMULLW    Y7, Y6, Y6;     \
	VPMULHRSW  Y9, Y6, Y6;     \
	VPADDW     Y0, Y6, Y6;     \
	VPACKUSWB  Y6, Y6, Y6;     \
	VPERMQ     $0xD8, Y6, Y6;  \
	VPMAXUB    X10, X6, X6;    \
	VPMINUB    X11, X6, X6;    \
	VMOVDQU    X6, (DI)

// func fgApplyRowAVX2(dst, src, scaling *uint8, grain *int16, n, mul, minValue, maxValue int)
TEXT ·fgApplyRowAVX2(SB), NOSPLIT, $0-64
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ scaling+16(FP), BX
	MOVQ grain+24(FP), DX
	MOVQ n+32(FP), CX

	MOVQ         mul+40(FP), R8
	MOVQ         R8, X9
	VPBROADCASTW X9, Y9
	MOVQ         minValue+48(FP), R9
	MOVQ         R9, X10
	VPBROADCASTB X10, Y10
	MOVQ         maxValue+56(FP), R10
	MOVQ         R10, X11
	VPBROADCASTB X11, Y11

	VPCMPEQD Y8, Y8, Y8
	VPSRLD   $24, Y8, Y8

loop:
	VPMOVZXBW (SI), Y0
	VPMOVZXBD (SI), Y1
	VPMOVZXBD 8(SI), Y2
	NOISE

	ADDQ $16, SI
	ADDQ $16, DI
	ADDQ $32, DX
	SUBQ $16, CX
	CMPQ CX, $16
	JGE  loop

	TESTQ CX, CX
	JZ    done

	// Overlap the block just done; every pixel is independent and dst is
	// never the source.
	MOVQ $16, AX
	SUBQ CX, AX
	SUBQ AX, SI
	SUBQ AX, DI
	SHLQ $1, AX
	SUBQ AX, DX
	MOVQ $16, CX
	JMP  loop

done:
	VZEROUPPER
	RET

// func fguvApplyRowAVX2(dst, src, luma, scaling *uint8, grain *int16, n, mul, minValue, maxValue int, p *fguvParams)
//
// Eight an iteration, not sixteen: a subsampled block minus its overlapped
// column leaves fifteen.
TEXT ·fguvApplyRowAVX2(SB), NOSPLIT, $0-80
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ luma+16(FP), R11
	MOVQ scaling+24(FP), BX
	MOVQ grain+32(FP), DX
	MOVQ n+40(FP), CX

	MOVQ         mul+48(FP), R8
	MOVQ         R8, X9
	VPBROADCASTW X9, X9
	MOVQ         minValue+56(FP), R9
	MOVQ         R9, X10
	VPBROADCASTB X10, X10
	MOVQ         maxValue+64(FP), R10
	MOVQ         R10, X11
	VPBROADCASTB X11, X11

	MOVQ p+72(FP), R12

	VPCMPEQD Y8, Y8, Y8
	VPSRLD   $24, Y8, Y8

	VPBROADCASTD 0(R12), X12  // [lumaMult, mult] per dword
	VPBROADCASTW 4(R12), X14  // pixelMax
	VPBROADCASTD 8(R12), X13  // offset
	MOVLQSX      12(R12), R13 // sx
	MOVLQSX      16(R12), R8  // chromaScalingFromLuma

	MOVQ  $8, R10 // luma advance per iteration
	TESTQ R13, R13
	JZ    loopuv
	SHLQ  $1, R10

loopuv:
	// avg = luma, pairwise averaged when chroma is subsampled
	TESTQ R13, R13
	JZ    noss
	VMOVDQU    (R11), X0
	VPMADDUBSW fgOnes<>(SB), X0, X0
	VPCMPEQW   X3, X3, X3
	VPSUBW     X3, X0, X0
	VPSRLW     $1, X0, X0
	JMP        haveavg

noss:
	VPMOVZXBW (R11), X0

haveavg:
	TESTQ R8, R8
	JNZ   fromluma

	// val = clip((avg*lumaMult + src*mult)>>6 + offset, 0, pixelMax)
	VPMOVZXBW  (SI), X15
	VPUNPCKLWD X15, X0, X4
	VPUNPCKHWD X15, X0, X5
	VPMADDWD   X12, X4, X4
	VPMADDWD   X12, X5, X5
	VPSRAD     $6, X4, X4
	VPSRAD     $6, X5, X5
	VPADDD     X13, X4, X4
	VPADDD     X13, X5, X5
	VPACKUSDW  X5, X4, X6
	VPMINUW    X14, X6, X6
	JMP        haveval

fromluma:
	VMOVDQU X0, X6

haveval:
	VPMOVZXWD  X6, Y1
	VPCMPEQD   Y3, Y3, Y3
	VPGATHERDD Y3, (BX)(Y1*1), Y4
	VPAND      Y8, Y4, Y4
	VPACKUSDW  Y4, Y4, Y4
	VPERMQ     $0x08, Y4, Y4

	VMOVDQU   (DX), X7
	VPMULLW   X7, X4, X6
	VPMULHRSW X9, X6, X6
	VPMOVZXBW (SI), X0
	VPADDW    X0, X6, X6
	VPACKUSWB X6, X6, X6
	VPMAXUB   X10, X6, X6
	VPMINUB   X11, X6, X6
	VMOVQ     X6, (DI)

	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $16, DX
	ADDQ R10, R11
	SUBQ $8, CX
	CMPQ CX, $8
	JGE  loopuv

	TESTQ CX, CX
	JZ    doneuv

	MOVQ  $8, AX
	SUBQ  CX, AX
	SUBQ  AX, SI
	SUBQ  AX, DI
	MOVQ  AX, R9
	TESTQ R13, R13
	JZ    nosub
	SHLQ  $1, R9

nosub:
	SUBQ R9, R11
	SHLQ $1, AX
	SUBQ AX, DX
	MOVQ $8, CX
	JMP  loopuv

doneuv:
	VZEROUPPER
	RET
