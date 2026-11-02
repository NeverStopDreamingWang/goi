package sqlite3

import (
	"reflect"
	"strings"
)

// whereIn 展开单个 IN 或 NOT IN 条件中的集合参数。
//
// 参数:
//   - leftPart: string 当前占位符前的 SQL 片段
//   - placeholder: string ORM 构建 SQL 时使用的占位符
//   - param: any 当前占位符对应的参数
//
// 返回:
//   - string: 带括号的占位符列表
//   - []any: 展开后的参数
//   - bool: 是否已展开；未匹配时返回 "", nil, false
//
// 说明:
//   - 左侧片段去除首尾空白后须以 IN 结尾，忽略大小写；不解析 SQL 语法
//   - 支持切片、数组及其非 nil 指针，[]byte 不展开；空集合展开为 ()
func whereIn(leftPart string, placeholder string, param any) (string, []any, bool) {
	leftPart = strings.TrimSpace(leftPart)
	if len(leftPart) < 2 || !strings.EqualFold(leftPart[len(leftPart)-2:], "in") {
		return "", nil, false
	}

	paramValue := reflect.ValueOf(param)
	for paramValue.IsValid() && paramValue.Kind() == reflect.Ptr {
		if paramValue.IsNil() {
			return "", nil, false
		}
		paramValue = paramValue.Elem()
	}
	if !paramValue.IsValid() ||
		(paramValue.Kind() != reflect.Slice && paramValue.Kind() != reflect.Array) ||
		paramValue.Kind() == reflect.Slice && paramValue.Type().Elem().Kind() == reflect.Uint8 {
		return "", nil, false
	}

	inPlaceholder := strings.TrimSuffix(
		strings.Repeat(placeholder+",", paramValue.Len()),
		",",
	)
	inPlaceholder = "(" + inPlaceholder + ")"
	inArgs := make([]any, 0, paramValue.Len())
	for i := 0; i < paramValue.Len(); i++ {
		inArgs = append(inArgs, paramValue.Index(i).Interface())
	}
	return inPlaceholder, inArgs, true
}
