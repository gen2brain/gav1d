//go:build amd64 && !noasm

#include "textflag.h"

// func fgApplyRowAVX2(dst, src, scaling *uint8, grain *int16, n, mul, minValue, maxValue int)
//
// Sixteen pixels an iteration. The scaling lookup is a dword gather, so the
// table is padded past its last entry; the multiply-high replaces the rounding
// shift, which would overflow a word before it could be shifted back down.
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
	VPSRLD   $24, Y8, Y8 // 0x000000ff per dword

loop:
	VPMOVZXBW (SI), Y0
	VPMOVZXBD (SI), Y1
	VPMOVZXBD 8(SI), Y2

	VPCMPEQD   Y3, Y3, Y3
	VPGATHERDD Y3, (BX)(Y1*1), Y4
	VPCMPEQD   Y3, Y3, Y3
	VPGATHERDD Y3, (BX)(Y2*1), Y5
	VPAND      Y8, Y4, Y4
	VPAND      Y8, Y5, Y5

	VPACKUSDW Y5, Y4, Y6
	VPERMQ    $0xD8, Y6, Y6

	VMOVDQU   (DX), Y7
	VPMULLW   Y7, Y6, Y6
	VPMULHRSW Y9, Y6, Y6
	VPADDW    Y0, Y6, Y6

	VPACKUSWB Y6, Y6, Y6
	VPERMQ    $0xD8, Y6, Y6
	VPMAXUB   X10, X6, X6
	VPMINUB   X11, X6, X6
	VMOVDQU   X6, (DI)

	ADDQ $16, SI
	ADDQ $16, DI
	ADDQ $32, DX
	SUBQ $16, CX
	CMPQ CX, $16
	JGE  loop

	TESTQ CX, CX
	JZ    done

	// Finish with a block that overlaps the one just done. Every pixel is an
	// independent function of src and grain, and dst is never the source.
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
