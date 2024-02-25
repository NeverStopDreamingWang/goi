package model

// 自定义设置
type MySettings map[string]any

// map 数据
type MapData map[string]any

// 迁移处理函数
type MigrationsHandler struct {
	BeforeFunc func() error // 迁移之前处理函数
	AfterFunc  func() error // 迁移之后处理函数
}
