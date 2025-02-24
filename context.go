package goi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Data 自定义标准的API响应格式
//
// 字段:
//   - Code int: 响应状态码
//   - Message string: 响应消息
//   - Results interface{}: 响应数据
type Data struct {
	Code    int         `json:"code"`    // 响应状态码
	Message string      `json:"message"` // 响应消息
	Results interface{} `json:"results"` // 响应数据
}

// Request 封装HTTP请求相关信息
//
// 字段:
//   - Object *http.Request: 原始HTTP请求对象
//   - PathParams ParamsValues: 路由参数
//
// 用于在处理请求时提供统一的访问接口
type Request struct {
	Object     *http.Request // 原始HTTP请求对象
	PathParams ParamsValues  // 路由参数
}

// QueryParams 解析并返回URL查询字符串参数
// 匹配模式: URL中的查询参数，格式为 ?key=value&key2=value2
// 示例: /api/users?page=1&size=10
// 返回一个ParamsValues类型的map，包含所有查询参数
//
// 返回:
//   - ParamsValues: URL查询参数的键值对映射，其中值为字符串切片
func (request *Request) QueryParams() ParamsValues {
	var QueryParams ParamsValues = make(ParamsValues)
	// 解析 Query 参数
	for name, values := range request.Object.URL.Query() {
		QueryParams[name] = append(QueryParams[name], values...)
	}
	return QueryParams
}

// BodyParams 解析并返回请求体中的参数
// 支持以下格式:
// - multipart/form-data
// - application/x-www-form-urlencoded
// - application/json
// 示例: {"name": "user", "age": 25}
// 返回一个BodyParamsValues类型的map，包含所有请求体参数
//
// 返回:
//   - BodyParamsValues: 请求体参数的键值对映射，支持多种格式的请求体解析
func (request *Request) BodyParams() BodyParamsValues {
	var err error
	var bodyParams BodyParamsValues = make(BodyParamsValues)
	// 解析 Body 参数
	err = request.Object.ParseMultipartForm(32 << 20)
	if err != nil {
		err = request.Object.ParseForm()
		if err != nil {
			parseBodyErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "context.parse_body_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			Log.Error(parseBodyErrorMsg)
			panic(parseBodyErrorMsg)
		}
	}
	for name, values := range request.Object.Form {
		for _, v := range values {
			bodyParams[name] = append(bodyParams[name], v)
		}
	}

	// 解析 json 数据
	body, err := io.ReadAll(request.Object.Body)
	if err != nil {
		readBodyErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "context.read_body_error",
			TemplateData: map[string]interface{}{
				"err": err,
			},
		})
		Log.Error(readBodyErrorMsg)
		panic(readBodyErrorMsg)
	}
	if len(body) != 0 {
		jsonData := make(map[string]interface{})
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			unmarshalBodyErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "context.unmarshal_body_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			Log.Error(unmarshalBodyErrorMsg)
			panic(unmarshalBodyErrorMsg)
		}
		for name, value := range jsonData {
			bodyParams[name] = append(bodyParams[name], value)
		}
	}
	return bodyParams
}

// WithContext 更新请求对象中的上下文信息
//
// 参数:
//   - ctx context.Context: 新的上下文对象
func (request *Request) WithContext(ctx context.Context) {
	// 创建一个新的 http.Request 并用新的上下文更新它
	request.Object = request.Object.WithContext(ctx)
}

// customResponseWriter 自定义响应写入器
//
// 字段:
//   - ResponseWriter http.ResponseWriter: 标准响应写入器
//   - StatusCode int: 响应状态码
//   - Bytes int64: 已写入的字节数
//
// 用于请求日志记录和响应监控
type customResponseWriter struct {
	http.ResponseWriter       // 内嵌http.ResponseWriter接口
	StatusCode          int   // 响应状态码
	Bytes               int64 // 已写入的字节数
}

// WriteHeader 设置HTTP响应状态码
//
// 参数:
//   - code int: HTTP状态码
func (w *customResponseWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write 写入响应数据
//
// 参数:
//   - b []byte: 要写入的数据
//
// 返回:
//   - int: 已写入的字节数
//   - error: 写入过程中的错误信息
func (w *customResponseWriter) Write(b []byte) (int, error) {
	bytesWritten, err := w.ResponseWriter.Write(b)
	w.Bytes += int64(bytesWritten)
	return bytesWritten, err
}

// Response 定义处理程序的响应格式
//
// 字段:
//   - Status int: HTTP状态码
//   - Data interface{}: 响应数据
//
// 示例: {Status: 200, Data: {"message": "success"}}
type Response struct {
	Status int         // HTTP状态码
	Data   interface{} // 响应数据
}
