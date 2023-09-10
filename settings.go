package hgee

// 数据库
type DataBase struct {
	ENGINE   string
	NAME     string
	USER     string
	PASSWORD string
	HOST     string
	PORT     uint16
}

// 项目设置
type metaSettings struct {
	version        string              // 当前版本
	SERVER_ADDRESS string              // 服务地址
	SERVER_PORT    uint16              // 服务端口
	BASE_DIR       string              // 项目根路径
	SECRET_KEY     string              // 项目密钥
	DATABASES      map[string]DataBase // 数据库配置

	userDefinedSettings map[string]any // 自定义设置
}

func newSettings() *metaSettings {
	return &metaSettings{
		version:        "1.0.5",
		SERVER_ADDRESS: "127.0.0.1",
		SERVER_PORT:    8000,
		BASE_DIR:       "",
		SECRET_KEY:     "",
		DATABASES:      make(map[string]DataBase),

		userDefinedSettings: make(map[string]any),
	}
}

// 设置配置
func (settings *metaSettings) Set(name string, value any) {
	settings.userDefinedSettings[name] = value
}

// 获取配置
func (settings metaSettings) Get(name string) any {
	return settings.userDefinedSettings[name]
}
