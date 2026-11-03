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

[快速开始](#快速开始) · [CLI 使用说明](cmd/goi/README.md) · [自定义 ORM](db/README.md) · [详细示例](https://github.com/NeverStopDreamingWang/goi_example) · [更新日志](https://github.com/NeverStopDreamingWang/goi/releases)

</div>

## goi 创建命令

创建规则、分组应用目录和完整示例见 [Goi CLI 使用说明](cmd/goi/README.md)。

### 安装 goi 命令行工具

推荐优先使用 `go install` 安装最新版本：

```shell
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

命令会安装为 `goi`（Windows 下为 `goi.exe`）。安装目录优先使用 `GOBIN`；未设置 `GOBIN` 时使用 `GOPATH/bin`。请将该目录加入 `PATH` 后再直接运行 `goi`。

`go install` 会根据发布标签中的源码在用户本机构建命令行工具，因此发布版本只需要提交源码并创建标签，无需提交预编译的 Windows 或 Linux 可执行文件。

### 本地构建 goi 命令行工具

需要修改源码或自定义构建参数时，再使用本地构建方式。

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

### goi 命令使用

```text
> goi

Usage（用法）:                                         
        goi <command> [arguments]                      
The commands are（命令如下）:                          
        goi create-project [path]<project-name>   创建项目
        goi create-app [path]<app-name>           创建app

```

`[path]` 是可选的相对父路径前缀，可包含多级目录；由用户按系统路径格式填写并连接名称，所需分隔符包含在该前缀中。`<project-name>`、`<app-name>` 是必填名称。方括号和尖括号都是语法标记，不需要实际输入。

示例

```shell
# 创建并进入外层项目根目录
> mkdir project
> cd project
# 配置包生成到 path/myproject
> goi create-project path/myproject
# 两条 goi 命令之间不再 cd
> goi create-app path/myapp
> go mod tidy

```

也可以不带父路径：在新建并进入项目根目录后，用下面两条命令替换上面的两条 `goi` 命令。两种方案二选一，不要依次重复执行：

```sh
goi create-project myproject
goi create-app myapp
```

省略父路径时，配置包和应用直接创建在当前根目录的 `myproject/`、`myapp/` 下。后文目录和导入示例仍采用带 `path/` 的方案；省略时应同步去掉导入中的 `/path`。

默认 module 使用执行 `create-project` 时当前项目根目录的末级目录名，此例为 `project`；也可通过 `--module example.com/team/myproject` 显式覆盖。示例中的 `path` 可替换、可多级，也可整体省略，末级 `myproject`、`myapp` 分别是项目配置包名和应用包名，不决定 module。`create-project` 在当前目录生成 `go.mod` 和简单的 `server/main.go`，参数只指定配置包的相对路径。`create-app` 同样接收相对路径，完整路径用于目录和导入，末级名称作为包名。

配置包同时生成 `urls.go`，初始仅包含 `package` 声明。

两种路径都支持 `/`，Windows 下也可使用 `\`；拒绝绝对路径、`..`、`.`、空路径段和非法末级包名。末级名称必须是 ASCII Go 包名，不能使用 `main`、`_` 或 Go 关键字。已有生成目标会被拒绝，不覆盖原代码、配置或密钥。

`create-app` 只检查当前目录的 `go.mod`，缺少时会报错，不向上搜索，也不从子目录执行；可运行 `goi help create-app` 查看用法。配置包在当前模块内递归识别，要求 `settings.go` 声明 `Server`；隐藏目录、`vendor`、`node_modules` 和嵌套 Go 模块会被跳过，多个候选配置包会报错。完整目录树见 [CLI 使用说明](cmd/goi/README.md)。

### 生成项目的数据库与应用初始化

新项目默认使用 SQLite，连接名称为 `default`，并导入 `github.com/mattn/go-sqlite3` 物理驱动。数据库目录在首次连接时创建，日志目录也会按目录权限创建。该 SQLite 驱动需要 `CGO_ENABLED=1` 和可用的 C 编译器；上面的 `CGO_ENABLED=0` 脚本只用于构建 Goi CLI，不是生成应用的构建配置。

新 app 的默认路由和表名使用末级名称，例如 `path/myapp` 默认使用 `myapp/` 和 `myapp_my_model`。不同分组使用相同末级名称时，需要自行修改默认路由、表名及导入别名，避免冲突。导入 app 只注册路由，不在 `init` 中自动执行数据库迁移；需要建表时，由应用在数据库配置完成后显式调用。手动修改 `server/main.go`，以下对应上面默认 module 为 `project` 的项目，直接使用配置包名 `myproject`：

```go
package main

import (
	"context"

	"project/path/myapp"
	"project/path/myproject"
)

func main() {
	myapp.Migrate(context.Background()) // 按需执行，不负责修改已有表结构
	myproject.Server.RunServer()
}
```

完成导入后，在同一项目根目录使用 `go run ./server` 启动。

生成的 `Validate`、`Create`、`Update`、`Delete` 方法均接收调用方的 `context.Context` 并传递给 ORM。处理 HTTP 请求时可使用 `request.Object.Context()`；不要用 `context.Background()` 替换请求上下文。`Update` 默认采用部分更新，省略的 nil 字段保留原值；需要清空可为 NULL 的字段时，应自行明确选择字段和写入规则。数据库错误会保留错误链，展示给用户的业务文案由调用方转换。

创建流程的普通测试不需要物理数据库驱动。实际生成项目、编译和 SQLite 行为测试可在具备 Go、CGO/C 编译器及模块缓存或网络的环境下执行：

```sh
go test -tags=integration ./cmd/goi
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
go get github.com/NeverStopDreamingWang/goi/v3@v3.0.1
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@v3.0.1
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

