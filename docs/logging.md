# 日志、切割与关闭

**简体中文** | [English](logging.en.md) · [文档目录](README.md)

`Logger` 提供级别日志、格式化输出、文件切割和显式关闭。`NewLogger` 不会自动设置 `goi.Log`，也不会自动注册切割任务或停机关闭回调；这些步骤由应用配置和生命周期决定。

## 配置默认日志

先创建日志并设置 `goi.Log`，再调用 `NewHTTPServer()`，这样 `Server.Log` 与包级默认日志指向同一个对象。完整服务示例：

```go
package main

import (
	"fmt"
	"os"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		panic(err)
	}
	logger := goi.NewLogger("logs/server.log")
	logger.Name = "server"
	logger.SplitSize = 10 * 1024 * 1024
	logger.SplitTime = "2006-01-02"
	goi.Log = logger
	server := goi.NewHTTPServer()
	server.Settings.Debug = false
	goi.RegisterOnStartup(logger) // 可选：启动后台切割检查
	defer func() {
		if err := logger.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	server.Router.Path("ping", "Ping", goi.ViewSet{
		GET: func(request *goi.Request) any { return "pong" },
	})
	logger.Info("starting")
	server.RunServer()
}
```

此程序运行后会创建文件并启动服务；仅检查编译时不会创建日志或监听端口。目录权限请按部署要求设置。当前 `NewLogger` 自行创建缺失目录时使用 `0644`，在 Unix 上可能缺少目录遍历权限，示例因此先显式创建目录；不要把其自动建目录行为当成跨平台权限配置方案。

如果先创建了 Engine，再设置 `goi.Log`，应同时更新 `Server.Log`；这两个字段保存的是指针值，后续赋值不会自动同步。不要在请求开始后并发替换默认日志或修改配置字段。

## 创建与输出

`NewLogger(path)` 立即打开文件，默认追加写入，文件权限参数为 `0644`。空路径返回 nil；建目录或打开文件失败会 panic，不是返回 error。框架默认 `goi.Log` 为 nil，启用服务前必须配置，否则框架日志和访问日志会访问无效日志对象。

| API | 用法 |
| --- | --- |
| `Debug` / `DebugF` | 调试日志；`Settings.Debug == false` 时，正常 `Log` 路径会跳过 Debug。 |
| `Info` / `InfoF` | 信息日志。 |
| `Warning` / `WarningF` | 警告日志。 |
| `Error` / `ErrorF` | 错误日志。 |
| `Log(level, values...)` / `LogF(level, format, values...)` | 指定 `goi.Level`；常量是 `goi.Debug`、`goi.INFO`、`goi.WARNING`、`goi.ERROR`。 |
| `Print(level, values...)` | 直接格式化并写入当前 Logger；不执行 Debug 过滤或控制台分流。 |

默认格式类似 `[2026-01-01 12:00:00] INFO message`，时间来自 `goi.GetTime()`。没有按级别高低配置的过滤阈值；除 Debug 的特殊处理外，等级主要用于输出标签。

`Settings.Debug == true` 时，`Log` 会额外写到控制台，再写当前日志文件；关闭 Debug 后只写文件并跳过 Debug 消息。直接调用 `Print(goi.Debug, ...)` 仍会写出。日志内容不自动脱敏，应避免记录密码、令牌和敏感请求体。

`Print`、`CheckSplit` 和 `Close` 在单个对象内互斥，但 `Log` 还会读写未单独同步的包级 `consoleLogger`。不能把前者的锁推导为所有并发 `Log` 调用都无数据竞争；多协程及多 Logger 场景应验证实际调用路径，配置与回调字段应在启动前固定。

## 访问日志

`Engine.ServeHTTP` 在请求结束路径中自动通过 `engine.Log.Info` 写访问记录，不需要单独注册访问日志中间件。记录包含远端地址、方法、URL 路径、响应字节数、协议、状态和响应头数量；不是完整请求/响应体转储。

当前 `ServeHTTP` 在注册 defer 时就计算传入的 `elapsed`，所以访问记录中的毫秒数不是完整请求处理耗时，不能据此作为性能指标或延迟监控。未处理异常的堆栈也会写入默认日志；访问记录是否送往控制台由 Debug 控制。

## 文件切割

`NewLogger` 的实际默认值是 `SplitSize = 10 * 1024 * 1024`（10 MiB）和 `SplitTime = "2006-01-02"`（按日期变化检查），以构造函数为准。

| 配置 / 方法 | 当前行为 |
| --- | --- |
| `SplitSize` | 文件达到该字节数时触发；0 禁用大小条件。 |
| `SplitTime` | 比较创建时间与当前时间经该 Go layout 格式化后的字符串；空字符串禁用时间条件。 |
| `CheckSplit()` | 加锁检查条件并切换文件；可由应用自行定期调用。关闭后返回 nil。 |
| `RegisterOnStartup(logger)` | 注册 Logger 的后台任务；应在 `RunServer` 前调用，任务每 1 秒执行 `CheckSplit`，停止时响应 ctx 取消。 |
| `SplitInterval` | 当前字段未被 `OnStartup` 使用，设置它不会改变固定 1 秒检查间隔。 |

切割不是每次写日志时触发，所以文件可能暂时超过大小阈值。默认切割先关闭旧文件，再重命名为 `server_日期.log`；同名已存在时追加序号，然后重新打开原路径。没有自动压缩、保留天数、归档清理或跨进程文件协调。

通常调用带锁的 `CheckSplit`，不要同时从多个位置直接调用底层 `SplitLogger`、`GetFile` 或改 `File`/`Logger`。`CheckSplit` 的部分错误返回 error，但文件 Stat 失败会 panic；后台任务遇到切割错误也会 panic，不是自动重试的容错服务。默认切割在失败前可能已经关闭旧文件，调用方需处理故障，不能承诺失败后仍可继续写入。写入方法没有 error 返回值，也不提供磁盘持久化或自动重试保证。

## 自定义格式与输出目标

下面只写标准输出，不创建文件：

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func main() {
	logger := &goi.Logger{
		Name:   "console",
		Logger: log.New(os.Stdout, "", 0),
		PrintFunc: func(logger *goi.Logger, level goi.Level, values ...any) {
			logger.Logger.Printf("%s | %s", level, fmt.Sprint(values...))
		},
	}
	logger.Print(goi.INFO, "hello")
	if err := logger.Close(); err != nil {
		panic(err)
	}
}
```

`PrintFunc` 在 Logger 的锁内执行，不能递归调用同一对象的 `Print`、`Log`、`Close` 或 `CheckSplit`，否则会死锁。应使用底层 `logger.Logger` 或自己的输出器。自定义格式需自行添加时间等字段。

`GetFileFunc(filePath)` 用于自定义文件打开；`SplitLoggerFunc(logger)` 用于自定义切割后的文件创建。`NewLogger` 首次打开文件时这些回调还是 nil，因此在构造后赋值只影响后续调用。切割回调执行前旧文件已经关闭；需要返回可写的新文件，由框架更新 `File` 和底层输出目标。

## Close 与停机顺序

`Close()` 与 `Print`、`CheckSplit` 使用同一把锁。可以重复调用，后续返回首次关闭结果；只关闭 `File`，当 `File == nil` 时不会关闭底层其他输出目标，例如上例的 `os.Stdout`。但 Logger 自身仍被标记为关闭，关闭后继续 `Print` 会 panic。

框架不会自动关闭所有 Logger。`Close` 也不会取消已经注册的切割任务；任务仍等到生命周期 ctx 取消才退出。通常在 `RunServer` 返回后关闭默认日志，此时请求、后台任务、停机回调和数据库关闭阶段都已完成。不要在 `OnShutdown` 中过早关闭默认日志，后续数据库关闭及停机信息仍会写日志。

多个自定义 Logger 也由应用保存并逐一关闭；关闭前确保其他协程不会继续写入。不要在关闭之后再使用该 Logger 报告关闭错误，可改用 `fmt.Fprintln(os.Stderr, err)`。

## 源码与关联指南

- [logger.go](../logger.go)：日志输出、切割、回调与 Close。
- [server.go](../server.go)、[recovery.go](../recovery.go)：默认日志引用、停机和访问记录。
- [startup.go](../startup.go)：切割任务接口与注册。
- [启动任务与优雅停机](lifecycle.md)、[配置与时间](configuration.md)。
