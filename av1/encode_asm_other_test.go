//go:build (!amd64 && !arm64) || noasm

package av1

func hasAsmResidual() bool { return false }

func hasAsmSatd() bool { return false }

func hasAsmSse() bool { return false }
