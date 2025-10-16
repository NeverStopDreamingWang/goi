package goi

import (
	"errors"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(request *Request) interface{}

// type HandlerFunc func(*http.Request)

// 路由视图
type ViewSet struct {
	GET      HandlerFunc
	HEAD     HandlerFunc
	POST     HandlerFunc
	PUT      HandlerFunc
	PATCH    HandlerFunc
	DELETE   HandlerFunc
	CONNECT  HandlerFunc
	OPTIONS  HandlerFunc
	TRACE    HandlerFunc
	NoMethod HandlerFunc // 未注册或不支持的HTTP方法时的处理函数，默认返回 405
}

func (viewSet ViewSet) GetHandlerFunc(method string) (HandlerFunc, error) {
	var handlerFunc HandlerFunc
	var err error
	switch method {
	case "GET":
		handlerFunc = viewSet.GET
	case "HEAD":
		handlerFunc = viewSet.HEAD
	case "POST":
		handlerFunc = viewSet.POST
	case "PUT":
		handlerFunc = viewSet.PUT
	case "PATCH":
		handlerFunc = viewSet.PATCH
	case "DELETE":
		handlerFunc = viewSet.DELETE
	case "CONNECT":
		handlerFunc = viewSet.CONNECT
	case "OPTIONS":
		handlerFunc = viewSet.OPTIONS
	case "TRACE":
		handlerFunc = viewSet.TRACE
	}
	if handlerFunc == nil {
		if viewSet.NoMethod != nil {
			handlerFunc = viewSet.NoMethod
		} else {
			methodNotAllowedMsg := i18n.T("server.method_not_allowed", map[string]interface{}{
				"method": method,
			})
			err = errors.New(methodNotAllowedMsg)
		}
	}
	return handlerFunc, err
}
