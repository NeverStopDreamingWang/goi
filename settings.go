package hgee

import (
	"time"
)

// 数据库
type MetaDataBase struct {
	ENGINE       string
	NAME         string
	USER         string
	PASSWORD     string
	HOST         string
	PORT         uint16
	SERVICE_NAME string
}

// 项目设置
type metaSettings struct {
	SERVER_ADDRESS string                  // 服务地址
	SERVER_PORT    uint16                  // 服务端口
	BASE_DIR       string                  // 项目根路径
	SECRET_KEY     string                  // 项目密钥
	DATABASES      map[string]MetaDataBase // 数据库配置
	TIME_ZONE      string                  // 时区
	LOCATION       *time.Location          // 地区时区

	// 自定义设置
	mySettings map[string]any
}

func newSettings() *metaSettings {
	return &metaSettings{
		SERVER_ADDRESS: "127.0.0.1",
		SERVER_PORT:    8000,
		BASE_DIR:       "",
		SECRET_KEY:     "",
		DATABASES:      make(map[string]MetaDataBase),
		TIME_ZONE:      "Asia/Shanghai",
		LOCATION:       nil,
		mySettings:     make(map[string]any),
	}
}

// 设置自定义配置
func (settings *metaSettings) Set(name string, value any) {
	settings.mySettings[name] = value
}

// 获取自定义配置
func (settings metaSettings) Get(name string) any {
	return settings.mySettings[name]
}
