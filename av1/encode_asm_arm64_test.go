//go:build arm64 && !noasm

package av1

func hasAsmResidual() bool { return true }

func hasAsmSatd4() bool { return true }

func hasAsmSatd8() bool { return false }

func hasAsmSse() bool { return true }
