package hgee

import (
	"net/http"
)

type RequestMiddleware func(*Request) any                       // 请求中间件
type ViewMiddleware func(*Request, HandlerFunc) any             // 视图中间件
type ResponseMiddleware func(*Request, http.ResponseWriter) any // 响应中间件

type MiddleWares struct {
	processRequest  RequestMiddleware  // 请求中间件
	processView     ViewMiddleware     // 视图中间件
	processResponse ResponseMiddleware // 响应中间件
}

// 注册请求中间件
func (middlewares *MiddleWares) RegisterRequest(processRequest RequestMiddleware) {
	middlewares.processRequest = processRequest
}

// 注册视图中间件
func (middlewares *MiddleWares) RegisterView(processView ViewMiddleware) {
	middlewares.processView = processView
}

// 注册响应中间件
func (middlewares *MiddleWares) RegisterResponse(processResponse ResponseMiddleware) {
	middlewares.processResponse = processResponse
}

// 创建中间件
func NewMiddleWares() *MiddleWares {
	return &MiddleWares{
		processRequest:  func(request *Request) any { return nil },
		processView:     func(request *Request, handlerFunc HandlerFunc) any { return nil },
		processResponse: func(request *Request, writer http.ResponseWriter) any { return nil },
	}
}
