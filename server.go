package goi

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

// Http 服务
var engine *Engine
var Settings *metaSettings
var Cache *MetaCache
var Log *MetaLogger
var version = "1.0.12"

// 版本
func Version() string {
	return version
}

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(*Request) any

// type HandlerFunc func(*http.Request)

// Engine 实现 ServeHTTP 接口
type Engine struct {
	Router      *metaRouter
	MiddleWares *metaMiddleWares
	Settings    *metaSettings
	Cache       *MetaCache
	Log         *MetaLogger
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	Settings = newSettings()
	Cache = newCache()
	Log = NewLogger()
	engine = &Engine{
		Router:      newRouter(),
		MiddleWares: newMiddleWares(),
		Settings:    Settings,
		Cache:       Cache,
		Log:         Log,
	}
	return engine
}

// 启动 http 服务
func (engine *Engine) RunServer() (err error) {
	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.Settings.SERVER_ADDRESS, engine.Settings.SERVER_PORT),
		Handler: engine,
	}
	return server.ListenAndServe()
	// return http.ListenAndServe(addr, engine)
}

// 初始化
func (engine *Engine) Init() {
	// 初始化时区
	location, err := time.LoadLocation(engine.Settings.TIME_ZONE)
	if err != nil {
		panic(fmt.Sprintf("初始化时区错误: %v\n", err))
	}
	engine.Settings.LOCATION = location

	// 初始化日志
	engine.Log.InitLogger()

	engine.Log.MetaLog(fmt.Sprintf("启动时间: %s", time.Now().In(Settings.LOCATION).Format("2006-01-02 15:04:05")))
	engine.Log.MetaLog(fmt.Sprintf("goi 版本: %v", version))
	engine.Log.MetaLog(fmt.Sprintf("监听地址: %v:%v", engine.Settings.SERVER_ADDRESS, engine.Settings.SERVER_PORT))

	engine.Log.MetaLog(fmt.Sprintf("当前时区: %v", engine.Settings.TIME_ZONE))

	// 初始化缓存
	engine.Cache.initCache()
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestID := time.Now().In(engine.Settings.LOCATION)
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
		panic(fmt.Sprintf("读取 Body 错误: %v\n", err))
	}
	if len(body) != 0 {
		jsonData := make(map[string]any)
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			panic(fmt.Sprintf("解析 json 错误: %v\n", err))
		}
		for name, value := range jsonData {
			requestContext.BodyParams[name] = append(requestContext.BodyParams[name], value)
		}
	}

	// 处理 HTTP 请求
	StatusCode := http.StatusOK
	var ResponseData []byte
	var write any

	responseData := engine.HandlerHTTP(requestContext, response)

	// 文件处理
	fileObject, isFile := responseData.(*os.File)
	if isFile {
		// 返回文件内容
		write, ResponseData = metaResponseStatic(fileObject, requestContext, response)
		log = fmt.Sprintf("- %v - %v %v %v byte status %v %v",
			request.Host,
			request.Method,
			request.URL.Path,
			write,
			request.Proto,
			StatusCode,
		)
		engine.Log.Info(log)
		return
	}

	// 响应处理
	responseObject, isResponse := responseData.(Response)
	if isResponse {
		StatusCode = responseObject.Status
		responseData = responseObject.Data
	}

	switch value := responseData.(type) {
	case nil:
		ResponseData, err = json.Marshal(value)
	case bool:
		ResponseData, err = json.Marshal(value)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		ResponseData, err = json.Marshal(value)
	case float32, float64:
		ResponseData, err = json.Marshal(value)
	case string:
		ResponseData = []byte(value)
	case []byte:
		ResponseData = responseData.([]byte)
	default:
		ResponseData, err = json.Marshal(value)
	}
	// 返回响应
	if err != nil {
		panic(fmt.Sprintf("响应 json 数据错误: %v\n", err))
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	write, err = response.Write(ResponseData)
	if err != nil {
		panic(fmt.Sprintf("响应错误: %v\n", err))
	}
	log = fmt.Sprintf("- %v - %v %v %v byte status %v %v",
		request.Host,
		request.Method,
		request.URL.Path,
		write,
		request.Proto,
		StatusCode,
	)
	if StatusCode == http.StatusOK {
		engine.Log.Info(log)
	} else {
		engine.Log.Error(log)
	}
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
