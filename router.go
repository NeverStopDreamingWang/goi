package goi

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// HandlerFunc 定义 goi 使用的请求处理程序
type HandlerFunc func(request *Request) interface{}

// type HandlerFunc func(*http.Request)

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
	file    string   // 文件
	dir     http.Dir // 文件夹
}

// 路由表
type metaRouter struct {
	path          string        // 路由
	desc          string        // 简介
	viewSet       AsView        // 视图方法
	includeRouter []*metaRouter // 子路由
}

// 路由信息
type RouteInfo struct {
	Path         string // 路由
	Desc         string // 简介
	ViewSet      AsView // 视图方法
	IsParent     bool   // 是否是父路由
	ParentRouter string // 父路由
}

// 路由参数
type routerParam struct {
	paramName string
	paramType string
}

// 创建路由
func newRouter() *metaRouter {
	return &metaRouter{
		path:          "",
		desc:          "根",
		viewSet:       AsView{},
		includeRouter: make([]*metaRouter, 0, 0),
	}
}

// 判断路由是否被注册
func (router metaRouter) isUrl(path string) {
	for _, itemRouter := range router.includeRouter {
		if itemRouter.path == path {
			panic(fmt.Sprintf("路由已存在: %s\n", path))
		}
		itemParams, itemConverterPattern := routerParse(itemRouter.path)
		params, converterPattern := routerParse(path)
		if len(itemParams) != 0 && len(params) != 0 && itemConverterPattern == converterPattern {
			panic(fmt.Sprintf("%s 冲突路由: %s\n", path, itemRouter.path))
		}
	}
}

// 添加父路由
func (router *metaRouter) Include(path string, desc string) *metaRouter {
	router.isUrl(path)

	includeRouter := &metaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{},
		includeRouter: make([]*metaRouter, 0, 0),
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
	return includeRouter
}

// 添加路由
func (router *metaRouter) UrlPatterns(path string, desc string, view AsView) {
	router.isUrl(path)
	includeRouter := &metaRouter{
		path:          path,
		desc:          desc,
		viewSet:       view,
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件静态路由
func (router *metaRouter) StaticFilePatterns(path string, desc string, FilePath string) {
	router.isUrl(path)
	includeRouter := &metaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{GET: metaStaticHandler, file: FilePath},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件夹静态路由
func (router *metaRouter) StaticDirPatterns(path string, desc string, DirPath http.Dir) {
	router.isUrl(path)

	if DirPath == "" {
		DirPath = "."
	}
	includeRouter := &metaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{GET: metaStaticHandler, dir: DirPath},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// Next 方法用于遍历每个路径并获取路由信息
func (router metaRouter) Next(routerChan chan<- RouteInfo) {
	// 遍历子路由
	for _, itemRouter := range router.includeRouter {
		// 发送路由的信息到通道
		routerChan <- RouteInfo{
			Path:         itemRouter.path,
			Desc:         itemRouter.desc,
			ViewSet:      itemRouter.viewSet,
			IsParent:     itemRouter.includeRouter != nil,
			ParentRouter: router.path,
		}
		if itemRouter.includeRouter != nil {
			// 递归调用 Next 方法获取子路由的信息
			itemRouter.Next(routerChan)
		}
	}
}

// 路由器处理程序
func (router metaRouter) routerHandlers(request *Request) (handlerFunc HandlerFunc, err string) {

	view_methods, isPattern := router.routeResolution(request.Object.URL.Path, request.PathParams)
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
func (router metaRouter) routeResolution(Path string, PathParams metaValues) (AsView, bool) {
	var re *regexp.Regexp
	params, converterPattern := routerParse(router.path)
	var reString string
	if len(params) == 0 { // 无参数直接匹配
		if len(router.includeRouter) == 0 && router.viewSet.file == "" && router.viewSet.dir == "" {
			reString = "^" + router.path + "$"
		} else if strings.HasSuffix(router.path, "/") == false {
			reString = "^" + router.path + "/"
		} else {
			reString = "^" + router.path
		}
	} else {
		if len(router.includeRouter) == 0 && router.viewSet.file == "" && router.viewSet.dir == "" {
			reString = "^" + converterPattern + "$"
		} else if strings.HasSuffix(router.path, "/") == false {
			reString = "^" + converterPattern + "/"
		} else {
			reString = "^" + converterPattern
		}
	}
	re = regexp.MustCompile(reString)

	paramsSlice := re.FindStringSubmatch(Path)
	if len(paramsSlice)-1 != len(params) || len(paramsSlice) == 0 {
		return AsView{}, false
	}
	paramsSlice = paramsSlice[1:]
	for i := 0; i < len(params); i++ {
		param := params[i]
		PathParams[param.paramName] = append(PathParams[param.paramName], paramsSlice[i]) // 添加参数
	}
	if router.viewSet.file != "" {
		PathParams["static_path"] = append(PathParams["static"], router.viewSet.file)
		return router.viewSet, true
	} else if router.viewSet.dir != "" { // 静态路由映射
		fileName := path.Clean("/" + Path[len(router.path):])
		dir := string(router.viewSet.dir)
		filePath := filepath.Join(dir, fileName)
		PathParams["static_path"] = append(PathParams["static"], filePath)
		return router.viewSet, true
	} else if len(router.includeRouter) == 0 { // API
		return router.viewSet, true
	}
	// 子路由
	for _, itemRouter := range router.includeRouter {
		view_methods, isPattern := itemRouter.routeResolution(Path[len(router.path):], PathParams)
		if isPattern == true {
			return view_methods, isPattern
		}
	}
	return AsView{}, false
}

// 路由参数解析
func routerParse(path string) ([]routerParam, string) {
	var params []routerParam
	var converterPattern string
	regexpStr := `<([^/<>]+):([^/<>]+)>`
	re := regexp.MustCompile(regexpStr)
	result := re.FindAllStringSubmatch(path, -1)
	for _, paramsSlice := range result {
		if len(paramsSlice) == 3 {
			converter, ok := metaConverter[paramsSlice[1]]
			if ok == false {
				continue
			}
			re = regexp.MustCompile(paramsSlice[0])
			path = re.ReplaceAllString(path, converter)
			converterPattern = path

			param := routerParam{
				paramName: paramsSlice[2],
				paramType: paramsSlice[1],
			}
			params = append(params, param)
		}
	}
	return params, converterPattern
}
