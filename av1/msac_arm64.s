//go:build arm64 && !noasm

#include "textflag.h"

#define MSAC_POS 24
#define MSAC_DIF 32
#define MSAC_RNG 40
#define MSAC_CNT 48
#define MSAC_UPD 56



TEXT msacRenorm<>(SB), NOSPLIT|NOFRAME, $0
	MOVD  MSAC_CNT(R0), R4;
	CLZW  R13, R5;
	SUB   $16, R5, R5;
	LSLW  R5, R13, R13;
	LSL   R5, R9, R9;
	MOVW  R13, MSAC_RNG(R0);
	CMPW  R5, R4;
	SUB   R5, R4, R4;
	BHS   done;
	MOVD  MSAC_POS(R0), R6;
	MOVD  8(R0), R7;
	MOVD  0(R0), R10;
	MOVD  $40, R5;
	SUB   R4, R5, R5;
fill:
	CMP   R7, R6;
	BHS   pad;
	MOVBU (R10)(R6), R11;
	EOR   $0xff, R11, R11;
	LSL   R5, R11, R11;
	ORR   R11, R9, R9;
	ADD   $1, R6, R6;
	SUB   $8, R5, R5;
	CMP   $0, R5;
	BGE   fill;
	B     filled;
pad:
	MOVD  $-256, R11;
	LSL   R5, R11, R11;
	MVN   R11, R11;
	ORR   R11, R9, R9;
filled:
	MOVD  R6, MSAC_POS(R0);
	MOVD  $40, R4;
	SUB   R5, R4, R4;
done:
	MOVD  R4, MSAC_CNT(R0);
	MOVD  R9, MSAC_DIF(R0)
	RET

#define BOOLBODY                      \
	LSL   $48, R3, R10;           \
	CMP   R10, R9;                \
	BLO   below;                  \
	SUB   R10, R9, R9;            \
	SUB   R3, R8, R13;            \
	MOVD  $0, R8;                 \
	B     gotbit;                 \
below:                                \
	MOVD  R3, R13;                \
	MOVD  $1, R8;                 \
gotbit:

// func msacBoolEquiNEON(s *msacContext) uint32
TEXT ·msacBoolEquiNEON(SB), NOSPLIT, $16-12
	MOVD  s+0(FP), R0
	MOVWU MSAC_RNG(R0), R8
	MOVD  MSAC_DIF(R0), R9

	LSR  $8, R8, R3
	LSL  $7, R3, R3
	ADD  $4, R3, R3

	BOOLBODY
	BL   msacRenorm<>(SB)
	MOVW R8, ret+8(FP)
	RET

// func msacBoolFNEON(s *msacContext, f uint32) uint32
TEXT ·msacBoolFNEON(SB), NOSPLIT, $16-20
	MOVD  s+0(FP), R0
	MOVWU f+8(FP), R2
	MOVWU MSAC_RNG(R0), R8
	MOVD  MSAC_DIF(R0), R9

	LSR  $8, R8, R3
	LSR  $6, R2, R2
	MULW R2, R3, R3
	LSR  $1, R3, R3
	ADD  $4, R3, R3

	BOOLBODY
	BL   msacRenorm<>(SB)
	MOVW R8, ret+16(FP)
	RET

// func msacBoolAdaptNEON(s *msacContext, cdf *uint16) uint32
TEXT ·msacBoolAdaptNEON(SB), NOSPLIT, $16-20
	MOVD  s+0(FP), R0
	MOVD  cdf+8(FP), R1
	MOVWU MSAC_RNG(R0), R8
	MOVD  MSAC_DIF(R0), R9

	MOVHU (R1), R2
	LSR   $8, R8, R3
	LSR   $6, R2, R2
	MULW  R2, R3, R3
	LSR   $1, R3, R3
	ADD   $4, R3, R3

	BOOLBODY

	MOVBU MSAC_UPD(R0), R5
	CBZ   R5, noupd

	MOVHU 2(R1), R7
	LSR   $4, R7, R5
	ADD   $4, R5, R5
	MOVHU (R1), R6
	CBZ   R8, down

	MOVD  $32768, R11
	SUB   R6, R11, R11
	LSRW  R5, R11, R11
	ADD   R11, R6, R6
	B     stored

down:
	LSRW  R5, R6, R11
	SUB   R11, R6, R6

stored:
	MOVH  R6, (R1)
	CMP   $32, R7
	BHS   nocount
	ADD   $1, R7, R7

nocount:
	MOVH  R7, 2(R1)

noupd:
	BL   msacRenorm<>(SB)
	MOVW R8, ret+16(FP)
	RET

#define UMULL(Vd, Vn, Vm)   WORD $(0x2E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define UMULL2(Vd, Vn, Vm)  WORD $(0x6E60C000 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SHRNH(Vd, Vn, SH)   WORD $(0x0F008400 | ((32 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define SHRNH2(Vd, Vn, SH)  WORD $(0x4F008400 | ((32 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define SHRNB(Vd, Vn, SH)   WORD $(0x0F008400 | ((16 - (SH)) << 16) | ((Vn) << 5) | (Vd))
#define UQSUBH(Vd, Vn, Vm)  WORD $(0x6E602C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define URHADDH(Vd, Vn, Vm) WORD $(0x6E601400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHLH(Vd, Vn, Vm)   WORD $(0x4E604400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define BICV(Vd, Vn, Vm)    WORD $(0x4E601C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define CMEQZH(Vd, Vn)      WORD $(0x4E609800 | ((Vn) << 5) | (Vd))

DATA msacMinProb<>+0(SB)/8, $0x003000340038003c
DATA msacMinProb<>+8(SB)/8, $0x002000240028002c
DATA msacMinProb<>+16(SB)/8, $0x001000140018001c
DATA msacMinProb<>+24(SB)/8, $0x000000040008000c
DATA msacMinProb<>+32(SB)/8, $0xff00ff00ff00ff00
DATA msacMinProb<>+40(SB)/8, $0xff00ff00ff00ff00
GLOBL msacMinProb<>(SB), RODATA|NOPTR, $48

#define LDCDF4 VLD1 (R1), [V0.H4]
#define LDCDF8 VLD1 (R1), [V0.H8]
#define STCDF4 VST1 [V0.H4], (R1)
#define STCDF8 VST1 [V0.H8], (R1)

#define SYMBODY(LD, ST) \
	MOVWU MSAC_RNG(R0), R8; \
	MOVD  MSAC_DIF(R0), R9; \
	LSR   $48, R9, R11; \
	LSR  $8, R8, R12; \
	VDUP R12, V1.H8; \
	LD; \
	VUSHR $6, V0.H8, V2.H8; \
	UMULL(3, 2, 1); \
	UMULL2(4, 2, 1); \
	SHRNH(5, 3, 1); \
	SHRNH2(5, 4, 1); \
	MOVD $msacMinProb<>(SB), R13; \
	MOVD $15, R14; \
	SUB  R2, R14, R14; \
	ADD  R14<<1, R13, R13; \
	VLD1 (R13), [V8.H8]; \
	VADD V8.H8, V5.H8, V5.H8; \
	MOVD RSP, R15; \
	MOVH R8, 14(R15); \
	ADD  $16, R15, R16; \
	VST1 [V5.H8], (R16); \
	VDUP R11, V6.H8; \
	UQSUBH(7, 5, 6); \
	CMEQZH(7, 7); \
	SHRNB(9, 7, 4); \
	VMOV V9.D[0], R14; \
	RBIT R14, R14; \
	CLZ  R14, R14; \
	LSR  $3, R14, R14; \
	ADD   R14<<1, R15, R16; \
	MOVHU 16(R16), R12; \
	MOVHU 14(R16), R13; \
	SUB   R12, R13, R13; \
	LSL   $48, R12, R10; \
	SUB   R10, R9, R9; \
	MOVD  R14, R8; \
	MOVBU MSAC_UPD(R0), R5; \
	CBZ   R5, noupd; \
	LSL   $1, R2, R6; \
	MOVHU (R1)(R6), R7; \
	LSR   $4, R7, R5; \
	ADD   $4, R5, R5; \
	CMP   $3, R2; \
	BLO   rateok; \
	ADD   $1, R5, R5; \
rateok:; \
	NEG  R5, R5; \
	VDUP R5, V10.H8; \
	VMOVI $0xff, V11.B16; \
	URHADDH(11, 11, 7); \
	VSUB V0.H8, V11.H8, V11.H8; \
	SSHLH(11, 11, 10); \
	VSUB V7.H8, V11.H8, V11.H8; \
	MOVD  $0xff00, R5; \
	VDUP  R5, V12.H8; \
	VCMEQ V12.H8, V8.H8, V12.H8; \
	BICV(11, 11, 12); \
	VADD  V11.H8, V0.H8, V0.H8; \
	ST; \
	CMP  $32, R7; \
	BHS  nocount; \
	ADD  $1, R7, R7; \
nocount:; \
	MOVH R7, (R1)(R6); \
noupd:; \
	BL   msacRenorm<>(SB); \
	MOVW R8, ret+24(FP)

// func msacSymbolAdapt4NEON(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdapt4NEON(SB), NOSPLIT, $64-28
	MOVD s+0(FP), R0
	MOVD cdf+8(FP), R1
	MOVD n+16(FP), R2
	SYMBODY(LDCDF4, STCDF4)
	BL   msacRenorm<>(SB)
	MOVW R8, ret+24(FP)
	RET

// func msacSymbolAdapt8NEON(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdapt8NEON(SB), NOSPLIT, $64-28
	MOVD s+0(FP), R0
	MOVD cdf+8(FP), R1
	MOVD n+16(FP), R2
	SYMBODY(LDCDF8, STCDF8)
	BL   msacRenorm<>(SB)
	MOVW R8, ret+24(FP)
	RET
