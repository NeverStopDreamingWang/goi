package goi

import (
	"fmt"
)

// FormatBytes 将字节大小格式化为人类可读的字符串
//
// 类型参数:
//   - T IntAll: 整数类型
//
// 参数:
//   - ByteSize T: 字节大小
//
// 返回:
//   - string: 格式化后的字符串，如 "1.25 MB"
//
// 示例:
//
//	FormatBytes(1024) // 返回 "1.00 KB"
//	FormatBytes(1536) // 返回 "1.50 KB"
func FormatBytes[T IntAll](ByteSize T) string {
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
