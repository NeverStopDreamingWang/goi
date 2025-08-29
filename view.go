package goi

import (
	"errors"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(request *Request) interface{}

// type HandlerFunc func(*http.Request)

// 路由视图
type ViewSet struct {
	GET     HandlerFunc
	HEAD    HandlerFunc
	POST    HandlerFunc
	PUT     HandlerFunc
	PATCH   HandlerFunc
	DELETE  HandlerFunc
	CONNECT HandlerFunc
	OPTIONS HandlerFunc
	TRACE   HandlerFunc
}

func (viewSet ViewSet) GetHandlerFunc(method string) (HandlerFunc, error) {
	var handlerFunc HandlerFunc
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
	default:
		methodNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.method_not_allowed",
			TemplateData: map[string]interface{}{
				"method": method,
			},
		})
		return nil, errors.New(methodNotAllowedMsg)
	}
	if handlerFunc == nil {
		methodNotAllowedMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "server.method_not_allowed",
			TemplateData: map[string]interface{}{
				"method": method,
			},
		})
		return nil, errors.New(methodNotAllowedMsg)
	}
	return handlerFunc, nil
}
