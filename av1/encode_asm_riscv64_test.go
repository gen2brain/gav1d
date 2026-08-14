//go:build riscv64 && riscv64.rva23u64 && !noasm

package av1

func hasAsmResidual() bool { return true }

func hasAsmSatd() bool { return false }

func hasAsmSse() bool { return true }
