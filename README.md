<div align="center">

# Goi

### 基于 Go 标准库的可扩展 Web 框架

从项目创建、请求处理到数据库访问与优雅停机，让 Web 开发更专注于业务。

基于 `net/http`，集成路由、中间件、参数校验、国际化和多数据库 ORM；支持自定义扩展，通过项目与应用脚手架减少重复工作。

[![Version](https://img.shields.io/github/v/release/NeverStopDreamingWang/goi?label=version&color=blue)](https://github.com/NeverStopDreamingWang/goi/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/NeverStopDreamingWang/goi?label=Go&logo=go&logoColor=white)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/NeverStopDreamingWang/goi/v3.svg)](https://pkg.go.dev/github.com/NeverStopDreamingWang/goi/v3)
[![Stars](https://img.shields.io/github/stars/NeverStopDreamingWang/goi?label=stars&color=blue)](https://github.com/NeverStopDreamingWang/goi/stargazers)
[![License](https://img.shields.io/github/license/NeverStopDreamingWang/goi?label=license)](LICENSE)

**简体中文** | [English](README.en.md)

[快速开始](#快速开始) · [其它特性](#其它特性) · [开发文档](docs/README.md) · [CLI 使用说明](cmd/goi/README.md) · [自定义 ORM](db/README.md) · [详细示例](https://github.com/NeverStopDreamingWang/goi_example) · [更新日志](https://github.com/NeverStopDreamingWang/goi/releases)

</div>

## goi 命令行工具

### 安装 goi 命令行工具

推荐使用 `go install` 安装最新版本：

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

命令会安装为 `goi`（Windows 下为 `goi.exe`）。安装目录优先使用 `GOBIN`，未设置时使用 `GOPATH/bin`；请将该目录加入 `PATH`。

### 命令使用说明

```text
goi create-project [path]<project-name>   创建项目配置包
goi create-app [path]<app-name>           创建应用
goi help [command]                      查看帮助
```

`[path]` 是可选的相对父路径前缀，名称为必填参数；方括号和尖括号不需要实际输入。

详情查看 [Goi CLI 使用说明](cmd/goi/README.md)。

## 快速开始

### 前置条件

Goi 需要 Go 1.24 或更高版本。

### 获取 Goi

通过 Go 的模块支持，当你在代码中添加导入时，`go [build|run|test]` 会自动获取必要的依赖：

```go
import "github.com/NeverStopDreamingWang/goi/v3"
```

或者，使用 `go get`：

```bash
go get github.com/NeverStopDreamingWang/goi/v3@latest
```


### 运行 Goi

#### 一个基础示例

```go
package main

import (
	"log"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

func Ping(request *goi.Request) any {
	goi.Log.DebugF("Test1")

	return response.Data{
		Code:    http.StatusOK,
		Message: "Hello World",
		Data:    nil,
	}
}

func main() {
	// 配置默认日志；NewHTTPServer 不会自动创建日志器。
	logger := goi.NewLogger("server.log")
	goi.Log = logger
	defer func() {
		if err := logger.Close(); err != nil {
			log.Print(err)
		}
	}()
	// 创建 Goi 服务器
	Server := goi.NewHTTPServer()
	// 网络协议
	Server.Settings.Network = "tcp" // 默认 "tcp"，HTTP 常用 "tcp"、"tcp4"、"tcp6"
	// 监听地址 0.0.0.0
	Server.Settings.BindAddress = "0.0.0.0" // 默认 127.0.0.1
	// 监听端口 8080
	Server.Settings.Port = 8080

	// 注册路由
	Server.Router.Path("ping", "测试接口", goi.ViewSet{GET: Ping})

	// 启动服务器
	Server.RunServer()
}
```

运行代码，使用 `go run` 命令：

```bash
go run main.go
```

然后在浏览器中访问 `http://127.0.0.1:8080/ping` 查看响应！

#### 从 CLI 开始

也可以使用 CLI 创建项目骨架，再按业务需要添加应用。先安装 CLI，并创建、进入项目根目录：

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
mkdir project
cd project
goi create-project path/myproject
goi create-app path/myapp
go mod tidy
```

生成的主要目录结构如下：

```text
project/
├── go.mod                    # module project
├── server/
│   └── main.go               # 服务入口
└── path/
    ├── myproject/            # 项目配置包
    │   ├── settings.go        # 服务与数据库等配置
    │   ├── urls.go            # 项目路由文件，初始仅有 package 声明
    │   └── ...                # 转换器、验证器、日志与 TLS 配置
    └── myapp/                # 应用包
        ├── models.go          # 模型与显式迁移入口
        ├── serializer.go      # 数据校验与写入
        ├── urls.go            # 应用路由注册
        ├── views.go           # 请求处理
        └── utils.go           # 应用帮助方法
```

`go.mod` 位于当前项目根目录，默认 module 为根目录名 `project`。`server/main.go` 是启动入口，`path/myproject` 保存项目配置，`path/myapp` 保存应用代码；`path` 是示例父路径，可按需要替换或省略。

创建应用后，编辑 `server/main.go`，导入应用以注册路由；需要建表时，在服务启动前显式调用迁移：

```go
package main

import (
	"context"

	"project/path/myapp"
	"project/path/myproject"
)

func main() {
	myapp.Migrate(context.Background()) // 按需创建数据表，不修改已有表结构
	myproject.Server.RunServer()
}
```

生成项目默认使用 SQLite，运行前需要启用 CGO 并安装可用的 C 编译器。在项目根目录启动：

```sh
go run ./server
```

推荐参考完整的 [goi_example 示例项目](https://github.com/NeverStopDreamingWang/goi_example)，了解项目组织、配置和应用开发方式。创建规则及完整流程见 [Goi CLI 使用说明](cmd/goi/README.md)。

### 停止服务

在终端按 `Ctrl+C`，或向进程发送 `SIGTERM`，即可停止服务。服务启动后，也可以从应用控制层或独立协程手动停止：

```go
if err := Server.StopServer(); err != nil {
	goi.Log.Error(err)
}
```

`StopServer()` 会等待清理完成并返回错误。不要在当前请求处理程序、受管理的后台任务或停机回调中同步调用它，以免等待自身退出。

## 其它特性

以下按用途列出框架的主要能力与扩展入口；详细用法见 [开发文档](docs/README.md)，完整应用可参考 [goi_example 示例项目](https://github.com/NeverStopDreamingWang/goi_example)。

| 特性 | 说明与入口 |
| --- | --- |
| 启动任务 | 实现 `Startup`，在 `RunServer()` 前调用 `RegisterOnStartup` 注册后台任务；任务应响应 `ctx.Done()`，退出时调用一次 `wg.Done()`。详见 [启动任务指南](docs/lifecycle.md)。 |
| 停机回调 | 实现 `ShutdownCallback`，通过 `RegisterOnShutdown` 注册清理逻辑；回调按注册顺序逆序执行，此时后台任务已退出、数据库尚未关闭。停机开始前可继续注册。详见 [停机回调指南](docs/lifecycle.md)。 |
| 路由与分组 | 使用 `Router.Path`、`Include` 和 `ViewSet` 组织路由与 HTTP 方法，并通过 `Router.NoRoute`、`ViewSet.NoMethod` 自定义未匹配路由和不支持方法的处理。详见 [路由与分组指南](docs/routing.md)。 |
| 路径参数转换 | 内置 `int`、`string`、`slug`、`path`、`uuid` 转换器；通过 `RegisterConverter` 扩展 `<type:name>` 的匹配与值转换。详见 [路径参数转换指南](docs/routing.md)。 |
| 中间件扩展 | 通过 `Router.Use` 配置全局、分组或单路由中间件；`Middleware` 提供请求、响应和异常处理入口。详见 [中间件扩展指南](docs/middleware.md)。 |
| 内置中间件 | 按需启用 CORS、安全响应头、HTTPS 重定向、防点击劫持及通用请求处理等中间件。详见 [内置中间件指南](docs/middleware.md)。 |
| 静态资源 | 支持本地文件、目录及 `embed.FS` 内嵌资源，通过 `StaticFile`、`StaticDir` 和对应的 `FS` 方法注册。详见 [静态资源指南](docs/routing.md)。 |
| 请求解析 | 读取查询参数、JSON、YAML、普通表单和 multipart 表单；通过 `parser.RegisterParser` 注册自定义请求体解析器。详见 [请求解析指南](docs/http.md)。 |
| 参数校验 | 使用 `Params.Get` 读取并赋值，或通过 `ParseParams` 调用验证器完成校验、转换和赋值；内置布尔、整数、字符串、时间、切片、map、slug 和 UUID 验证器，也可实现 `Validator.Parse` 扩展。见 [参数验证与类型转换](docs/validation.md)。 |
| 验证错误定制 | 实现 `ValidationError`，通过 `Server.Validation.SetValidationError` 定制错误消息和响应格式。详见 [验证错误定制指南](docs/validation.md)。 |
| 请求上下文 | 保留原始 `http.Request`，通过 `request.Object.Context()` 传递取消信号，也可使用 `Request.WithContext` 更新上下文。详见 [请求上下文指南](docs/http.md)。 |
| 响应控制 | 提供 `response.Data`、`response.Result` 和 `goi.Response`；可自定义状态码、响应头及文件响应，通过 `RawHandler` 控制原始响应写入。详见 [响应控制指南](docs/http.md)。 |
| 数据库与 ORM | 提供多数据库方言、事务、上下文操作和兼容模式路由；支持自定义引擎注册，以及标识符、占位符、列解析策略扩展。见 [自定义 ORM 开发指南](db/README.md)。 |
| 内存缓存 | 通过 `Server.Cache` 配置进程内缓存、TTL、清理、容量与淘汰行为；各模式的实际语义和现有限制详见 [内存缓存](docs/cache.md)。 |
| 日志 | 支持分级、格式化和访问日志、自定义输出及日志切割；切割任务可通过 `RegisterOnStartup` 注册，`Logger.Close()` 的调用时机由用户管理。详见 [日志指南](docs/logging.md)。 |
| HTTPS 与证书 | 通过 `Server.Settings.SSL` 配置证书和私钥启用 HTTPS，并提供 RSA、ECC 自签名证书生成工具。详见 [HTTPS 与证书指南](docs/configuration.md)。 |
| 国际化 | 使用 `Server.Settings.SetLanguage` 在框架内置的简体中文与英文消息间全局切换。详见 [国际化指南](docs/configuration.md)。 |
| 时间与时区 | 通过 `SetTimeZone`、`UseTZ` 配置时间表示，使用 `GetTime`、`GetLocation` 和 `Localize` 处理时间。详见 [时间与时区指南](docs/configuration.md)。 |
| 密码哈希 | `auth.MakePassword`、`auth.CheckPassword` 提供 PBKDF2-SHA256 加盐密码哈希与校验。详见 [密码哈希指南](docs/security.md)。 |
| 密码学工具 | 提供 AES、RSA 加解密、RSA 密钥生成及密码学随机字符串和字节工具。详见 [密码学工具指南](docs/security.md)。 |

