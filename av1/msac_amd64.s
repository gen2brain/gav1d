//go:build amd64 && !noasm

#include "textflag.h"

#define MSAC_POS 24
#define MSAC_DIF 32
#define MSAC_RNG 40
#define MSAC_CNT 48
#define MSAC_UPD 56

DATA msacMinProb<>+0(SB)/8, $0x003000340038003c
DATA msacMinProb<>+8(SB)/8, $0x002000240028002c
DATA msacMinProb<>+16(SB)/8, $0x001000140018001c
DATA msacMinProb<>+24(SB)/8, $0x000000040008000c
DATA msacMinProb<>+32(SB)/8, $0xff00ff00ff00ff00
DATA msacMinProb<>+40(SB)/8, $0xff00ff00ff00ff00
DATA msacMinProb<>+48(SB)/8, $0xff00ff00ff00ff00
DATA msacMinProb<>+56(SB)/8, $0xff00ff00ff00ff00
GLOBL msacMinProb<>(SB), RODATA|NOPTR, $64

DATA msacFF00<>+0(SB)/8, $0xff00ff00ff00ff00
DATA msacFF00<>+8(SB)/8, $0xff00ff00ff00ff00
DATA msacFF00<>+16(SB)/8, $0xff00ff00ff00ff00
DATA msacFF00<>+24(SB)/8, $0xff00ff00ff00ff00
GLOBL msacFF00<>(SB), RODATA|NOPTR, $32

#define LOAD4 MOVQ (SI), X0
#define LOAD8 MOVOU (SI), X0
#define STORE4 MOVQ X0, (SI)
#define STORE8 MOVOU X0, (SI)

#define SEARCH(LOAD)                 \
	MOVL MSAC_RNG(DI), R8;       \
	MOVQ MSAC_DIF(DI), R9;       \
	MOVQ R9, R11;                \
	SHRQ $48, R11;               \
	MOVD R8, X2;                 \
	PSHUFLW $0, X2, X2;          \
	PSHUFD $0, X2, X2;           \
	PAND msacFF00<>(SB), X2;     \
	LOAD;                        \
	MOVOU X0, X1;                \
	PSRLW $6, X1;                \
	PSLLW $7, X1;                \
	PMULHUW X2, X1;              \
	LEAQ msacMinProb<>(SB), BX;  \
	MOVL $15, AX;                \
	SUBL DX, AX;                 \
	MOVOU (BX)(AX*2), X3;        \
	MOVOU X3, X4;                \
	PADDW X3, X1;                \
	MOVW R8, 6(SP);              \
	MOVOU X1, 8(SP);             \
	MOVD R11, X3;                \
	PSHUFLW $0, X3, X3;          \
	PSHUFD $0, X3, X3;           \
	MOVOU X1, X6;                \
	PSUBUSW X3, X6;              \
	PXOR X7, X7;                 \
	PCMPEQW X7, X6;              \
	PMOVMSKB X6, AX;             \
	BSFL AX, AX;                 \
	MOVWLZX 8(SP)(AX*1), R12;    \
	MOVWLZX 6(SP)(AX*1), R13;    \
	SHRL $1, AX;                 \
	MOVL AX, R8;                 \
	SUBL R12, R13;               \
	MOVQ R12, R10;               \
	SHLQ $48, R10;               \
	SUBQ R10, R9

#define UPDATE(STORE)                \
	CMPB MSAC_UPD(DI), $0;       \
	JE   noupd;                  \
	MOVWLZX (SI)(DX*2), R10;     \
	MOVL R10, R11;               \
	SHRL $4, R11;                \
	CMPL DX, $3;                 \
	SBBL $-5, R11;               \
	CMPL R10, $32;               \
	ADCL $0, R10;                \
	MOVD R11, X3;                \
	PCMPEQW X2, X2;              \
	PAVGW X6, X2;                \
	PSUBW X0, X2;                \
	PSRAW X3, X2;                \
	PSUBW X6, X2;                \
	PCMPEQW msacFF00<>(SB), X4;  \
	PANDN X2, X4;                \
	PADDW X4, X0;                \
	STORE;                       \
	MOVW R10, (SI)(DX*2);        \
noupd:

#define RENORM                       \
	MOVL MSAC_CNT(DI), BX;       \
	BSRL R13, CX;                \
	XORL $15, CX;                \
	SHLL CX, R13;                \
	SHLQ CX, R9;                 \
	MOVL R13, MSAC_RNG(DI);      \
	SUBL CX, BX;                 \
	JAE  done;                   \
	MOVQ MSAC_POS(DI), R10;      \
	MOVQ 8(DI), R11;             \
	MOVQ 0(DI), R12;             \
	MOVL $40, CX;                \
	SUBL BX, CX;                 \
fill:                                \
	CMPQ R10, R11;               \
	JAE  pad;                    \
	MOVBLZX (R12)(R10*1), AX;    \
	XORL $0xff, AX;              \
	MOVQ AX, DX;                 \
	SHLQ CX, DX;                 \
	ORQ  DX, R9;                 \
	INCQ R10;                    \
	SUBL $8, CX;                 \
	JGE  fill;                   \
	JMP  filled;                 \
pad:                                 \
	MOVQ $-256, DX;              \
	SHLQ CX, DX;                 \
	NOTQ DX;                     \
	ORQ  DX, R9;                 \
filled:                              \
	MOVQ R10, MSAC_POS(DI);      \
	MOVL $40, BX;                \
	SUBL CX, BX;                 \
done:                                \
	MOVLQSX BX, AX;              \
	MOVQ AX, MSAC_CNT(DI);       \
	MOVQ R9, MSAC_DIF(DI)

// func msacSymbolAdapt4SSE(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdapt4SSE(SB), NOSPLIT, $64-28
	MOVQ s+0(FP), DI
	MOVQ cdf+8(FP), SI
	MOVQ n+16(FP), DX

	SEARCH(LOAD4)
	UPDATE(STORE4)
	RENORM
	MOVL R8, ret+24(FP)
	RET

// func msacSymbolAdapt8SSE(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdapt8SSE(SB), NOSPLIT, $64-28
	MOVQ s+0(FP), DI
	MOVQ cdf+8(FP), SI
	MOVQ n+16(FP), DX

	SEARCH(LOAD8)
	UPDATE(STORE8)
	RENORM
	MOVL R8, ret+24(FP)
	RET

// func msacSymbolAdapt16AVX2(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdapt16AVX2(SB), NOSPLIT, $64-28
	MOVQ s+0(FP), DI
	MOVQ cdf+8(FP), SI
	MOVQ n+16(FP), DX

	MOVL MSAC_RNG(DI), R8
	MOVQ MSAC_DIF(DI), R9
	MOVQ R9, R11
	SHRQ $48, R11

	VMOVD        R8, X2
	VPBROADCASTW X2, Y2
	VPAND        msacFF00<>(SB), Y2, Y2

	VMOVDQU (SI), Y0
	VPSRLW  $6, Y0, Y1
	VPSLLW  $7, Y1, Y1
	VPMULHUW Y2, Y1, Y1

	LEAQ    msacMinProb<>(SB), BX
	MOVL    $15, AX
	SUBL    DX, AX
	VMOVDQU (BX)(AX*2), Y3
	VMOVDQU Y3, Y4
	VPADDW  Y3, Y1, Y1

	MOVW    R8, 6(SP)
	VMOVDQU Y1, 8(SP)

	VMOVD        R11, X3
	VPBROADCASTW X3, Y3
	VPSUBUSW     Y3, Y1, Y6
	VPXOR        Y7, Y7, Y7
	VPCMPEQW     Y7, Y6, Y6
	VPMOVMSKB    Y6, AX
	BSFL         AX, AX

	MOVWLZX 8(SP)(AX*1), R12
	MOVWLZX 6(SP)(AX*1), R13
	SHRL    $1, AX
	MOVL    AX, R8
	SUBL    R12, R13
	MOVQ    R12, R10
	SHLQ    $48, R10
	SUBQ    R10, R9

	CMPB MSAC_UPD(DI), $0
	JE   noupd

	MOVWLZX (SI)(DX*2), R10
	MOVL    R10, R11
	SHRL    $4, R11
	CMPL    DX, $3
	SBBL    $-5, R11
	CMPL    R10, $32
	ADCL    $0, R10
	VMOVD    R11, X3
	VPCMPEQW Y2, Y2, Y2
	VPAVGW   Y6, Y2, Y2
	VPSUBW   Y0, Y2, Y2
	VPSRAW   X3, Y2, Y2
	VPSUBW   Y6, Y2, Y2
	VPCMPEQW msacFF00<>(SB), Y4, Y4
	VPANDN   Y2, Y4, Y4
	VPADDW   Y4, Y0, Y0
	VMOVDQU  Y0, (SI)
	MOVW     R10, (SI)(DX*2)

noupd:
	VZEROUPPER
	RENORM
	MOVL R8, ret+24(FP)
	RET

#define BOOLBODY                     \
	MOVQ AX, R10;                \
	SHLQ $48, R10;               \
	XORL R11, R11;               \
	CMPQ R9, R10;                \
	JB   below;                  \
	MOVL $1, R11;                \
	SUBQ R10, R9;                \
	MOVL R8, R13;                \
	SUBL AX, R13;                \
	JMP  gotbit;                 \
below:                               \
	MOVL AX, R13;                \
gotbit:                              \
	MOVL R11, R8;                \
	XORL $1, R8

// func msacBoolEquiSSE(s *msacContext) uint32
TEXT ·msacBoolEquiSSE(SB), NOSPLIT, $0-12
	MOVQ s+0(FP), DI
	MOVL MSAC_RNG(DI), R8
	MOVQ MSAC_DIF(DI), R9

	MOVL R8, AX
	SHRL $8, AX
	SHLL $7, AX
	ADDL $4, AX

	BOOLBODY
	RENORM
	MOVL R8, ret+8(FP)
	RET

// func msacBoolFSSE(s *msacContext, f uint32) uint32
TEXT ·msacBoolFSSE(SB), NOSPLIT, $0-20
	MOVQ s+0(FP), DI
	MOVL f+8(FP), DX
	MOVL MSAC_RNG(DI), R8
	MOVQ MSAC_DIF(DI), R9

	MOVL  R8, AX
	SHRL  $8, AX
	SHRL  $6, DX
	IMULL DX, AX
	SHRL  $1, AX
	ADDL  $4, AX

	BOOLBODY
	RENORM
	MOVL R8, ret+16(FP)
	RET

// func msacBoolAdaptSSE(s *msacContext, cdf *uint16) uint32
TEXT ·msacBoolAdaptSSE(SB), NOSPLIT, $0-20
	MOVQ s+0(FP), DI
	MOVQ cdf+8(FP), SI
	MOVL MSAC_RNG(DI), R8
	MOVQ MSAC_DIF(DI), R9

	MOVWLZX (SI), DX
	MOVL  R8, AX
	SHRL  $8, AX
	SHRL  $6, DX
	IMULL DX, AX
	SHRL  $1, AX
	ADDL  $4, AX

	BOOLBODY

	CMPB MSAC_UPD(DI), $0
	JE   noupd

	MOVWLZX 2(SI), R10
	MOVL    R10, R11
	SHRL    $4, R11
	ADDL    $4, R11
	MOVWLZX (SI), R12
	MOVL    R11, CX
	TESTL   R8, R8
	JZ      down
	MOVL $32768, AX
	SUBL R12, AX
	SHRL CX, AX
	ADDL AX, R12
	JMP  stored

down:
	MOVL R12, AX
	SHRL CX, AX
	SUBL AX, R12

stored:
	MOVW R12, (SI)
	CMPL R10, $32
	ADCL $0, R10
	MOVW R10, 2(SI)

noupd:
	RENORM
	MOVL R8, ret+16(FP)
	RET
