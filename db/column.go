package db

import (
	"reflect"
	"strings"
)

// ColumnParser 解析模型字段的列名、列类型及是否参与操作，可自行实现以替换默认规则。
type ColumnParser interface {
	// Parse 解析单个模型字段的列定义
	//
	// 参数:
	//   - field: reflect.StructField 模型字段
	//
	// 返回:
	//   - columnName: string 数据库列名，由调用方处理标识符引用
	//   - columnType: string 列类型及约束，供迁移和验证使用
	//   - ok: bool 是否参与数据库操作
	Parse(field reflect.StructField) (columnName string, columnType string, ok bool)
}

// columnParser 保存各数据库包共用的默认列解析器。
var columnParser ColumnParser = NewColumnParser("field_name", "field_type")

// SetColumnParser 设置公共默认列解析器
//
// 参数:
//   - parser: ColumnParser 默认解析器；nil 时忽略，保留当前配置
//
// 说明:
//   - 初始化阶段配置，不可与 Connect、ConnectDatabase 或 Validate 并发调用
//   - 仅影响后续创建的 Engine 和 Validate；数据库包或连接级配置优先，已有 Engine 不变
//   - 不深拷贝解析器；注册后不应修改共享状态，其并发安全由实现者负责
func SetColumnParser(parser ColumnParser) {
	if parser == nil {
		return
	}
	columnParser = parser
}

// GetColumnParser 获取公共默认列解析器，未设置时使用 field_name 和 field_type 标签。
// 返回值非 nil；仅在数据库包和连接均未覆盖时作为默认配置。
func GetColumnParser() ColumnParser {
	return columnParser
}

// DefaultColumnParser 按标签解析列定义，标签键在创建时固定。
//
// 说明:
//   - 列类型标签缺失、为空白或去除首尾空白后为 - 时忽略该字段
//   - 列名标签缺失时使用小写字段名；标签存在时原样使用，包括空值
type DefaultColumnParser struct {
	nameTagKey string
	typeTagKey string
}

// NewColumnParser 创建使用指定标签键的列解析器
//
// 参数:
//   - nameTagKey: string 列名标签键，例如 field_name
//   - typeTagKey: string 列类型标签键，例如 field_type
//
// 返回:
//   - DefaultColumnParser: 标签键固定的列解析器
func NewColumnParser(nameTagKey string, typeTagKey string) DefaultColumnParser {
	return DefaultColumnParser{
		nameTagKey: nameTagKey,
		typeTagKey: typeTagKey,
	}
}

// Parse 根据字段标签解析列定义
//
// 参数:
//   - field: reflect.StructField 模型结构体字段
//
// 返回:
//   - columnName: string 列名，列名标签缺失时使用小写字段名
//   - columnType: string 列类型标签的原始值
//   - ok: bool 列类型标签缺失、为空白或为 - 时返回 false
//
// 说明:
//   - 判断空白和 - 时忽略首尾空白，有效类型仍原样返回
//   - 不添加标识符定界符，不校验或转换列类型
func (parser DefaultColumnParser) Parse(field reflect.StructField) (columnName string, columnType string, ok bool) {
	columnType, ok = field.Tag.Lookup(parser.typeTagKey)
	trimmedColumnType := strings.TrimSpace(columnType)
	if !ok || trimmedColumnType == "" || trimmedColumnType == "-" {
		return "", "", false
	}
	columnName, hasColumnName := field.Tag.Lookup(parser.nameTagKey)
	if !hasColumnName {
		columnName = strings.ToLower(field.Name)
	}
	return columnName, columnType, true
}
