package goi

import (
	"embed"
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// 路由表
type MetaRouter struct {
	path          string        // 路由
	desc          string        // 描述
	viewSet       ViewSet       // 视图方法
	includeRouter []*MetaRouter // 子路由

	// 预编译
	pattern    string         // 路由正则表达式
	regex      *regexp.Regexp // 路由正则表达式匹配对象
	paramInfos []ParamInfo    // 参数信息列表
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
			pathAlreadyExistsMsg := i18n.T("router.path_already_exists", map[string]interface{}{
				"path": child.path,
			})
			panic(pathAlreadyExistsMsg)
		} else if itemRouter.pattern == child.pattern { // 指针引用相等
			pathCollisionMsg := i18n.T("router.path_regexp_collision", map[string]interface{}{
				"path":           itemRouter.path,
				"collision_path": child.path,
				"pattern":        child.pattern,
			})
			panic(pathCollisionMsg)
		}
	}
}

// Include 创建一个子路由
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
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
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
//   - viewSet: ViewSet 视图方法
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
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
//   - filePath: string 文件路径
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

// StaticDir 注册静态目录路由
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
//   - dirPath: http.Dir 静态映射路径
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

// StaticFileFS 注册 embed.FS 静态文件路由
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
//   - fileFS: embed.FS 嵌入式文件系统
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

// StaticDirFS 注册 embed.FS 静态目录路由
//
// 参数:
//   - path: string 路由
//   - desc: string 描述
//   - dirFS: embed.FS 嵌入式文件系统
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

// Route 为 MetaRouter 路由的副本
type Route struct {
	Path     string  // 路由
	Desc     string  // 描述
	ViewSet  ViewSet // 视图方法
	Children []Route // 子路由

	// 预编译
	Pattern    string         // 路由正则表达式
	Regex      *regexp.Regexp // 路由正则表达式匹配对象
	ParamInfos []ParamInfo    // 参数信息列表
}

// GetRoute 获取路由信息
//
// 返回:
//   - Route: 路由信息
func (router MetaRouter) GetRoute() Route {
	var children []Route
	if len(router.includeRouter) > 0 {
		children = make([]Route, 0, len(router.includeRouter))
		for _, itemRouter := range router.includeRouter {
			children = append(children, itemRouter.GetRoute())
		}
	}
	return Route{
		Path:       router.path,
		Desc:       router.desc,
		ViewSet:    router.viewSet,
		Children:   children,
		Pattern:    router.pattern,
		Regex:      router.regex,
		ParamInfos: router.paramInfos,
	}
}
