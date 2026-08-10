//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no vector multiply and no saturating narrow, so both
// are encoded by hand.
#define SQRDMULH(Vd, Vn, Vm) WORD $(0x6E60B400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SQXTUN(Vd, Vn)       WORD $(0x2E212800 | ((Vn) << 5) | (Vd))

// LOOKUP loads scaling[src[I]] shifted up by lshift; NEON has no gather.
#define LOOKUP(I)             \
	MOVBU I(R1), R8;      \
	MOVBU (R2)(R8), R9;   \
	LSL   R5, R9, R9;     \
	VMOV  R9, V1.H[I]

// func fgApplyRowNEON(dst, src, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int)
//
// Pre-shifting the scaling value turns the rounding shift into SQRDMULH, which
// cannot overflow a halfword.
TEXT ·fgApplyRowNEON(SB), NOSPLIT, $0-64
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD scaling+16(FP), R2
	MOVD grain+24(FP), R3
	MOVD n+32(FP), R4
	MOVD lshift+40(FP), R5
	MOVD minValue+48(FP), R6
	MOVD maxValue+56(FP), R7

	VDUP R6, V6.B16
	VDUP R7, V7.B16

loop:
	LOOKUP(0)
	LOOKUP(1)
	LOOKUP(2)
	LOOKUP(3)
	LOOKUP(4)
	LOOKUP(5)
	LOOKUP(6)
	LOOKUP(7)

	VLD1   (R1), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VLD1   (R3), [V2.H8]

	SQRDMULH(2, 2, 1)
	VADD V0.H8, V2.H8, V2.H8
	SQXTUN(2, 2)

	VUMAX V6.B8, V2.B8, V2.B8
	VUMIN V7.B8, V2.B8, V2.B8
	VST1  [V2.B8], (R0)

	ADD  $8, R0
	ADD  $8, R1
	ADD  $16, R3
	SUB  $8, R4
	CMP  $8, R4
	BGE  loop

	CBZ R4, done

	// Overlap the block just done; every pixel is independent and dst is
	// never the source.
	MOVD $8, R10
	SUB  R4, R10, R10
	SUB  R10, R0
	SUB  R10, R1
	LSL  $1, R10, R11
	SUB  R11, R3
	MOVD $8, R4
	B    loop

done:
	RET

// UVLOOKUP builds the chroma scaling index for lane I. The gather is scalar
// anyway, so the index is too, which avoids a widening multiply.
#define UVLOOKUP(I)            \
	MOVD  $I, R13;         \
	LSL   R7, R13, R13;    \
	MOVBU (R2)(R13), R14;  \
	ADD   R7, R13, R15;    \
	MOVBU (R2)(R15), R15;  \
	MUL   R7, R15, R15;    \
	ADD   R15, R14, R14;   \
	ADD   R7, R14, R14;    \
	LSR   R7, R14, R14;    \
	MOVBU I(R1), R15;      \
	MUL   R9, R14, R13;    \
	MADD  R10, R13, R15, R13; \
	ASR   $6, R13, R13;    \
	ADD   R11, R13, R13;   \
	CMP   ZR, R13;         \
	CSEL  LT, ZR, R13, R13; \
	CMP   R12, R13;        \
	CSEL  GT, R12, R13, R13; \
	CMP   ZR, R8;          \
	CSEL  NE, R14, R13, R13; \
	MOVBU (R3)(R13), R15;  \
	LSL   R5, R15, R15;    \
	VMOV  R15, V1.H[I]

// func fguvApplyRowNEON(dst, src, luma, scaling *uint8, grain *int16, n, lshift, minValue, maxValue int, p *fguvParams)
TEXT ·fguvApplyRowNEON(SB), NOSPLIT, $0-80
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD luma+16(FP), R2
	MOVD scaling+24(FP), R3
	MOVD grain+32(FP), R4
	MOVD lshift+48(FP), R5
	MOVD n+40(FP), R6

	MOVD minValue+56(FP), R13
	VDUP R13, V6.B16
	MOVD maxValue+64(FP), R13
	VDUP R13, V7.B16

	MOVD p+72(FP), R13
	MOVH 0(R13), R9   // lumaMult
	MOVH 2(R13), R10  // mult
	MOVH 4(R13), R12  // pixelMax
	MOVW 8(R13), R11  // offset
	MOVW 12(R13), R7  // sx
	MOVW 16(R13), R8  // chromaScalingFromLuma

uvloop:
	UVLOOKUP(0)
	UVLOOKUP(1)
	UVLOOKUP(2)
	UVLOOKUP(3)
	UVLOOKUP(4)
	UVLOOKUP(5)
	UVLOOKUP(6)
	UVLOOKUP(7)

	VLD1   (R1), [V0.B8]
	VUSHLL $0, V0.B8, V0.H8
	VLD1   (R4), [V2.H8]

	SQRDMULH(2, 2, 1)
	VADD V0.H8, V2.H8, V2.H8
	SQXTUN(2, 2)

	VUMAX V6.B8, V2.B8, V2.B8
	VUMIN V7.B8, V2.B8, V2.B8
	VST1  [V2.B8], (R0)

	ADD  $8, R0
	ADD  $8, R1
	MOVD $8, R13
	LSL  R7, R13, R13
	ADD  R13, R2
	ADD  $16, R4
	SUB  $8, R6
	CMP  $8, R6
	BGE  uvloop

	CBZ R6, uvdone

	MOVD $8, R13
	SUB  R6, R13, R13
	SUB  R13, R0
	SUB  R13, R1
	MOVD R13, R14
	LSL  R7, R14, R14
	SUB  R14, R2
	LSL  $1, R13, R14
	SUB  R14, R4
	MOVD $8, R6
	B    uvloop

uvdone:
	RET

#define SSHLS(Vd, Vn, Vm)  WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS4(Vd, Vn, Vm) WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS4(Vd, Vn, Vm) WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define MULS4(Vd, Vn, Vm)  WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLL4(Vd, Vn)     WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHR6(Vd, Vn)      WORD $(0x4F000400 | ((64 - 6) << 16) | ((Vn) << 5) | (Vd))
#define XTN4(Vd, Vn)       WORD $(0x0E612800 | ((Vn) << 5) | (Vd))

// GATHER4 reads four scaling entries, which NEON has no instruction for.
#define GATHER4                \
	VMOV  V2.S[0], R11;    \
	MOVBU (R3)(R11), R12;  \
	VMOV  R12, V4.S[0];    \
	VMOV  V2.S[1], R11;    \
	MOVBU (R3)(R11), R12;  \
	VMOV  R12, V4.S[1];    \
	VMOV  V2.S[2], R11;    \
	MOVBU (R3)(R11), R12;  \
	VMOV  R12, V4.S[2];    \
	VMOV  V2.S[3], R11;    \
	MOVBU (R3)(R11), R12;  \
	VMOV  R12, V4.S[3]

// NOISE16 turns the gathered scaling and the grain into the output sample.
#define NOISE16                    \
	VLD1  (R4), [V5.H4];       \
	SSHLL4(5, 5);              \
	MULS4(4, 4, 5);            \
	VADD  V25.S4, V4.S4, V4.S4; \
	SSHLS(4, 4, 24);           \
	VADD  V1.S4, V4.S4, V4.S4; \
	SMAXS4(4, 4, 26);          \
	SMINS4(4, 4, 27);          \
	XTN4(4, 4);                \
	VST1  [V4.D1], (R0);       \
	ADD   $8, R0, R0;          \
	ADD   $8, R1, R1;          \
	ADD   $8, R4, R4;          \
	SUBS  $4, R2, R2

// func fgApplyRow16NEON(dst, src *uint16, scaling *uint8, grain *int16, n, shift, minValue, maxValue int)
TEXT ·fgApplyRow16NEON(SB), NOSPLIT, $0-64
	MOVD  dst+0(FP), R0
	MOVD  src+8(FP), R1
	MOVD  scaling+16(FP), R3
	MOVD  grain+24(FP), R4
	MOVD  n+32(FP), R2
	MOVD  shift+40(FP), R5
	MOVD  $1, R6
	LSL   R5, R6, R6
	LSR   $1, R6, R6
	VDUP  R6, V25.S4
	NEG   R5, R5
	VDUP  R5, V24.S4
	MOVD  minValue+48(FP), R6
	VDUP  R6, V26.S4
	MOVD  maxValue+56(FP), R6
	VDUP  R6, V27.S4

fgrow16:
	VLD1   (R1), [V1.H4]
	VUSHLL $0, V1.H4, V1.S4
	VMOV   V1.B16, V2.B16
	GATHER4
	NOISE16
	BNE    fgrow16

	RET

// func fguvApplyRow16NEON(dst, src, luma *uint16, scaling *uint8, grain *int16, p *fguv16Params)
TEXT ·fguvApplyRow16NEON(SB), NOSPLIT, $0-48
	MOVD  dst+0(FP), R0
	MOVD  src+8(FP), R1
	MOVD  luma+16(FP), R7
	MOVD  scaling+24(FP), R3
	MOVD  grain+32(FP), R4
	MOVD  p+40(FP), R8
	MOVD  0(R8), R2
	MOVD  8(R8), R5
	MOVD  $1, R6
	LSL   R5, R6, R6
	LSR   $1, R6, R6
	VDUP  R6, V25.S4
	NEG   R5, R5
	VDUP  R5, V24.S4
	MOVD  16(R8), R6
	VDUP  R6, V26.S4
	MOVD  24(R8), R6
	VDUP  R6, V27.S4
	MOVD  32(R8), R9
	MOVD  40(R8), R6
	VDUP  R6, V28.S4
	MOVD  48(R8), R6
	VDUP  R6, V29.S4
	MOVD  56(R8), R6
	VDUP  R6, V30.S4
	MOVD  64(R8), R6
	VDUP  R6, V31.S4
	MOVD  72(R8), R10
	VMOVI $0, V23.B16
	MOVD  $1, R6
	NEG   R6, R6
	VDUP  R6, V22.S4

fguvrow16:
	CBZ    R10, nosubx
	VLD1   (R7), [V6.H8]
	VUZP1  V6.H8, V6.H8, V7.H8
	VUZP2  V6.H8, V6.H8, V8.H8
	VUSHLL $0, V7.H4, V7.S4
	VUSHLL $0, V8.H4, V8.S4
	VADD   V8.S4, V7.S4, V2.S4
	VSUB   V22.S4, V2.S4, V2.S4
	VUSHR  $1, V2.S4, V2.S4
	B      havelum

nosubx:
	VLD1   (R7), [V2.H4]
	VUSHLL $0, V2.H4, V2.S4

havelum:
	VLD1   (R1), [V1.H4]
	VUSHLL $0, V1.H4, V1.S4
	CBNZ   R9, idxdone
	MULS4(2, 2, 28)
	MULS4(10, 1, 29)
	VADD   V10.S4, V2.S4, V2.S4
	SSHR6(2, 2)
	VADD   V30.S4, V2.S4, V2.S4
	SMAXS4(2, 2, 23)
	SMINS4(2, 2, 31)

idxdone:
	GATHER4
	NOISE16
	ADD    $8, R7, R7
	CBZ    R10, nostep
	ADD    $8, R7, R7

nostep:
	BNE    fguvrow16

	RET

#define ARMULS(Vd, Vn, Vm) WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define ARSSHLL(Vd, Vn)    WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))

// func grainARNEON(pre *int32, row *int16, rowStride, lag int, coeffs *int8, n int)
TEXT ·grainARNEON(SB), NOSPLIT, $0-48
	MOVD pre+0(FP), R0
	MOVD row+8(FP), R1
	MOVD rowStride+16(FP), R2
	MOVD lag+24(FP), R3
	MOVD coeffs+32(FP), R4
	MOVD n+40(FP), R5

	LSL  $1, R2, R2
	MOVD R3, R6
	LSL  $1, R3, R7
	ADD  $1, R7, R7

argrow:
	MOVD R1, R8
	MOVD R7, R9

argtap:
	MOVB (R4), R10
	VDUP R10, V1.S4
	ADD  $1, R4, R4

	MOVD R8, R11
	MOVD R0, R12
	MOVD R5, R13

argcol:
	VLD1 (R11), [V0.H4]
	ARSSHLL(0, 0)
	ARMULS(0, 0, 1)
	VLD1 (R12), [V2.S4]
	VADD V2.S4, V0.S4, V0.S4
	VST1 [V0.S4], (R12)

	ADD  $8, R11
	ADD  $16, R12
	SUB  $4, R13
	CBNZ R13, argcol

	ADD  $2, R8
	SUB  $1, R9
	CBNZ R9, argtap

	ADD  R2, R1
	SUB  $1, R6
	CBNZ R6, argrow

	RET
