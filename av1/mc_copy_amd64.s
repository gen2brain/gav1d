//go:build amd64 && !noasm

#include "textflag.h"

// Two rows at a time, and narrow blocks move through general registers rather
// than vectors. Heights are always even.
#define ROW2(LOAD, STORE, RA, RB)  \
	LOAD  (SI), RA;            \
	LOAD  (SI)(R9*1), RB;      \
	STORE RA, (DI);            \
	STORE RB, (DI)(R8*1)

#define STEP2          \
	LEAQ (SI)(R9*2), SI; \
	LEAQ (DI)(R8*2), DI; \
	SUBQ $2, CX

// func mcPut8Asm(dst *uint8, dstStride int, src *uint8, srcStride int, w, h int)
TEXT ·mcPut8Asm(SB), NOSPLIT, $0-48
	MOVQ dst+0(FP), DI
	MOVQ dstStride+8(FP), R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ w+32(FP), BX
	MOVQ h+40(FP), CX

	CMPQ BX, $4
	JEQ  cw4
	CMPQ BX, $8
	JEQ  cw8
	CMPQ BX, $16
	JEQ  cw16
	CMPQ BX, $2
	JEQ  cw2
	CMPQ BX, $32
	JEQ  cw32
	JMP  cwide

cw2:
	ROW2(MOVWLZX, MOVW, R10, R11)
	STEP2
	JNZ  cw2
	RET

cw4:
	ROW2(MOVL, MOVL, R10, R11)
	STEP2
	JNZ  cw4
	RET

cw8:
	ROW2(MOVQ, MOVQ, R10, R11)
	STEP2
	JNZ  cw8
	RET

cw16:
	ROW2(VMOVDQU, VMOVDQU, X0, X1)
	STEP2
	JNZ  cw16
	VZEROUPPER
	RET

cw32:
	ROW2(VMOVDQU, VMOVDQU, Y0, Y1)
	STEP2
	JNZ  cw32
	VZEROUPPER
	RET

cwide:
	XORQ R10, R10

cwinner:
	VMOVDQU (SI)(R10*1), Y0
	VMOVDQU Y0, (DI)(R10*1)
	ADDQ    $32, R10
	CMPQ    R10, BX
	JLT     cwinner

	ADDQ R9, SI
	ADDQ R8, DI
	DECQ CX
	JNZ  cwide

	VZEROUPPER
	RET
