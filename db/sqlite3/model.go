package sqlite3

import (
	"github.com/NeverStopDreamingWang/goi/v2"
)

// MigrationsHandler 定义建表前后的回调。
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// Settings 定义 SQLite3 表、迁移及自定义配置。
type Settings struct {
	MigrationsHandler // 迁移处理函数

	Schema    string // 附加数据库名称，例如 main、temp；空时不限定数据库
	TableName string // 表名

	// 自定义配置
	Settings goi.Params
}

// Model 提供 SQLite3 模型的表及迁移配置。
type Model interface {
	// ModelSet 返回当前模型的配置。
	ModelSet() *Settings
}
