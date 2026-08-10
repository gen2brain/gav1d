//go:build noasm || (!amd64 && !arm64 && !(riscv64 && riscv64.rva23u64))

package avif

func rowFn(ssHor int) convertRow {
	if ssHor == 0 {
		return convertRow444Go
	}

	return convertRowGo
}

func rowFn16(ssHor int) convertRow16 {
	if ssHor == 0 {
		return convertRow16x444Go
	}

	return convertRow16Go
}
