# goi

基于 `net/http` 开发的一个 Web 框架，语法与 `Django` 类似，简单易用，快速上手，详情请查看示例

[详细示例：example](https://github.com/NeverStopDreamingWang/goi_example)

## goi 创建命令

使用 `go env GOMODCACHE` 获取 go 软件包路径：`mypath\Go\pkg\mod` + `github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe`

使用 `go env GOROOT` 获取 go 安装路径 `mypath\Go` + `bin`
将可执行文件复制到 Go\bin 目录下

**Windows**: `copy mypath\Go\pkg\mod\github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe mypath\Go\bin\goi.exe`

编译

```cmd
go build -p=1 -o goi.exe
```

**Linux**: `cp mypath\go\pkg\mod\github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi mypath\go\bin\goi`

自定义编译

```cmd
go build -o goi
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
import "github.com/NeverStopDreamingWang/goi"
```

或者，使用 `go get`：

```bash
go get -u github.com/NeverStopDreamingWang/goi
```

### 运行 Goi

一个基础示例：

```go
package main

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
)

func Ping(request *goi.Request) interface{} {
	goi.Log.DebugF("Test1")

	return goi.Data{
		Code:    http.StatusOK,
		Message: "Hello World",
		Results: nil,
	}
}

func main() {
	// 创建 Goi 服务器
	Server := goi.NewHttpServer()
	// 网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 监听地址 0.0.0.0
	Server.Settings.BIND_ADDRESS = "0.0.0.0" // 默认 127.0.0.1
	// 监听端口 8080
	Server.Settings.PORT = 8080

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

