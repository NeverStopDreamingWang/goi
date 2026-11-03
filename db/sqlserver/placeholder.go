package sqlserver

import "github.com/NeverStopDreamingWang/goi/v3/db"

// placeholderParser 保存后续新建 SQL Server Engine 使用的占位符解析器。
var placeholderParser db.PlaceholderParser = defaultPlaceholderParser{}

// SetPlaceholderParser 设置全局占位符解析器
//
// 参数:
//   - parser: db.PlaceholderParser 默认或自定义占位符解析器；nil 时忽略，保留原配置
//
// 说明:
//   - 仅影响新建 Engine，已有 Engine 继续使用原对象
//   - 连接前完成配置，不可与连接操作并发修改
//   - 注册后不应修改解析器；自定义实现负责内部并发安全
func SetPlaceholderParser(parser db.PlaceholderParser) {
	if parser == nil {
		return
	}
	placeholderParser = parser
}

// GetPlaceholderParser 获取当前全局占位符解析器
//
// 返回:
//   - db.PlaceholderParser: 当前占位符解析器
//
// 说明:
//   - 默认使用 ?，执行前依次转换为 @p1、@p2 等
//   - 默认按字符替换，不识别 SQL 字面量和注释中的 ?
func GetPlaceholderParser() db.PlaceholderParser {
	return placeholderParser
}

// defaultPlaceholderParser 保留 SQL Server 的顺序编号占位符规则。
type defaultPlaceholderParser struct{}

// Placeholder 返回 ORM 构建 SQL 使用的 ? 占位符
func (defaultPlaceholderParser) Placeholder() string {
	return "?"
}

// Parse 转换占位符，返回交给 sql.DB 或 sql.Tx 执行的 SQL；参数顺序不变
func (defaultPlaceholderParser) Parse(query string, args ...any) string {
	return convertPlaceholders(query, len(args))
}
