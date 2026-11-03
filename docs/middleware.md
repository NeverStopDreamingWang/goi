# 中间件注册、执行顺序与内置配置

**简体中文** | [English](middleware.en.md) · [文档目录](README.md) · [路由指南](routing.md)

Goi 的中间件属于路由树：根、路由组、叶子的中间件按匹配路径合并，再包装已选定的视图。它不是独立于路由匹配的全局 `net/http` 过滤器。请在开始处理请求前完成配置与 `Use` 注册。

## 完整示例：正常、短路、异常与预检

运行下面的程序只会通过 `httptest` 调用处理链，不启动监听、不创建日志文件。日志器在 `NewHTTPServer` 前设置；真实服务应配置自己的输出，再调用 `RunServer`。

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/clickjacking"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/common"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/corsheaders"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/security"
)

type traceMiddleware struct{ name string }

func (m traceMiddleware) ProcessRequest(request *goi.Request) any {
	fmt.Println("request", m.name)
	if m.name == "group" && request.Object.Header.Get("X-Block") == "1" {
		return goi.Response{Status: http.StatusForbidden, Data: "blocked"}
	}
	return nil
}

func (m traceMiddleware) ProcessException(_ *goi.Request, exception any) any {
	fmt.Println("exception", m.name)
	if m.name == "group" {
		return goi.Response{Status: http.StatusInternalServerError, Data: "handled panic"}
	}
	return nil
}

func (m traceMiddleware) ProcessResponse(_ *goi.Request, response *goi.Response) {
	fmt.Println("response", m.name)
	response.Header().Add("X-Trace", m.name)
}

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()

	cors := corsheaders.Default()
	cors.AllowedOrigins = []string{"https://client.example"}
	cors.AllowHeaders = append(cors.AllowHeaders, "Authorization")
	cors.ExposeHeaders = []string{"X-Trace"}
	server.Router.Use(common.Default(), security.Default(), clickjacking.Default(), cors, traceMiddleware{"root"})
	api := server.Router.Include("api", "API").Use(traceMiddleware{"group"})
	api.Path("ping", "ping", goi.ViewSet{
		GET: func(request *goi.Request) any {
			if request.Object.Header.Get("X-Panic") == "1" {
				panic("example")
			}
			return "ok"
		},
	}).Use(traceMiddleware{"leaf"})

	for _, mode := range []string{"normal", "blocked", "panic", "preflight"} {
		method := http.MethodGet
		if mode == "preflight" {
			method = http.MethodOptions
		}
		request := httptest.NewRequest(method, "/api/ping", nil)
		request.Header.Set("Origin", "https://client.example")
		switch mode {
		case "blocked":
			request.Header.Set("X-Block", "1")
		case "panic":
			request.Header.Set("X-Panic", "1")
		case "preflight":
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		fmt.Println(mode, recorder.Code, recorder.Header().Values("X-Trace"))
	}
}
```

四次请求的结果依次是 `normal 200 [leaf group root]`、`blocked 403 [group root]`、`panic 500 [leaf group root]`、`preflight 200 []`。预检由 CORS 层返回，后面的请求中间件和视图不会执行；放在 CORS 外层的安全响应头中间件仍能处理该响应。

## 接口、注册与生命周期

自定义类型必须实现 `goi.Middleware` 的全部三个方法；不需要的阶段返回 nil 或使用空方法体。

| 方法 | 作用 |
| --- | --- |
| `ProcessRequest(*goi.Request) any` | 返回 nil 进入内层；返回非 nil 则作为当前响应，停止进入内层中间件和视图。 |
| `ProcessException(*goi.Request, any) any` | 处理视图 panic；返回非 nil 表示已处理，后续外层异常钩子不再调用。 |
| `ProcessResponse(*goi.Request, *goi.Response)` | 修改返回的响应，例如状态、数据和响应头；通常不要在此直接写入网络。 |

在 `server.Router.Use(...)` 注册的对象参与这个根路由下**成功选定视图**的请求；`group.Use(...)` 作用于该组；`group.Path(...).Use(...)` 只作用于这个叶子。`Use` 返回当前路由；`Path` 和静态注册方法返回的叶子仅用于 `Use`，不要在它上面继续递归注册路径。

中间件实例会被多个请求复用。请求内数据可存入 `request.Params` 或通过 `request.WithContext` 更新上下文；不要把请求数据保存在共享实例字段中，除非自行同步。注册表和中间件配置不提供运行期修改的并发保证。内置 `Default()` 返回配置值，先修改配置再传给 `Use`，不要依赖注册后修改原值能更新已注册的副本。

这些配置是中间件结构体字段，不是 `Server.Settings` 的字段。`Server.Settings.SSL.Enabled` 控制服务监听是否使用 TLS，`SecurityMiddleware.SSLRedirect` 只是 HTTP 重定向，两者不同。

## 执行顺序与异常边界

只看示例中的三个 trace 层：

```text
root.ProcessRequest → group.ProcessRequest → leaf.ProcessRequest → view
root.ProcessResponse ← group.ProcessResponse ← leaf.ProcessResponse
```

当 group 的请求阶段短路时，会执行 group 和已经进入的 root 响应阶段，不进入 leaf。普通返回值通过框架转换成 `goi.Response`，直接返回 `goi.Response` 或 `*goi.Response` 可控制状态和数据。

视图 panic 时，`ProcessException` 按 leaf → group → root 尝试，遇到非 nil 响应就停止。被处理后，响应阶段继续向外执行。**中间件自身的 `ProcessRequest`、`ProcessResponse` panic 不会经过这组视图异常钩子**，包装层会尝试转换为 500；未处理异常、异常钩子再次 panic 或响应写入失败，也可能走外层恢复逻辑。不要假设所有故障都会执行所有 `ProcessResponse`。

`ProcessException` 针对 panic，不会自动处理普通返回的 `error` 或 `ValidationError`；业务代码应返回合适的响应，例如 `validationErr.Response()`。需要隐藏内部信息时，在异常钩子中生成安全文案，不要依赖默认恢复永远隐藏错误详情。`RawHandler` 在响应写入阶段才执行，其执行异常不经过这组视图异常钩子。

默认未匹配路由的 404、默认不支持方法的 405 在构建中间件链前返回，根路由 `Use` 也不会处理它们。可以显式设置 `NoRoute(ViewSet)` 和 `ViewSet.NoMethod`，让选定的 fallback 经过路由链。CORS 对默认 404/405 同样没有额外的全局拦截入口。

## common：通用请求处理

`common.Default()` 默认不启用任何过滤或跳转。

| 字段 | 默认值与实际行为 |
| --- | --- |
| `PrependWWW` | false；启用时，对不以 `www.` 开头的请求 Host 返回 301，保留 RequestURI。 |
| `DisallowedUserAgents` | 空列表；对非空 User-Agent 逐个匹配正则，匹配返回 403，空规则和无效正则被忽略。 |

跳转协议根据 `request.TLS` 或 `X-Forwarded-Proto: https` 判断。它直接使用请求 Host，不是主机白名单验证器；代理部署应先约束并清理传入的 Host 与转发头。User-Agent 过滤不是身份认证。

## security：安全响应头与 HTTPS 跳转

| 字段 | `security.Default()` 的值与行为 |
| --- | --- |
| `ContentTypeNoSniff` | true；未设置时补 `X-Content-Type-Options: nosniff`。 |
| `CrossOriginOpenerPolicy` | `same-origin`；非空且响应未设置该头时写入。 |
| `ReferrerPolicy` | `[]string{"same-origin"}`；按逗号拼接，已有响应头不覆盖。 |
| `SSLRedirect` | false；启用后，将非安全且未豁免的请求 301 跳转到 HTTPS。 |
| `SSLHost` | 空；重定向使用原 Host，非空时使用此值。 |
| `RedirectExempt` | 空列表；对去除开头 `/` 的路径先做正则匹配，正则无效时退化为前缀匹配。 |
| `HSTSSeconds` | 0；仅值大于 0、请求被判断为 HTTPS 且没有现有头时写入 HSTS 的 `max-age`。 |
| `HSTSIncludeSubdomains`、`HSTSPreload` | false；在满足 HSTS 条件时附加相应指令，不进行额外的注册或验证。 |

HTTPS 判断直接使用 TLS 状态或 `X-Forwarded-Proto` 的 `https` 值，没有可信代理名单。只在你能控制并清理转发头的部署下依赖该头。默认配置不强制 HTTPS、不启用 HSTS；这些头也不会自动创建证书或启动 TLS 监听。

## corsheaders：CORS 与预检

| 字段 | `corsheaders.Default()` 的值与行为 |
| --- | --- |
| `AllowAllOrigins` | false；true 时允许所有来源。与凭证同时启用时回显具体 Origin，不返回 `*`。 |
| `AllowedOrigins` | 空；精确白名单按 URL scheme 与 host（含端口）比较；未启用 `AllowAllOrigins` 时，字符串 `null` 需显式列入。 |
| `AllowedOriginRegexes` | 空；匹配原始 Origin 的正则列表，无效规则忽略。需要整串规则时自行添加锚点。 |
| `AllowCredentials` | false；允许的来源才添加 `Access-Control-Allow-Credentials: true`。 |
| `ExposeHeaders` | 空；对允许来源列出可暴露的响应头。 |
| `AllowHeaders` | `Accept`、`Accept-Language`、`Content-Language`、`Content-Type`；默认没有 `Authorization`。 |
| `AllowMethods` | `DELETE`、`GET`、`OPTIONS`、`PATCH`、`POST`、`PUT`；这是响应声明，不会代替 `ViewSet` 的方法配置。 |
| `PreflightMaxAge` | 86400 秒；大于 0 时写入预检缓存头，其他值不写。 |
| `URLRegex` | `^.*$`；仅对匹配的 URL.Path 生效；空或无效正则禁用处理。 |
| `AllowPrivateNetwork` | false；开启且请求携带对应 true 标记时添加允许私网响应头，不进行网络访问控制。 |

请求必须携带 Origin；预检还需满足 OPTIONS 和非空 `Access-Control-Request-Method`。在已匹配并进入中间件链的路由上，预检直接返回 200，再由响应阶段检查来源并添加允许头。**来源不允许时，也可能得到 200，但没有允许来源的 CORS 响应头**；普通请求不会因此被服务器拒绝。CORS 不是鉴权、CSRF 防护或服务端请求来源过滤。

允许来源的响应添加 `Vary: Origin`；`AllowHeaders`、`AllowMethods` 只在预检响应声明列表，不逐项校验请求中的方法与头。若认证中间件会阻止无凭证 OPTIONS，通常将 CORS 注册在它之前；是否允许某个 Origin、是否开启凭证由应用自行决定。

## clickjacking：X-Frame-Options

`clickjacking.Default()` 的 `XFrameOptions` 为 `DENY`；空值同样使用 `DENY`，设置值会转换为大写。已有 `X-Frame-Options` 响应头时不覆盖；常用配置为 `DENY` 或 `SAMEORIGIN`，实现不额外验证任意字符串，也不配置 CSP。

当前实现还检查响应头 map 中的**精确小写键** `xframe_options_exempt`，第一个值忽略大小写等于 `true` 时跳过。若使用此约定，需要直接操作 `response.Header()["xframe_options_exempt"]`；不要假设经过标准 `Header.Set` 规范化后的键一定能匹配。该标记不被自动删除，不应当作隐私或权限机制。

## 源码入口

[middleware.go](../middleware.go) 定义接口和包装顺序；[server.go](../server.go)、[recovery.go](../recovery.go) 与 [context.go](../context.go) 处理路由选择、恢复和响应写入；[settings.go](../settings.go) 定义服务设置。内置实现见 [common](../middleware/common/middleware.go)、[security](../middleware/security/middleware.go)、[corsheaders](../middleware/corsheaders/middleware.go)、[clickjacking](../middleware/clickjacking/middleware.go)。
