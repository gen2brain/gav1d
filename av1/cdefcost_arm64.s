//go:build arm64 && !noasm

#include "textflag.h"

#define MULS4(Vd, Vn, Vm) WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))

DATA cdefWDiag<>+0(SB)/8, $0x000001a400000348
DATA cdefWDiag<>+8(SB)/8, $0x000000d200000118
DATA cdefWDiag<>+16(SB)/8, $0x0000008c000000a8
DATA cdefWDiag<>+24(SB)/8, $0x0000006900000078
DATA cdefWDiag<>+32(SB)/8, $0x0000008c00000078
DATA cdefWDiag<>+40(SB)/8, $0x000000d2000000a8
DATA cdefWDiag<>+48(SB)/8, $0x000001a400000118
DATA cdefWDiag<>+56(SB)/8, $0x0000000000000348
GLOBL cdefWDiag<>(SB), RODATA|NOPTR, $64

DATA cdefWAlt<>+0(SB)/8, $0x000000d2000001a4
DATA cdefWAlt<>+8(SB)/8, $0x000000690000008c
DATA cdefWAlt<>+16(SB)/8, $0x0000006900000069
DATA cdefWAlt<>+24(SB)/8, $0x0000006900000069
DATA cdefWAlt<>+32(SB)/8, $0x000000d20000008c
DATA cdefWAlt<>+40(SB)/8, $0x00000000000001a4
DATA cdefWAlt<>+48(SB)/8, $0x0000000000000000
DATA cdefWAlt<>+56(SB)/8, $0x0000000000000000
GLOBL cdefWAlt<>(SB), RODATA|NOPTR, $64

// ACC squares one vector, weights it and accumulates into V0.
#define ACC(SRC, WSRC)          \
	VLD1.P 16(SRC), [V4.S4];  \
	VLD1.P 16(WSRC), [V5.S4]; \
	MULS4(4, 4, 4);           \
	MULS4(4, 4, 5);           \
	VADD V4.S4, V0.S4, V0.S4

// SUMSQ accumulates four vectors from SRC weighted by WSRC into R11.
#define SUMSQ(SRC, WSRC)        \
	VEOR V0.B16, V0.B16, V0.B16; \
	ACC(SRC, WSRC);           \
	ACC(SRC, WSRC);           \
	ACC(SRC, WSRC);           \
	ACC(SRC, WSRC);           \
	VADDV V0.S4, V1;          \
	VMOV  V1.S[0], R11

// func cdefDirCostsNEON(s *cdefSums, cost *[8]uint32)
TEXT ·cdefDirCostsNEON(SB), NOSPLIT, $0-16
	MOVD s+0(FP), R0
	MOVD cost+8(FP), R1

	MOVD $105, R13

	// cost[2] and cost[6]: the horizontal and vertical sums, all weight 105.
	VEOR V0.B16, V0.B16, V0.B16
	VLD1.P 16(R0), [V4.S4]
	MULS4(4, 4, 4)
	VADD V4.S4, V0.S4, V0.S4
	VLD1.P 16(R0), [V4.S4]
	MULS4(4, 4, 4)
	VADD V4.S4, V0.S4, V0.S4
	VADDV V0.S4, V1
	VMOV  V1.S[0], R11
	MUL   R13, R11, R11
	MOVW  R11, 8(R1)

	VEOR V0.B16, V0.B16, V0.B16
	VLD1.P 16(R0), [V4.S4]
	MULS4(4, 4, 4)
	VADD V4.S4, V0.S4, V0.S4
	VLD1.P 16(R0), [V4.S4]
	MULS4(4, 4, 4)
	VADD V4.S4, V0.S4, V0.S4
	VADDV V0.S4, V1
	VMOV  V1.S[0], R11
	MUL   R13, R11, R11
	MOVW  R11, 24(R1)

	// cost[0] and cost[4]: the diagonals, weighted by the divide table.
	MOVD $cdefWDiag<>(SB), R2
	SUMSQ(R0, R2)
	MOVW R11, 0(R1)

	MOVD $cdefWDiag<>(SB), R2
	SUMSQ(R0, R2)
	MOVW R11, 16(R1)

	// cost[1], cost[3], cost[5], cost[7]: the four alternates.
	MOVD $4, R14
	MOVD $1, R15

alt:
	MOVD $cdefWAlt<>(SB), R2
	SUMSQ(R0, R2)
	MOVD R15, R16
	LSL  $2, R16, R16
	ADD  R1, R16, R16
	MOVW R11, (R16)
	ADD  $2, R15, R15
	SUB  $1, R14, R14
	CBNZ R14, alt

	RET
