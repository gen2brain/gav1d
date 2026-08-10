//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// func cflAcNormRVV(ac *int16, n int, log2sz int)
TEXT ·cflAcNormRVV(SB), NOSPLIT, $0-24
	MOV ac+0(FP), X10
	MOV n+8(FP), X11
	MOV log2sz+16(FP), X12

	MOV     $8, X8
	VSETVLI X8, E32, M1, TA, MA, X7
	VMVVX   X0, V8
	MOV     X10, X13
	MOV     X11, X14

acsum:
	VSETVLI  X8, E16, MF2, TA, MA, X7
	VLE16V   (X13), V0
	VSETVLI  X8, E32, M1, TA, MA, X7
	VSEXTVF2 V0, V1
	VADDVV   V1, V8, V8
	ADD      $16, X13, X13
	ADD      $-8, X14, X14
	BNE      X0, X14, acsum

	VMVVX     X0, V6
	VREDSUMVS V6, V8, V7
	VMVXS     V7, X15

	MOV  $1, X16
	SLL  X12, X16, X16
	SRL  $1, X16, X16
	ADDW X16, X15, X15
	SRAW X12, X15, X15

	MOV     X10, X13
	MOV     X11, X14
	VSETVLI X8, E16, MF2, TA, MA, X7
	VMVVX   X15, V5

acsub:
	VLE16V (X13), V0
	VSUBVV V5, V0, V0
	VSE16V V0, (X13)
	ADD    $16, X13, X13
	ADD    $-8, X14, X14
	BNE    X0, X14, acsub

	RET
