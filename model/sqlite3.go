package model

// SQLite3 模型设置
type SQLite3Settings struct {
	MigrationsHandler // 迁移处理函数

	TABLE_NAME string // 表名

	// 自定义配置
	MySettings MySettings
}

// 设置自定义配置
func (modelsettings *SQLite3Settings) Set(name string, value interface{}) {
	modelsettings.MySettings[name] = value
}

// 获取自定义配置
func (modelsettings SQLite3Settings) Get(name string) interface{} {
	return modelsettings.MySettings[name]
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
