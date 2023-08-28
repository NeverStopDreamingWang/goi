package hgee

import (
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"regexp"
	"strconv"
)

type AsView struct {
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

type routerParam struct {
	paramName string
	paramType string
}

// 路由表
type Router struct {
	// 子路由
	includeRouter map[string]*Router
	// 视图方法
	viewSet AsView
	// 路由参数
	params []routerParam
	// 转换后的路由
	converterPattern string
}

// 创建路由
func NewRouter() *Router {
	return &Router{
		includeRouter:    make(map[string]*Router),
		viewSet:          AsView{},
		params:           make([]routerParam, 0),
		converterPattern: "",
	}
}

// 添加父路由
func (router *Router) Include(pattern string) *Router {
	if _, ok := router.includeRouter[pattern]; ok {
		panic(fmt.Sprintf("路由已存在：%s\n", pattern))
	}
	var re *regexp.Regexp
	for includePatternUri, Irouter := range router.includeRouter {
		if len(Irouter.includeRouter) != 0 { // 拥有子路由
			re = regexp.MustCompile(includePatternUri + "/")
		} else {
			re = regexp.MustCompile(includePatternUri)
		}
		if len(re.FindStringSubmatch(pattern)) != 0 {
			panic(fmt.Sprintf("%s 中包含的子路由已被注册: %s\n", pattern, includePatternUri))
		}
	}
	router.includeRouter[pattern] = &Router{
		includeRouter:    make(map[string]*Router),
		viewSet:          AsView{},
		params:           make([]routerParam, 0),
		converterPattern: "",
	}
	// 路由参数解析
	routerParse(pattern, router.includeRouter[pattern])
	return router.includeRouter[pattern]
}

// 添加路由
func (router *Router) UrlPatterns(pattern string, view AsView) {
	if _, ok := router.includeRouter[pattern]; ok {
		panic(fmt.Sprintf("路由已存在：%s\n", pattern))
	}
	var re *regexp.Regexp
	for includePatternUri, Irouter := range router.includeRouter {
		if len(Irouter.includeRouter) != 0 { // 拥有子路由
			re = regexp.MustCompile(includePatternUri + "/")
		} else {
			re = regexp.MustCompile(includePatternUri)
		}
		if len(re.FindStringSubmatch(pattern)) != 0 {
			panic(fmt.Sprintf("%s 中的父路由已被注册: %s\n", pattern, includePatternUri))
		}
	}

	router.includeRouter[pattern] = &Router{
		includeRouter:    make(map[string]*Router),
		viewSet:          view,
		params:           make([]routerParam, 0),
		converterPattern: "",
	}
	// 路由参数解析
	routerParse(pattern, router.includeRouter[pattern])
}

// 路由器处理程序
func (router *Router) routerHandlers(request *Request, response http.ResponseWriter, middlewares *MiddleWares) (int, any) {

	// 处理请求前的中间件
	result := middlewares.processRequest(request)
	if result != nil {
		return http.StatusOK, result
	}

	view_methods, isPattern := routeResolution(request.Object.URL.Path, router.includeRouter, request.PathParams)
	if isPattern == false {
		result = fmt.Sprintf("URL NOT FOUND: %s", request.Object.URL.Path)
		return http.StatusNotFound, result
	}

	var handlerFunc HandlerFunc
	switch request.Object.Method {
	case "GET":
		handlerFunc = view_methods.GET
	case "HEAD":
		handlerFunc = view_methods.HEAD
	case "POST":
		handlerFunc = view_methods.POST
	case "PUT":
		handlerFunc = view_methods.PUT
	case "PATCH":
		handlerFunc = view_methods.PATCH
	case "DELETE":
		handlerFunc = view_methods.DELETE
	case "CONNECT":
		handlerFunc = view_methods.CONNECT
	case "OPTIONS":
		handlerFunc = view_methods.OPTIONS
	case "TRACE":
		handlerFunc = view_methods.TRACE
	default:
		result = fmt.Sprintf("Method NOT FOUND: %s", request.Object.Method)
		return http.StatusNotFound, result
	}

	if handlerFunc == nil {
		result = fmt.Sprintf("Method NOT FOUND: %s", request.Object.Method)
		return http.StatusNotFound, result
	}

	// 视图前的中间件
	result = middlewares.processView(request, handlerFunc)
	if result != nil {
		return http.StatusOK, result
	}

	responseResult := handlerFunc(request)

	// 返回响应前的中间件
	result = middlewares.processResponse(request, response)
	if result != nil {
		return http.StatusOK, result
	}

	return http.StatusOK, responseResult
}

// 路由参数解析
func routerParse(pattern string, router *Router) {
	regexpStr := `<([^/<>]+):([^/<>]+)>`
	re := regexp.MustCompile(regexpStr)
	result := re.FindAllStringSubmatch(pattern, -1)
	for _, paramsSlice := range result {
		if len(paramsSlice) == 3 {
			converter, ok := settingConverter[paramsSlice[1]]
			if ok == false {
				continue
			}
			re = regexp.MustCompile(paramsSlice[0])
			pattern = re.ReplaceAllString(pattern, converter)
			router.converterPattern = pattern

			param := routerParam{
				paramName: paramsSlice[2],
				paramType: paramsSlice[1],
			}
			router.params = append(router.params, param)
		}
	}
}

// 路由解析
func routeResolution(requestPattern string, includeRouter map[string]*Router, PathParams Values) (AsView, bool) {
	var re *regexp.Regexp
	for includePatternUri, router := range includeRouter {
		if len(router.params) == 0 { // 无参数直接匹配
			if len(router.includeRouter) == 0 {
				re = regexp.MustCompile(includePatternUri)
			} else {
				re = regexp.MustCompile(includePatternUri + "/")
			}
		} else {
			if len(router.includeRouter) == 0 { // 是否有子路径
				re = regexp.MustCompile(router.converterPattern + "$")
			} else {
				re = regexp.MustCompile(router.converterPattern)
			}
		}

		paramsSlice := re.FindStringSubmatch(requestPattern)
		if len(paramsSlice)-1 != len(router.params) || len(paramsSlice) == 0 {
			continue
		}
		paramsSlice = paramsSlice[1:]
		for i := 0; i < len(router.params); i++ {
			param := router.params[i]
			value := ParseValue(param.paramType, paramsSlice[i])
			PathParams[param.paramName] = append(PathParams[param.paramName], value) // 添加参数
		}

		if len(router.includeRouter) == 0 {
			return router.viewSet, true
		} else {
			return routeResolution(requestPattern[len(includePatternUri):], router.includeRouter, PathParams)
		}
	}
	return AsView{}, false
}

// 参数解析
func ParseValue(paramType string, paramValue string) any {
	switch paramType {
	case "int":
		value, _ := strconv.Atoi(paramValue)
		return value
	case "uuid":
		value, _ := uuid.Parse(paramValue)
		return value
	default:
		return paramValue
	}
}
