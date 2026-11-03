<div align="center">

# Goi CLI

### Goi 项目与应用的脚手架工具

`goi` 用于创建项目配置包和应用脚手架。本指南说明安装、按需构建、创建、注册应用及运行流程。

[![Version](https://img.shields.io/github/v/release/NeverStopDreamingWang/goi?label=version&color=blue)](https://github.com/NeverStopDreamingWang/goi/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/NeverStopDreamingWang/goi?label=Go&logo=go&logoColor=white)](../../go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/NeverStopDreamingWang/goi/v3.svg)](https://pkg.go.dev/github.com/NeverStopDreamingWang/goi/v3)
[![Stars](https://img.shields.io/github/stars/NeverStopDreamingWang/goi?label=stars&color=blue)](https://github.com/NeverStopDreamingWang/goi/stargazers)
[![License](https://img.shields.io/github/license/NeverStopDreamingWang/goi?label=license)](../../LICENSE)

**简体中文** | [English](README.en.md)

[安装 CLI](#安装-cli) · [命令概览](#命令概览) · [创建项目和应用](#创建项目和应用) · [帮助与常见错误](#帮助和常见错误) · [项目主页](../../README.md)

</div>

## 安装 CLI

需要 Go 1.24 或更高版本。推荐优先使用 `go install` 安装最新版本：

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

需要固定版本时，例如安装 v3.0.2：

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@v3.0.2
```

命令安装为 `goi`（Windows 下为 `goi.exe`）。安装目录优先使用 `GOBIN`；未设置时使用 `GOPATH/bin`。可通过以下命令查看配置，并将对应目录加入 `PATH`：

```sh
go env GOBIN GOPATH
```

打开能读取新 `PATH` 的终端，验证安装：

```sh
goi help
```

帮助会显示 Goi 版本和可用命令。若提示找不到 `goi`，先检查安装目录是否已加入 `PATH`。`go install` 从版本源码在本机构建命令，不需要另行下载预编译的可执行文件。

## 按需本地构建

只有需要修改 CLI 源码或自定义构建参数时，再使用本地构建。源码入口为 `cmd/goi/`，构建脚本位于 `cmd/`。在 Goi 框架仓库根目录运行对应脚本。

Windows 环境：

```bat
cmd\build.bat
```

Linux 环境：

```sh
sh ./cmd/build.sh
```

两个脚本都会交叉编译 Windows 与 Linux 两个系统的 CLI，目标架构均为 `amd64`，生成：

- `windows/amd64`：`cmd/dist/windows-amd64/goi.exe`
- `linux/amd64`：`cmd/dist/linux-amd64/goi`

脚本设置 `CGO_ENABLED=0`，不依赖目标系统的 C 工具链；使用 `-trimpath` 移除本地路径，并通过 `-ldflags="-s -w"` 移除符号表和调试信息以减小体积。脚本会自动切换到自身目录，也可从其它工作目录通过脚本路径调用。

`cmd/dist/` 只保存本地构建产物，不纳入 Git 版本控制；不要将生成的二进制提交到源码仓库。这些脚本构建的是 Goi CLI，不是后续生成的 SQLite 应用。

## 命令概览

不带子命令运行 `goi`，或运行 `goi help`，均可查看版本和命令概览。命令语法如下：

```text
goi help [command]
goi create-project [path]<project-name> [--module <module-path>]
goi create-app [path]<app-name>
```

`[path]` 表示可选的相对父路径前缀，可以包含多级目录；由用户按系统路径格式填写并连接名称，所需分隔符包含在该前缀中。`<project-name>` 和 `<app-name>` 表示必填名称。方括号、尖括号均不实际输入。

`help` 的 `command` 可省略，也可指定 `create-project` 或 `create-app` 查看详细参数和示例。`--module <module-path>` 仅用于创建项目，显式指定写入 `go.mod` 的 module；省略时使用当前项目根目录名，不改变配置包或应用的名称。

## 创建项目和应用

先创建并进入外层项目目录，再连续执行两条创建命令：

```sh
mkdir project
cd project
goi create-project path/myproject
goi create-app path/myapp
go mod tidy
```

不需要父路径时，在新建并进入项目根目录后，改用下面两条命令。它们替换上面的两条 `goi` 命令，两种方案二选一，不要依次重复执行：

```sh
goi create-project myproject
goi create-app myapp
```

这会直接在当前根目录中创建 `myproject/` 配置包和 `myapp/` 应用。下文的目录树与导入示例采用带 `path/` 的方案；省略父路径时，导入路径也要同步去掉 `/path`。

进入 `project` 后，它就是当前项目根目录，两条 `goi` 命令之间不再 `cd`。默认 module 取执行 `create-project` 时当前根目录的末级目录名，此例为 `project`。参数末级 `myproject`、`myapp` 分别是项目配置包名和应用包名，不决定 module。`create-project` 在当前目录生成 `go.mod` 和 `server/main.go`，参数只指定配置包目录。

需要自定义模块路径时，将 `create-project` 命令替换为：

```sh
goi create-project path/myproject --module example.com/team/myproject
```

生成的 `server/main.go` 是简单的服务入口，不包含额外的 Cobra/web 子命令或业务代码。配置包同时生成 `urls.go`，初始仅包含 `package` 声明。创建命令拒绝覆盖已有生成目标，不覆盖代码、配置或密钥。

## 路径和应用规则

`create-app` 必须在含 `go.mod` 的项目根目录执行；不会向上查找项目，也不能从项目子目录执行。应用的导入路径使用当前 `go.mod` 中的真实 module，项目目录改名不影响它。

`path/myapp` 会自动创建父目录，在该完整路径中生成应用文件；实际 Go 包名使用末级名称 `myapp`，不是 `path_myapp`。两种创建命令都支持相对路径；Windows 下也可使用反斜杠：

```powershell
mkdir project
cd project
goi create-project path\myproject
goi create-app path\myapp
```

正反斜杠表示同一目标，不应重复执行。已有父目录可以复用，目标配置包或应用目录必须不存在；创建命令不提供自动合并。已有 `go.mod`、`server/main.go` 等生成目标也不会被覆盖。

路径不得是绝对路径，不得包含 `..`、`.` 或空路径段。例如 `../myapp`、`./myapp`、`path/../myapp`、`path//myapp` 都会被拒绝。配置包和应用的末级名称必须是合法的 ASCII Go 包名，不能使用 `main`、`_` 或 Go 关键字。配置包路径不能直接使用 `server`，因为该目录用于服务入口；`server/config` 等子路径不受此限制。

完成上述流程后，目录结构如下：

```text
project/                      # 当前运行目录，即项目根
├── go.mod                   # module project
├── server/
│   └── main.go               # 简单服务入口
└── path/                    # 可替换的相对父路径
    ├── myproject/           # 项目配置包
    │   ├── settings.go
    │   ├── urls.go          # 初始仅包含 package 声明
    │   ├── converter.go
    │   ├── validator.go
    │   ├── logger.go
    │   └── ssl.go
    └── myapp/
        ├── models.go
        ├── serializer.go
        ├── urls.go
        ├── views.go
        └── utils.go
```

创建应用时，在当前 Go 模块内递归查找通过 `settings.go` 声明 `Server` 的项目配置包；跳过隐藏目录、`vendor`、`node_modules` 和含独立 `go.mod` 的嵌套模块，不限于根目录或直接子目录。找不到配置包或存在多个候选时会报错，不根据目录名猜测。

## 导入应用和显式迁移

创建应用后，需要手动修改 `server/main.go`，导入应用并按需迁移。以下示例对应默认 module `project`，配置包名为 `myproject`，应用路径为 `path/myapp`：

```go
package main

import (
	"context"

	"project/path/myapp"
	"project/path/myproject"
)

func main() {
	ctx := context.Background()
	myapp.Migrate(ctx)
	myproject.Server.RunServer()
}
```

脚手架不会自动修改 `server/main.go` 注册新应用。只需要注册路由、不执行迁移时，也可使用空白导入，例如 `_ "project/path/myapp"`。

`--module example.com/team/myproject` 会显式覆盖默认 module；此时将上面的导入前缀 `project` 改为该完整路径，例如 `"example.com/team/myproject/path/myapp"`。生成的 `urls.go` 使用真实导入路径和识别到的配置包名，例如 `myproject.Server`，不固定使用 `project` 别名。应用被导入后注册路由，但不会在 `init` 中自动建表。需要建表时，在数据库配置完成、服务启动之前显式调用 `Migrate(ctx)`；迁移只创建不存在的表，不自动修改已有结构。

默认路由和表名沿用末级应用名：`path/myapp` 使用 `myapp/` 路由和 `myapp_my_model` 表。若不同分组都以 `myapp` 结尾，应自行修改 `urls.go` 的路由前缀、`ModelSet()` 中的表名，并使用不同导入别名，避免冲突；分组目录不会自动成为路由或表名前缀。

## 上下文和数据库

生成的 `Validate`、`Create`、`Update`、`Delete` 方法接收 `context.Context`，并将它传给 ORM。HTTP 处理程序应传入 `request.Object.Context()`，不要用 `context.Background()` 替换请求上下文；上面 main 中的后台上下文仅用于启动阶段迁移。

`Update` 是部分更新：nil 字段默认跳过，普通零值仍会写入。需要显式写入 NULL 时，应根据模型约束自行选择字段和写入规则。数据库错误保留原错误链，面向用户的业务消息由调用方处理。完整 ORM 行为见[数据库开发指南](../../db/README.md)。

项目默认配置 SQLite，连接名为 `default`，并导入 `github.com/mattn/go-sqlite3` 驱动。数据目录在首次连接时创建；运行生成的应用需要 `CGO_ENABLED=1` 和可用的 C 编译器。根目录构建脚本中的 `CGO_ENABLED=0` 只用于 Goi CLI，不适用于该 SQLite 应用配置。

完成导入和需要的迁移调用后，在生成项目的根目录运行：

```sh
go mod tidy
go run ./server
```

## 帮助和常见错误

```sh
goi help create-project
goi help create-app
```

`create-app` 检查当前目录的 `go.mod`。缺少该文件时会返回项目根目录错误，不会向上搜索，也不会写入应用文件；请切换到含 `go.mod` 的项目根后重试。对于尚未创建的项目，先在选定根目录执行 `create-project`，再直接执行 `create-app`。

路径非法、目标已存在或无法唯一识别配置包时同样会报错。应先修正路径或项目结构，不要通过重复执行覆盖现有文件。

## 验证 CLI

以下命令在 Goi 框架仓库根目录执行，不是在生成项目中执行：

```sh
go test ./cmd/goi
go test -race ./cmd/goi
go test -tags=integration ./cmd/goi
```

普通测试覆盖名称、路径、拒绝覆盖和模板生成。集成测试实际生成项目并编译、验证 SQLite 行为，需要 Go、CGO/C 编译器以及已缓存的依赖或可用网络；race 测试也需要对应平台支持的工具链。
