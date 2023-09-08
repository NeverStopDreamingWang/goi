package hgee

import (
	"fmt"
)

type formatBytesSize interface {
	int8 | int16 | int32 | int64 | int | uint8 | uint16 | uint32 | uint64 | uint | float32 | float64
}

func formatBytes[T formatBytesSize](ByteSize T) string {
	byteSize := float64(ByteSize)
	const unit = 1024.00
	exts := [7]string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

	for _, ext := range exts {
		if byteSize < unit {
			return fmt.Sprintf("%.2f %v", byteSize, ext)
		}
		byteSize /= unit
	}
	return ""
}
