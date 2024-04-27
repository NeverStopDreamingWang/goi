package goi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 内置响应数据格式
type Data struct {
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

type Request struct {
	Object      *http.Request
	Context     context.Context // 请求上下文
	PathParams  metaValues      // 路由参数
	QueryParams metaValues      // 查询字符串参数
	BodyParams  metaValues      // Body 传参
}

// 解析请求参数
func (request *Request) parseRequestParams() {
	var err error
	// 解析 Query 参数
	for name, values := range request.Object.URL.Query() {
		request.QueryParams[name] = append(request.QueryParams[name], values...)
	}

	// 解析 Body 参数
	err = request.Object.ParseMultipartForm(32 << 20)
	if err != nil {
		err = request.Object.ParseForm()
		if err != nil {
			panic(fmt.Sprintf("解析 Body 错误: %v\n", err))
		}
	}
	for name, values := range request.Object.Form {
		request.BodyParams[name] = append(request.BodyParams[name], values...)
	}

	// 解析 json 数据
	body, err := io.ReadAll(request.Object.Body)
	if err != nil {
		panic(fmt.Sprintf("读取 Body 错误: %v\n", err))
	}
	if len(body) != 0 {
		jsonData := make(map[string]interface{})
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			panic(fmt.Sprintf("解析 json 错误: %v\n", err))
		}
		for name, value := range jsonData {
			stringValue, ok := value.(string)
			if !ok {
				// 如果无法转换为字符串，则使用 fmt.Sprint 将其转换为字符串
				stringValue = fmt.Sprint(value)
			}
			request.BodyParams[name] = append(request.BodyParams[name], stringValue)
		}
	}
}

// 请求响应数据
type Response struct {
	Status int
	Data   interface{}
}
