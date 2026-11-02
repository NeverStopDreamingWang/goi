package postgres

import "github.com/NeverStopDreamingWang/goi/v2"

// MigrationsHandler 设置新建表前后的回调
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// Settings 定义模型的 Schema、表名和迁移配置
type Settings struct {
	MigrationsHandler // 迁移处理函数

	Schema    string // Schema 名称，空时使用 public
	TableName string // 表名

	// 自定义配置
	Settings goi.Params
}

// Model 提供模型的数据库配置
type Model interface {
	ModelSet() *Settings
}
