package middleware

import (
	"example/example"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 注册中间件
	// 注册请求中间件
	example.Server.MiddleWares.BeforeRequest(RequestMiddleWare)
	// 注册视图中间件
	// Server.MiddleWares.BeforeView()
	// 注册响应中间件
	// Server.MiddleWares.BeforeResponse()
}

// 请求中间件
func RequestMiddleWare(request *goi.Request) interface{} {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}

// 请求中间件
func ViewMiddleWare(request *goi.Request) interface{} {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}

// 请求中间件
func ResponseMiddleWare(request *goi.Request) interface{} {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}
