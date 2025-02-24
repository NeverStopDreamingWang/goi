package goi_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi"
)

// ExampleFormatBytes 展示如何使用 FormatBytes 函数格式化不同大小的字节数
func ExampleFormatBytes() {
	// 基本单位示例
	fmt.Println(goi.FormatBytes(512))  // 字节(B)
	fmt.Println(goi.FormatBytes(1024)) // 千字节(KB)
	fmt.Println(goi.FormatBytes(2048)) // 2KB

	// 更大单位示例
	fmt.Println(goi.FormatBytes(1024 * 1024))        // 兆字节(MB)
	fmt.Println(goi.FormatBytes(1024 * 1024 * 1024)) // 吉字节(GB)

	// 非整数倍示例
	fmt.Println(goi.FormatBytes(1536))              // 1.5KB
	fmt.Println(goi.FormatBytes(1024 * 1024 * 1.5)) // 1.5MB

	// 不同整数类型示例
	fmt.Println(goi.FormatBytes(int32(2048)))
	fmt.Println(goi.FormatBytes(int64(3072)))
	fmt.Println(goi.FormatBytes(uint(4096)))

	// Output:
	// 512.00 B
	// 1.00 KB
	// 2.00 KB
	// 1.00 MB
	// 1.00 GB
	// 1.50 KB
	// 1.50 MB
	// 2.00 KB
	// 3.00 KB
	// 4.00 KB
}
