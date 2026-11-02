package oracle

import (
	"fmt"
	"reflect"
	"strings"
)

// whereIn 展开 IN 或 NOT IN 的集合参数
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
//   - 支持切片、数组及其非 nil 指针；[]byte 和其他参数不展开
//   - 空集合展开为 ()；不匹配时返回空字符串、nil、false
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

// convertPlaceholders 将 ? 依次转换为 :1、:2 等占位符
//
// 参数:
//   - query: string 使用 ? 的 SQL
//   - argsCount: int 参数数量，也是最多替换的次数
//
// 返回:
//   - string: 转换后的 SQL，无参数时原样返回
//
// 说明:
//   - 不解析 SQL 语法，字面量和注释中的 ? 也会被替换
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
		// 写入 :N 占位符
		builder.WriteString(fmt.Sprintf(":%d", index))
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
