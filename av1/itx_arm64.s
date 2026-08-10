//go:build arm64 && !noasm

#include "textflag.h"

// Go's arm64 assembler has no signed word shift, min, max or narrow.
#define SSHLS(Vd, Vn, Vm)  WORD $(0x4EA04400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMAXS(Vd, Vn, Vm)  WORD $(0x4EA06400 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SMINS(Vd, Vn, Vm)  WORD $(0x4EA06C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHR4(Vd, Vn)      WORD $(0x4F3C0400 | ((Vn) << 5) | (Vd))
#define SQXTUNH(Vd, Vn)    WORD $(0x2E612800 | ((Vn) << 5) | (Vd))
#define SQXTUNB(Vd, Vn)    WORD $(0x2E212800 | ((Vn) << 5) | (Vd))
#define SSHLLW(Vd, Vn)     WORD $(0x0F10A400 | ((Vn) << 5) | (Vd))
#define SSHLL2W(Vd, Vn)    WORD $(0x4F10A400 | ((Vn) << 5) | (Vd))
#define MULS(Vd, Vn, Vm)   WORD $(0x4EA09C00 | ((Vm) << 16) | ((Vn) << 5) | (Vd))
#define SSHR8(Vd, Vn)      WORD $(0x4F380400 | ((Vn) << 5) | (Vd))

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

// func itxAdd8NEON(dst *uint8, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd8NEON(SB), NOSPLIT, $0-52
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tmp+16(FP), R2
	MOVD w+24(FP), R3
	MOVD h+32(FP), R4
	MOVD ts+40(FP), R11
	MOVW bitdepthMax+48(FP), R5
	VDUP R5, V7.S4
	VEOR V6.B16, V6.B16, V6.B16
	MOVD $8, R6
	VDUP R6, V5.S4
	LSL  $2, R11, R11

rows8:
	MOVD R0, R7
	MOVD R3, R8
	MOVD R2, R12

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

	ADD  R11, R12, R2
	ADD  R1, R0
	SUB  $1, R4
	CBNZ R4, rows8

	RET

// func itxAdd16NEON(dst *uint16, stride int, tmp *int32, w, h, ts int, bitdepthMax int32)
TEXT ·itxAdd16NEON(SB), NOSPLIT, $0-52
	MOVD dst+0(FP), R0
	MOVD stride+8(FP), R1
	MOVD tmp+16(FP), R2
	MOVD w+24(FP), R3
	MOVD h+32(FP), R4
	MOVD ts+40(FP), R11
	MOVW bitdepthMax+48(FP), R5
	VDUP R5, V7.S4
	VEOR V6.B16, V6.B16, V6.B16
	MOVD $8, R6
	VDUP R6, V5.S4
	LSL  $1, R1, R1
	LSL  $2, R11, R11

rows16:
	MOVD R0, R7
	MOVD R3, R8
	MOVD R2, R12

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

	ADD  R11, R12, R2
	ADD  R1, R0
	SUB  $1, R4
	CBNZ R4, rows16

	RET

// func itxTransposeNEON(wide, out *int32, w, ws, ts, n int)
TEXT ·itxTransposeNEON(SB), NOSPLIT, $0-48
	MOVD wide+0(FP), R0
	MOVD out+8(FP), R1
	MOVD w+16(FP), R2
	MOVD ws+24(FP), R3
	MOVD ts+32(FP), R5
	MOVD n+40(FP), R4
	LSL  $2, R3, R3
	LSL  $2, R5, R5

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

// func widenCoefs16NEON(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16NEON(SB), NOSPLIT, $0-56
	MOVD  dst+0(FP), R0
	MOVD  dstStride+8(FP), R1
	LSL   $2, R1, R1
	MOVD  src+16(FP), R2
	MOVD  srcStride+24(FP), R3
	LSL   $1, R3, R3
	MOVD  cols+32(FP), R4
	MOVD  n+40(FP), R5
	MOVD  lanes+48(FP), R6
	VMOVI $0, V5.B16
	VMOVI $0, V6.B16
	CBZ   R4, wdone

wcol:
	MOVD R0, R7
	MOVD R2, R8
	MOVD ZR, R9

w8:
	ADD  $8, R9, R10
	CMP  R5, R10
	BGT  w1
	VLD1 (R8), [V0.H8]
	SSHLL2W(2, 0)
	SSHLLW(1, 0)
	VST1 [V1.S4, V2.S4], (R7)
	ADD  $16, R8, R8
	ADD  $32, R7, R7
	MOVD R10, R9
	B    w8

w1:
	CMP  R5, R9
	BGE  wz8
	MOVH (R8), R11
	MOVW R11, (R7)
	ADD  $2, R8, R8
	ADD  $4, R7, R7
	ADD  $1, R9, R9
	B    w1

wz8:
	ADD  $8, R9, R10
	CMP  R6, R10
	BGT  wz1
	VST1 [V5.S4, V6.S4], (R7)
	ADD  $32, R7, R7
	MOVD R10, R9
	B    wz8

wz1:
	CMP  R6, R9
	BGE  wnext
	MOVW ZR, (R7)
	ADD  $4, R7, R7
	ADD  $1, R9, R9
	B    wz1

wnext:
	ADD  R1, R0, R0
	ADD  R3, R2, R2
	SUBS $1, R4, R4
	BNE  wcol

wdone:
	RET

// func widenCoefs16Rect2NEON(dst *int32, dstStride int, src *int16, srcStride, cols, n, lanes int)
TEXT ·widenCoefs16Rect2NEON(SB), NOSPLIT, $0-56
	MOVD  dst+0(FP), R0
	MOVD  dstStride+8(FP), R1
	LSL   $2, R1, R1
	MOVD  src+16(FP), R2
	MOVD  srcStride+24(FP), R3
	LSL   $1, R3, R3
	MOVD  cols+32(FP), R4
	MOVD  n+40(FP), R5
	MOVD  lanes+48(FP), R6
	VMOVI $0, V5.B16
	VMOVI $0, V6.B16
	MOVD  $181, R12
	VDUP  R12, V3.S4
	MOVD  $128, R13
	VDUP  R13, V4.S4
	CBZ   R4, rdone

rcol:
	MOVD R0, R7
	MOVD R2, R8
	MOVD ZR, R9

r8:
	ADD  $8, R9, R10
	CMP  R5, R10
	BGT  r1
	VLD1 (R8), [V0.H8]
	SSHLL2W(2, 0)
	SSHLLW(1, 0)
	MULS(1, 1, 3)
	MULS(2, 2, 3)
	VADD V4.S4, V1.S4, V1.S4
	VADD V4.S4, V2.S4, V2.S4
	SSHR8(1, 1)
	SSHR8(2, 2)
	VST1 [V1.S4, V2.S4], (R7)
	ADD  $16, R8, R8
	ADD  $32, R7, R7
	MOVD R10, R9
	B    r8

r1:
	CMP   R5, R9
	BGE   rz8
	MOVH  (R8), R11
	MOVD  $181, R14
	MUL   R14, R11, R11
	ADD   $128, R11, R11
	ASR   $8, R11, R11
	MOVW  R11, (R7)
	ADD   $2, R8, R8
	ADD   $4, R7, R7
	ADD   $1, R9, R9
	B     r1

rz8:
	ADD  $8, R9, R10
	CMP  R6, R10
	BGT  rz1
	VST1 [V5.S4, V6.S4], (R7)
	ADD  $32, R7, R7
	MOVD R10, R9
	B    rz8

rz1:
	CMP  R6, R9
	BGE  rnext
	MOVW ZR, (R7)
	ADD  $4, R7, R7
	ADD  $1, R9, R9
	B    rz1

rnext:
	ADD  R1, R0, R0
	ADD  R3, R2, R2
	SUBS $1, R4, R4
	BNE  rcol

rdone:
	RET
