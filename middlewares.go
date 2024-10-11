package goi

import (
	"net/http"
)

type RequestMiddleware func(*Request) interface{}                       // 请求中间件
type ViewMiddleware func(*Request) interface{}                          // 视图中间件
type ResponseMiddleware func(*Request, http.ResponseWriter) interface{} // 响应中间件

type metaMiddleWares struct {
	processRequest  RequestMiddleware  // 请求中间件
	processView     ViewMiddleware     // 视图中间件
	processResponse ResponseMiddleware // 响应中间件
}

// 注册请求中间件
func (middlewares *metaMiddleWares) BeforeRequest(processRequest RequestMiddleware) {
	middlewares.processRequest = processRequest
}

// 注册视图中间件
func (middlewares *metaMiddleWares) BeforeView(processView ViewMiddleware) {
	middlewares.processView = processView
}

// 注册响应中间件
func (middlewares *metaMiddleWares) BeforeResponse(processResponse ResponseMiddleware) {
	middlewares.processResponse = processResponse
}

// 创建中间件
func newMiddleWares() *metaMiddleWares {
	return &metaMiddleWares{
		processRequest:  func(request *Request) interface{} { return nil },
		processView:     func(request *Request) interface{} { return nil },
		processResponse: func(request *Request, writer http.ResponseWriter) interface{} { return nil },
	}
}
