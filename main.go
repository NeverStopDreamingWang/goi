package hgee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"
)

// HandlerFunc 定义 hgee 使用的请求处理程序
type HandlerFunc func(*Request) any

// type HandlerFunc func(*http.Request)

// Engine 实现 ServeHTTP 接口
type Engine struct {
	Router      *Router
	MiddleWares *MiddleWares
}

// 创建一个 Http 服务
func NewHttpServer() *Engine {
	return &Engine{
		Router:      NewRouter(),
		MiddleWares: NewMiddleWares(),
	}
}

// 启动 http 服务
func (engine *Engine) RunServer(addr string) (err error) {
	// engine.Router.CheckRouter()
	server := http.Server{
		// Addr:    fmt.Sprintf("%s:%d", SERVER_ADDRESS, SERVER_PORT),
		Addr:    addr,
		Handler: engine,
	}
	return server.ListenAndServe()
	// return http.ListenAndServe(addr, engine)
}

// 实现 ServeHTTP 接口
func (engine *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestID := time.Now()
	ctx := context.WithValue(request.Context(), "requestID", requestID)
	request = request.WithContext(ctx)

	queryParams := make(Values)

	for s, values := range request.URL.Query() {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := ParseValue(paramType, value)
			queryParams[s] = append(queryParams[s], data)
		}
	}

	requestContext := &Request{
		Object:      request,
		Context:     request.Context(),
		PathParams:  make(Values),
		QueryParams: queryParams,
		BodyParams:  make(map[string]any),
	}

	// 路由处理
	StatusCode, result := engine.Router.routerHandlers(requestContext, response, engine.MiddleWares)

	// 返回响应
	responseData, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(response, "json err: %s", err)
	}
	// 返回响应
	response.WriteHeader(StatusCode)
	write, err := response.Write(responseData)
	if err != nil {
		StatusCode = http.StatusInternalServerError
	}
	fmt.Printf("%s %s - %s %d byte status %d\n", request.Host, request.Method, request.URL.Path, write, StatusCode)
}
