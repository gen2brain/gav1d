//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

// CFLBODY scales the AC by alpha and folds it into the DC. There is no vector
// absolute, so the arithmetic shift gives a mask that both takes the sign off
// the magnitude and puts it back on the rounded result.
#define CFLBODY            \
	VSEXTVF2 V0, V1;   \
	VMULVX   X16, V1, V1; \
	VSRAVI   $31, V1, V2; \
	VXORVV   V2, V1, V3;  \
	VSUBVV   V2, V3, V3;  \
	VADDVX   X18, V3, V3; \
	VSRAVI   $6, V3, V3;  \
	VXORVV   V2, V3, V3;  \
	VSUBVV   V2, V3, V3;  \
	VADDVX   X14, V3, V3; \
	VMAXVX   X0, V3, V3;  \
	VMINVX   X17, V3, V3

#define CFLSETUP              \
	MOV  dst+0(FP), X10;  \
	MOV  stride+8(FP), X11; \
	MOV  w+16(FP), X12;   \
	MOV  h+24(FP), X13;   \
	MOVW dc+32(FP), X14;  \
	MOV  ac+40(FP), X15;  \
	MOVW alpha+48(FP), X16; \
	MOVW max+52(FP), X17; \
	MOV  $32, X18

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

// func cflPredict8RVV(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict8RVV(SB), NOSPLIT, $0-56
	CFLSETUP

cfl8row:
	MOV X15, X19
	MOV X10, X20
	MOV X12, X21

cfl8col:
	VSETVLI X21, E16, MF2, TA, MA, X22
	VLE16V  (X19), V0
	VSETVLI X21, E32, M1, TA, MA, X22
	CFLBODY
	VSETVLI X21, E16, MF2, TA, MA, X22
	VNSRLWI $0, V3, V4
	VSETVLI X21, E8, MF4, TA, MA, X22
	VNSRLWI $0, V4, V5
	VSE8V   V5, (X20)

	SLL $1, X22, X23
	ADD X23, X19, X19
	ADD X22, X20, X20
	SUB X22, X21, X21
	BNE X0, X21, cfl8col

	ADD X11, X10, X10
	SLL $1, X12, X23
	ADD X23, X15, X15
	ADD $-1, X13, X13
	BNE X0, X13, cfl8row

	RET

// func cflPredict16RVV(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict16RVV(SB), NOSPLIT, $0-56
	CFLSETUP
	SLL $1, X11, X11

cfl16row:
	MOV X15, X19
	MOV X10, X20
	MOV X12, X21

cfl16col:
	VSETVLI X21, E16, MF2, TA, MA, X22
	VLE16V  (X19), V0
	VSETVLI X21, E32, M1, TA, MA, X22
	CFLBODY
	VSETVLI X21, E16, MF2, TA, MA, X22
	VNSRLWI $0, V3, V4
	VSE16V  V4, (X20)

	SLL $1, X22, X23
	ADD X23, X19, X19
	ADD X23, X20, X20
	SUB X22, X21, X21
	BNE X0, X21, cfl16col

	ADD X11, X10, X10
	SLL $1, X12, X23
	ADD X23, X15, X15
	ADD $-1, X13, X13
	BNE X0, X13, cfl16row

	RET

// func z2Top8RVV(dst *uint8, edge *uint8, n int, frac int32)
TEXT ·z2Top8RVV(SB), NOSPLIT, $0-28
	MOV  dst+0(FP), X10
	MOV  edge+8(FP), X11
	MOV  n+16(FP), X12
	MOVW frac+24(FP), X13

	MOV  $64, X14
	SUBW X13, X14, X14
	MOV  $32, X15

z2t8:
	VSETVLI  X12, E8, MF2, TA, MA, X16
	VLE8V    (X11), V0
	ADD      $1, X11, X17
	VLE8V    (X17), V1
	VSETVLI  X12, E16, M1, TA, MA, X16
	VZEXTVF2 V0, V2
	VZEXTVF2 V1, V3
	VMULVX   X14, V2, V2
	VMACCVX  V3, X13, V2
	VADDVX   X15, V2, V2
	VSRLVI   $6, V2, V2
	VSETVLI  X12, E8, MF2, TA, MA, X16
	VNSRLWI  $0, V2, V4
	VSE8V    V4, (X10)

	ADD X16, X11, X11
	ADD X16, X10, X10
	SUB X16, X12, X12
	BNE X0, X12, z2t8

	RET

// func z2Top16RVV(dst *uint16, edge *uint16, n int, frac int32)
TEXT ·z2Top16RVV(SB), NOSPLIT, $0-28
	MOV  dst+0(FP), X10
	MOV  edge+8(FP), X11
	MOV  n+16(FP), X12
	MOVW frac+24(FP), X13

	MOV  $64, X14
	SUBW X13, X14, X14
	MOV  $32, X15

z2t16:
	VSETVLI  X12, E16, MF2, TA, MA, X16
	VLE16V   (X11), V0
	ADD      $2, X11, X17
	VLE16V   (X17), V1
	VSETVLI  X12, E32, M1, TA, MA, X16
	VZEXTVF2 V0, V2
	VZEXTVF2 V1, V3
	VMULVX   X14, V2, V2
	VMACCVX  V3, X13, V2
	VADDVX   X15, V2, V2
	VSRLVI   $6, V2, V2
	VSETVLI  X12, E16, MF2, TA, MA, X16
	VNSRLWI  $0, V2, V4
	VSE16V   V4, (X10)

	SLL $1, X16, X18
	ADD X18, X11, X11
	ADD X18, X10, X10
	SUB X16, X12, X12
	BNE X0, X12, z2t16

	RET

// There is no pairwise add, so the two halves of a chroma pair come from two
// strided loads and are added after widening.
// func cflAcMain8RVV(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)
TEXT ·cflAcMain8RVV(SB), NOSPLIT, $0-64
	MOV ac+0(FP), X10
	MOV acStride+8(FP), X11
	MOV ypx+16(FP), X12
	MOV stride+24(FP), X13
	MOV n+32(FP), X14
	MOV rows+40(FP), X15
	MOV ssHor+48(FP), X16
	MOV ssVer+56(FP), X17

	SLL $1, X11, X11
	MOV $2, X18
	BEQ X0, X16, ac444row
	BEQ X0, X17, ac422row

ac420row:
	MOV X12, X19
	MOV X10, X20
	MOV X14, X21

ac420col:
	VSETVLI  X21, E8, MF2, TA, MA, X22
	VLSE8V   (X19), X18, V0
	ADD      $1, X19, X23
	VLSE8V   (X23), X18, V1
	ADD      X13, X19, X24
	VLSE8V   (X24), X18, V2
	ADD      $1, X24, X24
	VLSE8V   (X24), X18, V3
	VSETVLI  X21, E16, M1, TA, MA, X22
	VZEXTVF2 V0, V4
	VZEXTVF2 V1, V5
	VZEXTVF2 V2, V6
	VZEXTVF2 V3, V7
	VADDVV   V5, V4, V4
	VADDVV   V7, V6, V6
	VADDVV   V6, V4, V4
	VSLLVI   $1, V4, V4
	VSE16V   V4, (X20)

	SLL $1, X22, X25
	ADD X25, X19, X19
	ADD X25, X20, X20
	SUB X22, X21, X21
	BNE X0, X21, ac420col

	ADD X11, X10, X10
	SLL $1, X13, X25
	ADD X25, X12, X12
	ADD $-1, X15, X15
	BNE X0, X15, ac420row

	RET

ac422row:
	MOV X12, X19
	MOV X10, X20
	MOV X14, X21

ac422col:
	VSETVLI  X21, E8, MF2, TA, MA, X22
	VLSE8V   (X19), X18, V0
	ADD      $1, X19, X23
	VLSE8V   (X23), X18, V1
	VSETVLI  X21, E16, M1, TA, MA, X22
	VZEXTVF2 V0, V4
	VZEXTVF2 V1, V5
	VADDVV   V5, V4, V4
	VSLLVI   $2, V4, V4
	VSE16V   V4, (X20)

	SLL $1, X22, X25
	ADD X25, X19, X19
	ADD X25, X20, X20
	SUB X22, X21, X21
	BNE X0, X21, ac422col

	ADD X11, X10, X10
	ADD X13, X12, X12
	ADD $-1, X15, X15
	BNE X0, X15, ac422row

	RET

ac444row:
	MOV X12, X19
	MOV X10, X20
	MOV X14, X21

ac444col:
	VSETVLI  X21, E8, MF2, TA, MA, X22
	VLE8V    (X19), V0
	VSETVLI  X21, E16, M1, TA, MA, X22
	VZEXTVF2 V0, V4
	VSLLVI   $3, V4, V4
	VSE16V   V4, (X20)

	ADD X22, X19, X19
	SLL $1, X22, X25
	ADD X25, X20, X20
	SUB X22, X21, X21
	BNE X0, X21, ac444col

	ADD X11, X10, X10
	ADD X13, X12, X12
	ADD $-1, X15, X15
	BNE X0, X15, ac444row

	RET
