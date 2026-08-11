//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

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

// WSUM squares sixteen words at X10, weights them from WSRC and leaves the
// total in X14, advancing X10.
#define WSUM(WSRC)                        \
	MOV     $16, X12;                 \
	VSETVLI X12, E32, M4, TA, MA, X13; \
	VLE32V  (X10), V8;                \
	VLE32V  (WSRC), V16;              \
	VMULVV  V8, V8, V8;               \
	VMULVV  V16, V8, V8;              \
	VMVVI   $0, V4;                   \
	VREDSUMVS V4, V8, V4;             \
	VMVXS   V4, X14;                  \
	ADD     $64, X10, X10

// func cdefDirCostsRVV(s *cdefSums, cost *[8]uint32)
TEXT ·cdefDirCostsRVV(SB), NOSPLIT, $0-16
	MOV s+0(FP), X10
	MOV cost+8(FP), X11
	MOV $105, X15

	MOV     $8, X12
	VSETVLI X12, E32, M2, TA, MA, X13
	VLE32V  (X10), V8
	VMULVV  V8, V8, V8
	VMVVI   $0, V4
	VREDSUMVS V4, V8, V4
	VMVXS   V4, X14
	MULW    X15, X14, X14
	MOVW    X14, 8(X11)
	ADD     $32, X10, X10

	MOV     $8, X12
	VSETVLI X12, E32, M2, TA, MA, X13
	VLE32V  (X10), V8
	VMULVV  V8, V8, V8
	VMVVI   $0, V4
	VREDSUMVS V4, V8, V4
	VMVXS   V4, X14
	MULW    X15, X14, X14
	MOVW    X14, 24(X11)
	ADD     $32, X10, X10

	MOV $cdefWDiag<>(SB), X16
	WSUM(X16)
	MOVW X14, 0(X11)
	MOV $cdefWDiag<>(SB), X16
	WSUM(X16)
	MOVW X14, 16(X11)

	MOV $cdefWAlt<>(SB), X16
	WSUM(X16)
	MOVW X14, 4(X11)
	MOV $cdefWAlt<>(SB), X16
	WSUM(X16)
	MOVW X14, 12(X11)
	MOV $cdefWAlt<>(SB), X16
	WSUM(X16)
	MOVW X14, 20(X11)
	MOV $cdefWAlt<>(SB), X16
	WSUM(X16)
	MOVW X14, 28(X11)

	RET
