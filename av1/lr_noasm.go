//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package av1

func lrInit[P pixel, C coef](*lrContext[P, C]) {}
