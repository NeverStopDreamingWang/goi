# goi

基于 `net/http` 开发的一个 Web 框架，语法与 `Django` 类似，简单易用，快速上手，详情请查看示例

[详细示例：goi_example](https://github.com/NeverStopDreamingWang/goi_example)

自定义 ORM 的公共接口、引擎注册、连接生命周期及 MySQL 方言实现示例见 [自定义 ORM 开发指南](db/README.md)。

## goi 创建命令

### 本地构建 goi 命令行工具

命令行源码入口位于 `cmd/goi/`，构建脚本位于 `cmd/`。Windows 和 Linux 分别提供对应环境的构建脚本；两个脚本都会交叉编译 Windows 与 Linux 两个系统的可执行文件。目标架构均为 `amd64`，并设置 `CGO_ENABLED=0`，因此不依赖目标系统的 C 工具链。构建命令使用 `-trimpath` 和 `-ldflags="-s -w"`，用于移除本地路径、符号表及调试信息，减小发布产物体积。

Windows 环境执行：

```bat
cmd\build.bat
```

Linux 环境执行：

```sh
sh ./cmd/build.sh
```

无论使用哪个脚本，都会生成以下两个构建产物：

- `windows/amd64`：`cmd/dist/windows-amd64/goi.exe`
- `linux/amd64`：`cmd/dist/linux-amd64/goi`

两个脚本都会自动切换到自身所在目录，因此也可以从其他工作目录通过脚本路径调用。

`cmd/dist/` 仅用于本地验证或按需生成下载附件，构建产物不纳入 Git 版本控制。

### 安装 goi 命令行工具

使用 `go install` 安装最新版本：

```shell
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

命令会安装为 `goi`（Windows 下为 `goi.exe`）。安装目录优先使用 `GOBIN`；未设置 `GOBIN` 时使用 `GOPATH/bin`。请将该目录加入 `PATH` 后再直接运行 `goi`。

`go install` 会根据发布标签中的源码在用户本机构建命令行工具，因此发布版本只需要提交源码并创建标签，无需提交预编译的 Windows 或 Linux 可执行文件。

### goi 命令使用

```shell
> goi

Usage（用法）:                                         
        goi <command> [arguments]                      
The commands are（命令如下）:                          
        create-project  myproject   创建项目       
        create-app      myapp       创建app

```

示例

```shell
# 创建项目
> goi create-project example

# 新建应用 app
> cd example
> goi create-app myapp

```

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

### 从 v2 迁移到 v3

v3 包含不兼容的 ORM API 调整。先将 Goi 及其子包的导入路径统一改为 `/v3`，再更新依赖；使用 CLI 时也需重新安装。不要修改第三方依赖自身的版本后缀。

```sh
go get github.com/NeverStopDreamingWang/goi/v3@v3.0.0
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@v3.0.0
go mod tidy
```

| 迁移项 | v3 用法 |
| --- | --- |
| 上下文 | 迁移、CRUD、计数、分页和事务方法直接接收 `context.Context`；例如 `Select(ctx, &list)`、`First(ctx, &item)`、`Delete(ctx)`。基础 `Execute`、`Query`、`QueryRow` 仍提供 `...Context` 入口。 |
| 写入 nil | `Insert(ctx, model, skipNil)`、`Update(ctx, model, skipNil)` 必须指定 `skipNil`：`true` 跳过 nil，`false` 将选中的 nil 写为 SQL NULL；普通零值不会被跳过。用 `Fields` 控制写入范围。 |
| 迁移命名空间 | 不再向 `Migrate` 传入 Schema，改在模型 `ModelSet()` 返回的 `Settings.Schema` 中设置，再调用 `Migrate(ctx, model)`。 |
| Kingbase | 通过 `kingbase.Config.Mode` 选择方言，改用对应原生包的 Engine、Model、Settings 和 Validate；例如 MySQL 模式使用 `db.Connect[*mysql.Engine]`。原包级策略配置改放 `Config.EngineOptions`。 |
| map 查询结果 | map 的键使用 Go 字段名，例如 `Name`，不是数据库列名 `name`；请同步调整读取代码。 |
| SeaBoxSQL | 新增 `seaboxsql` 路由，支持 `postgres`、`mysql`、`oracle`，空 Mode 默认 PostgreSQL；只选择 ORM 方言，不切换服务端模式或物理驱动。 |

`Update`、`Delete` 未设置 WHERE 时会操作整张表；升级后应检查筛选条件与 `Fields`。数据库模型、配置和完整示例见 [自定义 ORM 开发指南](db/README.md)。兼容模式路由不代表与对应原生数据库完全兼容，需针对目标实例验证。

### 运行 Goi

一个基础示例：

```go
package main

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

func Ping(request *goi.Request) any {
	goi.Log.DebugF("Test1")

	return response.Data{
		Code:    http.StatusOK,
		Message: "Hello World",
		Data: nil,
	}
}

func main() {
	// 创建 Goi 服务器
	Server := goi.NewHTTPServer()
	// 网络协议
	Server.Settings.Network = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6"
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

然后在浏览器中访问 `http://0.0.0.0:8080/ping` 查看响应！

### 优雅停机

Goi 会监听 `SIGINT`（例如 `Ctrl+C`）和 `SIGTERM`，也可以由业务控制层手动调用 `Server.StopServer()`。服务进入本轮运行后，两种入口共用同一轮停机流程；即使并发或重复调用，也只会执行一次清理，并向所有调用方返回同一个停机结果。正常停机导致 HTTP `Serve` 返回时不会再次发起停机；`Serve` 异常返回时仍会进入统一停机流程。`RunServer()` 会等待全部清理完成后再返回。

停机顺序为：停止 HTTP 监听并等待在途请求完成、取消并等待后台任务、逆序执行停机回调、最后关闭数据库。HTTP `Shutdown`、`OnShutdown` 和数据库 `Close` 返回的错误都会立即写入日志，剩余清理仍会继续，最终由 `StopServer()` 合并返回。

停机过程不设置超时，以确保请求或后台任务仍可能使用数据库时不会提前关闭依赖。因此，后台任务必须响应 `ctx.Done()` 并调用一次 `wg.Done()`。不要在当前 HTTP 请求处理程序、受管理的后台任务或 `OnShutdown` 回调中同步调用 `StopServer()`，否则会等待自身退出；应从独立协程或应用控制层发起。如果启动初始化、在途请求或后台任务永久阻塞，优雅停机也会一直等待，此时只能由操作系统强制终止进程。

每个 `Engine` 以及全局启动/停机任务管理器只支持一轮生命周期，停止后不能重新运行。`RegisterOnStartup` 应在 `RunServer()` 前调用，后台任务开始启动后注册不会参与本轮生命周期；`RegisterOnShutdown` 可以在服务运行期间调用，停机开始后注册不会参与本轮生命周期。在 `RunServer()` 开始前调用 `StopServer()` 会返回 `ErrServerNotRunning`。

