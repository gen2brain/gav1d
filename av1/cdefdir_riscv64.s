//go:build riscv64 && riscv64.rva23u64 && !noasm

#include "textflag.h"

#define HV0  0
#define HV1  32
#define DIA0 64
#define DIA1 128
#define ALT0 192
#define ALT1 256
#define ALT2 320
#define ALT3 384

// ADDAT accumulates a whole row into one of the padded arrays at an offset
// that only the row index decides.
#define ADDAT(OFF, SRC)      \
	ADD    $OFF, X11, X6; \
	VLE32V (X6), V4;     \
	VADDVV SRC, V4, V4;  \
	VSE32V V4, (X6)

// ROW folds one row of eight samples, held in V2, into all eight partial sums.
// HY is Y>>1, where the alternating sums step half as fast.
#define ROW(Y, HY)                 \
	VADDVV V2, V0, V0;         \
	VMVVX  X0, V6;             \
	VREDSUMVS V6, V2, V7;      \
	VMVXS  V7, X6;             \
	MOVW   X6, (HV0+4*(Y))(X11); \
	ADDAT(DIA0+4*(Y), V2);     \
	VRGATHERVV V12, V2, V6;    \
	ADDAT(DIA1+4*(Y), V6);     \
	ADDAT(ALT2+4*(3-(HY)), V2); \
	ADDAT(ALT3+4*(HY), V2);    \
	VRGATHERVV V13, V2, V6;    \
	VRGATHERVV V14, V2, V7;    \
	VADDVV V7, V6, V6;         \
	VSETVLI X5, E32, M1, TA, MA, X7; \
	ADD    $(ALT0+4*(Y)), X11, X6; \
	VLE32V (X6), V4;           \
	VADDVV V6, V4, V4;         \
	VSE32V V4, (X6);           \
	VRGATHERVV V15, V6, V7;    \
	ADD    $(ALT1+4*(Y)), X11, X6; \
	VLE32V (X6), V4;           \
	VADDVV V7, V4, V4;         \
	VSE32V V4, (X6);           \
	VSETVLI X8, E32, M1, TA, MA, X7

// func cdefSums8RVV(img *uint8, stride int, s *cdefSums, idx *[32]int32)
TEXT ·cdefSums8RVV(SB), NOSPLIT, $0-32
	MOV  img+0(FP), X10
	MOV  stride+8(FP), X12
	MOV  s+16(FP), X11
	MOV  idx+24(FP), X13

	MOV     $8, X8
	MOV     $4, X5
	VSETVLI X8, E32, M1, TA, MA, X7
	VLE32V  (X13), V12
	ADD     $32, X13, X6
	VLE32V  (X6), V13
	ADD     $64, X13, X6
	VLE32V  (X6), V14
	ADD     $96, X13, X6
	VLE32V  (X6), V15
	VMVVX   X0, V0
	MOV     $128, X9

#define LOAD8                          \
	VSETVLI X8, E8, MF4, TA, MA, X7; \
	VLE8V   (X10), V3;             \
	VSETVLI X8, E32, M1, TA, MA, X7; \
	VZEXTVF4 V3, V2;               \
	VSUBVX  X9, V2, V2

	LOAD8
	ROW(0, 0)
	ADD X12, X10, X10
	LOAD8
	ROW(1, 0)
	ADD X12, X10, X10
	LOAD8
	ROW(2, 1)
	ADD X12, X10, X10
	LOAD8
	ROW(3, 1)
	ADD X12, X10, X10
	LOAD8
	ROW(4, 2)
	ADD X12, X10, X10
	LOAD8
	ROW(5, 2)
	ADD X12, X10, X10
	LOAD8
	ROW(6, 3)
	ADD X12, X10, X10
	LOAD8
	ROW(7, 3)

	ADD    $HV1, X11, X6
	VSE32V V0, (X6)
	RET

// func cdefSums16RVV(img *uint16, stride int, s *cdefSums, idx *[32]int32, shift int)
TEXT ·cdefSums16RVV(SB), NOSPLIT, $0-40
	MOV  img+0(FP), X10
	MOV  stride+8(FP), X12
	SLLI $1, X12, X12
	MOV  s+16(FP), X11
	MOV  idx+24(FP), X13
	MOV  shift+32(FP), X14

	MOV     $8, X8
	MOV     $4, X5
	VSETVLI X8, E32, M1, TA, MA, X7
	VLE32V  (X13), V12
	ADD     $32, X13, X6
	VLE32V  (X6), V13
	ADD     $64, X13, X6
	VLE32V  (X6), V14
	ADD     $96, X13, X6
	VLE32V  (X6), V15
	VMVVX   X0, V0
	MOV     $128, X9

#define LOAD16                         \
	VSETVLI X8, E16, MF2, TA, MA, X7; \
	VLE16V  (X10), V3;             \
	VSETVLI X8, E32, M1, TA, MA, X7; \
	VZEXTVF2 V3, V2;               \
	VSRLVX  X14, V2, V2;           \
	VSUBVX  X9, V2, V2

	LOAD16
	ROW(0, 0)
	ADD X12, X10, X10
	LOAD16
	ROW(1, 0)
	ADD X12, X10, X10
	LOAD16
	ROW(2, 1)
	ADD X12, X10, X10
	LOAD16
	ROW(3, 1)
	ADD X12, X10, X10
	LOAD16
	ROW(4, 2)
	ADD X12, X10, X10
	LOAD16
	ROW(5, 2)
	ADD X12, X10, X10
	LOAD16
	ROW(6, 3)
	ADD X12, X10, X10
	LOAD16
	ROW(7, 3)

	ADD    $HV1, X11, X6
	VSE32V V0, (X6)
	RET
