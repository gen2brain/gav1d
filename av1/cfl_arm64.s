//go:build arm64 && !noasm

#include "textflag.h"

// Go's assembler has no SSHLL, so the widening of a signed half into a word is
// hand encoded, low half and high half.
#define SSHLLL(Vd, Vn) WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHLLH(Vd, Vn) WORD $(0x4F10A400 | ((Vn) << 5) | (Vd))

#define MULS4(Vd, Vn, Vm)  WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS4(Vd, Vn, Vm) WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS4(Vd, Vn, Vm) WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHRS(Vd, Vn, SH)  WORD $(0x4F000400 | ((64 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define XTN4(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))
#define XTNB(Vd, Vn)       WORD $(0x0E212800 | ((Vn) << 5) | (Vd))

// CFLROW scales four AC samples by alpha, rounds the magnitude, puts the sign
// back and folds the result into the DC. There is no NEON absolute here: the
// arithmetic shift gives a mask that both negates and restores.
#define CFLROW               \
	SSHLLL(1, 0);        \
	MULS4(1, 1, 17);     \
	SSHRS(2, 1, 31);     \
	VEOR V2.B16, V1.B16, V3.B16; \
	VSUB V2.S4, V3.S4, V3.S4;    \
	VADD V20.S4, V3.S4, V3.S4;   \
	SSHRS(3, 3, 6);      \
	VEOR V2.B16, V3.B16, V3.B16; \
	VSUB V2.S4, V3.S4, V3.S4;    \
	VADD V16.S4, V3.S4, V3.S4;   \
	SMAXS4(3, 3, 19);    \
	SMINS4(3, 3, 18);    \
	XTN4(4, 3)

#define CFLSETUP              \
	MOVD dst+0(FP), R0;   \
	MOVD stride+8(FP), R1; \
	MOVD w+16(FP), R2;    \
	MOVD h+24(FP), R3;    \
	MOVW dc+32(FP), R4;   \
	MOVD ac+40(FP), R5;   \
	MOVW alpha+48(FP), R6; \
	MOVW max+52(FP), R7;  \
	VDUP R4, V16.S4;      \
	VDUP R6, V17.S4;      \
	VDUP R7, V18.S4;      \
	VMOVI $0, V19.B16;    \
	MOVD $32, R8;         \
	VDUP R8, V20.S4

// func cflPredict8NEON(dst *uint8, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict8NEON(SB), NOSPLIT, $0-56
	CFLSETUP

cfl8row:
	MOVD $0, R10

cfl8col:
	ADD    R10<<1, R5, R11
	FMOVD  (R11), F0
	CFLROW
	XTNB(4, 4)
	ADD    R10, R0, R12
	FMOVS  F4, (R12)
	ADD    $4, R10, R10
	CMP    R2, R10
	BLT    cfl8col

	ADD  R1, R0, R0
	ADD  R2<<1, R5, R5
	SUBS $1, R3, R3
	BNE  cfl8row

	RET

// func cflPredict16NEON(dst *uint16, stride, w, h int, dc int32, ac *int16, alpha, max int32)
TEXT ·cflPredict16NEON(SB), NOSPLIT, $0-56
	CFLSETUP
	LSL $1, R1, R1

cfl16row:
	MOVD $0, R10

cfl16col:
	ADD    R10<<1, R5, R11
	FMOVD  (R11), F0
	CFLROW
	ADD    R10<<1, R0, R12
	FMOVD  F4, (R12)
	ADD    $4, R10, R10
	CMP    R2, R10
	BLT    cfl16col

	ADD  R1, R0, R0
	ADD  R2<<1, R5, R5
	SUBS $1, R3, R3
	BNE  cfl16row

	RET


// func cflAcNormNEON(ac *int16, n int, log2sz int)
TEXT ·cflAcNormNEON(SB), NOSPLIT, $0-24
	MOVD ac+0(FP), R0
	MOVD n+8(FP), R1
	MOVD log2sz+16(FP), R2

	VMOVI $0, V0.B16
	MOVD  R0, R3
	MOVD  R1, R4

acsum:
	VLD1.P 16(R3), [V1.H8]
	SSHLLL(2, 1)
	SSHLLH(3, 1)
	VADD   V2.S4, V0.S4, V0.S4
	VADD   V3.S4, V0.S4, V0.S4
	SUBS   $8, R4
	BNE    acsum

	VADDV V0.S4, V4
	VMOV  V4.S[0], R5

	MOVD $1, R6
	LSL  R2, R6, R6
	LSR  $1, R6, R6
	ADDW R6, R5, R5
	ASRW R2, R5, R5

	VDUP R5, V5.H8
	MOVD R0, R3
	MOVD R1, R4

acsub:
	VLD1   (R3), [V1.H8]
	VSUB   V5.H8, V1.H8, V1.H8
	VST1.P [V1.H8], 16(R3)
	SUBS   $8, R4
	BNE    acsub

	RET

#define UMULLH(Vd, Vn, Vm) WORD $(0x2E20C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMLALH(Vd, Vn, Vm) WORD $(0x2E208000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define URSHRH(Vd, Vn)     WORD $(0x6F1A2400 | ((Vn) << 5) | (Vd))
#define UMULLS(Vd, Vn, Vm) WORD $(0x2E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMLALS(Vd, Vn, Vm) WORD $(0x2E608000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define URSHRS(Vd, Vn)     WORD $(0x6F3A2400 | ((Vn) << 5) | (Vd))
#define DUPB(Vd, Rn)       WORD $(0x0E010C00 | ((Rn) << 5) | (Vd))
#define DUPH(Vd, Rn)       WORD $(0x0E020C00 | ((Rn) << 5) | (Vd))

// The rounding shift carries the +32, so the tap is a widening multiply, a
// widening accumulate and one shift.
// func z2Top8NEON(dst *uint8, edge *uint8, n int, frac int32)
TEXT ·z2Top8NEON(SB), NOSPLIT, $0-28
	MOVD dst+0(FP), R0
	MOVD edge+8(FP), R1
	MOVD n+16(FP), R2
	MOVW frac+24(FP), R3

	MOVD $64, R4
	SUBW R3, R4, R4
	DUPB(6, 4)
	DUPB(7, 3)
	MOVD $0, R5

z2t8:
	ADD    R5, R1, R6
	FMOVD  (R6), F0
	ADD    $1, R6, R6
	FMOVD  (R6), F1
	UMULLH(2, 0, 6)
	UMLALH(2, 1, 7)
	URSHRH(2, 2)
	XTNB(2, 2)
	ADD    R5, R0, R7
	FMOVD  F2, (R7)
	ADD    $8, R5, R5
	CMP    R2, R5
	BLT    z2t8

	RET

// func z2Top16NEON(dst *uint16, edge *uint16, n int, frac int32)
TEXT ·z2Top16NEON(SB), NOSPLIT, $0-28
	MOVD dst+0(FP), R0
	MOVD edge+8(FP), R1
	MOVD n+16(FP), R2
	MOVW frac+24(FP), R3

	MOVD $64, R4
	SUBW R3, R4, R4
	DUPH(6, 4)
	DUPH(7, 3)
	MOVD $0, R5

z2t16:
	ADD    R5<<1, R1, R6
	FMOVD  (R6), F0
	ADD    $2, R6, R6
	FMOVD  (R6), F1
	UMULLS(2, 0, 6)
	UMLALS(2, 1, 7)
	URSHRS(2, 2)
	XTN4(2, 2)
	ADD    R5<<1, R0, R7
	FMOVD  F2, (R7)
	ADD    $4, R5, R5
	CMP    R2, R5
	BLT    z2t16

	RET

#define UADDLPH(Vd, Vn) WORD $(0x6E202800 | ((Vn) << 5) | (Vd))
#define UXTLH(Vd, Vn)   WORD $(0x2F08A400 | ((Vn) << 5) | (Vd))
#define SHLH(Vd, Vn, SH) WORD $(0x4F105400 | ((SH) << 16) | ((Vn) << 5) | (Vd))

// The subsampled sum is a pairwise add of adjacent bytes, which UADDLP is, and
// the shift after it is whatever the layout leaves over.
// func cflAcMain8NEON(ac *int16, acStride int, ypx *uint8, stride, n, rows, ssHor, ssVer int)
TEXT ·cflAcMain8NEON(SB), NOSPLIT, $0-64
	MOVD ac+0(FP), R0
	MOVD acStride+8(FP), R1
	MOVD ypx+16(FP), R2
	MOVD stride+24(FP), R3
	MOVD n+32(FP), R4
	MOVD rows+40(FP), R5
	MOVD ssHor+48(FP), R6
	MOVD ssVer+56(FP), R7

	LSL  $1, R1, R1
	CBZ  R6, ac444row
	CBZ  R7, ac422row

ac420row:
	MOVD $0, R8

ac420col:
	ADD    R8<<1, R2, R9
	VLD1   (R9), [V0.B16]
	UADDLPH(0, 0)
	ADD    R3, R9, R10
	VLD1   (R10), [V1.B16]
	UADDLPH(1, 1)
	VADD   V1.H8, V0.H8, V0.H8
	SHLH(0, 0, 1)
	ADD    R8<<1, R0, R11
	VST1   [V0.H8], (R11)
	ADD    $8, R8, R8
	CMP    R4, R8
	BLT    ac420col

	ADD  R1, R0, R0
	ADD  R3<<1, R2, R2
	SUBS $1, R5, R5
	BNE  ac420row

	RET

ac422row:
	MOVD $0, R8

ac422col:
	ADD    R8<<1, R2, R9
	VLD1   (R9), [V0.B16]
	UADDLPH(0, 0)
	SHLH(0, 0, 2)
	ADD    R8<<1, R0, R11
	VST1   [V0.H8], (R11)
	ADD    $8, R8, R8
	CMP    R4, R8
	BLT    ac422col

	ADD  R1, R0, R0
	ADD  R3, R2, R2
	SUBS $1, R5, R5
	BNE  ac422row

	RET

ac444row:
	MOVD $0, R8

ac444col:
	ADD    R8, R2, R9
	FMOVD  (R9), F0
	UXTLH(0, 0)
	SHLH(0, 0, 3)
	ADD    R8<<1, R0, R11
	VST1   [V0.H8], (R11)
	ADD    $8, R8, R8
	CMP    R4, R8
	BLT    ac444col

	ADD  R1, R0, R0
	ADD  R3, R2, R2
	SUBS $1, R5, R5
	BNE  ac444row

	RET
