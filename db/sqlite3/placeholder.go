package sqlite3

import "github.com/NeverStopDreamingWang/goi/v2/db"

// placeholderParser 保存后续新建 SQLite3 Engine 使用的占位符解析器。
var placeholderParser db.PlaceholderParser = db.NewPlaceholderParser("?")

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
