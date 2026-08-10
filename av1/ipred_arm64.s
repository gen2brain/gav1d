//go:build arm64 && !noasm

#include "textflag.h"

#define UMULL8(Vd, Vn, Vm)  WORD $(0x2E20C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMLAL8(Vd, Vn, Vm)  WORD $(0x2E208000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define XTN8(Vd, Vn)        WORD $(0x0E212800 | ((Vn) << 5) | (Vd))
#define URSHR8H(Vd, Vn)     WORD $(0x6F002400 | ((32 - 8) << 16) | ((Vn) << 5) | (Vd))
#define UHADDH(Vd, Vn, Vm)  WORD $(0x6E600400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UQSUBB(Vd, Vn, Vm)  WORD $(0x6E202C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UQADDB(Vd, Vn, Vm)  WORD $(0x6E200C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define URHADDB(Vd, Vn, Vm) WORD $(0x6E201400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

#define SMV_ROW              \
	VLD1R  (R8), [V2.B8];    \
	ADD    $1, R8, R8;       \
	VEOR   V16.B8, V2.B8, V3.B8; \
	UMULL8(4, 1, 2);         \
	UMLAL8(4, 5, 3);         \
	VUADDW V5.B8, V4.H8, V4.H8; \
	URSHR8H(4, 4);           \
	XTN8(6, 4)

// func smoothV8NEON(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)
TEXT ·smoothV8NEON(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD weights+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5

	SUB   R5, R2, R11
	VLD1R (R11), [V5.B8]
	VMOVI $255, V16.B8

	CMP $4, R4
	BEQ smv4

	MOVD $0, R6

smv8col:
	ADD   R6, R2, R11
	ADD   $1, R11, R11
	VLD1  (R11), [V1.B8]
	MOVD  R5, R7
	MOVD  R3, R8
	ADD   R6, R0, R10

smv8row:
	SMV_ROW
	FMOVD F6, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smv8row

	ADD  $8, R6, R6
	CMP  R4, R6
	BLT  smv8col
	RET

smv4:
	ADD   $1, R2, R11
	FMOVS (R11), F1
	MOVD  R5, R7
	MOVD  R3, R8
	MOVD  R0, R10

smv4row:
	SMV_ROW
	FMOVS F6, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smv4row
	RET

#define SMH_ROW              \
	VLD1R  (R9), [V1.B8];    \
	SUB    $1, R9, R9;       \
	UMULL8(6, 1, 2);         \
	UMLAL8(6, 4, 3);         \
	VUADDW V4.B8, V6.H8, V6.H8; \
	URSHR8H(6, 6);           \
	XTN8(7, 6)

// func smoothH8NEON(dst *uint8, stride int, tl *uint8, weights *uint8, w, h int)
TEXT ·smoothH8NEON(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD weights+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5

	ADD   R4, R2, R11
	VLD1R (R11), [V4.B8]
	VMOVI $255, V16.B8

	CMP $4, R4
	BEQ smh4

	MOVD $0, R6

smh8col:
	ADD  R6, R3, R11
	VLD1 (R11), [V2.B8]
	VEOR V16.B8, V2.B8, V3.B8
	MOVD R5, R7
	SUB  $1, R2, R9
	ADD  R6, R0, R10

smh8row:
	SMH_ROW
	FMOVD F7, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smh8row

	ADD  $8, R6, R6
	CMP  R4, R6
	BLT  smh8col
	RET

smh4:
	VLD1 (R3), [V2.B8]
	VEOR V16.B8, V2.B8, V3.B8
	MOVD R5, R7
	SUB  $1, R2, R9
	MOVD R0, R10

smh4row:
	SMH_ROW
	FMOVS F7, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   smh4row
	RET

#define SM2_ROW               \
	VLD1R  (R8), [V7.B8];     \
	ADD    $1, R8, R8;        \
	VEOR   V16.B8, V7.B8, V8.B8; \
	UMULL8(9, 1, 7);          \
	UMLAL8(9, 5, 8);          \
	VUADDW V5.B8, V9.H8, V9.H8; \
	VLD1R  (R9), [V10.B8];    \
	SUB    $1, R9, R9;        \
	UMULL8(11, 10, 2);        \
	UMLAL8(11, 4, 3);         \
	VUADDW V4.B8, V11.H8, V11.H8; \
	UHADDH(9, 9, 11);         \
	URSHR8H(9, 9);            \
	XTN8(12, 9)

// func smooth8NEON(dst *uint8, stride int, tl *uint8, vw, hw *uint8, w, h int)
TEXT ·smooth8NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD vw+24(FP), R3
	MOVD hw+32(FP), R12
	MOVD w+40(FP), R4
	MOVD h+48(FP), R5

	SUB   R5, R2, R11
	VLD1R (R11), [V5.B8]
	ADD   R4, R2, R11
	VLD1R (R11), [V4.B8]
	VMOVI $255, V16.B8

	CMP $4, R4
	BEQ sm24

	MOVD $0, R6

sm28col:
	ADD  R6, R2, R11
	ADD  $1, R11, R11
	VLD1 (R11), [V1.B8]
	ADD  R6, R12, R11
	VLD1 (R11), [V2.B8]
	VEOR V16.B8, V2.B8, V3.B8
	MOVD R5, R7
	MOVD R3, R8
	SUB  $1, R2, R9
	ADD  R6, R0, R10

sm28row:
	SM2_ROW
	FMOVD F12, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   sm28row

	ADD  $8, R6, R6
	CMP  R4, R6
	BLT  sm28col
	RET

sm24:
	ADD   $1, R2, R11
	FMOVS (R11), F1
	VLD1  (R12), [V2.B8]
	VEOR  V16.B8, V2.B8, V3.B8
	MOVD  R5, R7
	MOVD  R3, R8
	SUB   $1, R2, R9
	MOVD  R0, R10

sm24row:
	SM2_ROW
	FMOVS F12, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   sm24row
	RET

#define PAETH_ROW             \
	VLD1R (R9), [V3.B16];     \
	SUB   $1, R9, R9;         \
	URHADDB(1, 6, 3);         \
	VEOR  V6.B16, V3.B16, V0.B16; \
	VAND  V15.B16, V0.B16, V0.B16; \
	UQSUBB(2, 5, 1);          \
	VSUB  V0.B16, V1.B16, V1.B16; \
	UQSUBB(1, 1, 5);          \
	VORR  V2.B16, V1.B16, V1.B16; \
	UQADDB(1, 1, 1);          \
	VORR  V0.B16, V1.B16, V1.B16; \
	UQSUBB(2, 5, 3);          \
	UQSUBB(0, 3, 5);          \
	VORR  V0.B16, V2.B16, V2.B16; \
	VUMIN V7.B16, V2.B16, V2.B16; \
	VCMEQ V7.B16, V2.B16, V8.B16; \
	VBSL  V6.B16, V3.B16, V8.B16; \
	VUMIN V2.B16, V1.B16, V1.B16; \
	VCMEQ V2.B16, V1.B16, V9.B16; \
	VBSL  V5.B16, V8.B16, V9.B16

// func paeth8NEON(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·paeth8NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	VLD1R (R2), [V5.B16]
	VMOVI $1, V15.B16

	CMP $4, R4
	BEQ pth4
	CMP $8, R4
	BEQ pth8

	MOVD $0, R6

pth16col:
	ADD  R6, R2, R11
	ADD  $1, R11, R11
	VLD1 (R11), [V6.B16]
	UQSUBB(7, 5, 6)
	UQSUBB(0, 6, 5)
	VORR V0.B16, V7.B16, V7.B16
	MOVD R5, R7
	SUB  $1, R2, R9
	ADD  R6, R0, R10

pth16row:
	PAETH_ROW
	VST1 [V9.B16], (R10)
	ADD  R1, R10, R10
	SUBS $1, R7, R7
	BNE  pth16row

	ADD  $16, R6, R6
	CMP  R4, R6
	BLT  pth16col
	RET

pth4:
	ADD   $1, R2, R11
	FMOVS (R11), F6
	UQSUBB(7, 5, 6)
	UQSUBB(0, 6, 5)
	VORR V0.B16, V7.B16, V7.B16
	MOVD R5, R7
	SUB  $1, R2, R9
	MOVD R0, R10

pth4row:
	PAETH_ROW
	FMOVS F9, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   pth4row
	RET

pth8:
	ADD   $1, R2, R11
	FMOVD (R11), F6
	UQSUBB(7, 5, 6)
	UQSUBB(0, 6, 5)
	VORR V0.B16, V7.B16, V7.B16
	MOVD R5, R7
	SUB  $1, R2, R9
	MOVD R0, R10

pth8row:
	PAETH_ROW
	FMOVD F9, (R10)
	ADD   R1, R10, R10
	SUBS  $1, R7, R7
	BNE   pth8row
	RET

// func splatDc8NEON(dst *uint8, stride, w, h int, dc uint8)
TEXT ·splatDc8NEON(SB), NOSPLIT, $0-33
	MOVD  dst+0(FP), R0
	MOVD  stride+8(FP), R1
	MOVD  w+16(FP), R4
	MOVD  h+24(FP), R5
	MOVBU dc+32(FP), R11

	VDUP R11, V0.B16
	VMOV V0.B16, V1.B16
	VMOV V0.B16, V2.B16
	VMOV V0.B16, V3.B16

	CMP $4, R4
	BEQ sp4
	CMP $8, R4
	BEQ sp8
	CMP $16, R4
	BEQ sp16
	CMP $32, R4
	BEQ sp32

sp64:
	VST1 [V0.B16, V1.B16, V2.B16, V3.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp64
	RET

sp4:
	FMOVS F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   sp4
	RET

sp8:
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   sp8
	RET

sp16:
	VST1 [V0.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp16
	RET

sp32:
	VST1 [V0.B16, V1.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  sp32
	RET

// func vpred8NEON(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·vpred8NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	ADD $1, R2, R11

	CMP $4, R4
	BEQ vp4
	CMP $8, R4
	BEQ vp8
	CMP $16, R4
	BEQ vp16
	CMP $32, R4
	BEQ vp32

	VLD1 (R11), [V0.B16, V1.B16, V2.B16, V3.B16]

vp64:
	VST1 [V0.B16, V1.B16, V2.B16, V3.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp64
	RET

vp4:
	FMOVS (R11), F0

vp4row:
	FMOVS F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   vp4row
	RET

vp8:
	FMOVD (R11), F0

vp8row:
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUBS  $1, R5, R5
	BNE   vp8row
	RET

vp16:
	VLD1 (R11), [V0.B16]

vp16row:
	VST1 [V0.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp16row
	RET

vp32:
	VLD1 (R11), [V0.B16, V1.B16]

vp32row:
	VST1 [V0.B16, V1.B16], (R0)
	ADD  R1, R0, R0
	SUBS $1, R5, R5
	BNE  vp32row
	RET

// func hpred8NEON(dst *uint8, stride int, tl *uint8, w, h int)
TEXT ·hpred8NEON(SB), NOSPLIT, $0-40
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD w+24(FP), R4
	MOVD h+32(FP), R5

	SUB $1, R2, R9

	CMP $4, R4
	BEQ hp4
	CMP $8, R4
	BEQ hp8
	CMP $16, R4
	BEQ hp16
	CMP $32, R4
	BEQ hp32

hp64:
	VLD1R (R9), [V0.B16]
	VMOV  V0.B16, V1.B16
	VMOV  V0.B16, V2.B16
	VMOV  V0.B16, V3.B16
	VST1  [V0.B16, V1.B16, V2.B16, V3.B16], (R0)
	ADD   R1, R0, R0
	SUB   $1, R9, R9
	SUBS  $1, R5, R5
	BNE   hp64
	RET

hp4:
	VLD1R (R9), [V0.B8]
	FMOVS F0, (R0)
	ADD   R1, R0, R0
	SUB   $1, R9, R9
	SUBS  $1, R5, R5
	BNE   hp4
	RET

hp8:
	VLD1R (R9), [V0.B8]
	FMOVD F0, (R0)
	ADD   R1, R0, R0
	SUB   $1, R9, R9
	SUBS  $1, R5, R5
	BNE   hp8
	RET

hp16:
	VLD1R (R9), [V0.B16]
	VST1  [V0.B16], (R0)
	ADD   R1, R0, R0
	SUB   $1, R9, R9
	SUBS  $1, R5, R5
	BNE   hp16
	RET

hp32:
	VLD1R (R9), [V0.B16]
	VMOV  V0.B16, V1.B16
	VST1  [V0.B16, V1.B16], (R0)
	ADD   R1, R0, R0
	SUB   $1, R9, R9
	SUBS  $1, R5, R5
	BNE   hp32
	RET

#define MLAIDX(Vd, Vn, Vm, IDX) WORD $(0x6F400000 | ((((IDX)>>2)&1)<<11) | ((((IDX)>>1)&1)<<21) | (((IDX)&1)<<20) | ((Vm)<<16) | ((Vn)<<5) | (Vd))
#define SQSHRUN4(Vd, Vn) WORD $(0x2F008400 | ((16 - 4) << 16) | ((Vn) << 5) | (Vd))

#define FI_BLOCK             \
	MOVWU (R8), R21;         \
	MOVBU (R10), R22;        \
	MOVBU (R11), R23;        \
	MOVBU (R9), R24;         \
	ORR   R22<<32, R21, R21; \
	ORR   R23<<40, R21, R21; \
	ORR   R24<<48, R21, R21; \
	FMOVD R21, F14;          \
	VUSHLL $0, V14.B8, V15.H8; \
	VMOV  V13.B16, V16.B16;  \
	MLAIDX(16, 0, 15, 6);    \
	MLAIDX(16, 1, 15, 0);    \
	MLAIDX(16, 2, 15, 1);    \
	MLAIDX(16, 3, 15, 2);    \
	MLAIDX(16, 4, 15, 3);    \
	MLAIDX(16, 5, 15, 4);    \
	MLAIDX(16, 6, 15, 5);    \
	SQSHRUN4(17, 16);        \
	VMOV  V17.S[0], R21;     \
	MOVW  R21, (R12);        \
	VMOV  V17.S[1], R22;     \
	MOVW  R22, (R13)

// func filterIntra8NEON(dst *uint8, stride int, tl *uint8, f *int16, w, h int)
TEXT ·filterIntra8NEON(SB), NOSPLIT, $0-48
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tl+16(FP), R2
	MOVD f+24(FP), R3
	MOVD w+32(FP), R4
	MOVD h+40(FP), R5

	VLD1.P 32(R3), [V0.H8, V1.H8]
	VLD1.P 32(R3), [V2.H8, V3.H8]
	VLD1.P 32(R3), [V4.H8, V5.H8]
	VLD1   (R3), [V6.H8]

	MOVD $8, R25
	VDUP R25, V13.H8

	ADD  $1, R2, R8
	MOVD $0, R7

fiRow:
	MOVD R0, R12
	ADD  R1, R0, R13

	SUB R7, R2, R9
	SUB $1, R9, R10
	SUB $2, R9, R11

	MOVD R8, R14
	MOVD $0, R6

fiCol:
	FI_BLOCK

	ADD $4, R8, R8
	SUB $1, R8, R9
	ADD $3, R12, R10
	ADD $3, R13, R11
	ADD $4, R12, R12
	ADD $4, R13, R13

	ADD $4, R6, R6
	CMP R4, R6
	BLT fiCol

	SUB  R6, R13, R8
	ADD  R1, R0, R0
	ADD  R1, R0, R0
	ADD  $2, R7, R7
	CMP  R5, R7
	BLT  fiRow
	RET

#define URSHR6H(Vd, Vn) WORD $(0x6F002400 | ((32 - 6) << 16) | ((Vn) << 5) | (Vd))

#define Z1_WEIGHTS       \
	AND  $0x3E, R11, R20; \
	MOVD $64, R21;       \
	SUB  R20, R21, R21;  \
	VDUP R21, V2.B8;     \
	VDUP R20, V3.B8;     \
	ASR  $6, R11, R22;   \
	ADD  R22, R2, R23

#define Z1_TAP           \
	UMULL8(4, 0, 2);     \
	UMLAL8(4, 1, 3);     \
	URSHR6H(4, 4);       \
	XTN8(5, 4)

// func z1Full8NEON(dst *uint8, stride int, top *uint8, w, rows, dx, xpos int)
TEXT ·z1Full8NEON(SB), NOSPLIT, $0-56
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD top+16(FP), R2
	MOVD w+24(FP), R4
	MOVD rows+32(FP), R5
	MOVD dx+40(FP), R9
	MOVD xpos+48(FP), R11

	CMP $4, R4
	BEQ z1w4

z1w8:
	Z1_WEIGHTS
	MOVD R0, R10
	MOVD $0, R6

z1w8col:
	ADD  R6, R23, R24
	VLD1 (R24), [V0.B8]
	ADD  $1, R24, R24
	VLD1 (R24), [V1.B8]
	Z1_TAP
	ADD   R6, R10, R24
	FMOVD F5, (R24)
	ADD   $8, R6, R6
	CMP   R4, R6
	BLT   z1w8col

	ADD  R1, R0, R0
	ADD  R9, R11, R11
	SUBS $1, R5, R5
	BNE  z1w8
	RET

z1w4:
	Z1_WEIGHTS
	VLD1  (R23), [V0.B8]
	ADD   $1, R23, R23
	VLD1  (R23), [V1.B8]
	Z1_TAP
	FMOVS F5, (R0)
	ADD   R1, R0, R0
	ADD   R9, R11, R11
	SUBS  $1, R5, R5
	BNE   z1w4
	RET
