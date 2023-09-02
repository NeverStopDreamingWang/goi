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
	address     string
	port        uint16
	Router      *Router
	MiddleWares *MiddleWares
}

// 创建一个 Http 服务
func NewHttpServer(address string, port uint16) *Engine {
	fmt.Printf("创建 Http 服务：%v:%v\n", address, port)
	return &Engine{
		address:     address,
		port:        port,
		Router:      NewRouter(),
		MiddleWares: NewMiddleWares(),
	}
}

// 启动 http 服务
func (engine *Engine) RunServer() (err error) {
	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", engine.address, engine.port),
		Handler: engine,
	}
	fmt.Printf("运行在 http://%v:%v\n", engine.address, engine.port)
	return server.ListenAndServe()
	// return http.ListenAndServe(addr, engine)
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestID := time.Now()
	ctx := context.WithValue(request.Context(), "requestID", requestID)
	request = request.WithContext(ctx)

	var err error
	// 初始化请求
	requestContext := &Request{
		Object:      request,
		Context:     request.Context(),
		PathParams:  make(Values),
		QueryParams: make(Values),
		BodyParams:  make(Values),
	}

	// 解析 Query 参数
	for name, values := range request.URL.Query() {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := ParseValue(paramType, value)
			requestContext.QueryParams[name] = append(requestContext.QueryParams[name], data)
		}
	}

	// 解析 Body 参数
	request.ParseMultipartForm(32 << 20)
	for name, values := range request.Form {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := ParseValue(paramType, value)
			requestContext.BodyParams[name] = append(requestContext.BodyParams[name], data)
		}
	}

	// 解析 json 数据
	body, err := io.ReadAll(request.Body)
	if err == nil {
		jsonData := make(map[string]interface{})
		json.Unmarshal(body, &jsonData)
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
		ResponseStatic(value, requestContext, response)
		return
	default:
		ResponseData, err = json.Marshal(value)
	}
	// 返回响应
	if err != nil {
		fmt.Fprintf(response, "response err: %s", err)
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	write, err := response.Write(ResponseData)
	if err != nil {
		StatusCode = http.StatusInternalServerError
	}
	fmt.Printf("[%v] - %v - %v %v %v byte status %v %v\n",
		time.Now().Format("2006-01-02 15:04:05"),
		request.Host,
		request.Method,
		request.URL.Path,
		write,
		request.Proto,
		StatusCode,
	)
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
