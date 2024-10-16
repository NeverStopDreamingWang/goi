package goi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 内置响应数据格式
type Data struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Results interface{} `json:"results"`
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
			parseBodyErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "context.parse_body_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			panic(parseBodyErrorMsg)
		}
	}
	for name, values := range request.Object.Form {
		request.BodyParams[name] = append(request.BodyParams[name], values...)
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
			panic(unmarshalBodyErrorMsg)
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

// 自定义响应
type customResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Bytes      int64
}

// 重写 WriteHeader 方法
func (w *customResponseWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// 重写 Write 方法
func (w *customResponseWriter) Write(b []byte) (int, error) {
	bytesWritten, err := w.ResponseWriter.Write(b)
	w.Bytes += int64(bytesWritten)
	return bytesWritten, err
}

// 请求响应数据
type Response struct {
	Status int
	Data   interface{}
}
