package goi

import (
	"strconv"
)

type Converter struct {
	Regex string
	// 将 URL 片段转换为 Go 值
	ToGo func(string) (interface{}, error)
}

// 内置转换器
// intConverter 用于匹配整数类型的URL参数
// 匹配模式: 一个或多个数字字符 (0-9)
// 示例: 123, 456789
var intConverter = Converter{
	Regex: `([0-9]+)`,
	ToGo: func(value string) (interface{}, error) {
		iv, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		return iv, nil
	},
}

// stringConverter 用于匹配字符串类型的URL参数
// 匹配模式: 除了斜杠(/)外的任意字符序列
// 示例: hello, user-123, article_title
var stringConverter = Converter{
	Regex: `([^/]+)`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// slugConverter 用于匹配URL友好的字符串参数
// 匹配模式: 由字母、数字、连字符和下划线组成的字符串
// 示例: my-blog-post, user_profile_123
var slugConverter = Converter{
	Regex: `([-a-zA-Z0-9_]+)`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// pathConverter 用于匹配完整路径类型的URL参数
// 匹配模式: 任意字符序列，包括斜杠
// 示例: /blog/2024/03, /users/photos/vacation
var pathConverter = Converter{
	Regex: `(.+)`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// uuidConverter 用于匹配UUID格式的URL参数
// 匹配模式: 标准的UUID格式 (8-4-4-4-12位十六进制数)
// 示例: 550e8400-e29b-41d4-a716-446655440000
var uuidConverter = Converter{
	Regex: `([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// metaConverter 存储URL路径参数的转换器映射
// key: 转换器名称
// value: 转换器
var metaConverter = map[string]Converter{
	"int":    intConverter,    // 匹配整数，如: 123
	"string": stringConverter, // 匹配除了'/'外的任意字符，如: hello
	"slug":   slugConverter,   // 匹配URL友好的字符串，如: my-post-title
	"path":   pathConverter,   // 匹配任意路径，如: /blog/2024/03
	"uuid":   uuidConverter,   // 匹配UUID格式，如: 550e8400-e29b-41d4-a716-446655440000
}

// RegisterConverter 注册自定义URL参数转换器
//
// 参数:
//   - converter Converter: 转换器
func RegisterConverter(converter Converter, type_name string) {
	metaConverter[type_name] = converter
}

// GetConverters 获取所有转换器
func GetConverters() map[string]Converter {
	return metaConverter
}
