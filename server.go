package hgee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"time"
)

// HandlerFunc 定义 hgee 使用的请求处理程序
type HandlerFunc func(*Request) any

// type HandlerFunc func(*http.Request)

// Engine 实现 ServeHTTP 接口
type Engine struct {
	Router      *metaRouter
	MiddleWares *metaMiddleWares
	Settings    *metaSettings
	Log         *metaLogger
}

// Http 服务
var engine *Engine

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	engine = &Engine{
		Router:      newRouter(),
		MiddleWares: newMiddleWares(),
		Settings:    newSettings(),
		Log:         newLogger(),
	}
	return engine
}

// 启动 http 服务
func (engine *Engine) RunServer() (err error) {
	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.Settings.SERVER_ADDRESS, engine.Settings.SERVER_PORT),
		Handler: engine,
	}
	engine.init() // 初始化
	return server.ListenAndServe()
	// return http.ListenAndServe(addr, engine)
}

// 初始化
func (engine *Engine) init() {
	// 初始化日志
	engine.initLogger()
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestID := time.Now()
	ctx := context.WithValue(request.Context(), "requestID", requestID)
	request = request.WithContext(ctx)

	var log string
	var err error
	// 初始化请求
	requestContext := &Request{
		Object:      request,
		Context:     request.Context(),
		PathParams:  make(metaValues),
		QueryParams: make(metaValues),
		BodyParams:  make(metaValues),
	}
	defer metaRecovery(requestContext, response) // 异常处理

	// 解析 Query 参数
	for name, values := range request.URL.Query() {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := parseValue(paramType, value)
			requestContext.QueryParams[name] = append(requestContext.QueryParams[name], data)
		}
	}

	// 解析 Body 参数
	request.ParseMultipartForm(32 << 20)
	for name, values := range request.Form {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := parseValue(paramType, value)
			requestContext.BodyParams[name] = append(requestContext.BodyParams[name], data)
		}
	}

	// 解析 json 数据
	body, err := io.ReadAll(request.Body)
	if err != nil {
		panic(fmt.Sprintf("读取 Body 错误：%v", err))
	}
	if len(body) != 0 {
		jsonData := make(map[string]any)
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			panic(fmt.Sprintf("解析 json 错误：%v", err))
		}
		for name, value := range jsonData {
			requestContext.BodyParams[name] = append(requestContext.BodyParams[name], value)
		}
	}

	// 处理 HTTP 请求
	StatusCode := http.StatusOK
	responseData := engine.HandlerHTTP(requestContext, response)

	responseObject, isResponse := responseData.(Response)
	if isResponse {
		StatusCode = responseObject.Status
		responseData = responseObject.Data
	}

	var ResponseData []byte

	switch value := responseData.(type) {
	case nil:
		ResponseData, err = json.Marshal(value)
	case bool:
		ResponseData, err = json.Marshal(value)
	case int:
		ResponseData, err = json.Marshal(value)
	case float64:
		ResponseData, err = json.Marshal(value)
	case func(int) float64:
		ResponseData, err = json.Marshal(value)
	case string:
		ResponseData = []byte(value)
	case []byte:
		ResponseData = responseData.([]byte)
	case *os.File:
		// 返回文件内容
		metaResponseStatic(value, requestContext, response)
		return
	default:
		ResponseData, err = json.Marshal(value)
	}
	// 返回响应
	if err != nil {
		panic(fmt.Sprintf("响应 json 数据错误：%v", err))
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	write, err := response.Write(ResponseData)
	if err != nil {
		panic(fmt.Sprintf("响应错误：%v", err))
	}
	log = fmt.Sprintf("- %v - %v %v %v byte status %v %v",
		request.Host,
		request.Method,
		request.URL.Path,
		write,
		request.Proto,
		StatusCode,
	)
	engine.Log.Info(log)
}

// 处理 HTTP 请求
func (engine *Engine) HandlerHTTP(request *Request, response http.ResponseWriter) (result any) {
	// 处理请求前的中间件
	result = engine.MiddleWares.processRequest(request)
	if result != nil {
		return result
	}

	// 路由处理
	handlerFunc, err := engine.Router.routerHandlers(request)
	if err != "" {
		return Response{Status: http.StatusNotFound, Data: err}
	}

	// 视图前的中间件
	result = engine.MiddleWares.processView(request, handlerFunc)
	if result != nil {
		return result
	}
	// 视图处理
	viewResponse := handlerFunc(request)

	// 返回响应前的中间件
	result = engine.MiddleWares.processResponse(request, response)
	if result != nil {
		return result
	}
	return viewResponse
}
