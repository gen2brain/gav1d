//go:build riscv64 && riscv64.rva23u64 && !noasm

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
GLOBL msacMinProb<>(SB), RODATA|NOPTR, $48

#define RENORM                        \
	MOV   MSAC_CNT(X10), X6;      \
	CLZW  X13, X7;                \
	ADD   $-16, X7, X7;           \
	SLL   X7, X13, X13;           \
	SLLI  $32, X13, X13;          \
	SRLI  $32, X13, X13;          \
	SLL   X7, X9, X9;             \
	MOVW  X13, MSAC_RNG(X10);     \
	SUB   X7, X6, X28;            \
	SLLI  $32, X6, X29;           \
	SRLI  $32, X29, X29;          \
	SLLI  $32, X7, X30;           \
	SRLI  $32, X30, X30;          \
	BGEU  X29, X30, skiprefill;   \
	JMP   dorefill;               \
skiprefill:                           \
	JMP   done;                   \
dorefill:                             \
	MOV   MSAC_POS(X10), X11;     \
	MOV   8(X10), X12;            \
	MOV   0(X10), X14;            \
	MOV   $40, X7;                \
	SUB   X28, X7, X7;            \
fill:                                 \
	BGEU  X11, X12, pad;          \
	ADD   X14, X11, X15;          \
	MOVBU (X15), X16;             \
	XOR   $0xff, X16, X16;        \
	SLL   X7, X16, X16;           \
	OR    X16, X9, X9;            \
	ADD   $1, X11, X11;           \
	ADD   $-8, X7, X7;            \
	BGE   X7, ZERO, fill;         \
	JMP   filled;                 \
pad:                                  \
	MOV   $-256, X16;             \
	SLL   X7, X16, X16;           \
	NOT   X16, X16;               \
	OR    X16, X9, X9;            \
filled:                               \
	MOV   X11, MSAC_POS(X10);     \
	MOV   $40, X28;               \
	SUB   X7, X28, X28;           \
done:                                 \
	MOV   X28, MSAC_CNT(X10);     \
	MOV   X9, MSAC_DIF(X10)

#define BOOLBODY                      \
	SLLI  $48, X5, X17;           \
	BLTU  X9, X17, below;         \
	SUB   X17, X9, X9;            \
	SUB   X5, X8, X13;            \
	MOV   $0, X8;                 \
	JMP   gotbit;                 \
below:                                \
	MOV   X5, X13;                \
	MOV   $1, X8;                 \
gotbit:

// func msacBoolEquiRVV(s *msacContext) uint32
TEXT ·msacBoolEquiRVV(SB), NOSPLIT, $0-12
	MOV   s+0(FP), X10
	MOVWU MSAC_RNG(X10), X8
	MOV   MSAC_DIF(X10), X9

	SRLI $8, X8, X5
	SLLI $7, X5, X5
	ADD  $4, X5, X5

	BOOLBODY
	RENORM
	MOVW X8, ret+8(FP)
	RET

// func msacBoolFRVV(s *msacContext, f uint32) uint32
TEXT ·msacBoolFRVV(SB), NOSPLIT, $0-20
	MOV   s+0(FP), X10
	MOVWU f+8(FP), X18
	MOVWU MSAC_RNG(X10), X8
	MOV   MSAC_DIF(X10), X9

	SRLI $8, X8, X5
	SRLI $6, X18, X18
	MUL  X18, X5, X5
	SRLI $1, X5, X5
	ADD  $4, X5, X5

	BOOLBODY
	RENORM
	MOVW X8, ret+16(FP)
	RET

// func msacBoolAdaptRVV(s *msacContext, cdf *uint16) uint32
TEXT ·msacBoolAdaptRVV(SB), NOSPLIT, $0-20
	MOV   s+0(FP), X10
	MOV   cdf+8(FP), X19
	MOVWU MSAC_RNG(X10), X8
	MOV   MSAC_DIF(X10), X9

	MOVHU (X19), X18
	SRLI  $8, X8, X5
	SRLI  $6, X18, X18
	MUL   X18, X5, X5
	SRLI  $1, X5, X5
	ADD   $4, X5, X5

	BOOLBODY

	MOVBU MSAC_UPD(X10), X20
	BEQZ  X20, noupd

	MOVHU 2(X19), X21
	SRLI  $4, X21, X20
	ADD   $4, X20, X20
	MOVHU (X19), X22
	BEQZ  X8, down

	MOV  $32768, X23
	SUB  X22, X23, X23
	SRL  X20, X23, X23
	ADD  X23, X22, X22
	JMP  stored

down:
	SRL  X20, X22, X23
	SUB  X23, X22, X22

stored:
	MOVH X22, (X19)
	MOV  $32, X23
	BGEU X21, X23, nocount
	ADD  $1, X21, X21

nocount:
	MOVH X21, 2(X19)

noupd:
	RENORM
	MOVW X8, ret+16(FP)
	RET

// func msacSymbolAdaptRVV(s *msacContext, cdf *uint16, n int) uint32
TEXT ·msacSymbolAdaptRVV(SB), NOSPLIT, $48-28
	MOV   s+0(FP), X10
	MOV   cdf+8(FP), X19
	MOV   n+16(FP), X18

	MOVWU MSAC_RNG(X10), X8
	MOV   MSAC_DIF(X10), X9
	SRLI  $48, X9, X17

	SRLI $8, X8, X20

	ADD $1, X18, X21
	VSETVLI X21, E16, M2, TA, MA, X22

	VLE16V  (X19), V8
	VSRLVI  $6, V8, V12
	VWMULUVX X20, V12, V16
	VNSRLWI $1, V16, V12

	MOV  $msacMinProb<>(SB), X23
	MOV  $15, X24
	SUB  X18, X24, X24
	SLLI $1, X24, X24
	ADD  X24, X23, X23
	VLE16V (X23), V14
	VADDVV V14, V12, V12

	MOV     $8, X25
	ADD     X2, X25, X25
	VSE16V  V12, (X25)

	VMSLEUVX X17, V12, V0
	VFIRSTM  V0, X26

	SLLI  $1, X26, X6
	ADD   X25, X6, X6
	MOVHU (X6), X15
	BNEZ  X26, notfirst
	MOV   X8, X16
	JMP   gotuv

notfirst:
	ADD   $-2, X6, X28
	MOVHU (X28), X16

gotuv:
	SUB   X15, X16, X13
	SLLI  $48, X15, X29
	SUB   X29, X9, X9
	MOV   X26, X8

	MOVBU MSAC_UPD(X10), X20
	BEQZ  X20, noupd

	SLLI  $1, X18, X21
	ADD   X19, X21, X21
	MOVHU (X21), X30
	SRLI  $4, X30, X20
	ADD   $4, X20, X20
	MOV   $3, X7
	BLTU  X18, X7, rateok
	ADD   $1, X20, X20

rateok:
	VSETVLI X18, E16, M2, TA, MA, X22

	MOV        $32768, X23
	VMVVX      X23, V14
	MOV        $65535, X24
	VMERGEVXM  X24, V14, V0, V14
	VSUBVV     V8, V14, V14
	VSRAVX     X20, V14, V14
	VMVVI      $0, V16
	VMERGEVIM  $-1, V16, V0, V16
	VSUBVV     V16, V14, V14
	VADDVV     V14, V8, V8
	VSE16V     V8, (X19)

	MOV  $32, X23
	BGEU X30, X23, nocount
	ADD  $1, X30, X30

nocount:
	MOVH X30, (X21)

noupd:
	RENORM
	MOVW X8, ret+24(FP)
	RET
