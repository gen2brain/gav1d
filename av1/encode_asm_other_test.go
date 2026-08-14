//go:build !amd64 || noasm

package av1

func hasAsmResidual() bool { return false }
