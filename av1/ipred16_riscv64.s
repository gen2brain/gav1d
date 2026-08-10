//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

#define SMV16_ROW              \
	MOVHU     (X18), X22;      \
	MOVHU     2(X18), X23;     \
	ADD       $4, X18, X18;    \
	VWMULUVX  X22, V1, V8;     \
	VWMACCUVX V2, X23, V8;     \
	VWADDUWX  X24, V8, V8;     \
	VNSRLWI   $8, V8, V7;      \
	VSE16V    V7, (X20);       \
	ADD       X6, X20, X20

// func smoothV16RVV(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)
TEXT ·smoothV16RVV(SB), NOSPLIT, $0-48
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV weights+24(FP), X10
	MOV w+32(FP), X12
	MOV h+40(FP), X13

	SLLI $1, X6, X6

	SLLI  $1, X13, X25
	SUB   X25, X7, X25
	MOVHU (X25), X21
	MOV   $128, X24

	MOV X12, X14
	MOV $0, X15

smv16col:
	VSETVLI X14, E16, M1, TA, MA, X16

	VMVVX X21, V2

	SLLI   $1, X15, X25
	ADD    X25, X7, X25
	ADD    $2, X25, X25
	VLE16V (X25), V1

	MOV  X13, X17
	MOV  X10, X18
	SLLI $1, X15, X20
	ADD  X20, X5, X20

smv16row:
	SMV16_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, smv16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, smv16col
	RET

#define SMH16_ROW              \
	MOVHU     (X19), X22;      \
	SUB       $2, X19, X19;    \
	VWMULUVX  X22, V2, V8;     \
	VWMACCUVV V3, V5, V8;      \
	VWADDUWX  X24, V8, V8;     \
	VNSRLWI   $8, V8, V7;      \
	VSE16V    V7, (X20);       \
	ADD       X6, X20, X20

// func smoothH16RVV(dst *uint16, stride int, tl *uint16, w, n *int16, wd, h int)
TEXT ·smoothH16RVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X10
	MOV n+32(FP), X11
	MOV wd+40(FP), X12
	MOV h+48(FP), X13

	SLLI $1, X6, X6

	SLLI  $1, X12, X25
	ADD   X25, X7, X25
	MOVHU (X25), X21
	MOV   $128, X24

	MOV X12, X14
	MOV $0, X15

smh16col:
	VSETVLI X14, E16, M1, TA, MA, X16

	VMVVX X21, V5

	SLLI   $1, X15, X25
	ADD    X25, X10, X25
	VLE16V (X25), V2
	SLLI   $1, X15, X25
	ADD    X25, X11, X25
	VLE16V (X25), V3

	MOV  X13, X17
	SUB  $2, X7, X19
	SLLI $1, X15, X20
	ADD  X20, X5, X20

smh16row:
	SMH16_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, smh16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, smh16col
	RET

#define SM216_ROW              \
	MOVHU     (X18), X22;      \
	MOVHU     2(X18), X23;     \
	ADD       $4, X18, X18;    \
	VWMULUVX  X22, V1, V8;     \
	VWMACCUVX V4, X23, V8;     \
	MOVHU     (X19), X22;      \
	SUB       $2, X19, X19;    \
	VWMULUVX  X22, V2, V10;    \
	VWMACCUVV V3, V5, V10;     \
	VSETVLI   X16, E32, M2, TA, MA, X26; \
	VADDVV    V10, V8, V8;     \
	VADDVX    X24, V8, V8;     \
	VSETVLI   X16, E16, M1, TA, MA, X26; \
	VNSRLWI   $9, V8, V7;      \
	VSE16V    V7, (X20);       \
	ADD       X6, X20, X20

// func smooth16RVV(dst *uint16, stride int, tl *uint16, vw, hw, hn *int16, w, h int)
TEXT ·smooth16RVV(SB), NOSPLIT, $0-64
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV vw+24(FP), X10
	MOV hw+32(FP), X11
	MOV hn+40(FP), X28
	MOV w+48(FP), X12
	MOV h+56(FP), X13

	SLLI $1, X6, X6

	SLLI  $1, X13, X25
	SUB   X25, X7, X25
	MOVHU (X25), X21
	SLLI  $1, X12, X25
	ADD   X25, X7, X25
	MOVHU (X25), X29
	MOV   $256, X24

	MOV X12, X14
	MOV $0, X15

sm216col:
	VSETVLI X14, E16, M1, TA, MA, X16

	VMVVX X21, V4
	VMVVX X29, V5

	SLLI   $1, X15, X25
	ADD    X25, X7, X25
	ADD    $2, X25, X25
	VLE16V (X25), V1

	SLLI   $1, X15, X25
	ADD    X25, X11, X25
	VLE16V (X25), V2
	SLLI   $1, X15, X25
	ADD    X25, X28, X25
	VLE16V (X25), V3

	MOV  X13, X17
	MOV  X10, X18
	SUB  $2, X7, X19
	SLLI $1, X15, X20
	ADD  X20, X5, X20

sm216row:
	SM216_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, sm216row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, sm216col
	RET

#define PAETH16_ROW            \
	MOVHU     (X19), X22;      \
	SUB       $2, X19, X19;    \
	SUB       X21, X22, X23;   \
	SUB       X23, X0, X25;    \
	MAX       X23, X25, X25;   \
	VADDVX    X23, V6, V2;     \
	VRSUBVX   X0, V2, V3;      \
	VMAXVV    V3, V2, V2;      \
	VMINVX    X25, V7, V9;     \
	VMSEQVV   V9, V7, V0;      \
	VMERGEVXM X22, V1, V0, V11; \
	VMINVV    V2, V9, V10;     \
	VMSNEVV   V10, V9, V0;     \
	VMERGEVXM X21, V11, V0, V12; \
	VSE16V    V12, (X20);      \
	ADD       X6, X20, X20

// func paeth16RVV(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·paeth16RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	SLLI $1, X6, X6

	MOVHU (X7), X21

	MOV X12, X14
	MOV $0, X15

pth16col:
	VSETVLI X14, E16, M1, TA, MA, X16

	SLLI   $1, X15, X25
	ADD    X25, X7, X25
	ADD    $2, X25, X25
	VLE16V (X25), V1

	VSUBVX  X21, V1, V6
	VRSUBVX X0, V6, V4
	VMAXVV  V4, V6, V7

	MOV  X13, X17
	SUB  $2, X7, X19
	SLLI $1, X15, X20
	ADD  X20, X5, X20

pth16row:
	PAETH16_ROW
	SUB  $1, X17, X17
	BNE  X0, X17, pth16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, pth16col
	RET

// func splatDc16RVV(dst *uint16, stride, w, h int, dc uint16)
TEXT ·splatDc16RVV(SB), NOSPLIT, $0-34
	MOV   dst+0(FP), X5
	MOV   stride+8(FP), X6
	MOV   w+16(FP), X12
	MOV   h+24(FP), X13
	MOVHU dc+32(FP), X21

	SLLI $1, X6, X6

	MOV X12, X14
	MOV $0, X15

sp16col:
	VSETVLI X14, E16, M1, TA, MA, X16
	VMVVX   X21, V1

	MOV  X13, X17
	SLLI $1, X15, X20
	ADD  X20, X5, X20

sp16row:
	VSE16V V1, (X20)
	ADD    X6, X20, X20
	SUB    $1, X17, X17
	BNE    X0, X17, sp16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, sp16col
	RET

// func vpred16RVV(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·vpred16RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	SLLI $1, X6, X6

	MOV X12, X14
	MOV $0, X15

vp16col:
	VSETVLI X14, E16, M1, TA, MA, X16

	SLLI   $1, X15, X25
	ADD    X25, X7, X25
	ADD    $2, X25, X25
	VLE16V (X25), V1

	MOV  X13, X17
	SLLI $1, X15, X20
	ADD  X20, X5, X20

vp16row:
	VSE16V V1, (X20)
	ADD    X6, X20, X20
	SUB    $1, X17, X17
	BNE    X0, X17, vp16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, vp16col
	RET

// func hpred16RVV(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·hpred16RVV(SB), NOSPLIT, $0-40
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV tl+16(FP), X7
	MOV w+24(FP), X12
	MOV h+32(FP), X13

	SLLI $1, X6, X6

	MOV X12, X14
	MOV $0, X15

hp16col:
	VSETVLI X14, E16, M1, TA, MA, X16

	MOV  X13, X17
	SUB  $2, X7, X19
	SLLI $1, X15, X20
	ADD  X20, X5, X20

hp16row:
	MOVHU  (X19), X22
	SUB    $2, X19, X19
	VMVVX  X22, V1
	VSE16V V1, (X20)
	ADD    X6, X20, X20
	SUB    $1, X17, X17
	BNE    X0, X17, hp16row

	SUB X16, X14, X14
	ADD X16, X15, X15
	BNE X0, X14, hp16col
	RET

#define FI16_BLOCK             \
	MOVHU     (X25), X22;      \
	VWMULVX   X22, V1, V16;    \
	MOVHU     0(X18), X22;     \
	VWMACCVX  V2, X22, V16;    \
	MOVHU     2(X18), X22;     \
	VWMACCVX  V3, X22, V16;    \
	MOVHU     4(X18), X22;     \
	VWMACCVX  V4, X22, V16;    \
	MOVHU     6(X18), X22;     \
	VWMACCVX  V5, X22, V16;    \
	MOVHU     (X26), X22;      \
	VWMACCVX  V6, X22, V16;    \
	MOVHU     (X28), X22;      \
	VWMACCVX  V7, X22, V16;    \
	VSETVLI   X16, E32, M2, TA, MA, X29; \
	VADDVX    X23, V16, V16;   \
	VSRAVI    $4, V16, V16;    \
	VMAXVX    X0, V16, V16;    \
	VMINVX    X24, V16, V16;   \
	VSETVLI   X16, E16, M1, TA, MA, X29; \
	VNSRLWI   $0, V16, V18;    \
	VSETIVLI  $4, E16, M1, TA, MA, X29; \
	VSE16V    V18, (X20);      \
	VSLIDEDOWNVI $4, V18, V19; \
	VSE16V    V19, (X21);      \
	VSETIVLI  $8, E16, M1, TA, MA, X29

// func filterIntra16RVV(dst *uint16, stride int, tl *uint16, f *int16, w, h int, bitdepthMax int32)
TEXT ·filterIntra16RVV(SB), NOSPLIT, $0-52
	MOV  dst+0(FP), X5
	MOV  stride+8(FP), X6
	MOV  tl+16(FP), X7
	MOV  f+24(FP), X10
	MOV  w+32(FP), X12
	MOV  h+40(FP), X13
	MOVW bitdepthMax+48(FP), X24

	SLLI $1, X6, X6

	VSETIVLI $8, E16, M1, TA, MA, X29

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
	MOV $8, X16

	ADD $2, X7, X18
	MOV $0, X17

fi16Row:
	MOV X5, X20
	ADD X6, X5, X21

	SLLI $1, X17, X25
	SUB  X25, X7, X25
	SUB  $2, X25, X26
	SUB  $4, X25, X28

	MOV $0, X15

fi16Col:
	FI16_BLOCK

	ADD  $8, X18, X18
	SUB  $2, X18, X25
	ADD  $6, X20, X26
	ADD  $6, X21, X28
	ADD  $8, X20, X20
	ADD  $8, X21, X21

	ADD $4, X15, X15
	BLT X15, X12, fi16Col

	SLLI $1, X12, X25
	SUB  X25, X21, X18
	ADD  X6, X5, X5
	ADD  X6, X5, X5
	ADD  $2, X17, X17
	BLT  X17, X13, fi16Row
	RET

#define Z116_ROW               \
	AND       $0x3E, X17, X21; \
	MOV       $64, X22;        \
	SUB       X21, X22, X22;   \
	SRA       $6, X17, X23;    \
	SLLI      $1, X23, X23;    \
	ADD       X23, X7, X23;    \
	MOV       $0, X15;         \
	MOV       X12, X14

// func z1Full16RVV(dst *uint16, stride int, top *uint16, w, rows, dx, xpos int)
TEXT ·z1Full16RVV(SB), NOSPLIT, $0-56
	MOV dst+0(FP), X5
	MOV stride+8(FP), X6
	MOV top+16(FP), X7
	MOV w+24(FP), X12
	MOV rows+32(FP), X13
	MOV dx+40(FP), X18
	MOV xpos+48(FP), X17

	SLLI $1, X6, X6
	MOV  $32, X26

z116row:
	Z116_ROW
	SLLI $1, X15, X20
	ADD  X20, X5, X20

z116col:
	VSETVLI X14, E16, M1, TA, MA, X16

	SLLI   $1, X15, X25
	ADD    X25, X23, X25
	VLE16V (X25), V0
	ADD    $2, X25, X25
	VLE16V (X25), V1

	VWMULUVX  X22, V0, V8
	VWMACCUVX V1, X21, V8
	VWADDUWX  X26, V8, V8
	VNSRLWI   $6, V8, V7
	VSE16V    V7, (X20)

	SLLI $1, X16, X25
	ADD  X25, X20, X20
	ADD  X16, X15, X15
	SUB  X16, X14, X14
	BNE  X0, X14, z116col

	ADD X6, X5, X5
	ADD X18, X17, X17
	SUB $1, X13, X13
	BNE X0, X13, z116row
	RET
