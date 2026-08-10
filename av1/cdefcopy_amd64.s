//go:build amd64 && !noasm

#include "textflag.h"

// The rows are at most twelve samples, so the wide case covers them with two
// overlapping stores rather than a tail loop.

// func cdefCopy8AVX2(tmp *int16, tmpStride int, src *uint8, srcStride, n, h int)
TEXT ·cdefCopy8AVX2(SB), NOSPLIT, $0-48
	MOVQ tmp+0(FP), DI
	MOVQ tmpStride+8(FP), R8
	SHLQ $1, R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	MOVQ n+32(FP), CX
	MOVQ h+40(FP), DX

	CMPQ CX, $2
	JEQ  two8
	CMPQ CX, $8
	JLT  mid8
	JEQ  exact8
	LEAQ -8(SI)(CX*1), R10
	LEAQ -16(DI)(CX*2), R11

wide8:
	VPMOVZXBW (SI), X0
	VMOVDQU   X0, (DI)
	VPMOVZXBW (R10), X1
	VMOVDQU   X1, (R11)
	ADDQ      R9, SI
	ADDQ      R9, R10
	ADDQ      R8, DI
	ADDQ      R8, R11
	DECQ      DX
	JNZ       wide8
	RET

exact8:
	VPMOVZXBW (SI), X0
	VMOVDQU   X0, (DI)
	ADDQ      R9, SI
	ADDQ      R8, DI
	DECQ      DX
	JNZ       exact8
	RET

// A two wide column reads exactly its two bytes, so it never runs past a
// left or right edge array.
two8:
	MOVWLZX   (SI), AX
	VMOVD     AX, X0
	VPMOVZXBW X0, X0
	VMOVD     X0, (DI)
	ADDQ      R9, SI
	ADDQ      R8, DI
	DECQ      DX
	JNZ       two8
	RET

mid8:
	CMPQ CX, $4
	JLT  slow8
	JEQ  exact48
	LEAQ -4(SI)(CX*1), R10
	LEAQ -8(DI)(CX*2), R11

loopmid8:
	VPMOVZXBW (SI), X0
	VMOVQ     X0, (DI)
	VPMOVZXBW (R10), X1
	VMOVQ     X1, (R11)
	ADDQ      R9, SI
	ADDQ      R9, R10
	ADDQ      R8, DI
	ADDQ      R8, R11
	DECQ      DX
	JNZ       loopmid8
	RET

exact48:
	VPMOVZXBW (SI), X0
	VMOVQ     X0, (DI)
	ADDQ      R9, SI
	ADDQ      R8, DI
	DECQ      DX
	JNZ       exact48
	RET

slow8:
	TESTQ CX, CX
	JZ    done8

rows8:
	XORQ R12, R12

cols8:
	MOVBLZX (SI)(R12*1), AX
	MOVW    AX, (DI)(R12*2)
	INCQ    R12
	CMPQ    R12, CX
	JLT     cols8
	ADDQ    R9, SI
	ADDQ    R8, DI
	DECQ    DX
	JNZ     rows8

done8:
	RET

// func cdefCopy16AVX2(tmp *int16, tmpStride int, src *uint16, srcStride, n, h int)
TEXT ·cdefCopy16AVX2(SB), NOSPLIT, $0-48
	MOVQ tmp+0(FP), DI
	MOVQ tmpStride+8(FP), R8
	SHLQ $1, R8
	MOVQ src+16(FP), SI
	MOVQ srcStride+24(FP), R9
	SHLQ $1, R9
	MOVQ n+32(FP), CX
	MOVQ h+40(FP), DX

	CMPQ CX, $2
	JEQ  two16
	CMPQ CX, $8
	JLT  mid16
	JEQ  exact816
	LEAQ -16(SI)(CX*2), R10
	LEAQ -16(DI)(CX*2), R11

wide16:
	VMOVDQU (SI), X0
	VMOVDQU X0, (DI)
	VMOVDQU (R10), X1
	VMOVDQU X1, (R11)
	ADDQ    R9, SI
	ADDQ    R9, R10
	ADDQ    R8, DI
	ADDQ    R8, R11
	DECQ    DX
	JNZ     wide16
	RET

exact816:
	VMOVDQU (SI), X0
	VMOVDQU X0, (DI)
	ADDQ    R9, SI
	ADDQ    R8, DI
	DECQ    DX
	JNZ     exact816
	RET

two16:
	MOVL (SI), AX
	MOVL AX, (DI)
	ADDQ R9, SI
	ADDQ R8, DI
	DECQ DX
	JNZ  two16
	RET

mid16:
	CMPQ CX, $4
	JLT  slow16
	JEQ  exact416
	LEAQ -8(SI)(CX*2), R10
	LEAQ -8(DI)(CX*2), R11

loopmid16:
	MOVQ (SI), AX
	MOVQ AX, (DI)
	MOVQ (R10), AX
	MOVQ AX, (R11)
	ADDQ R9, SI
	ADDQ R9, R10
	ADDQ R8, DI
	ADDQ R8, R11
	DECQ DX
	JNZ  loopmid16
	RET

exact416:
	MOVQ (SI), AX
	MOVQ AX, (DI)
	ADDQ R9, SI
	ADDQ R8, DI
	DECQ DX
	JNZ  exact416
	RET

slow16:
	TESTQ CX, CX
	JZ    done16

rows16:
	XORQ R12, R12

cols16:
	MOVWLZX (SI)(R12*2), AX
	MOVW    AX, (DI)(R12*2)
	INCQ    R12
	CMPQ    R12, CX
	JLT     cols16
	ADDQ    R9, SI
	ADDQ    R8, DI
	DECQ    DX
	JNZ     rows16

done16:
	RET
