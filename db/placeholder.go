package db

// PlaceholderParser 定义 ORM 占位符，并在执行前转换 SQL 和参数，可自行实现以适配驱动。
type PlaceholderParser interface {
	// Placeholder 返回 ORM 构建 SQL 时使用的占位符。
	Placeholder() string
	// Parse 返回最终传给 sql.DB 或 sql.Tx 执行的 SQL 和参数。
	Parse(query string, args ...any) (string, []any)
}

// DefaultPlaceholderParser 使用固定的 ORM 占位符，原样返回 SQL 和参数。
//
// 说明:
//   - 适用于 ORM 与驱动使用相同占位符的情况
//   - 需要展开集合、转换占位符或识别 SQL 语法时，请自行实现 PlaceholderParser
type DefaultPlaceholderParser struct {
	placeholder string
}

// NewPlaceholderParser 创建使用指定占位符的默认解析器
//
// 参数:
//   - placeholder: string ORM 构建 SQL 时使用的占位符
//
// 返回:
//   - DefaultPlaceholderParser: 使用指定占位符、原样返回 SQL 和参数的解析器
func NewPlaceholderParser(placeholder string) DefaultPlaceholderParser {
	return DefaultPlaceholderParser{placeholder: placeholder}
}

// Placeholder 返回 ORM 构建 SQL 时使用的占位符
//
// 返回:
//   - string: ORM 占位符
func (parser DefaultPlaceholderParser) Placeholder() string {
	return parser.placeholder
}

// Parse 原样返回待执行的 SQL 和参数
//
// 参数:
//   - query: string ORM 构建的 SQL
//   - args: ...any SQL 参数
//
// 返回:
//   - string: 交给 sql.DB 或 sql.Tx 执行的 SQL，此处与 query 相同
//   - []any: 与 SQL 对应的执行参数，此处与 args 相同
func (parser DefaultPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	return query, args
}
