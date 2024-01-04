package middleware

import "github.com/NeverStopDreamingWang/hgee"

// 请求中间件
func RequestMiddleWare(request *hgee.Request) any {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}
