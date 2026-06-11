package goi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
	"github.com/NeverStopDreamingWang/goi/parser"
)

const ContentType = "Content-Type"

// Request 封装HTTP请求相关信息
//
// 参数:
//   - Object *http.Request: 原始HTTP请求对象，包含完整的HTTP请求信息
//   - PathParams Params: 路由参数，存储URL路径中的动态参数值
//   - Params Params: 自定义参数，可在整个请求处理过程中传递和共享数据
//
// 用于在处理请求时提供统一的访问接口
type Request struct {
	Object     *http.Request
	PathParams Params
	Params     Params
}

// WithContext 更新请求对象中的上下文信息
//
// 参数:
//   - ctx context.Context: 新的上下文对象
func (request *Request) WithContext(ctx context.Context) {
	// 创建一个新的 http.Request 并用新的上下文更新它
	request.Object = request.Object.WithContext(ctx)
}

// QueryParams 解析并返回URL查询字符串参数
// 匹配模式: URL中的查询参数，格式为 ?key=value&key2=value2
// 示例: /api/users?page=1&size=10
// 返回一个Params类型的map，包含所有查询参数
//
// 返回:
//   - Params: URL查询参数的键值对映射，其中值为字符串切片
func (request *Request) QueryParams() Params {
	params := make(Params)

	queryValues := request.Object.URL.Query()
	for name, values := range queryValues {
		if len(values) == 1 {
			if values[0] == "" {
				params[name] = nil
			} else {
				params[name] = values[0]
			}
		} else {
			params[name] = values
		}
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
//   - JSON数据会被解析为any类型
//   - 如果解析失败会触发panic
func (request *Request) BodyParams() Params {
	parsing := parser.GetParser(request.Object.Header.Get(ContentType))
	return request.BodyParamsParsing(parsing)
}

func (request *Request) BodyParamsJSON() Params {
	return request.BodyParamsParsing(parser.JSON)
}

func (request *Request) BodyParamsXML() Params {
	return request.BodyParamsParsing(parser.XML)
}

func (request *Request) BodyParamsYAML() Params {
	return request.BodyParamsParsing(parser.YAML)
}

func (request *Request) BodyParamsParsing(parsing parser.Parser) Params {
	params, err := parsing.Parse(request.Object)
	if err != nil {
		panic(err)
	}
	return Params(params)
}

// ResponseWriter 自定义响应写入器
//
// 字段:
//   - ResponseWriter http.ResponseWriter: 标准响应写入器
//   - Status int: HTTP状态码
//   - bytes int64: 已写入的字节数
//
// 用于请求日志记录和响应监控
type ResponseWriter struct {
	http.ResponseWriter     // 内嵌http.ResponseWriter接口
	Status              int // 响应状态码

	bytes int64 // 已写入的字节数
}

// Header 返回响应头
func (responseWriter *ResponseWriter) Header() http.Header {
	return responseWriter.ResponseWriter.Header()
}

// WriteHeader 设置HTTP响应状态码
//
// 参数:
// - code int: HTTP状态码
func (responseWriter *ResponseWriter) WriteHeader(code int) {
	responseWriter.Status = code
	responseWriter.ResponseWriter.WriteHeader(code)
}

// Write 写入响应数据
//
// 参数:
//   - b []byte: 要写入的数据
//
// 返回:
//   - int: 已写入的字节数
//   - error: 写入过程中的错误信息
func (responseWriter *ResponseWriter) Write(b []byte) (int, error) {
	// 调用嵌入的http.ResponseWriter的Write方法
	bytesWritten, err := responseWriter.ResponseWriter.Write(b)
	responseWriter.bytes += int64(bytesWritten)
	return bytesWritten, err
}

// Flush 若底层 http.ResponseWriter 支持 http.Flusher 则触发分块刷新，否则静默 no-op
//
// 用于流式响应、SSE、长连接等场景，由 RawHandler 视图主动调用
func (responseWriter *ResponseWriter) Flush() {
	if f, ok := responseWriter.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RawHandler 视图函数返回 RawHandler 时，框架不会再对返回值做 json.Marshal 与
// Content-Length 设置，转而由 fn 自行写入 status / header / body 并按需 flush
//
// 适用于：流式响应、SSE、反向代理透传、自定义二进制协议等需要直接操作 ResponseWriter 的场景
//
// 参数:
//   - w *ResponseWriter: 响应写入器
//   - r *Request: HTTP 请求对象
//
// 返回:
//   - error: 写入过程中的错误信息
type RawHandler func(w http.ResponseWriter, r *Request) error

// Response 定义处理程序的响应格式
//
// 字段:
//   - Status int: HTTP状态码
//   - Data any: 响应数据
//
// 示例: {Status: 200, Data: {"message": "success"}}
type Response struct {
	Status  int // HTTP状态码
	Data    any // 响应数据
	headers http.Header
}

func (response *Response) Header() http.Header {
	if response.headers == nil {
		response.headers = make(http.Header)
	}
	return response.headers
}

func (response *Response) write(request *Request, responseWriter *ResponseWriter) (int, error) {
	var err error
	var dataByte []byte
	var contentType string
	for key, value := range response.Header() {
		for _, v := range value {
			if responseWriter.Header().Values(key) == nil {
				responseWriter.Header().Set(key, v)
			} else {
				responseWriter.Header().Add(key, v)
			}
		}
	}
	switch value := response.Data.(type) {
	case nil:
		return 0, nil
	case RawHandler:
		err = value(responseWriter, request)
		return int(responseWriter.bytes), err
	case string:
		dataByte = []byte(value)
		contentType = "text/plain"
	case []byte:
		dataByte = value
		contentType = "application/octet-stream"
	case *os.File:
		defer value.Close()
		fileInfo, err := value.Stat()
		if err != nil || fileInfo.IsDir() {
			http.NotFound(responseWriter, request.Object)
			return 0, nil
		}
		http.ServeContent(responseWriter, request.Object, fileInfo.Name(), fileInfo.ModTime(), value)
		return 0, nil
	case http.File:
		defer value.Close()
		fileInfo, err := value.Stat()
		if err != nil || fileInfo.IsDir() {
			http.NotFound(responseWriter, request.Object)
			return 0, nil
		}
		http.ServeContent(responseWriter, request.Object, fileInfo.Name(), fileInfo.ModTime(), value)
		return 0, nil
	case FS:
		http.ServeFileFS(responseWriter, request.Object, value.FS, value.Name)
		return 0, nil
	case fs.FS:
		http.ServeFileFS(responseWriter, request.Object, value, request.Object.URL.Path)
		return 0, nil
	case embed.FS:
		http.ServeFileFS(responseWriter, request.Object, value, request.Object.URL.Path)
		return 0, nil
	case io.Reader:
		// 注意：本分支放在所有具名 reader 类型（*os.File / http.File 等）之后，
		// 避免它们被 io.Reader 接口吞掉，绕过原有的 ServeContent 处理
		if closer, ok := value.(io.Closer); ok {
			defer closer.Close()
		}
		if responseWriter.Header().Get(ContentType) == "" {
			responseWriter.Header().Set(ContentType, "application/octet-stream")
		}
		responseWriter.WriteHeader(response.Status)
		written, copyErr := io.Copy(responseWriter, value)
		return int(written), copyErr
	default:
		dataByte, err = json.Marshal(value)
		contentType = "application/json"
	}
	if err != nil {
		responseJsonErrorMsg := i18n.T("server.response_error", map[string]any{
			"err": err,
		})
		return 0, errors.New(responseJsonErrorMsg)
	}
	// 若未显式设置 Content-Length，则为非流式响应补上
	if responseWriter.Header().Get("Content-Length") == "" {
		responseWriter.Header().Set("Content-Length", strconv.Itoa(len(dataByte)))
	}
	if responseWriter.Header().Get(ContentType) == "" && contentType != "" {
		responseWriter.Header().Set(ContentType, contentType)
	}
	// 写入响应状态码
	responseWriter.WriteHeader(response.Status)
	// 写入响应数据
	return responseWriter.Write(dataByte)
}
