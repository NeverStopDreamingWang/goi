<div align="center">

# Goi CLI

### Scaffolding for Goi projects and applications

`goi` creates project configuration packages and application scaffolding. This guide covers installation, optional local builds, creation, application registration, and running a project.

[![Version](https://img.shields.io/github/v/release/NeverStopDreamingWang/goi?label=version&color=blue)](https://github.com/NeverStopDreamingWang/goi/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/NeverStopDreamingWang/goi?label=Go&logo=go&logoColor=white)](../../go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/NeverStopDreamingWang/goi/v3.svg)](https://pkg.go.dev/github.com/NeverStopDreamingWang/goi/v3)
[![Stars](https://img.shields.io/github/stars/NeverStopDreamingWang/goi?label=stars&color=blue)](https://github.com/NeverStopDreamingWang/goi/stargazers)
[![License](https://img.shields.io/github/license/NeverStopDreamingWang/goi?label=license)](../../LICENSE)

[简体中文](README.md) | **English**

[Install the CLI](#install-the-cli) · [Commands](#command-overview) · [Create a Project and App](#create-a-project-and-an-application) · [Help and Common Errors](#help-and-common-errors) · [Project README](../../README.en.md)

</div>

## Install the CLI

Go 1.24 or later is required. The recommended approach is to install the latest version with `go install`:

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@latest
```

To pin a version, for example v3.0.4:

```sh
go install github.com/NeverStopDreamingWang/goi/v3/cmd/goi@v3.0.4
```

The executable is named `goi` (`goi.exe` on Windows). It is installed into `GOBIN`, or `GOPATH/bin` if `GOBIN` is unset. Inspect the configuration with the following command, then add the appropriate directory to `PATH`:

```sh
go env GOBIN GOPATH
```

Open a terminal that picks up the updated `PATH` and verify the installation:

```sh
goi help
```

The help output shows the Goi version and available commands. If `goi` cannot be found, first check whether its installation directory is in `PATH`. `go install` builds the command locally from versioned source, so you do not need to download a separate prebuilt executable.

## Optional Local Build

Use a local build only when you need to modify the CLI source or customize build options. The source entry point is `cmd/goi/`, and the build scripts are in `cmd/`. Run the appropriate script from the Goi framework repository root.

On Windows:

```bat
cmd\build.bat
```

On Linux:

```sh
sh ./cmd/build.sh
```

Both scripts cross-compile the CLI for Windows and Linux, targeting `amd64`, and generate:

- `windows/amd64`: `cmd/dist/windows-amd64/goi.exe`
- `linux/amd64`: `cmd/dist/linux-amd64/goi`

The scripts set `CGO_ENABLED=0`, so no target-system C toolchain is required. They use `-trimpath` to remove local paths and `-ldflags="-s -w"` to remove symbol tables and debugging information, reducing executable size. Each script switches to its own directory automatically and can also be invoked by path from another working directory.

`cmd/dist/` only stores local build artifacts and is not tracked in Git. Do not commit the generated binaries to the source repository. These scripts build the Goi CLI, not the SQLite application generated later.

## Command Overview

Run `goi` without a subcommand, or run `goi help`, to see the version and command overview. The command syntax is:

```text
goi help [command]
goi create-project [path]<project-name> [--module <module-path>]
goi create-app [path]<app-name>
```

`[path]` is an optional relative parent-path prefix and may contain multiple directory levels. Supply it in your system's path format and include the separator needed to join it to the name. `<project-name>` and `<app-name>` are required names. Do not type the square brackets or angle brackets literally.

The `command` argument to `help` is optional. Specify `create-project` or `create-app` to see detailed arguments and examples. `--module <module-path>` is only used when creating a project and explicitly sets the module written to `go.mod`. If omitted, the current project root directory name is used. This does not change the configuration-package or application name.

## Create a Project and an Application

First create and enter the outer project directory, then run the two creation commands in sequence:

```sh
mkdir project
cd project
goi create-project path/myproject
goi create-app path/myapp
go mod tidy
```

If you do not need a parent path, use the following pair after creating and entering a new project root. These replace the two `goi` commands above. Choose one approach; do not run both pairs in sequence:

```sh
goi create-project myproject
goi create-app myapp
```

This creates the `myproject/` configuration package and the `myapp/` application directly under the current root. The directory tree and import examples below use `path/`; if you omit the parent path, also remove `/path` from the example imports.

Once you enter `project`, it is the current project root. Do not change directories between the two `goi` commands. The default module name is the last directory name of the current root when `create-project` runs, which is `project` in this example. The final argument names `myproject` and `myapp` are the configuration-package and application-package names; they do not determine the module name. `create-project` generates `go.mod` and `server/main.go` in the current directory. Its argument only specifies the configuration-package directory.

To customize the module path, replace the `create-project` command with:

```sh
goi create-project path/myproject --module example.com/team/myproject
```

The generated `server/main.go` is a simple service entry point without additional Cobra/web subcommands or business code. The configuration package also includes a `urls.go` file containing only a `package` declaration initially. Creation commands reject existing generation targets and do not overwrite source code, configuration, or keys.

## Path and Application Rules

Run `create-app` from a project root containing `go.mod`. It does not search parent directories and cannot be run from a project subdirectory. Application import paths use the actual module in the current `go.mod`, so renaming the project directory does not affect them.

`path/myapp` creates parent directories automatically and generates the application files at that full path. The final name, `myapp`, becomes the Go package name, not `path_myapp`. Both creation commands accept relative paths. On Windows, you can also use backslashes:

```powershell
mkdir project
cd project
goi create-project path\myproject
goi create-app path\myapp
```

Forward slashes and backslashes refer to the same target; do not run both examples. Existing parent directories can be reused, but the target configuration-package or application directory must not exist. The commands do not merge files automatically. Existing generation targets such as `go.mod` and `server/main.go` are not overwritten either.

Paths must not be absolute or contain `..`, `.`, or empty segments. For example, `../myapp`, `./myapp`, `path/../myapp`, and `path//myapp` are rejected. The final configuration-package and application names must be valid ASCII Go package names and cannot be `main`, `_`, or a Go keyword. The configuration-package path cannot be exactly `server`, because that directory is reserved for the service entry point. Subpaths such as `server/config` are not subject to this restriction.

After completing the flow above, the directory structure is:

```text
project/                      # Current working directory and project root
├── go.mod                   # module project
├── server/
│   └── main.go               # Simple service entry point
└── path/                    # Replaceable relative parent path
    ├── myproject/           # Project configuration package
    │   ├── settings.go
    │   ├── urls.go          # Initially contains only a package declaration
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

When creating an application, the CLI searches the current Go module recursively for a project configuration package whose `settings.go` declares `Server`. It skips hidden directories, `vendor`, `node_modules`, and nested modules with their own `go.mod`; the search is not limited to the root or its direct children. Missing or multiple candidates cause an error. The CLI does not guess the package from directory names.

## Import Applications and Run Migrations Explicitly

After creating an application, edit `server/main.go` manually to import it and run migrations as needed. The following example uses the default module `project`, configuration-package name `myproject`, and application path `path/myapp`:

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

The scaffolding does not modify `server/main.go` automatically to register new applications. To register routes without running migrations, you can also use a blank import, such as `_ "project/path/myapp"`.

`--module example.com/team/myproject` explicitly overrides the default module. In that case, replace the import prefix `project` above with the full module path, for example `"example.com/team/myproject/path/myapp"`. The generated `urls.go` uses the actual import path and discovered configuration-package name, such as `myproject.Server`, rather than a fixed `project` alias. Importing an application registers its routes but does not create tables automatically in `init`. If tables need to be created, call `Migrate(ctx)` explicitly after database configuration and before starting the service. Migration only creates missing tables; it does not alter existing structures automatically.

Default routes and table names use the final application name: `path/myapp` uses the `myapp/` route and the `myapp_my_model` table. If multiple groups end in `myapp`, adjust the route prefix in `urls.go`, the table name in `ModelSet()`, and import aliases to avoid conflicts. Group directories do not automatically become route or table-name prefixes.

## Context and Database

The generated `Validate`, `Create`, `Update`, and `Delete` methods accept `context.Context` and pass it to the ORM. HTTP handlers should pass `request.Object.Context()` instead of replacing the request context with `context.Background()`. The background context in the main function above is only for startup migrations.

`Update` performs partial updates: nil fields are skipped by default, while ordinary zero values are still written. To write NULL explicitly, choose fields and write rules according to your model constraints. Database errors retain their original error chain; the caller handles user-facing business messages. See the [Database Development Guide (CN)](../../db/README.md) for complete ORM behavior.

Projects configure SQLite by default under the connection name `default` and import the `github.com/mattn/go-sqlite3` driver. The data directory is created on the first connection. Running the generated application requires `CGO_ENABLED=1` and an available C compiler. The repository build scripts use `CGO_ENABLED=0` only for the Goi CLI, not for this SQLite application configuration.

After adding imports and any required migration calls, run the following from the generated project root:

```sh
go mod tidy
go run ./server
```

## Help and Common Errors

```sh
goi help create-project
goi help create-app
```

`create-app` checks `go.mod` in the current directory. If the file is missing, it returns a project-root error without searching parent directories or writing application files. Switch to the project root containing `go.mod` and retry. For a project that has not been created yet, run `create-project` in the chosen root first, then run `create-app` directly from the same directory.

Invalid paths, existing targets, or an ambiguous configuration package also cause errors. Fix the path or project structure first rather than trying to overwrite existing files by rerunning the command.

## Verify the CLI

Run these commands from the Goi framework repository root, not from a generated project:

```sh
go test ./cmd/goi
go test -race ./cmd/goi
go test -tags=integration ./cmd/goi
```

Ordinary tests cover names, paths, overwrite prevention, and template generation. Integration tests generate and compile a real project and verify SQLite behavior. They require Go, CGO/a C compiler, and cached dependencies or network access. Race tests also require a toolchain supported by the target platform.
