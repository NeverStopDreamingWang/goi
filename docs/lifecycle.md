# 服务生命周期与优雅停机

**简体中文** | [English](lifecycle.en.md) · [文档目录](README.md) · [请求与响应](http.md)

Goi 将 HTTP 服务、启动任务、停机回调和已注册数据库的关闭组织在同一轮生命周期中。`RunServer()` 阻塞运行；信号和手动调用最终都通过 `StopServer()` 停机。本文描述当前实现，不添加应用级超时或异常捕获策略。

源码：[server.go](../server.go)、[startup.go](../startup.go)、[shutdown.go](../shutdown.go)、[database.go](../database.go)、[logger.go](../logger.go)。

## 1. 创建与启动

使用 `goi.NewHTTPServer()` 创建 Engine，不要手工使用 `goi.Engine{}` 代替。构造函数初始化生命周期通道、路由，并引用当时的包级 Settings、Cache、Log 和 Validation。

`goi.Log` 默认是 nil；调用 `RunServer` 前必须配置有效日志器。若把日志器赋给 `goi.Log`，应在 `NewHTTPServer()` 之前完成，使 Engine 获得对应引用。示例使用当前目录的 `server.log`，避免依赖预先存在的日志子目录。

启动的主要步骤是：

1. 将本 Engine 状态从“已创建”切换为“启动中”，输出启动信息并配置 HTTP/TLS 服务。
2. 成功创建网络监听器；TLS 启用时，证书文件会在此前加载。
3. 初始化缓存及内部任务，关闭启动任务注册入口，然后逐个启动已注册任务的协程。
4. 输出监听地址，发布运行状态，注册 `os.Interrupt` / `SIGTERM` 信号处理，再调用 HTTP `Serve` / `ServeTLS`。
5. 服务监听结束后，等待整轮停机完成，才从 `RunServer()` 返回或继续抛出 Serve 错误。

任务的 `OnStartup` 是通过 `go` 启动的，不是就绪屏障：框架不等待任务完成内部初始化。因此不能仅凭监听日志推断所有后台组件已经准备好接收业务操作。需要任务就绪协调时由应用自己安排。

同一个 Engine 只能进入一次运行生命周期，停止或启动失败后都不能再调用 `RunServer()` 重启；重复调用会 panic。

### 启动和监听失败

`RunServer()` 没有 error 返回值。启动阶段 panic 会在记录失败、冻结/取消启动任务并等待任务退出后，继续抛出原 panic；启动期间等待的 `StopServer()` 调用会收到失败终态，而不是一直等待启动通道。

注意“启动失败清理”不等同于完整停机：发布运行状态之前失败的分支不会执行用户 `OnShutdown` 或统一关闭数据库，不应假设所有启动副作用都已回滚。运行状态发布后，如果 `Serve` 返回非 `http.ErrServerClosed` 错误，则进入 `StopServer()`，等待完整清理后再 panic 该 Serve 错误。

## 2. 启动任务

```text
type Startup interface {
    StartupName() string
    OnStartup(ctx context.Context, wg *sync.WaitGroup)
}
goi.RegisterOnStartup(task)
```

应在调用 `RunServer()` 前完成注册。框架在启动任务前执行内部 `claim`，此后注册会被忽略，注册函数没有错误返回值。

每个任务启动前，框架会对共享 WaitGroup 调用一次 `Add(1)`；任务应：

- 在自己的 `OnStartup` 中 `defer wg.Done()`，退出时恰好通知一次，不重复 Add/Done 框架已经管理的这一次计数。
- 响应 `ctx.Done()`，自行结束循环、释放任务私有资源并返回。
- 在返回前等到自己启动的子协程完成，或用自己的同步机制管理它们；只调用 Done 不代表额外协程已退出。
- 不依赖随后才执行的 `OnShutdown` 来使自己结束，否则停机会卡在等待任务的阶段。

传入的 ctx 是启动管理器的生命周期上下文，不是某个 HTTP 请求的上下文。`StopServer()` 会在 HTTP 请求排空后取消它；任务正常结束并不会自动停止 HTTP 服务。

## 3. 停机回调

```text
type ShutdownCallback interface {
    ShutdownName() string
    OnShutdown() error
}
goi.RegisterOnShutdown(callback)
```

回调可在服务运行期间注册，直到停机执行者锁定注册入口；此后注册会被忽略。回调按照注册顺序逆序执行，适合先创建的依赖后释放。执行回调时，受管理的后台任务已经退出，但数据库尚未关闭，可以在这里做最终的数据写入或其他清理。

`OnShutdown()` 返回 error 时会写日志并继续执行后续回调与数据库关闭。框架不会把回调 panic 转换为 error；panic 保持 Go 的正常语义，后续清理不保证继续执行。停机执行者的 defer 仍会发布终态并唤醒等待者，但这不代表被中断的清理步骤已经完成。

不要在停机回调里关闭框架仍需使用的主日志器：数据库关闭及停机完成信息还会继续写日志。日志文件的最终 `Close()` 由用户管理，通常放在 `RunServer()` 返回之后。

## 4. 两种停机入口与顺序

正常入口是收到 `os.Interrupt` / `SIGTERM`，或应用从独立协程调用 `server.StopServer()`。信号处理协程会调用同一个 StopServer，不存在另一套独立的清理顺序；`RunServer()` 离开时会停止信号投递并通知尚在等待信号的协程结束。

停机顺序固定为：

```text
http.Server.Shutdown(context.Background())
  → 停止监听、关闭空闲连接，并等待在途 HTTP 请求完成
startup.cancel() + WaitGroup.Wait()
  → 通知并等待受管理的后台任务退出
OnShutdown()，按注册顺序逆序执行
  → 此时仍可使用已注册数据库
Settings.Databases 中每个 Database.Close()
  → 最后输出停机信息并通知等待者
```

当前实现没有设置停机超时。如果在途请求或任务一直不结束，`StopServer()` 和等待它的 `RunServer()` 也会一直等待。标准库 Shutdown 不等待被劫持的连接等特殊连接，相关资源应由应用自行管理；不要把“HTTP 排空”理解为所有自定义连接都已结束。

### 避免同步等待自己

不要在 HTTP 请求处理函数、受管理的 Startup 任务或 OnShutdown 回调内部同步调用 StopServer：它可能正在等待当前请求、当前任务，或当前轮停机本身结束，从而造成自等待。

需要从这些位置发起停机时，让独立协程执行 StopServer，而当前处理流程应尽快完成自己的返回/Done。下方 `/stop/` 示例只异步发出停机请求；不能把返回 202 理解成清理已经完成。

## 5. 并发调用、错误和资源边界

| 调用时机 | StopServer 行为 |
| --- | --- |
| Engine 尚未运行 | 返回 `goi.ErrServerNotRunning`。 |
| 正在启动 | 等待启动成功或失败后，再按最终状态处理。 |
| 正在运行 | 只有成功取得停机执行权的一个调用者执行清理。 |
| 已在停机 | 等待同一个 `stopDone`，不重复清理。 |
| 已停止或启动失败 | 返回本轮已收集错误的合并结果。 |

正常 error 会在发生时记录到日志，停机完成后通过 `errors.Join` 汇总；没有错误时返回 nil。HTTP Shutdown 和 OnShutdown 的原始 error 被保留，可用 `errors.Is` / `errors.As` 检查。数据库 Close 的错误目前会先转成含数据库名称/引擎的国际化错误，再加入结果，不保留原始数据库错误的 unwrap 链。

Settings.Databases 是 map，数据库之间的关闭顺序没有保证。Database.Close 对未建立连接的项目返回 nil；尝试关闭后会清空缓存指针，后续再调用 Database.DB 仍可能重新创建连接池。因此业务应先停止所有使用者，不要在停机后继续开启数据库操作。数据库注册与 ORM 连接说明见 [自定义 ORM 文档](../db/README.md)。

多个 Engine 的路由和生命周期状态是各自的，但构造时引用的是包级共享配置、缓存、日志和验证管理器；Startup/Shutdown 注册表也是包级对象，不是每个 Engine 的隔离实例。框架没有禁止创建多个 Engine，但不能据此假设它们的任务、取消信号和数据库生命周期互不影响，需要由应用协调共享资源。

`RunTime()` 依赖已记录的启动时刻，应在服务进入运行状态后使用，不能在刚创建 Engine 时直接调用；它按调用时刻计算时间差，不是停止后冻结的运行时长。

## 6. 完整示例

下面展示一个响应取消的后台任务、一个停机回调，以及异步手动停机入口。运行会监听默认的 `127.0.0.1:8080` 并创建 `server.log`；也可用 Ctrl+C 停止。

```go
package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

type worker struct{}

func (worker) StartupName() string { return "example worker" }
func (worker) OnStartup(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			goi.Log.Debug("worker tick")
		}
	}
}

type cleanup struct{}

func (cleanup) ShutdownName() string { return "example cleanup" }
func (cleanup) OnShutdown() error {
	log.Print("background tasks have exited; databases are still available")
	return nil
}

func main() {
	logger := goi.NewLogger("server.log")
	goi.Log = logger
	defer func() {
		if err := logger.Close(); err != nil {
			log.Print(err)
		}
	}()
	server := goi.NewHTTPServer()
	goi.RegisterOnStartup(worker{})
	goi.RegisterOnShutdown(cleanup{})
	server.Router.Path("stop/", "request shutdown", goi.ViewSet{
		POST: func(request *goi.Request) any {
			go func() {
				if err := server.StopServer(); err != nil {
					log.Printf("shutdown result: %v", err)
				}
			}()
			return goi.Response{Status: http.StatusAccepted, Data: "shutdown requested"}
		},
	})
	server.RunServer()
}
```
