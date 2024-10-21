package goi

import (
	"context"
	"encoding/json"
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
	Object     *http.Request
	Context    context.Context // 请求上下文
	PathParams ParamsValues    // 路由参数
}

// 解析 查询字符串参数
func (request *Request) QueryParams() ParamsValues {
	var QueryParams ParamsValues = make(ParamsValues)
	// 解析 Query 参数
	for name, values := range request.Object.URL.Query() {
		QueryParams[name] = append(QueryParams[name], values...)
	}
	return QueryParams
}

// 解析 Body 传参
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
			bodyParams[name] = append(bodyParams[name], value)
		}
	}
	return bodyParams
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
