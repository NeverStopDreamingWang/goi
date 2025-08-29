package goi

import (
	"embed"
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 路由表
type MetaRouter struct {
	path          string        // 路由
	desc          string        // 简介
	viewSet       ViewSet       // 视图方法
	includeRouter []*MetaRouter // 子路由

	// 预编译
	pattern    string
	regex      *regexp.Regexp
	paramInfos []ParamInfo // 参数信息列表
}

// 路由信息
type RouteInfo struct {
	Path         string  // 路由
	Desc         string  // 简介
	ViewSet      ViewSet // 视图方法
	IsParent     bool    // 是否是父路由
	ParentRouter string  // 父路由
}

// 创建路由
func newRouter() *MetaRouter {
	return &MetaRouter{
		path:          "/",
		desc:          "/",
		viewSet:       ViewSet{},
		includeRouter: make([]*MetaRouter, 0, 0),
		pattern:       "/",
		regex:         regexp.MustCompile("^/"),
		paramInfos:    make([]ParamInfo, 0),
	}
}

// hasChildRouter 判断是否包含指定子路由（通过指针引用校验）
// 返回 true 表示当前路由的 includeRouter 中已包含该子路由
func (router MetaRouter) hasChildRouter(child *MetaRouter) {
	if router.includeRouter == nil || child == nil {
		return
	}
	for _, itemRouter := range router.includeRouter {
		if itemRouter.path == child.path { // 指针引用相等
			pathAlreadyExistsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "router.path_already_exists",
				TemplateData: map[string]interface{}{
					"path": child.path,
				},
			})
			panic(pathAlreadyExistsMsg)
		} else if itemRouter.pattern == child.pattern { // 指针引用相等
			pathCollisionMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "router.path_regexp_collision",
				TemplateData: map[string]interface{}{
					"path":           itemRouter.path,
					"collision_path": child.path,
					"pattern":        child.pattern,
				},
			})
			panic(pathCollisionMsg)
		}
	}
}

// Include 创建一个子路由
func (router *MetaRouter) Include(path string, desc string) *MetaRouter {
	if strings.HasSuffix(path, "/") == false {
		path = path + "/"
	}
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       ViewSet{},
		includeRouter: make([]*MetaRouter, 0, 0),
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
	router.includeRouter = append(router.includeRouter, includeRouter)
	return includeRouter
}

// Path 注册一个路由
func (router *MetaRouter) Path(path string, desc string, viewSet ViewSet) {
	includeRouter := &MetaRouter{
		path:          path,
		desc:          desc,
		viewSet:       viewSet,
		includeRouter: nil,
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// StaticFile 注册静态文件路由
func (router *MetaRouter) StaticFile(path string, desc string, filePath string) {
	includeRouter := &MetaRouter{
		path: path,
		desc: desc,
		viewSet: ViewSet{
			GET: StaticFileView(filePath),
		},
		includeRouter: nil,
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件夹静态路由
// StaticDir 注册静态目录路由
func (router *MetaRouter) StaticDir(path string, desc string, dirPath http.Dir) {
	if dirPath == "" {
		dirPath = "."
	}
	if strings.HasSuffix(path, "/") == false {
		path = path + "/"
	}
	path = path + "<path:fileName>"
	includeRouter := &MetaRouter{
		path: path,
		desc: desc,
		viewSet: ViewSet{
			GET: StaticDirView(dirPath),
		},
		includeRouter: nil,
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件静态路由
// StaticFileFS 注册 embed.FS 静态文件路由
func (router *MetaRouter) StaticFileFS(path string, desc string, fileFS embed.FS) {
	includeRouter := &MetaRouter{
		path: path,
		desc: desc,
		viewSet: ViewSet{
			GET: StaticFileFSView(fileFS),
		},
		includeRouter: nil,
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
	router.includeRouter = append(router.includeRouter, includeRouter)
}

// 添加文件夹静态路由
// StaticDirFS 注册 embed.FS 静态目录路由
func (router *MetaRouter) StaticDirFS(path string, desc string, dirFS embed.FS) {
	if strings.HasSuffix(path, "/") == false {
		path = path + "/"
	}
	path = path + "<path:fileName>"
	includeRouter := &MetaRouter{
		path: path,
		desc: desc,
		viewSet: ViewSet{
			GET: StaticDirFSView(dirFS),
		},
		includeRouter: nil,
	}
	includeRouter.compilePattern()
	router.hasChildRouter(includeRouter)
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
