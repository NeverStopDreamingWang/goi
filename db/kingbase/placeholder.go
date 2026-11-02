package kingbase

import (
	"fmt"
	"strings"
)

// gokbPlaceholderParser 将 ORM 的 ? 转为 Gokb 的顺序编号占位符。
type gokbPlaceholderParser struct{}

func (gokbPlaceholderParser) Placeholder() string { return "?" }

// Parse 返回驱动 SQL，最多替换参数个数对应的 ?，不解析 SQL 字符串或注释。
func (gokbPlaceholderParser) Parse(query string, args ...any) string {
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
