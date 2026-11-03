# 请求、解析器与响应

**简体中文** | [English](http.en.md) · [文档目录](README.md) · [参数验证](validation.md)

本文说明 HTTP 数据如何进入 `Request`、何时解析和校验，以及视图返回值如何写入响应。路由视图的签名为 `func(*goi.Request) any`，不是直接接收 `http.ResponseWriter` 的标准库处理函数。

源码：[context.go](../context.go)、[params.go](../params.go)、[view.go](../view.go)、[parser/](../parser/)、[response/](../response/)。

## 1. 请求对象与参数来源

| 入口 | 内容与规则 |
| --- | --- |
| `request.Object` | 原始 `*http.Request`；从这里读取 Header、Cookie、Body、MultipartForm 和 Context。 |
| `request.PathParams` | 路由 `<type:name>` 捕获的参数；值已经过路由 `Converter.ToGo` 转换。它与参数验证器 `Validator.Parse` 是两套接口。 |
| `request.QueryParams()` | 每次根据 URL 生成新的 `Params`：单值返回 `string`，单个空值返回 `nil`，重复键返回 `[]string`。 |
| `request.BodyParams()` | 根据请求 `Content-Type` 选择解析器，再读取请求体。表单解析器也可能合并 URL 查询参数，见下文。 |
| `request.Params` | 框架为每个请求创建的自定义 `Params`，供视图和中间件共享；不会自动填入查询或请求体参数。 |

例如 `?name=goi&empty=&tag=a&tag=b` 对应 `name: "goi"`、`empty: nil`、`tag: []string{"a", "b"}`。这与表单中的空字符串规则不同，不要把所有输入源当作同一种类型。

`Params.Get(key, &dest)` 读取单个键并调用 `SetValue` 赋值，不会执行 `type` 标签验证器；`Params.ParseParams(&struct)` 根据标签调用 `Validator.Parse`。两者都返回 `ValidationError`，视图中应检查后返回 `err.Response()`，而不是把错误对象直接当作成功数据返回。字段标签、`required` 和 `allow_null` 规则见 [参数验证](validation.md)。

### 请求上下文

用 `request.Object.Context()` 将请求的取消信号传给 ORM、HTTP 客户端或其他支持 `context.Context` 的操作。`request.WithContext(ctx)` 会替换 `Object` 为 `Object.WithContext(ctx)` 的结果，不会新建 `PathParams` 或 `Params`。

扩展请求上下文时应从原上下文派生，避免用新的 `context.Background()` 丢掉请求取消链。`ServeHTTP` 会在原上下文上加入框架的 `requestID`；业务自定义值宜使用自己的键类型，避免与其他组件冲突。

## 2. 请求体解析

解析器接口和注册方法如下；这里的注册名是用于查找的 MIME 字符串，不是 `Name()` 的返回值：

```text
type Parser interface {
    Name() string
    Parse(*http.Request) (parser.Params, error)
}
parser.RegisterParser(mimeType, implementation)
parser.GetParser(contentType)
```

| 解析器 | 默认匹配的 Content-Type | 返回值或限制 |
| --- | --- | --- |
| `parser.JSON` | `application/json` | 解码到 `map[string]any`；普通数字默认是 `float64`。顶层 JSON 对象可用，`null` 返回 nil map，非对象的数组/标量会报错。 |
| `parser.YAML` | `application/x-yaml` | 用 YAML 解码器写入 `parser.Params`。 |
| `parser.FormMultipart` | `multipart/form-data` | 使用 32 MiB 内存阈值解析 multipart；返回 `request.Form` 中的普通字段，不包含上传文件。 |
| `parser.Form` | 未匹配 MIME 时的默认解析器 | 解析 URL 查询参数及表单，支持 URL 编码和 multipart；单值是 `string`，重复键是 `[]string`，空字符串不会改成 nil。 |
| `parser.FormPost` | 未单独注册 | 仅从 `request.PostForm` 取 URL 编码表单值；需要显式使用或自行注册。 |
| `parser.XML` | `application/xml`、`text/xml` | 当前实现把标准库 XML 解码器的目标设为 map，不能作为通用 XML 参数解析使用。请自定义面向结构体的解析器；不要依赖此入口正常读取 XML。 |

### MIME 匹配是精确匹配

`GetParser` 直接用传入的完整字符串查表，不自动去除 `charset`、multipart `boundary` 等参数，也不自动识别 `+json`。因此 `application/json; charset=utf-8` 不会直接命中 JSON；未知或空 Content-Type 会回退到 `Form`，不是自动返回 415。

需要兼容带参数的 MIME 时，先用标准库 `mime.ParseMediaType` 取得媒体类型，再调用 `parser.GetParser(mediaType).Parse(request.Object)`，保留原始请求头供 multipart 解码器读取 boundary。若应用只允许特定格式，应在选择解析器前自行判断并返回 415。下方完整示例采用这种方式。

### 便捷方法、错误与重复读取

- `BodyParamsJSON()`、`BodyParamsYAML()` 固定选用对应解析器；`BodyParamsParsing(p)` 使用指定解析器。`BodyParamsXML()` 存在上述 XML 限制。
- 所有 `BodyParams*` 便捷方法在解析器返回 error 时都会 `panic(err)`，没有 error 返回值；不要据此假设会自动返回 400。未由异常中间件处理的 panic 会进入框架异常处理，且已经写出的响应无法重新替换。
- 如果希望明确把格式错误返回为 400，直接调用 `Parser.Parse` 并检查 error。传给 `BodyParamsParsing` 或注册的解析器不能为 nil。
- JSON/YAML 解析不缓存请求体；重复调用会继续消费同一个 Body，可能得到 EOF。不要先读完 Body，再期望便捷方法重新解析。
- JSON 当前只执行一次 `Decode`，不额外检查后续内容或做请求体大小限制。`EnableDecoderUseNumber` 会把普通数字保留为 `json.Number`；应用验证器也需支持该类型。`EnableDecoderDisallowUnknownFields` 对当前 map 目标不能提供结构体式未知字段检查。

### multipart 文件

普通字段与文件分开读取。可通过 `request.Object.FormFile("file")` 或解析后的 `request.Object.MultipartForm.File` 获取文件；检查错误后关闭打开的文件，必要时调用 `MultipartForm.RemoveAll()` 清理临时文件。不要直接把客户端提供的文件名当作可信保存路径。

32 MiB 是 `ParseMultipartForm` 的内存阈值，不是上传请求总大小上限；超过阈值的文件内容可能落到临时磁盘。需要总量限制时，应在读取前通过应用的 HTTP 层或代理配置限制。不要把 `BodyParams()` 返回的字段 map 当作文件集合。

## 3. 返回值与响应写入

一般视图返回值会被转换为 HTTP 200 的 `goi.Response`；返回已有的 `goi.Response` 或 `*goi.Response` 则使用其配置，不会自动给零值 `Status` 补 200。因此显式创建 `Response` 时应设置合法状态码。

| `Response.Data` 或直接返回值 | 写入方式 |
| --- | --- |
| `string` | 文本；未手动设置时 Content-Type 为 `text/plain`。 |
| `[]byte` | 二进制；默认 `application/octet-stream`。 |
| 结构体、map、`response.Data`、`response.Result` 等 | `json.Marshal`；默认 `application/json`。`Data` 的字段是 `code/message/data`，`Result` 则是 `code/message/result`，不会自动包裹所有返回值。 |
| `*os.File`、`http.File` | 调用 `http.ServeContent`，结束后关闭文件；不存在或为目录时返回 404。 |
| `goi.FS`、`fs.FS` / `embed.FS` | 交给 `http.ServeFileFS`；`goi.FS.Name` 可指定文件名，直接返回文件系统则使用请求 URL 路径。 |
| `io.Reader` | 以 `io.Copy` 写入，使用 `Response.Status`；若同时实现 `io.Closer`，结束后关闭。 |
| `goi.RawHandler` | 由回调直接写状态、响应头和正文，框架不做 JSON 编码或自动设置 Content-Length。 |
| `nil`、nil `*goi.Response` | 不写正文。当前 `Data == nil` 分支也不会调用 `WriteHeader`，未由其他代码写状态时最终是标准库默认的 200。 |

需要带明确状态的空响应时，可返回 `goi.Response{Status: http.StatusNoContent, Data: ""}`，或在 `RawHandler` 中显式写状态；不要依赖 `Response{Status: 204, Data: nil}` 写出 204。

用 `resp.Header().Set(...)` 添加响应头。普通文本、字节和 JSON 分支会在未指定时补 `Content-Length`；文件和 RawHandler 等特殊分支有各自的状态和输出控制，不能假设 `Response.Status` 会覆盖底层标准库处理结果。返回 `error` 本身没有自动的错误响应语义，应自行转换为 `Response`。

`RawHandler` 的真实签名是 `func(http.ResponseWriter, *goi.Request) error`。回调在路由中间件返回后的响应写入阶段执行，其返回的 error 或 panic 不经过视图的 `ProcessException` 链，详见 [中间件指南](middleware.md)。回调可通过 `http.Flusher` 刷新输出；框架包装器的 `Flush` 在底层不支持时是 no-op，不保证所有流式能力。它也没有透传所有可选接口，例如 `http.Hijacker`。回调返回 error 会进入响应错误处理；若状态或正文已经写出，不能指望框架再完整重写错误响应。

## 4. 完整示例

以下程序展示路径/查询参数、带参数 MIME 的解析、自定义 MIME、multipart 文件和原始响应。需要在项目中引入 Goi 模块后构建；运行会监听本地 8080 并创建当前目录的 `server.log`。示例仅演示读取上传信息，不保存文件。

```go
package main

import (
	"context"
	"io"
	"mime"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/parser"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

type requestKey struct{}
type vendorJSONParser struct{}

func (vendorJSONParser) Name() string { return "vendor-json" }
func (vendorJSONParser) Parse(r *http.Request) (parser.Params, error) {
	return parser.JSON.Parse(r)
}

func detail(request *goi.Request) any {
	var id int
	if err := request.PathParams.Get("id", &id); err != nil {
		return err.Response()
	}
	var query struct {
		Page int `name:"page" type:"int"`
	}
	if err := request.QueryParams().ParseParams(&query); err != nil {
		return err.Response()
	}
	request.Params.Set("entity_id", id)
	request.WithContext(context.WithValue(request.Object.Context(), requestKey{}, id))
	resp := goi.Response{Status: http.StatusOK, Data: response.Data{
		Code: 200, Message: "ok", Data: map[string]any{"id": id, "page": query.Page},
	}}
	resp.Header().Set("X-Example", "goi")
	return resp
}

func echo(request *goi.Request) any {
	mediaType, _, err := mime.ParseMediaType(request.Object.Header.Get(goi.ContentType))
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	switch mediaType {
	case parser.MIMEJSON, parser.MIMEYAML, "application/vnd.example+json":
	default:
		return goi.Response{Status: http.StatusUnsupportedMediaType, Data: "unsupported content type"}
	}
	params, err := parser.GetParser(mediaType).Parse(request.Object)
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	return goi.Response{Status: http.StatusCreated, Data: response.Result{
		Code: 201, Message: "ok", Result: params,
	}}
}

func upload(request *goi.Request) any {
	if err := request.Object.ParseMultipartForm(32 << 20); err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	if request.Object.MultipartForm != nil {
		defer request.Object.MultipartForm.RemoveAll()
	}
	file, header, err := request.Object.FormFile("file")
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	defer file.Close()
	return response.Data{Code: 200, Message: "ok", Data: map[string]any{
		"name": header.Filename, "size": header.Size,
	}}
}

func stream(request *goi.Request) any {
	return goi.RawHandler(func(w http.ResponseWriter, r *goi.Request) error {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "hello\n"); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	})
}

func main() {
	logger := goi.NewLogger("server.log")
	goi.Log = logger
	defer logger.Close()
	server := goi.NewHTTPServer()
	parser.RegisterParser("application/vnd.example+json", vendorJSONParser{})
	server.Router.Path("items/<int:id>/", "item", goi.ViewSet{GET: detail})
	server.Router.Path("echo/", "echo", goi.ViewSet{POST: echo})
	server.Router.Path("upload/", "upload", goi.ViewSet{POST: upload})
	server.Router.Path("stream/", "stream", goi.ViewSet{GET: stream})
	server.RunServer()
}
```

停机和后台任务的生命周期见 [服务生命周期](lifecycle.md)。请求体解析、参数验证和响应编码是三个不同阶段；选择解析器不会自动验证业务字段，验证成功也不会自动决定响应格式。
