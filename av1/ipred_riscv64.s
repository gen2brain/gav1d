//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

#define SMV_ROW               \
	MOVBU     (X18), X22;      \
	ADD       $1, X18, X18;     \
	VMVVX     X22, V2;        \
	VXORVI    $-1, V2, V3;    \
	VWMULUVV  V1, V2, V8;     \
	VWMACCUVV V4, V3, V8;     \
	VWADDUWV  V4, V8, V8;     \
	VWADDUWV  V6, V8, V8;     \
	VNSRLWI   $8, V8, V7;     \
	VSE8V     V7, (X20);      \
	ADD       X6, X20, X20

// func smoothV8RVV(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)
TEXT ·smoothV8RVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV weights+24(FP), X10
	MOV w+32(FP), X12
	MOV h+40(FP), X13

	SUB   X13, X7, X25
	MOVBU (X25), X21
	MOV   $128, X24

	MOV X12, X14
	MOV $0, X15

smvcol:
	VSETVLI X14, E8, M1, TA, MA, X16

	VMVVX X21, V4
	VMVVX X24, V6

	ADD   X15, X7, X25
	ADD   $1, X25, X25
	VLE8V (X25), V1

	MOV X13, X17
	MOV X10, X18
	ADD X15, X5, X20

smvrow:
	SMV_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, smvrow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, smvcol
	RET

#define SMH_ROW               \
	MOVBU     (X19), X22;      \
	SUB       $1, X19, X19;     \
	VMVVX     X22, V1;        \
	VWMULUVV  V1, V2, V8;     \
	VWMACCUVV V5, V3, V8;     \
	VWADDUWV  V5, V8, V8;     \
	VWADDUWV  V6, V8, V8;     \
	VNSRLWI   $8, V8, V7;     \
	VSE8V     V7, (X20);      \
	ADD       X6, X20, X20

// func smoothH8RVV(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)
TEXT ·smoothH8RVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV weights+24(FP), X10
	MOV w+32(FP), X12
	MOV h+40(FP), X13

	ADD   X12, X7, X25
	MOVBU (X25), X21
	MOV   $128, X24

	MOV X12, X14
	MOV $0, X15

smhcol:
	VSETVLI X14, E8, M1, TA, MA, X16

	VMVVX X21, V5
	VMVVX X24, V6

	ADD    X15, X10, X25
	VLE8V  (X25), V2
	VXORVI $-1, V2, V3

	MOV X13, X17
	SUB $1, X7, X19
	ADD X15, X5, X20

smhrow:
	SMH_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, smhrow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, smhcol
	RET

#define SM2_ROW                \
	MOVBU     (X18), X22;       \
	ADD       $1, X18, X18;      \
	VMVVX     X22, V12;        \
	VXORVI    $-1, V12, V13;   \
	VWMULUVV  V1, V12, V8;     \
	VWMACCUVV V4, V13, V8;     \
	VWADDUWV  V4, V8, V8;      \
	MOVBU     (X19), X22;       \
	SUB       $1, X19, X19;      \
	VMVVX     X22, V14;        \
	VWMULUVV  V14, V2, V10;    \
	VWMACCUVV V5, V3, V10;     \
	VWADDUWV  V5, V10, V10;    \
	VSETVLI   X16, E16, M2, TA, MA, X26; \
	VSRLVI    $1, V8, V16;     \
	VSRLVI    $1, V10, V18;    \
	VANDVV    V8, V10, V20;    \
	VANDVI    $1, V20, V20;    \
	VADDVV    V16, V18, V16;   \
	VADDVV    V20, V16, V16;   \
	VADDVX    X24, V16, V16;   \
	VSETVLI   X16, E8, M1, TA, MA, X26; \
	VNSRLWI   $8, V16, V7;     \
	VSE8V     V7, (X20);       \
	ADD       X6, X20, X20

// func smooth8RVV(dst *uint8, stride int, tl *uint8, vw, hw *uint8, w, h int)
TEXT ·smooth8RVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV vw+24(FP), X10
	MOV hw+32(FP), X11
	MOV w+40(FP), X12
	MOV h+48(FP), X13

	SUB   X13, X7, X25
	MOVBU (X25), X21
	ADD   X12, X7, X25
	MOVBU (X25), X23
	MOV   $128, X24

	MOV X12, X14
	MOV $0, X15

sm2col:
	VSETVLI X14, E8, M1, TA, MA, X16

	VMVVX X21, V4
	VMVVX X23, V5

	ADD   X15, X7, X25
	ADD   $1, X25, X25
	VLE8V (X25), V1

	ADD    X15, X11, X25
	VLE8V  (X25), V2
	VXORVI $-1, V2, V3

	MOV X13, X17
	MOV X10, X18
	SUB $1, X7, X19
	ADD X15, X5, X20

sm2row:
	SM2_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, sm2row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, sm2col
	RET

#define PAETH_ROW             \
	MOVBU     (X19), X22;      \
	SUB       $1, X19, X19;     \
	VMVVX     X22, V3;        \
	VXORVV    V6, V3, V0;     \
	VORVV     V6, V3, V1;     \
	VSRLVI    $1, V0, V2;     \
	VSUBVV    V2, V1, V1;     \
	VANDVI    $1, V0, V0;     \
	VSSUBUVV  V1, V5, V2;     \
	VSUBVV    V0, V1, V1;     \
	VSSUBUVV  V5, V1, V1;     \
	VORVV     V2, V1, V1;     \
	VSADDUVV  V1, V1, V1;     \
	VORVV     V0, V1, V1;     \
	VSSUBUVV  V3, V5, V2;     \
	VSSUBUVV  V5, V3, V0;     \
	VORVV     V0, V2, V2;     \
	VMINUVV   V7, V2, V2;     \
	VMSEQVV   V7, V2, V0;     \
	VMERGEVVM V3, V6, V0, V11; \
	VMINUVV   V2, V1, V1;     \
	VMSEQVV   V2, V1, V0;     \
	VMERGEVVM V11, V5, V0, V12; \
	VSE8V     V12, (X20);     \
	ADD       X6, X20, X20

// func paeth8RVV(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·paeth8RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	MOVBU (X7), X24

	MOV X12, X14
	MOV $0, X15

pthcol:
	VSETVLI X14, E8, M1, TA, MA, X16

	VMVVX X24, V5

	ADD   X15, X7, X25
	ADD   $1, X25, X25
	VLE8V (X25), V6

	VSSUBUVV V6, V5, V7
	VSSUBUVV V5, V6, V4
	VORVV    V4, V7, V7

	MOV X13, X17
	SUB $1, X7, X19
	ADD X15, X5, X20

pthrow:
	PAETH_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, pthrow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, pthcol
	RET

// func splatDc8RVV(dst *uint8, stride, w, h int, dc uint8)
TEXT ·splatDc8RVV(SB), NOSPLIT, $0-33
	MOV   dst+0(FP), X5
	MOV   stride+8(FP), X6
	MOV   w+16(FP), X12
	MOV   h+24(FP), X13
	MOVBU dc+32(FP), X21

	MOV X12, X14
	MOV $0, X15

spcol:
	VSETVLI X14, E8, M1, TA, MA, X16
	VMVVX   X21, V1

	MOV X13, X17
	ADD X15, X5, X20

sprow:
	VSE8V V1, (X20)
	ADD   X6, X20, X20
	SUB   $1, X17, X17
	BNE   X0, X17, sprow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, spcol
	RET

// func vpred8RVV(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·vpred8RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	MOV X12, X14
	MOV $0, X15

vpcol:
	VSETVLI X14, E8, M1, TA, MA, X16

	ADD   X15, X7, X25
	ADD   $1, X25, X25
	VLE8V (X25), V1

	MOV X13, X17
	ADD X15, X5, X20

vprow:
	VSE8V V1, (X20)
	ADD   X6, X20, X20
	SUB   $1, X17, X17
	BNE   X0, X17, vprow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, vpcol
	RET

// func hpred8RVV(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·hpred8RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	MOV X12, X14
	MOV $0, X15

hpcol:
	VSETVLI X14, E8, M1, TA, MA, X16

	MOV X13, X17
	SUB $1, X7, X19
	ADD X15, X5, X20

hprow:
	MOVBU (X19), X22
	SUB   $1, X19, X19
	VMVVX X22, V1
	VSE8V V1, (X20)
	ADD   X6, X20, X20
	SUB   $1, X17, X17
	BNE   X0, X17, hprow

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, hpcol
	RET

#define FI_BLOCK              \
	VMVVX     X23, V8;        \
	MOVBU     (X25), X22;     \
	VMACCVX   V1, X22, V8;    \
	MOVBU     0(X18), X22;    \
	VMACCVX   V2, X22, V8;    \
	MOVBU     1(X18), X22;    \
	VMACCVX   V3, X22, V8;    \
	MOVBU     2(X18), X22;    \
	VMACCVX   V4, X22, V8;    \
	MOVBU     3(X18), X22;    \
	VMACCVX   V5, X22, V8;    \
	MOVBU     (X26), X22;     \
	VMACCVX   V6, X22, V8;    \
	MOVBU     (X28), X22;     \
	VMACCVX   V7, X22, V8;    \
	VSRAVI    $4, V8, V8;     \
	VMAXVX    X0, V8, V8;     \
	VMINVX    X24, V8, V8;    \
	VSLIDEDOWNVI $4, V8, V9;  \
	VSETIVLI  $4, E8, MF2, TA, MA, X30; \
	VNSRLWI   $0, V8, V10;    \
	VNSRLWI   $0, V9, V11;    \
	VSE8V     V10, (X20);     \
	VSE8V     V11, (X21);     \
	VSETIVLI  $8, E16, M1, TA, MA, X30

// func filterIntra8RVV(dst *uint8, stride int, tl *uint8, f *int16, w, h int)
TEXT ·filterIntra8RVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV f+24(FP), X10
	MOV w+32(FP), X12
	MOV h+40(FP), X13

	VSETIVLI $8, E16, M1, TA, MA, X30

	VLE16V (X10), V1
	ADD    $16, X10, X10
	VLE16V (X10), V2
	ADD    $16, X10, X10
	VLE16V (X10), V3
	ADD    $16, X10, X10
	VLE16V (X10), V4
	ADD    $16, X10, X10
	VLE16V (X10), V5
	ADD    $16, X10, X10
	VLE16V (X10), V6
	ADD    $16, X10, X10
	VLE16V (X10), V7

	MOV $8, X23
	MOV $255, X24

	ADD $1, X7, X18
	MOV $0, X17

fiRow:
	MOV X5, X20
	ADD X6, X5, X21

	SUB X17, X7, X25
	SUB $1, X25, X26
	SUB $2, X25, X28

	MOV $0, X15

fiCol:
	FI_BLOCK

	ADD $4, X18, X18
	SUB $1, X18, X25
	ADD $3, X20, X26
	ADD $3, X21, X28
	ADD $4, X20, X20
	ADD $4, X21, X21

	ADD $4, X15, X15
	BLT X15, X12, fiCol

	SUB X12, X21, X18
	ADD X6, X5, X5
	ADD X6, X5, X5
	ADD $2, X17, X17
	BLT X17, X13, fiRow
	RET

#define Z1_ROW                \
	AND       $0x3E, X17, X21; \
	MOV       $64, X22;       \
	SUB       X21, X22, X22;  \
	SRA       $6, X17, X23;   \
	ADD       X23, X7, X23;   \
	MOV       $0, X15;        \
	MOV       X12, X14

// func z1Full8RVV(dst *uint8, stride int, top *uint8, w, rows, dx, xpos int)
TEXT ·z1Full8RVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV top+16(FP), X7
	MOV w+24(FP), X12
	MOV rows+32(FP), X13
	MOV dx+40(FP), X18
	MOV xpos+48(FP), X17

	MOV $32, X26

z1row:
	Z1_ROW
	ADD X15, X5, X20

z1col:
	VSETVLI X14, E8, M1, TA, MA, X16

	VMVVX X22, V2
	VMVVX X21, V3
	VMVVX X26, V4

	ADD    X15, X23, X25
	VLE8V  (X25), V0
	ADD    $1, X25, X25
	VLE8V  (X25), V1

	VWMULUVV  V0, V2, V8
	VWMACCUVV V1, V3, V8
	VWADDUWV  V4, V8, V8
	VNSRLWI   $6, V8, V7
	VSE8V     V7, (X20)

	ADD X16, X20, X20
	ADD X16, X15, X15
	SUB X16, X14, X14
	BNE X0, X14, z1col

	ADD X6, X5, X5
	ADD X18, X17, X17
	SUB $1, X13, X13
	BNE X0, X13, z1row
	RET
