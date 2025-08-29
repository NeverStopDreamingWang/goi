package sqlite3

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/model"
)

// SQLite3 模型设置
type SQLite3Settings struct {
	model.MigrationsHandler // 迁移处理函数

	TABLE_NAME string // 表名

	// 自定义配置
	Settings goi.Params
}

// 模型类
type SQLite3Model interface {
	ModelSet() *SQLite3Settings
}

// SQLite3 创建迁移
type SQLite3MakeMigrations struct {
	DATABASES []string
	MODELS    []SQLite3Model
}
