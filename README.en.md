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

[Quick Start](#quick-start) · [Other Features](#other-features) · [CLI Guide](cmd/goi/README.en.md) · [Custom ORM (CN)](db/README.md) · [Example](https://github.com/NeverStopDreamingWang/goi_example) · [Changelog](https://github.com/NeverStopDreamingWang/goi/releases)

</div>

## Goi CLI

### Install the Goi CLI

The recommended approach is to install the latest version with `go install`:

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

The executable is named `goi` (`goi.exe` on Windows). It is installed into `GOBIN`, or `GOPATH/bin` if `GOBIN` is unset. Add that directory to `PATH` to run `goi` directly.

### Command Usage

```text
goi create-project [path]<project-name>   Create a project configuration package
goi create-app [path]<app-name>           Create an application
goi help [command]                      Show help
```

`[path]` is an optional relative parent-path prefix; the name is required. Do not type the square brackets or angle brackets literally.

For details, see the [Goi CLI Guide](cmd/goi/README.en.md).

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

### Run Goi

#### A Basic Example

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

#### Start with CLI

You can also use the CLI to create a project skeleton, then add applications as needed. First install the CLI, create a project root directory, and enter it:

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
mkdir project
cd project
goi create-project path/myproject
goi create-app path/myapp
go mod tidy
```

The main generated directory structure is:

```text
project/
├── go.mod                    # module project
├── server/
│   └── main.go               # Service entry point
└── path/
    ├── myproject/            # Project configuration package
    │   ├── settings.go        # Server and database configuration
    │   ├── urls.go            # Project routes; initially only a package declaration
    │   └── ...                # Converters, validators, logging, and TLS configuration
    └── myapp/                # Application package
        ├── models.go          # Models and an explicit migration entry point
        ├── serializer.go      # Data validation and writes
        ├── urls.go            # Application route registration
        ├── views.go           # Request handlers
        └── utils.go           # Application helpers
```

`go.mod` is created in the current project root, and the default module name is the root directory name, `project`. `server/main.go` is the entry point, `path/myproject` contains project configuration, and `path/myapp` contains application code. `path` is an example parent path; replace or omit it as needed.

After creating an application, edit `server/main.go` and import the application to register its routes. If tables need to be created, call migration explicitly before starting the server:

```go
package main

import (
	"context"

	"project/path/myapp"
	"project/path/myproject"
)

func main() {
	myapp.Migrate(context.Background()) // Create tables when needed; does not alter existing table structures
	myproject.Server.RunServer()
}
```

Generated projects use SQLite by default. Enable CGO and install an available C compiler before running the application. Start it from the project root:

```sh
go run ./server
```

We recommend the complete [goi_example project](https://github.com/NeverStopDreamingWang/goi_example) for project organization, configuration, and application development. See the [Goi CLI Guide](cmd/goi/README.en.md) for creation rules and the full workflow.

### Stop the Server

Press `Ctrl+C` in the terminal or send `SIGTERM` to the process to stop the server. After the server starts, you can also stop it manually from the application control layer or a separate goroutine:

```go
if err := Server.StopServer(); err != nil {
	goi.Log.Error(err)
}
```

`StopServer()` waits for cleanup to finish and returns any errors. Do not call it synchronously from the current request handler, a managed background task, or a shutdown callback, as it would wait for itself to exit.

## Other Features

The following table groups the framework's main capabilities and extension points by purpose. Refer to the linked source files and the [goi_example project](https://github.com/NeverStopDreamingWang/goi_example) for complete usage examples.

| Feature | Description and entry points |
| --- | --- |
| Startup tasks | Implement `Startup` and call `RegisterOnStartup` before `RunServer()` to register background tasks. Tasks should respond to `ctx.Done()` and call `wg.Done()` once when exiting. See [startup.go](startup.go). |
| Shutdown callbacks | Implement `ShutdownCallback` and register cleanup logic with `RegisterOnShutdown`. Callbacks run in reverse registration order, after background tasks have exited and before databases are closed. Registration remains open until shutdown begins. See [shutdown.go](shutdown.go). |
| Routing and groups | Organize routes and HTTP methods with `Router.Path`, `Include`, and `ViewSet`; customize unmatched-route and unsupported-method handling with `Router.NoRoute` and `ViewSet.NoMethod`. See [router.go](router.go) and [view.go](view.go). |
| Path parameter conversion | Built-in converters include `int`, `string`, `slug`, `path`, and `uuid`. Extend `<type:name>` matching and value conversion with `RegisterConverter`. See [converter.go](converter.go). |
| Middleware extensions | Configure global, group, or individual-route middleware with `Router.Use`. `Middleware` provides request, response, and exception handling hooks. See [middleware.go](middleware.go). |
| Built-in middleware | Enable CORS, security response headers, HTTPS redirects, clickjacking protection, and common request handling as needed. See [middleware/](middleware/). |
| Static resources | Serve local files, directories, and resources embedded with `embed.FS` using `StaticFile`, `StaticDir`, and their corresponding `FS` methods. See [router.go](router.go). |
| Request parsing | Read query parameters, JSON, YAML, regular forms, and multipart forms. Register custom request body parsers with `parser.RegisterParser`. See [context.go](context.go) and [parser/](parser/). |
| Parameter validation | Read and validate parameters with `Params.Get` and `ParseParams`. Built-in validators cover booleans, integers, strings, time, slices, maps, slugs, and UUIDs. Implement `Validator` and call `RegisterValidator` to add more. See [params.go](params.go) and [validator.go](validator.go). |
| Custom validation errors | Implement `ValidationError` and use `Server.Validation.SetValidationError` to customize error messages and response formats. See [validator.go](validator.go). |
| Request context | Access the original `http.Request` and propagate cancellation through `request.Object.Context()`. Update the context with `Request.WithContext`. See [context.go](context.go). |
| Response control | Use `response.Data`, `response.Result`, and `goi.Response` to customize status codes, headers, and file responses; use `RawHandler` to control raw response writes. See [response/](response/) and [context.go](context.go). |
| Databases and ORM | Use multiple database dialects, transactions, context-aware operations, and compatibility-mode routing. Register custom engines and extend identifier, placeholder, and column parsing strategies. See the [Custom ORM Guide (CN)](db/README.md). |
| In-memory cache | Read and write process-local cache through `Server.Cache`, with TTL, expiration cleanup, capacity limits, and eviction policies such as random, LRU, and LFU. See [cache.go](cache.go). |
| Logging | Support leveled, formatted, and access logs, custom output, and log rotation. Register rotation tasks with `RegisterOnStartup`; the user decides when to call `Logger.Close()`. See [logger.go](logger.go). |
| HTTPS and certificates | Configure certificates and private keys with `Server.Settings.SSL` to enable HTTPS. Tools are also provided for generating RSA and ECC self-signed certificates. See [settings.go](settings.go) and [certificate.go](certificate.go). |
| Internationalization | Use `Server.Settings.SetLanguage` to switch globally between the framework's built-in Simplified Chinese and English messages. See [settings.go](settings.go). |
| Time and time zones | Configure time representation with `SetTimeZone` and `UseTZ`, and handle time with `GetTime`, `GetLocation`, and `Localize`. See [time.go](time.go). |
| Password hashing | `auth.MakePassword` and `auth.CheckPassword` provide salted PBKDF2-SHA256 password hashing and verification. See [auth/](auth/). |
| Cryptographic tools | Use AES and RSA encryption/decryption, RSA key generation, and cryptographically secure random string and byte tools. See [crypto/](crypto/). |
