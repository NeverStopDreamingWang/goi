package goi

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// SSL
type MetaSSL struct {
	STATUS    bool
	TYPE      string
	CERT_PATH string
	KEY_PATH  string
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
	NET_WORK     string                  // 网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	BIND_ADDRESS string                  // 监听地址
	PORT         uint16                  // 服务端口
	BIND_DOMAIN  string                  // 绑定域名
	BASE_DIR     string                  // 项目根路径
	SECRET_KEY   string                  // 项目 AES 密钥
	PRIVATE_KEY  string                  // 项目 RSA 私钥
	PUBLIC_KEY   string                  // 项目 RSA 公钥
	SSL          MetaSSL                 // SSL
	DATABASES    map[string]MetaDataBase // 数据库配置
	TIME_ZONE    string                  // 时区
	LOCATION     *time.Location          // 地区时区

	// 自定义设置
	mySettings map[string]interface{}
}

func newSettings() *metaSettings {
	return &metaSettings{
		NET_WORK:     "tcp",
		BIND_ADDRESS: "127.0.0.1",
		PORT:         8080,
		BIND_DOMAIN:  "",
		BASE_DIR:     "",
		SECRET_KEY:   "",
		PRIVATE_KEY:  "",
		PUBLIC_KEY:   "",
		SSL:          MetaSSL{},
		DATABASES:    make(map[string]MetaDataBase),
		TIME_ZONE:    "Asia/Shanghai",
		LOCATION:     time.Now().Location(),
		mySettings:   make(map[string]interface{}),
	}
}

// 设置自定义配置
func (settings *metaSettings) Set(key string, value interface{}) {
	settings.mySettings[key] = value
}

// 获取自定义配置
func (settings metaSettings) Get(key string, dest interface{}) error {
	// 获取目标变量的反射值
	destValue := reflect.ValueOf(dest)
	// 检查目标变量是否为指针类型
	if destValue.Kind() != reflect.Ptr {
		return errors.New("参数必须是指针类型")
	}
	destValue = destValue.Elem()
	// 检查目标变量的可设置性
	if !destValue.CanSet() {
		return errors.New("目标变量不可设置")
	}

	value, ok := settings.mySettings[key]
	if ok == false {
		return errors.New(fmt.Sprintf("%s 不存在", key))
	}
	// 获取值的类型
	valueType := reflect.TypeOf(value)

	destKind := destValue.Type().Kind()
	valueKind := valueType.Kind()
	if destKind != valueKind {
		return errors.New(fmt.Sprintf("无法将 %s 类型的值赋给 %s 类型的 dest 变量", valueKind.String(), destKind.String()))
	}
	switch valueKind {
	case reflect.String:
		destValue.SetString(reflect.ValueOf(value).String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		destValue.SetInt(reflect.ValueOf(value).Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		destValue.SetUint(reflect.ValueOf(value).Uint())
	case reflect.Float32, reflect.Float64:
		destValue.SetFloat(reflect.ValueOf(value).Float())
	case reflect.Slice, reflect.Map:
		destValue.Set(reflect.ValueOf(value))
	default:
		return errors.New("不支持的目标变量类型")
	}
	return nil
}
