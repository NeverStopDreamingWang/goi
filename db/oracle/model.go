package oracle

import "github.com/NeverStopDreamingWang/goi"

// 迁移处理函数
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// Oracle 模型设置
// 尽量与 mysql/sqlite3/postgresql 的 Settings 保持结构风格一致
//
// - TABLE_NAME: 表名
// - SCHEMA: 可选 schema 名（如果留空则由具体引擎决定是否使用默认）
// - Settings: 自定义扩展配置（例如 encrypt_fields 等）
type Settings struct {
	MigrationsHandler // 迁移处理函数

	TABLE_NAME string // 表名

	// 自定义配置
	Settings goi.Params
}

// 模型接口
// 每个模型需要实现 ModelSet 返回自身的 Settings
// 以便引擎在迁移和构造 SQL 时读取元信息
type Model interface {
	ModelSet() *Settings
}
