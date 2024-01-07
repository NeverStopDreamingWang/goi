package goi

import (
	"fmt"
)

func formatBytes[T IntAll](ByteSize T) string {
	byteSize := float64(ByteSize)
	const unit = 1024.00
	exts := [7]string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

	for _, ext := range exts {
		if byteSize < unit {
			return fmt.Sprintf("%.2f %v", byteSize, ext)
		}
		byteSize /= unit
	}
	return fmt.Sprintf("%.2f %v", byteSize, exts[len(exts)-1])
}
