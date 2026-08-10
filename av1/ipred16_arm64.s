//go:build arm64 && !noasm

#include "textflag.h"

#define UMULLS(Vd, Vn, Vm)   WORD $(0x2E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMLALS(Vd, Vn, Vm)   WORD $(0x2E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define URSHRS(Vd, Vn, SH)   WORD $(0x6F002400 | ((64 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define SSHRS(Vd, Vn, SH)    WORD $(0x4F000400 | ((64 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define XTNH(Vd, Vn)         WORD $(0x0E612800 | ((Vn) << 5) | (Vd))
#define SQXTUNH(Vd, Vn)      WORD $(0x2E612800 | ((Vn) << 5) | (Vd))
#define ABSH(Vd, Vn)         WORD $(0x4E60B800 | ((Vn) << 5) | (Vd))
#define SMINH(Vd, Vn, Vm)    WORD $(0x4E606C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMINH(Vd, Vn, Vm)    WORD $(0x6E606C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

#define SMLALI(Vd, Vn, Vm, IDX)  WORD $(0x0F402000 | ((((IDX)>>2)&1)<<11) | ((((IDX)>>1)&1)<<21) | (((IDX)&1)<<20) | ((Vm)<<16) | ((Vn)<<5) | (Vd))
#define SMLALI2(Vd, Vn, Vm, IDX) WORD $(0x4F402000 | ((((IDX)>>2)&1)<<11) | ((((IDX)>>1)&1)<<21) | (((IDX)&1)<<20) | ((Vm)<<16) | ((Vn)<<5) | (Vd))

#define SMV16_ROW           \
	MOVHU  (R8), R20;       \
	MOVHU  2(R8), R21;      \
	ADD    $4, R8, R8;      \
	VDUP   R20, V2.H4;      \
	VDUP   R21, V3.H4;      \
	UMULLS(4, 1, 2);        \
	UMLALS(4, 5, 3);        \
	URSHRS(4, 4, 8);        \
	XTNH(6, 4)

// func smoothV16NEON(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)
TEXT ·smoothV16NEON(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD weights+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5

	LSL $1, R1, R1

	SUB   R5, R2, R11
	SUB   R5, R11, R11
	VLD1R (R11), [V5.H4]

	MOVD $0, R6

smv16col:
	ADD  R6, R6, R11
	ADD  R11, R2, R11
	ADD  $2, R11, R11
	VLD1 (R11), [V1.H4]
	MOVD R5, R7
	MOVD R3, R8
	ADD  R6, R6, R10
	ADD  R10, R0, R10

smv16row:
	SMV16_ROW
	FMOVD F6, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smv16row

	ADD $4, R6, R6
	CMP R4, R6
	BLT smv16col
	RET

#define SMH16_ROW           \
	VLD1R  (R9), [V1.H4];   \
	SUB    $2, R9, R9;      \
	UMULLS(6, 1, 2);        \
	UMLALS(6, 4, 3);        \
	URSHRS(6, 6, 8);        \
	XTNH(7, 6)

// func smoothH16NEON(dst *uint16, stride int, tl *uint16, weights *int16, w, h int)
TEXT ·smoothH16NEON(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD weights+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5

	LSL $1, R1, R1

	ADD   R4, R4, R11
	ADD   R11, R2, R11
	VLD1R (R11), [V4.H4]

	MOVD $0, R6

smh16col:
	ADD  R6, R6, R11
	ADD  R11, R11, R11
	ADD  R11, R3, R11
	VLD2 (R11), [V2.H4, V3.H4]
	MOVD R5, R7
	SUB  $2, R2, R9
	ADD  R6, R6, R10
	ADD  R10, R0, R10

smh16row:
	SMH16_ROW
	FMOVD F7, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smh16row

	ADD $4, R6, R6
	CMP R4, R6
	BLT smh16col
	RET

#define SM216_ROW           \
	MOVHU  (R8), R20;       \
	MOVHU  2(R8), R21;      \
	ADD    $4, R8, R8;      \
	VDUP   R20, V12.H4;     \
	VDUP   R21, V13.H4;     \
	UMULLS(9, 1, 12);       \
	UMLALS(9, 5, 13);       \
	VLD1R  (R9), [V10.H4];  \
	SUB    $2, R9, R9;      \
	UMULLS(11, 10, 2);      \
	UMLALS(11, 4, 3);       \
	VADD   V11.S4, V9.S4, V9.S4; \
	URSHRS(9, 9, 9);        \
	XTNH(14, 9)

// func smooth16NEON(dst *uint16, stride int, tl *uint16, vw, hw *int16, w, h int)
TEXT ·smooth16NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD vw+24(FP), R3
	MOVD hw+32(FP), R12
	MOVD w+40(FP), R4
	MOVD h+48(FP), R5

	LSL $1, R1, R1

	SUB   R5, R2, R11
	SUB   R5, R11, R11
	VLD1R (R11), [V5.H4]
	ADD   R4, R4, R11
	ADD   R11, R2, R11
	VLD1R (R11), [V4.H4]

	MOVD $0, R6

sm216col:
	ADD  R6, R6, R11
	ADD  R11, R2, R11
	ADD  $2, R11, R11
	VLD1 (R11), [V1.H4]
	ADD  R6, R6, R11
	ADD  R11, R11, R11
	ADD  R11, R12, R11
	VLD2 (R11), [V2.H4, V3.H4]
	MOVD R5, R7
	MOVD R3, R8
	SUB  $2, R2, R9
	ADD  R6, R6, R10
	ADD  R10, R0, R10

sm216row:
	SM216_ROW
	FMOVD F14, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   sm216row

	ADD $4, R6, R6
	CMP R4, R6
	BLT sm216col
	RET

#define PAETH16_ROW              \
	VLD1R (R9), [V3.H8];         \
	SUB   $2, R9, R9;            \
	VSUB  V5.H8, V3.H8, V2.H8;   \
	ABSH(8, 2);                  \
	VADD  V6.H8, V2.H8, V2.H8;   \
	ABSH(2, 2);                  \
	SMINH(9, 7, 8);              \
	VCMEQ V9.H8, V7.H8, V10.H8;  \
	VMOV  V10.B16, V11.B16;      \
	VBSL  V1.B16, V3.B16, V11.B16; \
	SMINH(10, 9, 2);             \
	VCMEQ V10.H8, V9.H8, V12.H8; \
	VBSL  V5.B16, V11.B16, V12.B16

// func paeth16NEON(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·paeth16NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	LSL $1, R1, R1

	VLD1R (R2), [V5.H8]

	CMP $4, R4
	BEQ pth164

	MOVD $0, R6

pth168col:
	ADD  R6, R6, R11
	ADD  R11, R2, R11
	ADD  $2, R11, R11
	VLD1 (R11), [V1.H8]
	VSUB V5.H8, V1.H8, V6.H8
	ABSH(7, 6)
	MOVD R5, R7
	SUB  $2, R2, R9
	ADD  R6, R6, R10
	ADD  R10, R0, R10

pth168row:
	PAETH16_ROW
	VST1 [V12.H8], (R10)
	ADD  R1, R10, R10
	SUBS $1, R7, R7
	BNE  pth168row

	ADD $8, R6, R6
	CMP R4, R6
	BLT pth168col
	RET

pth164:
	ADD  $2, R2, R11
	VLD1 (R11), [V1.H4]
	VSUB V5.H8, V1.H8, V6.H8
	ABSH(7, 6)
	MOVD R5, R7
	SUB  $2, R2, R9
	MOVD R0, R10

pth164row:
	PAETH16_ROW
	FMOVD F12, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   pth164row
	RET

// func splatDc16NEON(dst *uint16, stride, w, h int, dc uint16)
TEXT ·splatDc16NEON(SB), NOSPLIT, $0-34
	MOVD  dst+0(FP), R0
	MOVD  stride+8(FP), R1
	MOVD  w+16(FP), R4
	MOVD  h+24(FP), R5
	MOVHU dc+32(FP), R11

	LSL $1, R1, R1

	VDUP R11, V0.H8
	VMOV V0.B16, V1.B16
	VMOV V0.B16, V2.B16
	VMOV V0.B16, V3.B16

	CMP $4, R4
	BEQ sp164
	CMP $8, R4
	BEQ sp168
	CMP $16, R4
	BEQ sp1616
	CMP $32, R4
	BEQ sp1632

sp1664:
	VST1.P [V0.H8, V1.H8, V2.H8, V3.H8], 64(R0)
	VST1   [V0.H8, V1.H8, V2.H8, V3.H8], (R0)
	SUB    $64, R0, R0
	ADD    R1, R0, R0
	SUBS   $1, R5, R5
	BNE    sp1664
	RET

sp164:
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   sp164
	RET

sp168:
	VST1 [V0.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp168
	RET

sp1616:
	VST1 [V0.H8, V1.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp1616
	RET

sp1632:
	VST1 [V0.H8, V1.H8, V2.H8, V3.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp1632
	RET

// func vpred16NEON(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·vpred16NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	LSL $1, R1, R1
	ADD $2, R2, R11

	CMP $4, R4
	BEQ vp164
	CMP $8, R4
	BEQ vp168
	CMP $16, R4
	BEQ vp1616
	CMP $32, R4
	BEQ vp1632

	VLD1.P 64(R11), [V0.H8, V1.H8, V2.H8, V3.H8]
	VLD1   (R11), [V4.H8, V5.H8, V6.H8, V7.H8]

vp1664:
	VST1.P [V0.H8, V1.H8, V2.H8, V3.H8], 64(R0)
	VST1   [V4.H8, V5.H8, V6.H8, V7.H8], (R0)
	SUB    $64, R0, R0
	ADD    R1, R0, R0
	SUBS   $1, R5, R5
	BNE    vp1664
	RET

vp164:
	FMOVD (R11), F0

vp164row:
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   vp164row
	RET

vp168:
	VLD1 (R11), [V0.H8]

vp168row:
	VST1 [V0.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp168row
	RET

vp1616:
	VLD1 (R11), [V0.H8, V1.H8]

vp1616row:
	VST1 [V0.H8, V1.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp1616row
	RET

vp1632:
	VLD1 (R11), [V0.H8, V1.H8, V2.H8, V3.H8]

vp1632row:
	VST1 [V0.H8, V1.H8, V2.H8, V3.H8], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp1632row
	RET

// func hpred16NEON(dst *uint16, stride int, tl *uint16, w, h int)
TEXT ·hpred16NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	LSL $1, R1, R1
	SUB $2, R2, R9

	CMP $4, R4
	BEQ hp164
	CMP $8, R4
	BEQ hp168
	CMP $16, R4
	BEQ hp1616
	CMP $32, R4
	BEQ hp1632

hp1664:
	VLD1R  (R9), [V0.H8]
	VMOV   V0.B16, V1.B16
	VMOV   V0.B16, V2.B16
	VMOV   V0.B16, V3.B16
	VST1.P [V0.H8, V1.H8, V2.H8, V3.H8], 64(R0)
	VST1   [V0.H8, V1.H8, V2.H8, V3.H8], (R0)
	SUB    $64, R0, R0
	ADD    R1, R0, R0
	SUB    $2, R9, R9
	SUBS   $1, R5, R5
	BNE    hp1664
	RET

hp164:
	VLD1R (R9), [V0.H4]
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUB   $2, R9, R9
	SUBS  $1, R5, R5
	BNE   hp164
	RET

hp168:
	VLD1R (R9), [V0.H8]
	VST1  [V0.H8], (R0)
	ADD   R1, R0, R0
	SUB   $2, R9, R9
	SUBS  $1, R5, R5
	BNE   hp168
	RET

hp1616:
	VLD1R (R9), [V0.H8]
	VMOV  V0.B16, V1.B16
	VST1  [V0.H8, V1.H8], (R0)
	ADD   R1, R0, R0
	SUB   $2, R9, R9
	SUBS  $1, R5, R5
	BNE   hp1616
	RET

hp1632:
	VLD1R (R9), [V0.H8]
	VMOV  V0.B16, V1.B16
	VMOV  V0.B16, V2.B16
	VMOV  V0.B16, V3.B16
	VST1  [V0.H8, V1.H8, V2.H8, V3.H8], (R0)
	ADD   R1, R0, R0
	SUB   $2, R9, R9
	SUBS  $1, R5, R5
	BNE   hp1632
	RET

#define FI16_BLOCK               \
	MOVD  (R8), R21;             \
	MOVHU (R10), R22;            \
	MOVHU (R11), R23;            \
	MOVHU (R9), R24;             \
	ORR   R23<<16, R22, R22;     \
	ORR   R24<<32, R22, R22;     \
	FMOVD R21, F15;              \
	VMOV  R22, V15.D[1];         \
	VMOV  V20.B16, V16.B16;      \
	VMOV  V20.B16, V17.B16;      \
	SMLALI(16, 0, 15, 6);        \
	SMLALI2(17, 0, 15, 6);       \
	SMLALI(16, 1, 15, 0);        \
	SMLALI2(17, 1, 15, 0);       \
	SMLALI(16, 2, 15, 1);        \
	SMLALI2(17, 2, 15, 1);       \
	SMLALI(16, 3, 15, 2);        \
	SMLALI2(17, 3, 15, 2);       \
	SMLALI(16, 4, 15, 3);        \
	SMLALI2(17, 4, 15, 3);       \
	SMLALI(16, 5, 15, 4);        \
	SMLALI2(17, 5, 15, 4);       \
	SMLALI(16, 6, 15, 5);        \
	SMLALI2(17, 6, 15, 5);       \
	SSHRS(16, 16, 4);            \
	SSHRS(17, 17, 4);            \
	SQXTUNH(18, 16);             \
	SQXTUNH(19, 17);             \
	UMINH(18, 18, 21);           \
	UMINH(19, 19, 21);           \
	FMOVD F18, (R12);            \
	FMOVD F19, (R13)

// func filterIntra16NEON(dst *uint16, stride int, tl *uint16, f *int16, w, h int, bitdepthMax int32)
TEXT ·filterIntra16NEON(SB), NOSPLIT, $0-52
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD f+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5
	MOVW bitdepthMax+48(FP), R25

	LSL $1, R1, R1

	VLD1.P 32(R3), [V0.H8, V1.H8]
	VLD1.P 32(R3), [V2.H8, V3.H8]
	VLD1.P 32(R3), [V4.H8, V5.H8]
	VLD1   (R3), [V6.H8]

	MOVD $8, R20
	VDUP R20, V20.S4
	VDUP R25, V21.H8

	ADD  $2, R2, R8
	MOVD $0, R7

fi16Row:
	MOVD R0, R12
	ADD  R1, R0, R13

	SUB R7, R2, R9
	SUB R7, R9, R9
	SUB $2, R9, R10
	SUB $4, R9, R11

	MOVD R8, R14
	MOVD $0, R6

fi16Col:
	FI16_BLOCK

	ADD $8, R8, R8
	SUB $2, R8, R9
	ADD $6, R12, R10
	ADD $6, R13, R11
	ADD $8, R12, R12
	ADD $8, R13, R13

	ADD $4, R6, R6
	CMP R4, R6
	BLT fi16Col

	ADD R6, R6, R14
	SUB R14, R13, R8
	ADD R1, R0, R0
	ADD R1, R0, R0
	ADD $2, R7, R7
	CMP R5, R7
	BLT fi16Row
	RET

#define Z116_WEIGHTS      \
	AND  $0x3E, R11, R20; \
	MOVD $64, R21;        \
	SUB  R20, R21, R21;   \
	VDUP R21, V2.H4;      \
	VDUP R20, V3.H4;      \
	ASR  $6, R11, R22;    \
	ADD  R22, R22, R22;   \
	ADD  R22, R2, R23

// func z1Full16NEON(dst *uint16, stride int, top *uint16, w, rows, dx, xpos int)
TEXT ·z1Full16NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD top+16(FP), R2
	MOVD w+24(FP), R4
	MOVD rows+32(FP), R5
	MOVD dx+40(FP), R9
	MOVD xpos+48(FP), R11

	LSL $1, R1, R1

z116row:
	Z116_WEIGHTS
	MOVD R0, R10
	MOVD $0, R6

z116col:
	ADD    R6, R6, R24
	ADD    R24, R23, R24
	VLD1   (R24), [V0.H4]
	ADD    $2, R24, R24
	VLD1   (R24), [V1.H4]
	UMULLS(4, 0, 2)
	UMLALS(4, 1, 3)
	URSHRS(4, 4, 6)
	XTNH(5, 4)
	ADD    R6, R6, R24
	ADD    R24, R10, R24
	FMOVD  F5, (R24)
	ADD    $4, R6, R6
	CMP    R4, R6
	BLT    z116col

	ADD  R1, R0, R0
	ADD  R9, R11, R11
	SUBS $1, R5, R5
	BNE  z116row
	RET
