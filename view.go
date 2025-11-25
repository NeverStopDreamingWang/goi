package goi

import (
	"errors"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(request *Request) any

// type HandlerFunc func(*http.Request)

// 路由视图
type ViewSet struct {
	GET      HandlerFunc // 获取资源
	HEAD     HandlerFunc // 获取响应头
	POST     HandlerFunc // 创建资源
	PUT      HandlerFunc // 更新资源
	PATCH    HandlerFunc // 部分更新
	DELETE   HandlerFunc // 删除资源
	CONNECT  HandlerFunc // 建立隧道
	OPTIONS  HandlerFunc // 查询支持的方法
	TRACE    HandlerFunc // 回显请求
	NoMethod HandlerFunc // 未注册或不支持的HTTP方法时的处理函数，默认返回 405
}

// GetHandlerFunc 获取视图方法
//
// 参数:
//   - method string: HTTP方法
//
// 返回:
//   - HandlerFunc: 处理函数
//   - error: 错误信息
func (viewSet ViewSet) GetHandlerFunc(method string) (HandlerFunc, error) {
	var handlerFunc HandlerFunc
	var err error
	switch method {
	case http.MethodGet:
		handlerFunc = viewSet.GET
	case http.MethodHead:
		handlerFunc = viewSet.HEAD
		// HEAD 未定义时，回退到 GET，但不返回 body
		if handlerFunc == nil && viewSet.GET != nil {
			handlerFunc = viewSet.GET
		}
	case http.MethodPost:
		handlerFunc = viewSet.POST
	case http.MethodPut:
		handlerFunc = viewSet.PUT
	case http.MethodPatch:
		handlerFunc = viewSet.PATCH
	case http.MethodDelete:
		handlerFunc = viewSet.DELETE
	case http.MethodConnect:
		handlerFunc = viewSet.CONNECT
	case http.MethodOptions:
		handlerFunc = viewSet.OPTIONS
		// OPTIONS 未定义时，使用默认处理函数
		if handlerFunc == nil {
			handlerFunc = defaultOptionsHandler
		}
	case http.MethodTrace:
		handlerFunc = viewSet.TRACE
	}
	if handlerFunc == nil {
		if viewSet.NoMethod != nil {
			handlerFunc = viewSet.NoMethod
		} else {
			methodNotAllowedMsg := i18n.T("server.method_not_allowed", map[string]any{
				"method": method,
			})
			err = errors.New(methodNotAllowedMsg)
		}
	}
	return handlerFunc, err
}

// GetMethods 获取允许的HTTP方法列表
//
// 返回:
//   - []string: 允许的HTTP方法列表
func (viewSet ViewSet) GetMethods() []string {
	var methods []string
	if viewSet.GET != nil {
		methods = append(methods, http.MethodGet)
	}
	if viewSet.HEAD != nil {
		methods = append(methods, http.MethodHead)
	} else if viewSet.GET != nil {
		methods = append(methods, http.MethodHead)
	}
	if viewSet.POST != nil {
		methods = append(methods, http.MethodPost)
	}
	if viewSet.PUT != nil {
		methods = append(methods, http.MethodPut)
	}
	if viewSet.PATCH != nil {
		methods = append(methods, http.MethodPatch)
	}
	if viewSet.DELETE != nil {
		methods = append(methods, http.MethodDelete)
	}
	if viewSet.CONNECT != nil {
		methods = append(methods, http.MethodConnect)
	}
	methods = append(methods, http.MethodOptions)
	if viewSet.TRACE != nil {
		methods = append(methods, http.MethodTrace)
	}
	return methods
}

// defaultOptionsHandler 默认的 OPTIONS 请求处理函数
//
// 当用户未自定义 OPTIONS 处理函数时，自动调用此方法
// 返回 200 状态码，并设置 Content-Length 响应头
// Allow 响应头由框架自动设置
//
// 参数:
//   - request *Request: HTTP请求对象
//
// 返回:
//   - Response: HTTP响应对象，包含允许的方法列表
func defaultOptionsHandler(request *Request) any {
	response := Response{
		Status: 200, // http.StatusOK
		Data:   nil,
	}
	response.Header().Set("Content-Length", "0")
	return response
}
