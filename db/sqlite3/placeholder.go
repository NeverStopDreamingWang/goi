package sqlite3

import (
	"strings"

	"github.com/NeverStopDreamingWang/goi/v3/db"
)

// placeholderParser 保存后续新建 SQLite3 Engine 使用的占位符解析器。
var placeholderParser db.PlaceholderParser = defaultPlaceholderParser{}

// SetPlaceholderParser 设置后续 SQLite3 Engine 使用的占位符解析器。
//
// 参数:
//   - parser: db.PlaceholderParser 默认或自定义占位符解析器；nil 时忽略，保留原配置
//
// 说明:
//   - 仅影响新建 Engine，已有实例不变
//   - 须在连接前配置，不可与 Connect 或 ConnectDatabase 并发调用
//   - 注册后的实现不应再修改，共享状态的并发安全由实现者保证
func SetPlaceholderParser(parser db.PlaceholderParser) {
	if parser == nil {
		return
	}
	placeholderParser = parser
}

// GetPlaceholderParser 获取当前 SQLite3 包级占位符解析器。
//
// 返回:
//   - db.PlaceholderParser: 当前 SQLite3 包级占位符解析器
//
// 说明:
//   - 默认 ORM 和 SQLite3 驱动均使用 ?；Engine 在创建时保存解析器
func GetPlaceholderParser() db.PlaceholderParser {
	return placeholderParser
}

// defaultPlaceholderParser 使用 ? 占位符并展开 IN / NOT IN 集合参数。
type defaultPlaceholderParser struct{}

// Placeholder 返回 ORM 使用的 ? 占位符。
func (defaultPlaceholderParser) Placeholder() string { return "?" }

// Parse 展开集合参数，返回交给 sql.DB 或 sql.Tx 执行的 SQL 和参数。
//
// 说明:
//   - ORM 占位符由 Placeholder 指定，匹配后统一转换为 SQLite3 驱动使用的 ?
//   - IN ?、NOT IN ? 支持切片、数组及其非 nil 指针，[]byte 除外
//   - IN (?) 等带括号的写法不展开；空集合生成 ()，是否合法由数据库处理
//   - 按字符串匹配占位符，不解析 SQL 字符串或注释，也不校验参数数量
//   - 不修改传入的参数切片，未匹配的 SQL 和参数保持原样
func (parser defaultPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	placeholder := parser.Placeholder()
	if len(args) == 0 || !strings.Contains(query, placeholder) {
		return query, args
	}

	var builder strings.Builder
	builder.Grow(len(query))
	expandedArgs := make([]any, 0, len(args))
	for i, param := range args {
		pos := strings.Index(query, placeholder)
		if pos < 0 {
			expandedArgs = append(expandedArgs, args[i:]...)
			break
		}
		leftPart := query[:pos]
		builder.WriteString(leftPart)
		inPlaceholder, inArgs, ok := expandInParameter(leftPart, placeholder, param)
		if ok {
			// 仅转换展开片段中的占位符，保留帮助方法生成的括号和分隔符。
			for _, inArg := range inArgs {
				inPos := strings.Index(inPlaceholder, placeholder)
				builder.WriteString(inPlaceholder[:inPos])
				builder.WriteString("?")
				inPlaceholder = inPlaceholder[inPos+len(placeholder):]
				expandedArgs = append(expandedArgs, inArg)
			}
			builder.WriteString(inPlaceholder)
		} else {
			builder.WriteString("?")
			expandedArgs = append(expandedArgs, param)
		}
		query = query[pos+len(placeholder):]
	}
	builder.WriteString(query)
	return builder.String(), expandedArgs
}
