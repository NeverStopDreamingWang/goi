package goi_test

import "github.com/NeverStopDreamingWang/goi"

func ExampleRegisterConverter() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter("my_phone", `(1[3456789]\d{9})`)

	// 邮箱
	goi.RegisterConverter("my_email", `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

	// URL
	goi.RegisterConverter("my_url", `(https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|www\.[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9]+\.[^\s]{2,}|www\.[a-zA-Z0-9]+\.[^\s]{2,})`)

	// 日期 (YYYY-MM-DD)
	goi.RegisterConverter("my_date", `(\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))`)

	// 时间 (HH:MM:SS)
	goi.RegisterConverter("my_time", `((?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d)`)

	// IP地址 (IPv4)
	goi.RegisterConverter("my_ipv4", `((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`)

	// 用户名 (字母开头，允许字母数字下划线，4-16位)
	goi.RegisterConverter("my_username", `([a-zA-Z][a-zA-Z0-9_]{3,15})`)
}
