package goi

import (
	"time"
)

// SSL
type MetaSSL struct {
	STATUS    bool
	CERT_PATH string
	KEY_PATH  string
	CSR_PATH  string
}

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
	NET_WORK   string                  // 网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	ADDRESS    string                  // 服务地址
	PORT       uint16                  // 服务端口
	BASE_DIR   string                  // 项目根路径
	SECRET_KEY string                  // 项目密钥
	SSL        MetaSSL                 // SSL
	DATABASES  map[string]MetaDataBase // 数据库配置
	TIME_ZONE  string                  // 时区
	LOCATION   *time.Location          // 地区时区

	// 自定义设置
	mySettings map[string]any
}

func newSettings() *metaSettings {
	return &metaSettings{
		NET_WORK:   "tcp",
		ADDRESS:    "127.0.0.1",
		PORT:       8080,
		BASE_DIR:   "",
		SECRET_KEY: "",
		SSL:        MetaSSL{},
		DATABASES:  make(map[string]MetaDataBase),
		TIME_ZONE:  "Asia/Shanghai",
		LOCATION:   time.Now().Location(),
		mySettings: make(map[string]any),
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
