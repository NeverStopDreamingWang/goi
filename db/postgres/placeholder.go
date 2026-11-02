package postgres

import "github.com/NeverStopDreamingWang/goi/v2/db"

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

// Parse 返回交给 sql.DB 或 sql.Tx 执行的 SQL，将 ? 转为 $1、$2 等，参数顺序不变
func (defaultPlaceholderParser) Parse(query string, args ...any) string {
	return convertPlaceholders(query, len(args))
}
