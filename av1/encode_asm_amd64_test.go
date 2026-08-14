//go:build amd64 && !noasm

package av1

func hasAsmResidual() bool { return hasAVX2 }

func hasAsmSatd4() bool { return hasAVX2 }

func hasAsmSatd8() bool { return hasAVX2 }

func hasAsmSse() bool { return hasAVX2 }
