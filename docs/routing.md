# 路由、路径参数与静态资源

**简体中文** | [English](routing.en.md) · [文档目录](README.md)

`Engine.Router` 维护路由树，`ViewSet` 按 HTTP 方法选择视图。路由先匹配路径，再选择方法，最后执行匹配路由上的中间件与视图。请在接收请求前完成注册，不要在处理请求时修改路由树。

## 一个不启动监听的完整示例

下面用 `httptest` 直接调用 `ServeHTTP`，不创建网络监听。先配置日志器，因为 `NewHTTPServer` 不会自动创建默认日志器；示例将日志丢弃，实际项目应配置自己的日志输出。

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/security"
)

func missing(*goi.Request) any {
	return goi.Response{Status: http.StatusNotFound, Data: "not found"}
}

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()

	goi.RegisterConverter("hex", goi.Converter{
		Regex: `([0-9a-fA-F]+)`,
		ToGo: func(raw string) (any, error) {
			return strconv.ParseUint(raw, 16, 16)
		},
	})

	api := server.Router.Include("api", "API")
	api.Path("items/<int:id>/", "item", goi.ViewSet{
		GET: func(request *goi.Request) any {
			var id int
			if err := request.PathParams.Get("id", &id); err != nil {
				return err.Response()
			}
			return fmt.Sprintf("item=%d", id)
		},
		NoMethod: func(*goi.Request) any {
			return goi.Response{Status: http.StatusMethodNotAllowed, Data: "method not allowed"}
		},
	}).Use(security.Default())
	api.Path("hex/<hex:code>", "hex value", goi.ViewSet{
		GET: func(request *goi.Request) any {
			return fmt.Sprintf("%T:%v", request.PathParams["code"], request.PathParams["code"])
		},
	})
	server.Router.Path("raw", "direct writer", goi.ViewSet{
		GET: func(*goi.Request) any {
			return goi.RawHandler(func(w http.ResponseWriter, _ *goi.Request) error {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusAccepted)
				_, err := io.WriteString(w, "raw")
				return err
			})
		},
	})
	server.Router.NoRoute(goi.ViewSet{GET: missing, Options: missing, NoMethod: missing})

	for _, test := range []struct{ method, path string }{
		{http.MethodGet, "/api/items/7/"},
		{http.MethodPost, "/api/items/7/"},
		{http.MethodGet, "/api/hex/ff"},
		{http.MethodGet, "/raw"},
		{http.MethodGet, "/missing"},
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		fmt.Println(recorder.Code, recorder.Body.String())
	}
}
```

结果依次为 `200 item=7`、`405 method not allowed`、`200 uint64:255`、`202 raw`、`404 not found`。实际启动服务时，在完成日志、设置与路由注册后调用 `server.RunServer()`；不要同时运行这段请求演示循环。

## 路由树与斜杠

根路由已经消费请求路径开头的 `/`。子路由使用不带开头 `/` 的相对路径，例如 `Include("api", ...)` 下的 `Path("users/", ...)` 对应 `/api/users/`。

| 方法 | 用途与返回值 |
| --- | --- |
| `Include(path, desc)` | 创建可继续注册子路由的路由组；缺少结尾 `/` 时自动补上，返回该路由组。 |
| `Use(middlewares...)` | 按传入顺序追加中间件，返回当前路由，可继续对这个路由调用 `Use`。 |
| `Path(path, desc, viewSet)` | 创建一个叶子路由，原样保留路径及其结尾斜杠。返回的是新注册的叶子，不是父组。 |
| `StaticFile`、`StaticDir`、`StaticFileFS`、`StaticDirFS` | 创建静态资源叶子，返回该叶子，详见下节。 |

`Path` 和静态注册方法返回的 `*Router` **仅用于调用 `Use` 添加该叶子的中间件**，不要在返回值上递归调用 `Include`、`Path` 或静态注册方法；这不是受支持的路由组用法，可能产生未知行为。需要继续注册同级路由时，保留并使用父组变量。

叶子路由整串匹配；`users` 与 `users/` 不同，框架不自动补斜杠重定向，也不会规范化重复斜杠。相同父组内按注册顺序尝试，先成功匹配的路由生效；完全相同的路径或编译后模式会在注册时 panic，但并不检测所有正则重叠。

`Include` 请使用固定前缀。当前解析器按声明路径的长度裁剪子路径，不适合在组路径中放动态占位符；将动态参数放在叶子 `Path` 中。`desc` 是描述信息，不影响路径匹配。

## 路径参数与 Converter

占位符写作 `<type:name>`，例如 `<int:id>`。匹配成功后，通过 `request.PathParams.Get("id", &id)` 读取转换后的值；它与请求查询参数、请求体参数是不同的来源。

| 内置转换器 | 匹配及返回值 |
| --- | --- |
| `int` | 一个或多个 `0-9`，通过 `strconv.Atoi` 返回 int；不接受正负号。 |
| `string` | 非空且不含 `/` 的片段，返回 string。 |
| `slug` | 非空 ASCII 字母、数字、`-`、`_`，返回 string。 |
| `path` | 非空字符串，可以包含 `/`，返回 string。 |
| `uuid` | 小写十六进制 `8-4-4-4-12` 格式，返回 string。 |

自定义 `Converter.Regex` 必须为每个占位符提供**恰好一个捕获组**；内部额外分组应使用 `(?:...)`。解析器核对的是整条路由的捕获组总数与参数总数，不相等时不匹配。即使某个转换器多组、另一个少组导致总数偶然相同，也仍是不受支持的用法，参数可能错位。普通路径文字会被转义，不作为任意正则执行。

在使用转换器注册路由前调用 `RegisterConverter`。路由编译时保存转换器副本，同名重新注册不会改写已编译的路由。未知转换器或无效正则会在路由注册时 panic；`ToGo` 返回 error 则把当前路由视为未匹配，继续尝试其他路由或 `NoRoute`，不会自动生成参数验证的 400 响应。

路由的 `Converter.ToGo` 与 [参数验证器 `Validator.Parse`](validation.md) 是两个独立接口，不要一同改名或混用规则。

## HTTP 方法、NoRoute 与 NoMethod

`ViewSet` 支持 `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE`、`CONNECT`、`Options`、`TRACE`。字段名是 `Options`，不是 `OPTIONS`。

- `HEAD` 未配置时回退到 `GET`；未配置 `Options` 时有默认 OPTIONS 处理器。
- 已匹配路径但没有对应视图时，调用这个 `ViewSet.NoMethod`；未设置则默认 405。
- 某路由组的前缀匹配、但其子路由均不匹配时，可使用该组的 `NoRoute(viewSet)`；没有可用 fallback 时默认 404。
- `NoRoute` 仍是一个 `ViewSet`，会再次按请求方法选择处理器。示例同时设置 `GET`、`Options`、`NoMethod`，避免未配置 OPTIONS 时使用默认 200。
- 对已选中的 `ViewSet`，框架按 `GetMethods()` 设置 `Allow`；`NoMethod` 不会自动把未注册方法加入列表。

默认 404 和默认 405 在构建路由中间件链前返回；不能假设根路由 `Use` 一定处理这些响应。需要中间件参与时，可显式注册 `NoRoute`、`NoMethod` 处理器，详见 [中间件指南](middleware.md)。

`RawHandler` 用于自行写入 status、headers、body；其返回的数据不再进行 JSON 编码或自动补 `Content-Length`。它在响应写入阶段执行，此时路由中间件已返回；它的执行错误也不经过视图的 `ProcessException` 链。流式写入前应考虑请求上下文取消，不能在响应已写出后依赖框架改写状态码。

## 本地文件与 embed.FS

本地路径相对于进程当前工作目录，不会自动拼接 `Settings.BaseDir`。只映射适合公开访问的目录；注册静态资源不等于访问鉴权。

| 注册方法 | 路径含义 |
| --- | --- |
| `StaticFile(path, desc, filePath)` | 将一个 URL 映射到本地文件；每次请求打开文件，打开失败返回 404。 |
| `StaticDir(path, desc, http.Dir(...))` | 自动追加 `/<path:fileName>`，从该目录打开匹配的文件；空目录参数按 `.` 处理。 |
| `StaticFileFS(path, desc, fileFS, defaultPath)` | 将 URL 映射到 embed.FS 内的 `defaultPath`。 |
| `StaticDirFS(path, desc, dirFS, basePath)` | 自动追加 `/<path:fileName>`；先以 `fs.Sub` 锚定 `basePath`，再用 `fileName` 查找；空 `basePath` 按 `.` 处理。 |

这些方法注册 GET 与 HEAD。目录路由的 `fileName` 必须非空，因此 `/assets/` 本身不等于 `/assets/index.html`。本地目录对象不提供目录列表或自动首页，读到目录时返回 404；嵌入式文件响应交给 `net/http.ServeFileFS` 处理，包括它对 `index.html` 请求的重定向规则，不是 SPA fallback。

以下示例先准备 `public/index.html`（例如内容为 `<h1>Goi</h1>`）和 `public/hello.txt`（内容为 `hello`），与 `main.go` 放在同一目录树下，再编译运行。`embed` 内部路径使用 `/`，不使用操作系统的绝对文件路径。

```go
package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/NeverStopDreamingWang/goi/v3"
)

//go:embed public
var site embed.FS

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()
	server.Router.StaticFileFS("", "home", site, "public/index.html")
	server.Router.StaticDirFS("assets", "embedded assets", site, "public")
	server.Router.StaticFile("download", "local file", "public/index.html")
	server.Router.StaticDir("files", "local directory", http.Dir("public"))

	for _, path := range []string{"/", "/assets/hello.txt", "/download", "/files/index.html"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		fmt.Println(path, recorder.Code)
	}
}
```

直接组合视图时，`StaticFileFSView` 接收 `fs.FS` 与文件名，`StaticDirFSView` 只接收一个 `fs.FS`，要求路由提供名为 `fileName` 的路径参数。自行使用 `fs.Sub` 时，也要明确文件系统根目录。成功返回的本地文件对象由响应层关闭，不要在视图返回前关闭它。

## 源码入口

[router.go](../router.go) 定义注册接口；[resolvers.go](../resolvers.go) 编译及匹配路径；[converter.go](../converter.go) 定义路径转换器；[view.go](../view.go) 选择 HTTP 方法；[static.go](../static.go) 与 [context.go](../context.go) 处理文件及原始响应；[server.go](../server.go) 连接请求处理流程。
