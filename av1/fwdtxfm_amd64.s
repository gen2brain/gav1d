//go:build amd64 && !noasm

#include "textflag.h"

// func fwdDot4AVX2(dst *float64, b *float64, src *float64, n int)
TEXT ·fwdDot4AVX2(SB), NOSPLIT, $0-32
	MOVQ dst+0(FP), DI
	MOVQ b+8(FP), SI
	MOVQ src+16(FP), DX
	MOVQ n+24(FP), CX

	MOVQ CX, R8
	SHLQ $3, R8

	MOVQ CX, R9

rows:
	VXORPD Y0, Y0, Y0
	VXORPD Y1, Y1, Y1
	MOVQ   SI, R10
	MOVQ   DX, R11
	MOVQ   CX, R12

cols:
	CMPQ R12, $8
	JLT  tail

	VMOVUPD   (R10), Y2
	VMOVUPD   32(R10), Y3
	VMOVUPD   (R11), Y4
	VMOVUPD   32(R11), Y5
	VFMADD231PD Y4, Y2, Y0
	VFMADD231PD Y5, Y3, Y1

	ADDQ $64, R10
	ADDQ $64, R11
	SUBQ $8, R12
	JMP  cols

tail:
	TESTQ R12, R12
	JZ    reduce

	VMOVUPD     (R10), Y2
	VMOVUPD     (R11), Y4
	VFMADD231PD Y4, Y2, Y0

	ADDQ $32, R10
	ADDQ $32, R11
	SUBQ $4, R12
	JMP  tail

reduce:
	VADDPD      Y1, Y0, Y0
	VEXTRACTF128 $1, Y0, X1
	VADDPD      X1, X0, X0
	VHADDPD     X0, X0, X0
	VMOVSD      X0, (DI)

	ADDQ R8, SI
	ADDQ $8, DI
	DECQ R9
	JNZ  rows

	VZEROUPPER
	RET
