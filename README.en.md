<div align="center">

# Goi

### An extensible web framework built on Go's standard library

From project scaffolding and request handling to database access and graceful shutdown, keep web development focused on business logic.

Built on `net/http`, Goi integrates routing, middleware, parameter validation, internationalization, and a multi-database ORM. Custom extensions and project/application scaffolding help reduce repetitive work.

[![Version](https://img.shields.io/github/v/release/NeverStopDreamingWang/goi?label=version&color=blue)](https://github.com/NeverStopDreamingWang/goi/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/NeverStopDreamingWang/goi?label=Go&logo=go&logoColor=white)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/NeverStopDreamingWang/goi/v3.svg)](https://pkg.go.dev/github.com/NeverStopDreamingWang/goi/v3)
[![Stars](https://img.shields.io/github/stars/NeverStopDreamingWang/goi?label=stars&color=blue)](https://github.com/NeverStopDreamingWang/goi/stargazers)
[![License](https://img.shields.io/github/license/NeverStopDreamingWang/goi?label=license)](LICENSE)

[简体中文](README.md) | **English**

[Quick Start](#quick-start) · [CLI Guide (CN)](cmd/goi/README.md) · [Custom ORM (CN)](db/README.md) · [Example](https://github.com/NeverStopDreamingWang/goi_example) · [Changelog](https://github.com/NeverStopDreamingWang/goi/releases)

</div>

## Goi CLI

See the [Goi CLI Guide (CN)](cmd/goi/README.md) for creation rules, grouped application directories, and complete examples.

### Install the Goi CLI

The recommended approach is to install the latest version with `go install`:

```shell
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

The executable is named `goi` (`goi.exe` on Windows). It is installed into `GOBIN`, or `GOPATH/bin` if `GOBIN` is unset. Add that directory to `PATH` to run `goi` directly.

`go install` builds the CLI locally from the source associated with the release tag. Publishing a version therefore only requires committing the source and creating a tag; prebuilt Windows or Linux executables do not need to be committed.

### Build the Goi CLI Locally

Use a local build when you need to modify the source or customize build options.

The CLI entry point is `cmd/goi/`, and the build scripts are in `cmd/`. Separate scripts are provided for Windows and Linux; both cross-compile executables for both operating systems. Both targets use `amd64` and `CGO_ENABLED=0`, so no target-system C toolchain is required. The build commands use `-trimpath` and `-ldflags="-s -w"` to remove local paths, symbol tables, and debugging information, reducing artifact size.

On Windows:

```bat
cmd\build.bat
```

On Linux:

```sh
sh ./cmd/build.sh
```

Either script produces both artifacts:

- `windows/amd64`: `cmd/dist/windows-amd64/goi.exe`
- `linux/amd64`: `cmd/dist/linux-amd64/goi`

Both scripts switch to their own directory automatically, so they can also be invoked by path from another working directory.

`cmd/dist/` is only for local verification or generating optional download assets. Build artifacts are not tracked in Git.

### Use the Goi CLI

```text
> goi

Usage:
        goi <command> [arguments]
The commands are:
        goi create-project [path]<project-name>   Create project
        goi create-app [path]<app-name>           Create app

```

`[path]` is an optional relative parent-path prefix and may contain multiple directory levels. Supply it in your system's path format and include the separator needed to join it to the name. `<project-name>` and `<app-name>` are required names. Square brackets and angle brackets are syntax notation; do not type them literally.

Example:

```shell
# Create and enter the outer project root
> mkdir project
> cd project
# Generate the configuration package at path/myproject
> goi create-project path/myproject
# Do not change directories between the two goi commands
> goi create-app path/myapp
> go mod tidy

```

You can also omit the parent path. After creating and entering a new project root, replace the two `goi` commands above with the following pair. Choose one approach; do not run both pairs in sequence:

```sh
goi create-project myproject
goi create-app myapp
```

Without a parent path, the configuration package and application are created directly under the current root as `myproject/` and `myapp/`. The directory and import examples below still use `path/`; if you omit it, also remove `/path` from the example imports.

The default module name is the last directory name of the current project root when `create-project` runs, which is `project` in this example. Override it explicitly with `--module example.com/team/myproject`. The example `path` can be replaced, span multiple levels, or be omitted entirely. The final names `myproject` and `myapp` are the configuration package name and application package name; they do not determine the module name. `create-project` generates `go.mod` and a simple `server/main.go` in the current directory; its argument only specifies the relative configuration-package path. `create-app` also accepts a relative path: the full path is used for directories and imports, while the final name becomes the package name.

The configuration package also includes a `urls.go` file containing only a `package` declaration initially.

Both paths support `/`; Windows also supports `\`. Absolute paths, `..`, `.`, empty path segments, and invalid final package names are rejected. The final name must be an ASCII Go package name and cannot be `main`, `_`, or a Go keyword. Existing generation targets are rejected: source code, configuration, and keys are not overwritten.

`create-app` only checks `go.mod` in the current directory. It fails if that file is missing, does not search parent directories, and cannot be run from a subdirectory. Run `goi help create-app` for usage. The configuration package is discovered recursively within the current module by finding a `settings.go` that declares `Server`. Hidden directories, `vendor`, `node_modules`, and nested Go modules are skipped. Multiple candidates cause an error. See the [CLI Guide (CN)](cmd/goi/README.md) for the complete directory tree.

### Initialize the Generated Database and Application

New projects use SQLite by default, with the connection name `default`, and import the `github.com/mattn/go-sqlite3` SQL driver. The database directory is created on the first connection, and the log directory is created with directory permissions. This SQLite driver requires `CGO_ENABLED=1` and an available C compiler. The `CGO_ENABLED=0` scripts above build the Goi CLI, not the generated application.

Default application routes and table names use the final application name. For example, `path/myapp` uses `myapp/` and `myapp_my_model`. If different groups share the same final name, adjust their default routes, table names, and import aliases to avoid conflicts. Importing an app registers its routes but does not automatically migrate the database in `init`. If tables need to be created, call migration explicitly after database configuration. Edit `server/main.go` manually; the following example uses the default module `project` above and refers directly to the configuration package as `myproject`:

```go
package main

import (
	"context"

	"project/path/myapp"
	"project/path/myproject"
)

func main() {
	myapp.Migrate(context.Background()) // Run when needed; does not alter existing table structures
	myproject.Server.RunServer()
}
```

After adding the imports, run `go run ./server` from the same project root.

The generated `Validate`, `Create`, `Update`, and `Delete` methods accept the caller's `context.Context` and pass it to the ORM. In an HTTP handler, use `request.Object.Context()` rather than replacing the request context with `context.Background()`. `Update` performs partial updates by default: omitted nil fields retain their existing values. To clear a nullable field, explicitly choose the fields and write rules required by your use case. Database errors retain their error chain; the caller is responsible for converting them into user-facing business messages.

Ordinary creation tests do not require a physical database driver. Tests that generate and compile a real project and exercise SQLite can be run in an environment with Go, CGO/a C compiler, and cached dependencies or network access:

```sh
go test -tags=integration ./cmd/goi
```

## Quick Start

### Prerequisites

Goi requires Go 1.24 or later.

### Get Goi

With Go module support, adding an import allows `go [build|run|test]` to fetch the required dependencies automatically:

```go
import "github.com/NeverStopDreamingWang/goi/v3"
```

Alternatively, use `go get`:

```bash
go get github.com/NeverStopDreamingWang/goi/v3@latest
```

### Migrate from v2 to v3

v3 introduces incompatible ORM API changes. First update imports for Goi and its subpackages to `/v3`, then update the dependency. Reinstall the CLI if you use it. Do not change version suffixes belonging to third-party dependencies.

```sh
go get github.com/NeverStopDreamingWang/goi/v3@v3.0.1
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@v3.0.1
go mod tidy
```

| Migration item | v3 usage |
| --- | --- |
| Context | Migration, CRUD, counting, pagination, and transaction methods accept `context.Context` directly; for example, `Select(ctx, &list)`, `First(ctx, &item)`, and `Delete(ctx)`. The basic `Execute`, `Query`, and `QueryRow` methods still provide `...Context` variants. |
| Writing nil | `Insert(ctx, model, skipNil)` and `Update(ctx, model, skipNil)` require `skipNil`: `true` skips nil values, while `false` writes selected nil values as SQL NULL. Ordinary zero values are not skipped. Use `Fields` to control the write scope. |
| Migration namespace | Instead of passing Schema to `Migrate`, set `Settings.Schema` in the value returned by the model's `ModelSet()`, then call `Migrate(ctx, model)`. |
| Kingbase | Select the dialect with `kingbase.Config.Mode` and use the corresponding native package's Engine, Model, Settings, and Validate. For example, use `db.Connect[*mysql.Engine]` in MySQL mode. Move the former package-level strategy configuration into `Config.EngineOptions`. |
| Map query results | Map keys use Go field names, such as `Name`, rather than database column names such as `name`. Update code that reads these results accordingly. |
| SeaBoxSQL | The new `seaboxsql` router supports `postgres`, `mysql`, and `oracle`. An empty Mode defaults to PostgreSQL. It selects only the ORM dialect; it does not switch the server mode or physical driver. |

`Update` and `Delete` affect the entire table when no WHERE condition is set. Review filters and `Fields` when upgrading. For database models, configuration, and complete examples, see the [Custom ORM Guide (CN)](db/README.md). Compatibility-mode routing does not imply complete equivalence to the corresponding native database; verify it against your target instance.

### Run Goi

A basic example:

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
	// Create a Goi server
	Server := goi.NewHTTPServer()
	// Network protocol
	Server.Settings.Network = "tcp" // Default: "tcp"; common protocols: "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6"
	// Bind address: 0.0.0.0
	Server.Settings.BindAddress = "0.0.0.0" // Default: 127.0.0.1
	// Listen on port 8080
	Server.Settings.Port = 8080

	// Register a route
	Server.Router.Path("ping", "Test endpoint", goi.ViewSet{GET: Ping})

	// Start the server
	Server.RunServer()
}
```

Run the code with `go run`:

```bash
go run main.go
```

Then visit `http://0.0.0.0:8080/ping` in your browser to see the response.

### Graceful Shutdown

Goi listens for `SIGINT` (for example, `Ctrl+C`) and `SIGTERM`. The application control layer can also call `Server.StopServer()` manually. Once the server enters its running lifecycle, both entry points share the same shutdown process. Concurrent or repeated calls run cleanup only once and return the same shutdown result to all callers. A normal HTTP `Serve` return during shutdown does not initiate another shutdown; an unexpected `Serve` return still enters the unified shutdown process. `RunServer()` waits for all cleanup to finish before returning.

Shutdown proceeds in this order: stop HTTP listening and wait for in-flight requests, cancel and wait for background tasks, run shutdown callbacks in reverse registration order, and finally close the databases. Errors returned by HTTP `Shutdown`, `OnShutdown`, or database `Close` are logged immediately. Remaining cleanup continues, and `StopServer()` returns the combined errors.

There is no shutdown timeout, ensuring dependencies are not closed while requests or background tasks may still be using them. Background tasks must therefore respond to `ctx.Done()` and call `wg.Done()` once. Do not synchronously call `StopServer()` from the current HTTP handler, a managed background task, or an `OnShutdown` callback: it would wait for itself to finish. Initiate shutdown from a separate goroutine or the application control layer instead. If startup initialization, an in-flight request, or a background task blocks indefinitely, graceful shutdown will also wait indefinitely, requiring the operating system to force termination.

Each `Engine` and the global startup/shutdown task managers support only one lifecycle and cannot run again after stopping. Call `RegisterOnStartup` before `RunServer()`; tasks registered after background-task startup begins do not participate in the current lifecycle. `RegisterOnShutdown` may be called while the service is running, but callbacks registered after shutdown begins do not participate in that lifecycle. Calling `StopServer()` before `RunServer()` starts returns `ErrServerNotRunning`.
