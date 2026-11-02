package postgres

import (
	"fmt"
	"reflect"
	"strings"
)

// whereIn 展开 IN 或 NOT IN 的集合参数
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
//   - 左侧去除空格后须以 IN 结尾，参数须为切片或数组（字节切片除外）
//   - 集合指针会解引用；空集合生成 ()，其他情况由 Where 按单个参数处理
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

// convertPlaceholders 将 ? 转为 $1、$2 等驱动占位符
//
// 参数:
//   - query: string 原始 SQL
//   - argsCount: int 参数个数，最多替换相同数量的占位符
//
// 返回:
//   - string: 转换后的 SQL
//
// 说明:
//   - 按字符替换，不识别 SQL 字符串或注释中的 ?，也不校验占位符数量
func convertPlaceholders(query string, argsCount int) string {
	if argsCount == 0 || query == "" {
		return query
	}
	if !strings.Contains(query, "?") {
		return query
	}

	var (
		builder strings.Builder
		index   = 1
		start   = 0
	)

	for start < len(query) && index <= argsCount {
		pos := strings.IndexByte(query[start:], '?')
		if pos < 0 {
			break
		}
		pos += start

		// 写入 ? 之前的部分
		builder.WriteString(query[start:pos])
		// 写入 $N 占位符
		builder.WriteString(fmt.Sprintf("$%d", index))
		index++
		start = pos + 1
	}

	// 追加剩余部分
	if start < len(query) {
		builder.WriteString(query[start:])
	}

	// 如果没有发生替换则返回原始字符串
	if builder.Len() == 0 {
		return query
	}
	return builder.String()
}
