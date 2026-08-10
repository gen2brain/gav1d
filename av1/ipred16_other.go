//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package av1

const ipredHas16Deep = false
