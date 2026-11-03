# Server lifecycle and graceful shutdown

[简体中文](lifecycle.md) | **English** · [Documentation](README.en.md) · [Requests and responses](http.en.md)

Goi coordinates the HTTP server, startup tasks, shutdown callbacks, and registered database closure within one lifecycle. RunServer blocks; both signal-driven and manual shutdown ultimately use StopServer. This guide describes the current implementation without adding application-level timeouts or panic-recovery policies.

Source: [server.go](../server.go), [startup.go](../startup.go), [shutdown.go](../shutdown.go), [database.go](../database.go), [logger.go](../logger.go).

## 1. Construction and startup

Create an Engine with `goi.NewHTTPServer()`, not a manually initialized `goi.Engine{}`. The constructor initializes lifecycle channels and routing, and references the package-level Settings, Cache, Log, and Validation values that exist at construction time.

`goi.Log` is nil by default. Configure a usable logger before RunServer. When assigning it to goi.Log, do so before NewHTTPServer so the Engine receives that reference. The examples use `server.log` in the current directory rather than relying on an existing log subdirectory.

The main startup steps are:

1. Change this Engine from created to starting, log startup information, and configure HTTP/TLS.
2. Create the network listener successfully; when TLS is enabled, certificate files are loaded before this step.
3. Initialize the cache and internal tasks, close startup registration, then start a goroutine for each registered task.
4. Log the listening address, publish the running state, register `os.Interrupt` / `SIGTERM` handling, and call HTTP Serve / ServeTLS.
5. After serving ends, wait for the entire shutdown before returning from RunServer or panicking with a Serve error.

OnStartup is started with `go`; it is not a readiness barrier. Goi does not wait for tasks to finish their own initialization. A listening log message therefore does not prove that all background components are ready for business operations. Application-specific readiness coordination remains the application's responsibility.

Each Engine can enter its running lifecycle only once. It cannot restart after stopping or startup failure; another RunServer call panics.

### Startup and serving failures

RunServer has no error return value. A startup panic is rethrown after recording the failure, freezing/cancelling startup tasks, and waiting for them to exit. StopServer calls waiting during startup observe the failed terminal state rather than waiting forever on a startup channel.

Startup-failure cleanup is not the full shutdown sequence. A failure before publishing the running state does not execute user OnShutdown callbacks or close all registered databases; do not assume all startup side effects are rolled back. After running state is published, a Serve error other than `http.ErrServerClosed` invokes StopServer, waits for full cleanup, and then panics with that Serve error.

## 2. Startup tasks

```text
type Startup interface {
    StartupName() string
    OnStartup(ctx context.Context, wg *sync.WaitGroup)
}
goi.RegisterOnStartup(task)
```

Register tasks before calling RunServer. The framework internally calls claim before starting tasks; registration afterward is ignored, and RegisterOnStartup does not return an error.

Goi calls `Add(1)` on the shared WaitGroup before starting each task. A task should:

- Use `defer wg.Done()` in its OnStartup, notifying completion exactly once without repeating the Add/Done pair already managed by Goi.
- Observe `ctx.Done()`, finish its loop, release its own resources, and return.
- Wait for any child goroutines before returning, or manage them with its own synchronization. Calling Done alone does not mean those extra goroutines have exited.
- Not depend on a later OnShutdown callback to make itself exit, because shutdown waits for tasks before executing callbacks.

The supplied ctx belongs to the startup manager's lifecycle, not an HTTP request. StopServer cancels it after draining HTTP requests. A task that completes normally does not automatically stop the HTTP server.

## 3. Shutdown callbacks

```text
type ShutdownCallback interface {
    ShutdownName() string
    OnShutdown() error
}
goi.RegisterOnShutdown(callback)
```

Callbacks can be registered while serving until shutdown claims the registration list. Later registrations are ignored. They run in reverse registration order, allowing dependencies created first to be released last. Managed background tasks have already exited when callbacks run, but registered databases are still available for final writes or cleanup.

An error returned by OnShutdown is logged without preventing subsequent callbacks and database closure. Callback panics are not converted to errors: normal Go panic behavior applies, and remaining cleanup is not guaranteed to run. The shutdown owner's defer still publishes the stopped state and wakes waiters, but that does not mean interrupted cleanup steps completed.

Do not close the main framework logger in a shutdown callback: database closure and final shutdown messages still use it. The user controls final Logger.Close timing, normally after RunServer returns.

## 4. Shutdown entry points and order

Normal shutdown starts with `os.Interrupt` / `SIGTERM` or an application call to `server.StopServer()` from an independent goroutine. The signal goroutine calls the same StopServer; there is no separate signal-only cleanup sequence. When RunServer exits, it stops signal delivery and tells a signal goroutine that is still waiting to exit.

The shutdown order is:

```text
http.Server.Shutdown(context.Background())
  → close listeners and idle connections; wait for in-flight HTTP requests
startup.cancel() + WaitGroup.Wait()
  → notify managed background tasks and wait for their exit
OnShutdown(), in reverse registration order
  → registered databases remain available here
Database.Close() for every entry in Settings.Databases
  → log shutdown completion and notify waiters
```

The current implementation sets no shutdown timeout. A request or task that never finishes also keeps StopServer and its waiting RunServer blocked. Standard library Shutdown does not wait for special connections such as hijacked connections; the application must manage those resources. HTTP draining does not imply every custom connection has ended.

### Avoid waiting on yourself

Do not call StopServer synchronously inside an HTTP handler, managed Startup task, or OnShutdown callback. It may wait for the current request, current task, or the same shutdown round itself, creating a self-wait.

To initiate shutdown from these locations, use an independent goroutine for StopServer while the current handler/task completes its own return/Done promptly. The `/stop/` example below only requests shutdown asynchronously; its 202 response does not mean cleanup has finished.

## 5. Concurrent calls, errors, and resource boundaries

| Call timing | StopServer behavior |
| --- | --- |
| Engine has not started | Returns `goi.ErrServerNotRunning`. |
| Startup is in progress | Waits for startup success/failure, then acts on the resulting state. |
| Engine is running | Only one caller obtains shutdown ownership and executes cleanup. |
| Shutdown is in progress | Waits for the same stopDone without repeating cleanup. |
| Stopped or startup failed | Returns the joined errors collected for this lifecycle. |

Ordinary errors are logged when they occur and combined with `errors.Join` after shutdown; no errors means nil. Original HTTP Shutdown and OnShutdown errors are preserved and can be checked with errors.Is / errors.As. Database.Close errors are currently converted into localized errors containing the database name/engine before joining, so their original unwrap chain is not preserved.

Settings.Databases is a map, so database closure order is unspecified. Database.Close returns nil for a connection that was never created. After attempting closure, it clears the cached pointer; a later Database.DB call can create another pool. Stop all users first and avoid reopening database activity after shutdown. See [Custom ORM development](../db/README.md) for database registration and connection ownership.

Each Engine has its own router and lifecycle state, but construction references shared package-level configuration, cache, logger, and validation manager. Startup/Shutdown registries are also package-level objects rather than isolated instances per Engine. Multiple Engines are not prohibited, but their tasks, cancellation signals, and database lifecycles must not be assumed independent; applications must coordinate shared resources.

RunTime depends on a recorded startup time. Use it after the server reaches running state, not immediately after constructing an Engine. It calculates the difference at the moment of the call rather than freezing duration when the server stops.

## 6. Complete example

This example includes a cancellation-aware background task, a shutdown callback, and an asynchronous manual shutdown endpoint. Running it listens on the default `127.0.0.1:8080` and creates `server.log`. Ctrl+C also stops it.

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
