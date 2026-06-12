package goi

import (
	"os"
	"time"

	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

type Language string

// 框架语言
const (
	ZhCN Language = "zh-CN" // 中文-简体
	EnUS Language = "en-US" // 英文-美国
)

// SSL
type SSL struct {
	Enabled  bool
	Type     string
	CertPath string
	KeyPath  string
}

// 项目设置
type settings struct {
	Debug       bool                 // 是否开启 Debug 模式
	Network     string               // 网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	BindAddress string               // 监听地址
	Port        uint16               // 服务端口
	BindDomain  string               // 绑定域名
	BaseDir     string               // 项目根路径
	SecretKey   string               // 项目 AES 密钥
	PrivateKey  string               // 项目 RSA 私钥
	PublicKey   string               // 项目 RSA 公钥
	SSL         SSL                  // SSL
	Databases   map[string]*Database // 数据库配置

	// TIMEZONE
	UseTZ    bool           // UseTZ=true: 返回 GetLocation() 时区的时间 “有感知时区”；UseTZ=false: 返回 GetLocation() 时区的时间，但时区标注为 UTC “无感知时区”，避免任何时区换算，直存直取
	timeZone string         // 地区时区默认为空，本地时区
	location *time.Location // 地区时区
	language Language       // 项目语言

	// 自定义设置
	Params Params
}

func newSettings() *settings {
	i18n.SetLocalize(string(ZhCN))
	return &settings{
		Debug:       true,
		Network:     "tcp",
		BindAddress: "127.0.0.1",
		Port:        8080,
		BindDomain:  "",
		BaseDir:     "",
		SecretKey:   "",
		PrivateKey:  "",
		PublicKey:   "",
		SSL:         SSL{},
		Databases:   make(map[string]*Database),

		// TIMEZONE
		UseTZ:    true,
		timeZone: "",
		location: time.Local,
		language: ZhCN,

		Params: make(Params),
	}
}

// 设置时区
//
// 参数:
//   - timeZone string: 时区
func (self *settings) SetTimeZone(timeZone string) error {
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return err
	}
	self.timeZone = timeZone
	self.location = location

	if self.UseTZ == true {
		// 同时设置系统环境变量，确保子进程和其他组件使用相同时区
		err = os.Setenv("TZ", self.timeZone)
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
	return self.timeZone
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
//   - lang Language: 语言 ZhCN、EnUS
func (self settings) SetLanguage(lang Language) {
	self.language = lang
	i18n.SetLocalize(string(self.language))
}

// 获取代码语言
//
// 返回:
//   - Language: 语言 ZhCN、EnUS
func (self settings) GetLanguage() Language {
	return self.language
}
