package postgres

import (
	"fmt"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v3/db"
)

// placeholderParser 保存后续新建 PostgreSQL Engine 使用的占位符解析器。
var placeholderParser db.PlaceholderParser = defaultPlaceholderParser{}

// SetPlaceholderParser 设置后续新建实例使用的占位符解析器
//
// 参数:
//   - parser: db.PlaceholderParser 默认或自定义解析器；nil 时忽略，保留原配置
//
// 说明:
//   - 在连接前设置，勿与 Connect、ConnectDatabase 并发调用；已有实例不变
//   - 注册后不要修改解析器；自定义实现需保证内部并发安全
func SetPlaceholderParser(parser db.PlaceholderParser) {
	if parser == nil {
		return
	}
	placeholderParser = parser
}

// GetPlaceholderParser 获取当前包级占位符解析器
//
// 返回:
//   - db.PlaceholderParser: 当前解析器，默认将 ORM 的 ? 转为驱动的 $1、$2 等
//
// 说明:
//   - 新建实例保存当前解析器，后续包级替换不影响已有实例
func GetPlaceholderParser() db.PlaceholderParser {
	return placeholderParser
}

// defaultPlaceholderParser 将通用占位符转换为顺序编号占位符
type defaultPlaceholderParser struct{}

// Placeholder 返回 ORM 使用的 ? 占位符
func (defaultPlaceholderParser) Placeholder() string {
	return "?"
}

// Parse 展开集合参数并生成 $1、$2 等编号，返回交给 sql.DB 或 sql.Tx 的 SQL 和参数。
//
// 说明:
//   - IN ?、NOT IN ? 支持切片、数组及其非 nil 指针，不限制元素类型
//   - IN (?) 等带括号的写法不展开；空集合生成 ()，是否合法由数据库处理
//   - 单次扫描 SQL，不解析 SQL 字符串或注释，也不校验参数数量
//   - 不修改传入的参数切片，未匹配的 SQL 和多余参数保持原样
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
			// 仅替换展开片段中的占位符，括号和分隔符沿用帮助方法的结果。
			for _, inArg := range inArgs {
				inPos := strings.Index(inPlaceholder, placeholder)
				builder.WriteString(inPlaceholder[:inPos])
				builder.WriteString(fmt.Sprintf("$%d", len(expandedArgs)+1))
				inPlaceholder = inPlaceholder[inPos+len(placeholder):]
				expandedArgs = append(expandedArgs, inArg)
			}
			builder.WriteString(inPlaceholder)
		} else {
			builder.WriteString(fmt.Sprintf("$%d", len(expandedArgs)+1))
			expandedArgs = append(expandedArgs, param)
		}
		query = query[pos+len(placeholder):]
	}
	builder.WriteString(query)
	return builder.String(), expandedArgs
}
