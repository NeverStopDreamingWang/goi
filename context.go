package goi

import (
	"context"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/parse"
)

const ContentType = "Content-Type"

// Request 封装HTTP请求相关信息
//
// 参数:
//   - Object *http.Request: 原始HTTP请求对象，包含完整的HTTP请求信息
//   - PathParams Params: 路由参数，存储URL路径中的动态参数值
//
// 用于在处理请求时提供统一的访问接口
type Request struct {
	Object     *http.Request // 原始HTTP请求对象
	PathParams Params        // 路由参数
}

// QueryParams 解析并返回URL查询字符串参数
// 匹配模式: URL中的查询参数，格式为 ?key=value&key2=value2
// 示例: /api/users?page=1&size=10
// 返回一个Params类型的map，包含所有查询参数
//
// 返回:
//   - Params: URL查询参数的键值对映射，其中值为字符串切片
func (request *Request) QueryParams() Params {
	var params Params

	queryValues := request.Object.URL.Query()
	for name, values := range queryValues {
		params[name] = values
	}
	return params
}

// Params 解析并返回请求体中的参数
//
// 支持的Content-Type:
//   - multipart/form-data
//   - application/x-www-form-urlencoded
//   - application/json
//
// 返回:
//   - Params: 请求体参数的键值对映射
//
// 注意:
//   - JSON数据会被解析为interface{}类型
//   - 如果解析失败会触发panic
func (request *Request) BodyParams() Params {
	parsing := parse.GetParsing(request.Object.Header.Get(ContentType))
	return request.BodyParamsParsing(parsing)
}

func (request *Request) BodyParamsJSON() Params {
	return request.BodyParamsParsing(parse.JSON)
}

func (request *Request) BodyParamsXML() Params {
	return request.BodyParamsParsing(parse.XML)
}

func (request *Request) BodyParamsYAML() Params {
	return request.BodyParamsParsing(parse.YAML)
}

func (request *Request) BodyParamsParsing(parsing parse.Parsing) Params {
	params, err := parsing.Parse(request.Object)
	if err != nil {
		panic(err)
	}
	return Params(params)
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
