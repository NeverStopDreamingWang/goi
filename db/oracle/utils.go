package oracle

import (
	"fmt"
	"strings"
)

// convertPlaceholders 将统一的 ? 占位符转换为 Oracle/godror 支持的 :1,:2,...
//
// 参数:
//   - query: string 原始 SQL，使用 ? 作为占位符
//   - argsCount: int 参数个数
//
// 返回:
//   - string: 转换后的 SQL
//
// 说明:
//   - 使用顺序字符串遍历替代正则实现，依次将每个 ? 替换为 :1,:2,...
//   - 仅做占位符级别的替换，不解析 SQL 语法，假设不会在字符串字面量中出现未转义的 ?
func convertPlaceholders(query string, argsCount int) string {
	if argsCount == 0 || query == "" {
		return query
	}
	if !strings.Contains(query, "?") {
		return query
	}

	var (
		builder strings.Builder
		index   = 1
		start   = 0
	)

	for start < len(query) && index <= argsCount {
		pos := strings.IndexByte(query[start:], '?')
		if pos < 0 {
			break
		}
		pos += start

		// 写入 ? 之前的部分
		builder.WriteString(query[start:pos])
		// 写入 :N 占位符
		builder.WriteString(fmt.Sprintf(":%d", index))
		index++
		start = pos + 1
	}

	// 追加剩余部分
	if start < len(query) {
		builder.WriteString(query[start:])
	}

	// 如果没有发生替换则返回原始字符串
	if builder.Len() == 0 {
		return query
	}
	return builder.String()
}
