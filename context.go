package goi

import (
	"context"
	"net/http"
)

type Request struct {
	Object      *http.Request
	Context     context.Context // 请求上下文
	PathParams  metaValues      // 路由参数
	QueryParams metaValues      // 查询字符串参数
	BodyParams  metaValues      // Body 传参
}

// 请求响应数据
type Response struct {
	Status int
	Data   any
}

// 内置响应数据格式
type Data struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   any    `json:"data"`
}

type metaValues map[string][]any

// 根据键获取一个值, 获取不到返回 nil
func (values metaValues) Get(key string) any {
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
func (values metaValues) Set(key, value string) {
	values[key] = []any{value}
}

// 根据键添加一个值
func (values metaValues) Add(key, value string) {
	values[key] = append(values[key], value)
}

// 删除一个值
func (values metaValues) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values metaValues) Has(key string) bool {
	_, ok := values[key]
	return ok
}
