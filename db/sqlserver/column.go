package sqlserver

import "github.com/NeverStopDreamingWang/goi/v2/db"

// columnParser 保存本包覆盖，nil 时使用 db 的默认规则。
var columnParser db.ColumnParser

// SetColumnParser 设置本包的列解析器
//
// 参数:
//   - parser: db.ColumnParser 本包覆盖；nil 时保持原配置
//
// 说明:
//   - 优先级为连接级 EngineOptions.ColumnParser > 本包覆盖 > db 默认规则
//   - 仅在初始化阶段配置，不可与连接或校验操作并发调用
//   - 不深拷贝解析器，注册后不应修改共享状态
func SetColumnParser(parser db.ColumnParser) {
	if parser == nil {
		return
	}
	columnParser = parser
}

// GetColumnParser 获取当前生效的列解析器
//
// 返回:
//   - db.ColumnParser: 本包非 nil 覆盖，否则返回 db.GetColumnParser()
//
// 说明:
//   - Engine 在创建时保存解析器，已有实例不受后续默认配置替换影响
//   - 独立 Validate 在每次调用开始时获取当前规则
func GetColumnParser() db.ColumnParser {
	if columnParser != nil {
		return columnParser
	}
	return db.GetColumnParser()
}
