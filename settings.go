package goi

import (
	"os"
	"time"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

type Language string

// 框架语言
const (
	ZH_CN Language = "zh-CN" // 中文-简体
	EN_US Language = "en-US" // 英文-美国
)

// SSL
type SSL struct {
	STATUS    bool
	TYPE      string
	CERT_PATH string
	KEY_PATH  string
}

// 项目设置
type settings struct {
	NET_WORK     string               // 网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	BIND_ADDRESS string               // 监听地址
	PORT         uint16               // 服务端口
	BIND_DOMAIN  string               // 绑定域名
	BASE_DIR     string               // 项目根路径
	SECRET_KEY   string               // 项目 AES 密钥
	PRIVATE_KEY  string               // 项目 RSA 私钥
	PUBLIC_KEY   string               // 项目 RSA 公钥
	SSL          SSL                  // SSL
	DATABASES    map[string]*DataBase // 数据库配置

	// TIMEZONE
	USE_TZ    bool           // USE_TZ=true: 返回 GetLocation() 时区的时间 “有感知时区”；USE_TZ=false: 返回 GetLocation() 时区的时间，但时区标注为 UTC “无感知时区”，避免任何时区换算，直存直取
	time_zone string         // 地区时区默认为空，本地时区
	location  *time.Location // 地区时区
	language  Language       // 项目语言

	// 自定义设置
	Params Params
}

func newSettings() *settings {
	i18n.SetLocalize(string(ZH_CN))
	return &settings{
		NET_WORK:     "tcp",
		BIND_ADDRESS: "127.0.0.1",
		PORT:         8080,
		BIND_DOMAIN:  "",
		BASE_DIR:     "",
		SECRET_KEY:   "",
		PRIVATE_KEY:  "",
		PUBLIC_KEY:   "",
		SSL:          SSL{},
		DATABASES:    make(map[string]*DataBase),

		// TIMEZONE
		USE_TZ:    true,
		time_zone: "",
		location:  time.Local,
		language:  ZH_CN,

		Params: make(Params),
	}
}

// 设置时区
//
// 参数:
//   - time_zone string: 时区
func (self *settings) SetTimeZone(time_zone string) error {
	location, err := time.LoadLocation(time_zone)
	if err != nil {
		return err
	}
	self.time_zone = time_zone
	self.location = location

	if self.USE_TZ == true {
		// 同时设置系统环境变量，确保子进程和其他组件使用相同时区
		err = os.Setenv("TZ", self.time_zone)
		if err != nil {
			return err
		}
	} else {
		err = os.Setenv("TZ", "UTC")
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取时区
//
// 返回:
//   - string: 时区
func (self settings) GetTimeZone() string {
	return self.time_zone
}

// 获取时区
//
// 返回:
//   - *time.Location: 时区 Location
func (self settings) GetLocation() *time.Location {
	return self.location
}

// 设置代码语言
//
// 参数:
//   - lang Language: 语言 ZH_CN、EN_US
func (self settings) SetLanguage(lang Language) {
	self.language = lang
	i18n.SetLocalize(string(self.language))
}

// 获取代码语言
//
// 返回:
//   - Language: 语言 ZH_CN、EN_US
func (self settings) GetLanguage() Language {
	return self.language
}
