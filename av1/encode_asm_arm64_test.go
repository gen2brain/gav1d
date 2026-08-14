//go:build arm64 && !noasm

package av1

func hasAsmResidual() bool { return true }

func hasAsmSatd() bool { return false }

func hasAsmSse() bool { return false }
