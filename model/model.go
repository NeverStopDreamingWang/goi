package model

// 自定义设置
type MySettings map[string]interface{}

// map 数据
type MapData map[string]interface{}

// 迁移处理函数
type MigrationsHandler struct {
	BeforeHandler func() error // 迁移之前处理函数
	AfterHandler  func() error // 迁移之后处理函数
}
