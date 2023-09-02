package hgee

import (
	"context"
	"net/http"
)

type Values map[string][]any

// 根据键获取一个值, 获取不到返回 nil
func (values Values) Get(key string) any {
	if values == nil {
		return nil
	}
	vs := values[key]
	if len(vs) == 0 {
		return nil
	}
	return vs[0]
}

// 设置一个键值，会覆盖原来的值
func (values Values) Set(key, value string) {
	values[key] = []any{value}
}

// 根据键添加一个值
func (values Values) Add(key, value string) {
	values[key] = append(values[key], value)
}

// 删除一个值
func (values Values) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values Values) Has(key string) bool {
	_, ok := values[key]
	return ok
}

type Request struct {
	Object      *http.Request
	Context     context.Context // 请求上下文
	PathParams  Values          // 路由参数
	QueryParams Values          // 查询字符串参数
	BodyParams  Values          // Body 传参
}

// 内置响应数据格式
type Data struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   any    `json:"data"`
}

// 请求响应数据
type Response struct {
	Status int
	Data   any
}
