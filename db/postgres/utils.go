package postgres

import (
	"reflect"
	"strings"
)

// expandInParameter 展开 IN 或 NOT IN 的集合参数
//
// 参数:
//   - leftPart: string 当前占位符左侧的 SQL
//   - placeholder: string ORM 占位符
//   - param: any 当前参数
//
// 返回:
//   - string: 展开后的占位符片段，未匹配时为空
//   - []any: 展开后的参数，未匹配时为 nil
//   - bool: 是否展开，空集合也为 true
//
// 说明:
//   - 左侧去除空格后须以 IN 结尾，参数须为切片或数组，不限制元素类型
//   - 集合指针会解引用；空集合生成 ()，其他情况按单个参数处理
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
