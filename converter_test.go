package goi_test

import (
	"github.com/NeverStopDreamingWang/goi"
)

// 手机号
var phoneConverter = goi.Converter{
	Regex: `(1[3456789]\d{9})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 邮箱
var emailConverter = goi.Converter{
	Regex: `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// URL
var urlConverter = goi.Converter{
	Regex: `(https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|www\.[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9]+\.[^\s]{2,}|www\.[a-zA-Z0-9]+\.[^\s]{2,})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 日期 (YYYY-MM-DD)
var dateConverter = goi.Converter{
	Regex: `(\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 时间 (HH:MM:SS)
var timeConverter = goi.Converter{
	Regex: `((?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d)`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// IP地址 (IPv4)
var ipv4Converter = goi.Converter{
	Regex: `((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 用户名 (字母开头，允许字母数字下划位)
var usernameConverter = goi.Converter{
	Regex: `([a-zA-Z][a-zA-Z0-9_]{3,15})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

func ExampleRegisterConverter() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter(phoneConverter, "my_phone")

	// 邮箱
	goi.RegisterConverter(emailConverter, "my_email")

	// URL
	goi.RegisterConverter(urlConverter, "my_url")

	// 日期 (YYYY-MM-DD)
	goi.RegisterConverter(dateConverter, "my_date")

	// 时间 (HH:MM:SS)
	goi.RegisterConverter(timeConverter, "my_time")

	// IP地址 (IPv4)
	goi.RegisterConverter(ipv4Converter, "my_ipv4")

	// 用户名 (字母开头，允许字母数字下划位)
	goi.RegisterConverter(usernameConverter, "my_username")
}
