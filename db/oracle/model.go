package oracle

import "github.com/NeverStopDreamingWang/goi/v3"

// MigrationsHandler 定义建表前后的回调
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}

// Settings 定义 Oracle 模型设置
type Settings struct {
	MigrationsHandler // 迁移处理函数

	Schema    string // Schema（Owner）名称，空时使用当前 Schema；保留原始大小写
	TableName string // 表名

	// 自定义配置
	Settings goi.Params
}

// Model 提供表名、Schema 和迁移设置
type Model interface {
	// ModelSet 返回当前模型的设置
	ModelSet() *Settings
}
