package goi

import (
	"embed"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 路由表
type MetaRouter struct {
	path          string        // 路由
	desc          string        // 简介
	viewSet       AsView        // 视图方法
	includeRouter []*MetaRouter // 子路由
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
func newRouter() *MetaRouter {
	return &MetaRouter{
		path:          "/",
		desc:          "根",
		viewSet:       AsView{},
		includeRouter: make([]*MetaRouter, 0, 0),
	}
}

// 判断路由是否被注册
func (router MetaRouter) isUrl(path string) {
	for _, itemRouter := range router.includeRouter {
		if itemRouter.path == path {
			pathAlreadyExistsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "router.path_already_exists",
				TemplateData: map[string]interface{}{
					"path": path,
				},
			})
			panic(pathAlreadyExistsMsg)
		}
		itemParams, itemConverterPattern := routerParse(itemRouter.path)
		params, converterPattern := routerParse(path)
		if len(itemParams) != 0 && len(params) != 0 && itemConverterPattern == converterPattern {
			pathCollisionMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "router.path_collision",
				TemplateData: map[string]interface{}{
					"path":           path,
					"collision_path": itemRouter.path,
				},
			})
			panic(pathCollisionMsg)
		}
	}
}

// 添加父路由
func (router *MetaRouter) Include(path string, desc string) *MetaRouter {
	router.isUrl(path)

	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{},
		includeRouter: make([]*MetaRouter, 0, 0),
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
	return includeRouter
}

// 添加路由
func (router *MetaRouter) UrlPatterns(path string, desc string, view AsView) {
	router.isUrl(path)
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       view,
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件静态路由
func (router *MetaRouter) StaticFilePatterns(path string, desc string, FilePath string) {
	router.isUrl(path)
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{file: FilePath},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件夹静态路由
func (router *MetaRouter) StaticDirPatterns(path string, desc string, DirPath http.Dir) {
	router.isUrl(path)

	if DirPath == "" {
		DirPath = "."
	}
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{dir: DirPath},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件静态路由
func (router *MetaRouter) StaticFilePatternsFS(path string, desc string, FileFS embed.FS) {
	router.isUrl(path)
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{fileFS: &FileFS},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件夹静态路由
func (router *MetaRouter) StaticDirPatternsFS(path string, desc string, DirFS embed.FS) {
	router.isUrl(path)

	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       AsView{dirFS: &DirFS},
		includeRouter: nil,
	}
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// Next 方法用于遍历每个路径并获取路由信息
func (router MetaRouter) Next(routerChan chan<- RouteInfo) {
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

// 路由解析
func (router MetaRouter) routeResolution(Path string, PathParams Params) (AsView, bool) {
	var re *regexp.Regexp
	params, converterPattern := routerParse(router.path)
	var reString string
	if len(params) == 0 { // 无参数直接匹配
		if len(router.includeRouter) == 0 && router.viewSet.dir == "" && router.viewSet.dirFS == nil {
			reString = "^" + router.path + "$"
		} else if strings.HasSuffix(router.path, "/") == false {
			reString = "^" + router.path + "/"
		} else {
			reString = "^" + router.path
		}
	} else {
		if len(router.includeRouter) == 0 && router.viewSet.dir == "" && router.viewSet.dirFS == nil {
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
		PathParams[params[i].paramName] = paramsSlice[i]
	}

	if router.viewSet.dir != "" || router.viewSet.dirFS != nil { // 静态路由映射
		PathParams["fileName"] = path.Clean("/" + Path[len(router.path):])
		return router.viewSet, true
	} else if len(router.includeRouter) == 0 { // API
		return router.viewSet, true
	}
	// 子路由
	for _, itemRouter := range router.includeRouter {
		viewSet, isPattern := itemRouter.routeResolution(Path[len(router.path):], PathParams)
		if isPattern == true {
			return viewSet, isPattern
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
			if !ok {
				converterIsNotExistsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "router.converter_is_not_exists",
					TemplateData: map[string]interface{}{
						"name": paramsSlice[1],
					},
				})
				panic(converterIsNotExistsMsg)
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
