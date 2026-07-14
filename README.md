# goi

基于 `net/http` 开发的一个 Web 框架，语法与 `Django` 类似，简单易用，快速上手，详情请查看示例

[详细示例：goi_example](https://github.com/NeverStopDreamingWang/goi_example)

## goi 创建命令

### 构建 goi 命令行工具

构建入口位于 `cmd/` 目录。Windows 和 Linux 分别提供对应环境的构建脚本；两个脚本都会交叉编译 Windows 与 Linux 两个系统的可执行文件。目标架构均为 `amd64`，并设置 `CGO_ENABLED=0`，因此不依赖目标系统的 C 工具链。构建命令使用 `-trimpath` 和 `-ldflags="-s -w"`，用于移除本地路径、符号表及调试信息，减小发布产物体积。

Windows 环境执行：

```bat
cmd\build.bat
```

Linux 环境执行：

```sh
sh ./cmd/build.sh
```

无论使用哪个脚本，都会生成以下两个构建产物：

- `windows/amd64`：`cmd/goi.exe`
- `linux/amd64`：`cmd/goi`

两个脚本都会自动切换到自身所在目录，因此也可以从其他工作目录通过脚本路径调用。

### 安装 goi 命令行工具

使用 `go env GOMODCACHE` 获取 go 软件包路径：`mypath\Go\pkg\mod` + `github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe`

使用 `go env GOROOT` 获取 Go 安装目录，然后将对应系统的构建产物复制到其 `bin` 目录。

Windows PowerShell：

```powershell
$goroot = go env GOROOT
Copy-Item .\cmd\goi.exe "$goroot\bin\goi.exe"
```

Linux：

```sh
cp ./cmd/goi "$(go env GOROOT)/bin/goi"
```

### goi 命令使用

```shel
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
import "github.com/NeverStopDreamingWang/goi/v2"
```

或者，使用 `go get`：

```bash
go get -u github.com/NeverStopDreamingWang/goi/v2
```

### 运行 Goi

一个基础示例：

```go
package main

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v2"
	"github.com/NeverStopDreamingWang/goi/v2/response"
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

