package middleware

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"example/example"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

func init() {
	// 注册中间件
	// 注册请求中间件
	example.Server.MiddleWares.BeforeRequest(RequestMiddleWare)
	// 注册视图中间件
	example.Server.MiddleWares.BeforeView(ViewMiddleWare)
	// 注册响应中间件
	example.Server.MiddleWares.BeforeResponse(ResponseMiddleWare)
}

// 请求中间件
func RequestMiddleWare(request *goi.Request) interface{} {
	// fmt.Println("请求中间件", request.Object.URL)

	var apiList = []string{
		"^/api",
	}
	var re *regexp.Regexp
	for _, api := range apiList {
		re = regexp.MustCompile(api)
		// 跳过验证
		if re.MatchString(request.Object.URL.Path) {
			return nil
		}
	}

	token := request.Object.Header.Get("Authorization")

	payloads := &example.Payloads{}
	err := jwt.CkeckToken(token, goi.Settings.SECRET_KEY, payloads)
	if errors.Is(err, jwt.ErrDecode) { // token 解码错误
		return goi.Data{
			Code:    http.StatusUnauthorized,
			Message: "token 解码错误",
			Results: err,
		}
	} else if errors.Is(err, jwt.ErrExpiredSignature) { // token 已过期
		return goi.Data{
			Code:    http.StatusUnauthorized,
			Message: "token 已过期",
			Results: err,
		}
	}
	ctx := context.WithValue(request.Object.Context(), "user_id", payloads.User_id)
	request.WithContext(ctx)

	return nil
}

// 视图中间件
func ViewMiddleWare(request *goi.Request) interface{} {
	// fmt.Println("视图中间件", request.Object.URL)
	return nil
}

// 响应中间件
func ResponseMiddleWare(request *goi.Request, viewResponse interface{}) interface{} {
	// fmt.Println("响应中间件", request.Object.URL)
	return nil
}
