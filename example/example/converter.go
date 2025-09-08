package example

import (
	"github.com/NeverStopDreamingWang/goi"
)

// 手机号
var phoneConverter = goi.Converter{
	Regex: `(1[3456789]\d{9})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

func init() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter(phoneConverter, "phone")
}
