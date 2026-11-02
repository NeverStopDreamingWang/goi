package db

import (
	"reflect"
)

// ModelFields 获取模型的结构体字段名和数据库列名
//
// 参数:
//   - modelValue: reflect.Value 经 ModelStructValue 验证的结构体值
//   - parser: ColumnParser 列定义解析器
//
// 返回:
//   - []string: 参与操作的结构体字段名，按声明顺序排列
//   - []string: 与字段名一一对应的数据库列名
func ModelFields(modelValue reflect.Value, parser ColumnParser) ([]string, []string) {
	modelType := modelValue.Type()
	fields := make([]string, 0, modelType.NumField())
	fieldSQL := make([]string, 0, modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		columnName, _, ok := parser.Parse(field)
		if !ok {
			continue
		}

		fields = append(fields, field.Name)
		fieldSQL = append(fieldSQL, columnName)
	}
	return fields, fieldSQL
}

// ModelFieldValue 读取用于写入数据库的模型字段值
//
// 参数:
//   - modelValue: reflect.Value 已验证的结构体值
//   - fieldName: string 结构体字段名
//
// 返回:
//   - any: 原始字段值，nil 指针返回 nil
//   - bool: 字段是否存在且可读取；nil 指针仍返回 true
//
// 说明:
//   - 字段不存在或不可读取时返回 nil, false
//   - 非 nil 指针不解引用，以保留 driver.Valuer 等接口；参数转换交给 database/sql
//   - 是否跳过字段由调用方决定，返回 nil 不表示必须跳过
func ModelFieldValue(modelValue reflect.Value, fieldName string) (any, bool) {
	field := modelValue.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}
	if field.Kind() == reflect.Ptr && field.IsNil() {
		return nil, true
	}
	return field.Interface(), true
}
