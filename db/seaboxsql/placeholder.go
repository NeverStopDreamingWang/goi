package seaboxsql

import (
	"fmt"
	"strings"
)

// defaultPlaceholderParser 将 ORM 的 ? 转为 PostgreSQL 参数绑定使用的顺序编号占位符。
// 此规则由底层驱动决定，不随 SeaBoxSQL 的 SQL 兼容模式切换。
type defaultPlaceholderParser struct{}

func (defaultPlaceholderParser) Placeholder() string { return "?" }

// Parse 返回传给 sql.DB 或 sql.Tx 的 SQL，最多替换参数个数对应的 ?。
// 不解析 SQL 字符串、注释或运算符；复杂语句可通过 EngineOptions 自定义解析器。
func (defaultPlaceholderParser) Parse(query string, args ...any) string {
	if len(args) == 0 || !strings.Contains(query, "?") {
		return query
	}
	var builder strings.Builder
	start := 0
	for index := range args {
		position := strings.IndexByte(query[start:], '?')
		if position < 0 {
			break
		}
		position += start
		builder.WriteString(query[start:position])
		builder.WriteString(fmt.Sprintf("$%d", index+1))
		start = position + 1
	}
	builder.WriteString(query[start:])
	return builder.String()
}
