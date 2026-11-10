package sqlserver

import (
	"reflect"
	"strings"
)

// expandInParameter 展开 IN 或 NOT IN 的集合参数
//
// 参数:
//   - leftPart: string 当前占位符左侧的 SQL 片段
//   - placeholder: string ORM 占位符
//   - param: any 当前参数
//
// 返回:
//   - string: 展开后的占位符；不匹配时为空
//   - []any: 展开后的参数
//   - bool: 是否展开，空集合也为 true
//
// 说明:
//   - 仅处理 IN 占位符形式，已带括号的写法不展开
//   - 支持切片、数组及其非 nil 指针，不限制元素类型
//   - 空集合展开为 ()；不匹配时返回空字符串、nil、false
func expandInParameter(leftPart string, placeholder string, param any) (string, []any, bool) {
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
	if !paramValue.IsValid() || (paramValue.Kind() != reflect.Slice && paramValue.Kind() != reflect.Array) {
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
