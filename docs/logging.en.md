# Logging, Rotation, and Closing

[简体中文](logging.md) | **English** · [Documentation](README.en.md)

`Logger` provides level-based messages, formatting, file rotation, and explicit closing. `NewLogger` does not assign `goi.Log`, register rotation, or register a shutdown close callback. The application owns those configuration and lifecycle decisions.

## Configure the Default Logger

Create the logger and assign `goi.Log` before `NewHTTPServer()` so that `Server.Log` references the same object. Complete server example:

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
	goi.RegisterOnStartup(logger) // Optional background rotation checks.
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

Running this program creates files and starts a server; compile-only checks do neither. Choose directory permissions for your deployment. The current `NewLogger` uses `0644` when creating missing directories, which may omit directory traversal permission on Unix. The example therefore creates the directory explicitly; automatic directory creation is not a portable permission-configuration mechanism.

If the Engine already exists when `goi.Log` is assigned, update `Server.Log` too. Both fields contain pointer values and later assignments do not synchronize them automatically. Do not concurrently replace the default logger or modify configuration after serving begins.

## Construction and Output

`NewLogger(path)` opens the file immediately, appending by default with a `0644` file mode. An empty path returns nil. Directory or open failures panic rather than returning an error. The framework initializes `goi.Log` to nil; configure it before serving, since framework and access logging require a valid logger.

| API | Use |
| --- | --- |
| `Debug` / `DebugF` | Debug messages; the normal `Log` path skips them when `Settings.Debug == false`. |
| `Info` / `InfoF` | Informational messages. |
| `Warning` / `WarningF` | Warnings. |
| `Error` / `ErrorF` | Errors. |
| `Log(level, values...)` / `LogF(level, format, values...)` | Explicit `goi.Level`; constants are `goi.Debug`, `goi.INFO`, `goi.WARNING`, and `goi.ERROR`. |
| `Print(level, values...)` | Formats and writes directly to this Logger, without Debug filtering or console routing. |

The default format resembles `[2026-01-01 12:00:00] INFO message`; time comes from `goi.GetTime()`. There is no configured severity threshold. Apart from the special Debug handling, levels primarily label output.

With `Settings.Debug == true`, `Log` writes to the console and then the current file. With Debug disabled it writes only to the file and skips Debug messages. Direct `Print(goi.Debug, ...)` still writes. Messages are not redacted automatically; avoid recording passwords, tokens, and sensitive request bodies.

`Print`, `CheckSplit`, and `Close` are mutually exclusive on one object, but `Log` also reads and writes the package-level `consoleLogger` without a separate synchronization boundary. The object lock is not a guarantee that all concurrent `Log` calls are race-free. Verify actual multi-goroutine and multi-Logger paths, and finalize configuration and callbacks before starting.

## Access Records

`Engine.ServeHTTP` writes access records through `engine.Log.Info` on the request completion path; no separate access-log middleware is required. Records contain remote address, method, URL path, response bytes, protocol, status, and header count, not complete request or response bodies.

Currently `ServeHTTP` calculates the `elapsed` argument while registering its defer, so the millisecond value is not total request-processing duration. Do not use it as a performance or latency metric. Unhandled exception stacks also use the default logger. Debug controls whether records are copied to the console.

## File Rotation

The constructor's actual defaults are `SplitSize = 10 * 1024 * 1024` (10 MiB) and `SplitTime = "2006-01-02"` (check for a date change).

| Setting / method | Current behavior |
| --- | --- |
| `SplitSize` | Triggers when the file reaches this byte count; zero disables the size condition. |
| `SplitTime` | Compares creation and current times after formatting with this Go layout. An empty string disables the time condition. |
| `CheckSplit()` | Locks, checks conditions, and switches files. Applications may call it periodically. It returns nil after closing. |
| `RegisterOnStartup(logger)` | Registers the Logger's background task before `RunServer`. It calls `CheckSplit` every second and exits on lifecycle cancellation. |
| `SplitInterval` | Currently unused by `OnStartup`; assigning it does not change the fixed one-second interval. |

Rotation is not checked on every write, so files may temporarily exceed the threshold. The default closes the old file, renames it to `server_date.log` with a numeric suffix if needed, and reopens the original path. There is no automatic compression, retention period, archive cleanup, or cross-process coordination.

Normally call the locked `CheckSplit`, rather than concurrently invoking lower-level `SplitLogger`, `GetFile`, or changing `File`/`Logger`. Some rotation errors are returned, but a failed file Stat panics. The background task also panics on rotation errors; it is not a fault-tolerant retry service. Default rotation may already have closed the old file before failure, so continued writing is not guaranteed. Write methods return no error and provide no durability or automatic retry guarantee.

## Custom Format and Output

This program writes only to standard output and creates no file:

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

`PrintFunc` runs under the Logger lock. Do not recursively call the same object's `Print`, `Log`, `Close`, or `CheckSplit`, which would deadlock. Use the underlying `logger.Logger` or your own writer instead. Custom formats must add their own timestamps and fields.

`GetFileFunc(filePath)` customizes file opening; `SplitLoggerFunc(logger)` supplies the file after custom rotation. These callbacks are nil during `NewLogger`'s initial open, so assigning them after construction affects later calls only. The old file has already been closed before the rotation callback runs; return a writable replacement for the framework to update `File` and the underlying output.

## Close and Shutdown Order

`Close()` shares a lock with `Print` and `CheckSplit`. It is repeatable and subsequent calls return the first close result. It closes only `File`; if `File == nil`, it does not close other outputs such as `os.Stdout` in the example. The Logger is still marked closed, and a later `Print` panics.

The framework does not close all Loggers automatically. `Close` does not cancel an already registered rotation task; that task exits when its lifecycle context is canceled. Normally close the default logger after `RunServer` returns, once requests, background tasks, shutdown callbacks, and database cleanup have finished. Do not close it prematurely in `OnShutdown`, because later database and shutdown messages still need it.

Applications must retain and close additional custom Loggers too. Ensure no other goroutine continues writing before closing. Do not use a closed Logger to report its close error; use an independent sink such as `fmt.Fprintln(os.Stderr, err)`.

## Source and Related Guides

- [logger.go](../logger.go): output, rotation, callbacks, and Close.
- [server.go](../server.go), [recovery.go](../recovery.go): default references, shutdown, and access records.
- [startup.go](../startup.go): rotation task registration.
- [Startup and Graceful Shutdown](lifecycle.en.md), [Configuration and Time](configuration.en.md).
