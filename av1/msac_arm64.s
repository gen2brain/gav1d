//go:build arm64 && !noasm

#include "textflag.h"

#define MSAC_POS 24
#define MSAC_DIF 32
#define MSAC_RNG 40
#define MSAC_CNT 48
#define MSAC_UPD 56



#define RENORM                        \
	MOVD  MSAC_CNT(R0), R4;       \
	CLZW  R13, R5;                \
	SUB   $16, R5, R5;            \
	LSLW  R5, R13, R13;           \
	LSL   R5, R9, R9;             \
	MOVW  R13, MSAC_RNG(R0);      \
	CMPW  R5, R4;                 \
	SUB   R5, R4, R4;             \
	BHS   done;                   \
	MOVD  MSAC_POS(R0), R6;       \
	MOVD  8(R0), R7;              \
	MOVD  0(R0), R10;             \
	MOVD  $40, R5;                \
	SUB   R4, R5, R5;             \
fill:                                 \
	CMP   R7, R6;                 \
	BHS   pad;                    \
	MOVBU (R10)(R6), R11;         \
	EOR   $0xff, R11, R11;        \
	LSL   R5, R11, R11;           \
	ORR   R11, R9, R9;            \
	ADD   $1, R6, R6;             \
	SUB   $8, R5, R5;             \
	CMP   $0, R5;                 \
	BGE   fill;                   \
	B     filled;                 \
pad:                                  \
	MOVD  $-256, R11;             \
	LSL   R5, R11, R11;           \
	MVN   R11, R11;               \
	ORR   R11, R9, R9;            \
filled:                               \
	MOVD  R6, MSAC_POS(R0);       \
	MOVD  $40, R4;                \
	SUB   R5, R4, R4;             \
done:                                 \
	MOVD  R4, MSAC_CNT(R0);       \
	MOVD  R9, MSAC_DIF(R0)

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
TEXT ·msacBoolEquiNEON(SB), NOSPLIT, $0-12
	MOVD  s+0(FP), R0
	MOVWU MSAC_RNG(R0), R8
	MOVD  MSAC_DIF(R0), R9

	LSR  $8, R8, R3
	LSL  $7, R3, R3
	ADD  $4, R3, R3

	BOOLBODY
	RENORM
	MOVW R8, ret+8(FP)
	RET

// func msacBoolFNEON(s *msacContext, f uint32) uint32
TEXT ·msacBoolFNEON(SB), NOSPLIT, $0-20
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
	RENORM
	MOVW R8, ret+16(FP)
	RET

// func msacBoolAdaptNEON(s *msacContext, cdf *uint16) uint32
TEXT ·msacBoolAdaptNEON(SB), NOSPLIT, $0-20
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
	RENORM
	MOVW R8, ret+16(FP)
	RET
