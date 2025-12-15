package sqlserver

import (
	"fmt"
	"strings"
)

// convertPlaceholders 将统一的 ? 占位符转换为 @p1,@p2,... 形式（适配 go-mssqldb）
//
// 说明:
//   - 仅做简单的占位符替换，不解析 SQL 语法
//   - 假设 SQL 中不会在字符串字面量里包含未转义的 ?
func convertPlaceholders(query string, argsCount int) string {
	if argsCount == 0 {
		return query
	}

	// 快速统计是否存在 ?，避免无谓分配
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
		// 写入 @pN 占位符
		builder.WriteString(fmt.Sprintf("@p%d", index))
		index++
		start = pos + 1
	}

	// 追加剩余部分
	if start < len(query) {
		builder.WriteString(query[start:])
	}

	return builder.String()
}
