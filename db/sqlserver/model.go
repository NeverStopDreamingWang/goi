package sqlserver

import "github.com/NeverStopDreamingWang/goi"

// 迁移处理函数
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// SQL Server 模型设置
type Settings struct {
	MigrationsHandler // 迁移处理函数

	TABLE_NAME string // 表名

	// 自定义配置
	Settings goi.Params
}

// 模型接口
type Model interface {
	ModelSet() *Settings
}
