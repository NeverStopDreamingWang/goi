package goi

import (
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// 路由视图
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
	File    string   // 文件
	DIR     http.Dir // 文件夹
}

// 路由表
type metaRouter struct {
	includeRouter map[string]*metaRouter // 子路由
	viewSet       AsView                 // 视图方法
}

// 路由参数
type routerParam struct {
	paramName string
	paramType string
}

// 创建路由
func newRouter() *metaRouter {
	return &metaRouter{
		includeRouter: make(map[string]*metaRouter),
		viewSet:       AsView{},
	}
}

// 判断路由是否被注册
func (router *metaRouter) isUrl(UrlPath string) {
	if _, ok := router.includeRouter[UrlPath]; ok {
		panic(fmt.Sprintf("路由已存在: %s\n", UrlPath))
	}

	var re *regexp.Regexp
	for includePatternUri, Irouter := range router.includeRouter {
		params, converterPattern := routerParse(includePatternUri)
		var reString string
		if len(params) == 0 { // 无参数直接匹配
			if len(Irouter.includeRouter) == 0 || Irouter.viewSet.File != "" {
				reString = includePatternUri + "&"
			} else if strings.HasSuffix(includePatternUri, "/") == false {
				reString = includePatternUri + "/"
			} else {
				reString = includePatternUri
			}
		} else {
			if len(Irouter.includeRouter) == 0 || Irouter.viewSet.File != "" {
				reString = converterPattern + "$"
			} else if strings.HasSuffix(includePatternUri, "/") == false {
				reString = converterPattern + "/"
			} else {
				reString = converterPattern
			}
		}
		re = regexp.MustCompile(reString)

		if len(re.FindStringSubmatch(UrlPath)) != 0 {
			panic(fmt.Sprintf("%s 冲突路由: %s\n", UrlPath, includePatternUri))
		}
	}
}

// 添加父路由
func (router *metaRouter) Include(UrlPath string) *metaRouter {
	router.isUrl(UrlPath)
	router.includeRouter[UrlPath] = &metaRouter{
		includeRouter: make(map[string]*metaRouter),
		viewSet:       AsView{},
	}
	return router.includeRouter[UrlPath]
}

// 添加路由
func (router *metaRouter) UrlPatterns(UrlPath string, view AsView) {
	router.isUrl(UrlPath)
	router.includeRouter[UrlPath] = &metaRouter{
		includeRouter: nil,
		viewSet:       view,
	}
}

// 添加文件静态路由
func (router *metaRouter) StaticFilePatterns(UrlPath string, FilePath string) {
	router.isUrl(UrlPath)
	router.includeRouter[UrlPath] = &metaRouter{
		includeRouter: nil,
		viewSet:       AsView{GET: metaStaticHandler, File: FilePath},
	}
}

// 添加文件夹静态路由
func (router *metaRouter) StaticDirPatterns(UrlPath string, DirPath http.Dir) {
	router.isUrl(UrlPath)

	if DirPath == "" {
		DirPath = "."
	}
	router.includeRouter[UrlPath] = &metaRouter{
		includeRouter: nil,
		viewSet:       AsView{GET: metaStaticHandler, DIR: DirPath},
	}
}

// 路由器处理程序
func (router *metaRouter) routerHandlers(request *Request) (handlerFunc HandlerFunc, err string) {

	view_methods, isPattern := routeResolution(request.Object.URL.Path, router.includeRouter, request.PathParams)
	if isPattern == false {
		err = fmt.Sprintf("URL NOT FOUND: %s", request.Object.URL.Path)
		return nil, err
	}
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
		err = fmt.Sprintf("Method NOT FOUND: %s", request.Object.Method)
		return nil, err
	}

	if handlerFunc == nil {
		err = fmt.Sprintf("Method NOT FOUND: %s", request.Object.Method)
		return nil, err
	}
	return handlerFunc, ""
}

// 路由解析
func routeResolution(requestPattern string, includeRouter map[string]*metaRouter, PathParams metaValues) (AsView, bool) {
	var re *regexp.Regexp
	for includePatternUri, Irouter := range includeRouter {
		params, converterPattern := routerParse(includePatternUri)
		var reString string
		if len(params) == 0 { // 无参数直接匹配
			if len(Irouter.includeRouter) == 0 || Irouter.viewSet.File != "" {
				reString = includePatternUri + "&"
			} else if strings.HasSuffix(includePatternUri, "/") == false {
				reString = includePatternUri + "/"
			} else {
				reString = includePatternUri
			}
		} else {
			if len(Irouter.includeRouter) == 0 || Irouter.viewSet.File != "" {
				reString = converterPattern + "$"
			} else if strings.HasSuffix(includePatternUri, "/") == false {
				reString = converterPattern + "/"
			} else {
				reString = converterPattern
			}
		}
		re = regexp.MustCompile(reString)

		paramsSlice := re.FindStringSubmatch(requestPattern)
		if len(paramsSlice)-1 != len(params) || len(paramsSlice) == 0 {
			continue
		}
		paramsSlice = paramsSlice[1:]
		for i := 0; i < len(params); i++ {
			param := params[i]
			value := parseValue(param.paramType, paramsSlice[i])
			PathParams[param.paramName] = append(PathParams[param.paramName], value) // 添加参数
		}
		if Irouter.viewSet.DIR != "" { // 静态路由映射
			fileName := path.Clean("/" + requestPattern[len(includePatternUri):])
			dir := string(Irouter.viewSet.DIR)
			filePath := filepath.Join(dir, fileName)
			PathParams["staticPath"] = append(PathParams["static"], filePath)
			return Irouter.viewSet, true
		} else if Irouter.viewSet.File != "" {
			PathParams["staticPath"] = append(PathParams["static"], Irouter.viewSet.File)
			return Irouter.viewSet, true
		} else if len(Irouter.includeRouter) == 0 { // API
			return Irouter.viewSet, true
		} else { // 子路由
			return routeResolution(requestPattern[len(includePatternUri):], Irouter.includeRouter, PathParams)
		}
	}
	return AsView{}, false
}

// 路由参数解析
func routerParse(UrlPath string) ([]routerParam, string) {
	var params []routerParam
	var converterPattern string
	regexpStr := `<([^/<>]+):([^/<>]+)>`
	re := regexp.MustCompile(regexpStr)
	result := re.FindAllStringSubmatch(UrlPath, -1)
	for _, paramsSlice := range result {
		if len(paramsSlice) == 3 {
			converter, ok := metaConverter[paramsSlice[1]]
			if ok == false {
				continue
			}
			re = regexp.MustCompile(paramsSlice[0])
			UrlPath = re.ReplaceAllString(UrlPath, converter)
			converterPattern = UrlPath

			param := routerParam{
				paramName: paramsSlice[2],
				paramType: paramsSlice[1],
			}
			params = append(params, param)
		}
	}
	return params, converterPattern
}

// 参数解析
func parseValue(paramType string, paramValue string) any {
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
