//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no signed word shift, min, max or narrow.
#define SSHLS(Vd, Vn, Vm)  WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS(Vd, Vn, Vm)  WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm)  WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHR4(Vd, Vn)      WORD $(0x4F3C0400 | ((Vn) << 5) | (Vd))
#define SQXTUNH(Vd, Vn)    WORD $(0x2E612800 | ((Vn) << 5) | (Vd))
#define SQXTUNB(Vd, Vn)    WORD $(0x2E212800 | ((Vn) << 5) | (Vd))

// func itxClipNEON(tmp *int32, n int, rnd int32, shift int, lo, hi int32)
TEXT ·itxClipNEON(SB), NOSPLIT, $0-40
	MOVD tmp+0(FP), R0
	MOVD n+8(FP), R1
	MOVW rnd+16(FP), R2
	VDUP R2, V1.S4
	MOVD shift+24(FP), R3
	NEG  R3, R3
	VDUP R3, V2.S4
	MOVW lo+32(FP), R4
	VDUP R4, V3.S4
	MOVW hi+36(FP), R5
	VDUP R5, V4.S4

loop:
	VLD1 (R0), [V0.S4]
	VADD V1.S4, V0.S4, V0.S4
	SSHLS(0, 0, 2)
	SMAXS(0, 0, 3)
	SMINS(0, 0, 4)
	VST1 [V0.S4], (R0)

	ADD  $16, R0
	SUB  $4, R1
	CBNZ R1, loop

	RET

// RESIDUAL rounds one vector of residual and folds the pixels into it.
#define RESIDUAL                  \
	VLD1 (R2), [V0.S4];       \
	VADD V5.S4, V0.S4, V0.S4; \
	SSHR4(0, 0);              \
	VADD V1.S4, V0.S4, V0.S4; \
	SMAXS(0, 0, 6);           \
	SMINS(0, 0, 7)

// func itxAdd8NEON(dst *uint8, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd8NEON(SB), NOSPLIT, $0-44
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tmp+16(FP), R2
	MOVD w+24(FP), R3
	MOVD h+32(FP), R4
	MOVW bitdepthMax+40(FP), R5
	VDUP R5, V7.S4
	VEOR V6.B16, V6.B16, V6.B16
	MOVD $8, R6
	VDUP R6, V5.S4

rows8:
	MOVD R0, R7
	MOVD R3, R8

cols8:
	VLD1   (R7), [V1.S4]
	VUSHLL $0, V1.B8, V1.H8
	VUSHLL $0, V1.H4, V1.S4
	RESIDUAL
	SQXTUNH(0, 0)
	SQXTUNB(0, 0)
	VMOV   V0.S[0], R9
	MOVW   R9, (R7)

	ADD  $16, R2
	ADD  $4, R7
	SUB  $4, R8
	CBNZ R8, cols8

	ADD  R1, R0
	SUB  $1, R4
	CBNZ R4, rows8

	RET

// func itxAdd16NEON(dst *uint16, stride int, tmp *int32, w, h int, bitdepthMax int32)
TEXT ·itxAdd16NEON(SB), NOSPLIT, $0-44
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tmp+16(FP), R2
	MOVD w+24(FP), R3
	MOVD h+32(FP), R4
	MOVW bitdepthMax+40(FP), R5
	VDUP R5, V7.S4
	VEOR V6.B16, V6.B16, V6.B16
	MOVD $8, R6
	VDUP R6, V5.S4
	LSL  $1, R1, R1

rows16:
	MOVD R0, R7
	MOVD R3, R8

cols16:
	VLD1   (R7), [V1.H4]
	VUSHLL $0, V1.H4, V1.S4
	RESIDUAL
	SQXTUNH(0, 0)
	VMOV   V0.D[0], R9
	MOVD   R9, (R7)

	ADD  $16, R2
	ADD  $8, R7
	SUB  $4, R8
	CBNZ R8, cols16

	ADD  R1, R0
	SUB  $1, R4
	CBNZ R4, rows16

	RET

// func itxTransposeNEON(wide, out *int32, w, sh, n int)
TEXT ·itxTransposeNEON(SB), NOSPLIT, $0-40
	MOVD wide+0(FP), R0
	MOVD out+8(FP), R1
	MOVD w+16(FP), R2
	MOVD sh+24(FP), R3
	MOVD n+32(FP), R4
	LSL  $2, R3, R3
	LSL  $2, R2, R5

trlane:
	MOVD R0, R6
	MOVD R1, R7
	MOVD R2, R8

trtap:
	MOVD R6, R9
	VLD1 (R9), [V0.S4]
	ADD  R3, R9
	VLD1 (R9), [V1.S4]
	ADD  R3, R9
	VLD1 (R9), [V2.S4]
	ADD  R3, R9
	VLD1 (R9), [V3.S4]

	VZIP1 V2.S4, V0.S4, V4.S4
	VZIP1 V3.S4, V1.S4, V5.S4
	VZIP2 V2.S4, V0.S4, V6.S4
	VZIP2 V3.S4, V1.S4, V7.S4
	VZIP1 V5.S4, V4.S4, V0.S4
	VZIP2 V5.S4, V4.S4, V1.S4
	VZIP1 V7.S4, V6.S4, V2.S4
	VZIP2 V7.S4, V6.S4, V3.S4

	MOVD R7, R9
	VST1 [V0.S4], (R9)
	ADD  R5, R9
	VST1 [V1.S4], (R9)
	ADD  R5, R9
	VST1 [V2.S4], (R9)
	ADD  R5, R9
	VST1 [V3.S4], (R9)

	ADD  R3<<2, R6, R6
	ADD  $16, R7
	SUB  $4, R8
	CBNZ R8, trtap

	ADD  $16, R0
	ADD  R5<<2, R1, R1
	SUB  $4, R4
	CBNZ R4, trlane

	RET
