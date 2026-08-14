//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package av1

func hasAsmResidual() bool { return false }

func hasAsmSatd4() bool { return false }

func hasAsmSatd8() bool { return false }

func hasAsmSse() bool { return false }
