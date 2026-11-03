# Routing, Path Parameters, and Static Resources

[简体中文](routing.md) | **English** · [Documentation](README.en.md)

`Engine.Router` owns a route tree, and `ViewSet` selects a view by HTTP method. Goi resolves the path, selects the method, and then invokes the matched middleware chain and view. Finish registration before serving requests; do not mutate the tree from request handlers.

## A Complete Example Without a Listener

This example calls `ServeHTTP` through `httptest` without opening a network listener. Configure logging first: `NewHTTPServer` does not create a default logger. Logs are discarded here; configure an appropriate output in an application.

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/security"
)

func missing(*goi.Request) any {
	return goi.Response{Status: http.StatusNotFound, Data: "not found"}
}

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()

	goi.RegisterConverter("hex", goi.Converter{
		Regex: `([0-9a-fA-F]+)`,
		ToGo: func(raw string) (any, error) {
			return strconv.ParseUint(raw, 16, 16)
		},
	})

	api := server.Router.Include("api", "API")
	api.Path("items/<int:id>/", "item", goi.ViewSet{
		GET: func(request *goi.Request) any {
			var id int
			if err := request.PathParams.Get("id", &id); err != nil {
				return err.Response()
			}
			return fmt.Sprintf("item=%d", id)
		},
		NoMethod: func(*goi.Request) any {
			return goi.Response{Status: http.StatusMethodNotAllowed, Data: "method not allowed"}
		},
	}).Use(security.Default())
	api.Path("hex/<hex:code>", "hex value", goi.ViewSet{
		GET: func(request *goi.Request) any {
			return fmt.Sprintf("%T:%v", request.PathParams["code"], request.PathParams["code"])
		},
	})
	server.Router.Path("raw", "direct writer", goi.ViewSet{
		GET: func(*goi.Request) any {
			return goi.RawHandler(func(w http.ResponseWriter, _ *goi.Request) error {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusAccepted)
				_, err := io.WriteString(w, "raw")
				return err
			})
		},
	})
	server.Router.NoRoute(goi.ViewSet{GET: missing, Options: missing, NoMethod: missing})

	for _, test := range []struct{ method, path string }{
		{http.MethodGet, "/api/items/7/"},
		{http.MethodPost, "/api/items/7/"},
		{http.MethodGet, "/api/hex/ff"},
		{http.MethodGet, "/raw"},
		{http.MethodGet, "/missing"},
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		fmt.Println(recorder.Code, recorder.Body.String())
	}
}
```

The results are `200 item=7`, `405 method not allowed`, `200 uint64:255`, `202 raw`, and `404 not found`. To serve real traffic, configure logging, settings, and routes, then call `server.RunServer()` instead of running the demonstration request loop.

## The Route Tree and Slashes

The root router already consumes the leading `/`. Use relative child paths without a leading slash: `Path("users/", ...)` under `Include("api", ...)` corresponds to `/api/users/`.

| Method | Purpose and return value |
| --- | --- |
| `Include(path, desc)` | Creates a group that can contain more routes. Adds a trailing `/` when absent and returns that group. |
| `Use(middlewares...)` | Appends middleware in order and returns the current router. More `Use` calls can configure that same router. |
| `Path(path, desc, viewSet)` | Creates a leaf, preserving the path and its trailing slash. Returns the new leaf, not its parent. |
| `StaticFile`, `StaticDir`, `StaticFileFS`, `StaticDirFS` | Create static-resource leaves and return those leaves; see below. |

The `*Router` returned by `Path` or a static registration method is **only for adding middleware through `Use`**. Do not recursively register groups, paths, or static resources on that leaf; this is unsupported group usage and may cause undefined behavior. Retain the parent group to register siblings.

Leaves match the entire path. `users` and `users/` differ: Goi does not automatically redirect to add a slash or normalize repeated slashes. Siblings are tried in registration order, and the first successful match wins. Identical sibling paths or compiled patterns panic during registration, but not every overlapping regular expression is detected.

Use fixed prefixes in `Include`. The current resolver slices child paths using the declared path length, making dynamic placeholders unsuitable for group paths; put dynamic parameters in leaf `Path` declarations. `desc` is descriptive metadata and does not affect matching.

## Path Parameters and Converter

Write placeholders as `<type:name>`, such as `<int:id>`. Read converted values through `request.PathParams.Get("id", &id)`. Path parameters are separate from query and request-body parameters.

| Built-in converter | Match and return value |
| --- | --- |
| `int` | One or more `0-9` characters, converted to int with `strconv.Atoi`; signs are not accepted. |
| `string` | A nonempty segment without `/`, returned as string. |
| `slug` | Nonempty ASCII letters, digits, `-`, and `_`, returned as string. |
| `path` | A nonempty string that may contain `/`, returned as string. |
| `uuid` | A lowercase hexadecimal `8-4-4-4-12` string. |

Each placeholder's `Converter.Regex` must supply **exactly one capturing group**. Use `(?:...)` for additional internal groups. The resolver compares the total capture count for the whole route with the total parameter count; unequal totals prevent a match. One converter having extra groups and another having too few remains unsupported even when their totals happen to match, because values may be assigned to the wrong parameters. Ordinary path text is escaped rather than interpreted as arbitrary regex.

Call `RegisterConverter` before registering routes that use it. Compiled routes retain converter copies; replacing a registered name does not rewrite existing routes. Unknown converters or invalid regexes panic during registration. An error returned by `ToGo` makes that route fail to match, allowing other routes or `NoRoute`; it does not automatically produce a parameter-validation 400 response.

`Converter.ToGo` and [parameter-validator `Validator.Parse`](validation.en.md) are independent APIs. Do not rename them together or assume they accept the same formats.

## HTTP Methods, NoRoute, and NoMethod

`ViewSet` supports `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `Options`, and `TRACE`. The field is named `Options`, not `OPTIONS`.

- An unset `HEAD` falls back to `GET`. An unset `Options` uses the default OPTIONS handler.
- For a matched path without an applicable view, `ViewSet.NoMethod` is used; otherwise the default is 405.
- A group can set `NoRoute(viewSet)` when its prefix matches but none of its children do. Without an applicable fallback, the default is 404.
- `NoRoute` is still a `ViewSet`, so method selection also applies. The example sets `GET`, `Options`, and `NoMethod` to avoid the default OPTIONS 200 response.
- Goi sets `Allow` from the selected `ViewSet.GetMethods()`. `NoMethod` does not add otherwise unregistered methods to that list.

Default 404 and default 405 responses are returned before the route middleware chain is built. Do not assume root `Use` covers these responses. Register explicit `NoRoute` and `NoMethod` handlers when middleware must participate; see [Middleware](middleware.en.md).

`RawHandler` writes its own status, headers, and body, without JSON encoding or automatic `Content-Length`. It runs during response writing, after route middleware has returned; its execution errors do not pass through the view's `ProcessException` chain. Consider request cancellation before streaming and do not expect the framework to replace a status after bytes have been sent.

## Local Files and embed.FS

Local paths are relative to the process working directory; `Settings.BaseDir` is not automatically prepended. Map only directories intended for public access. Static registration does not perform access authorization.

| Registration | Path meaning |
| --- | --- |
| `StaticFile(path, desc, filePath)` | Maps a URL to a local file, opened per request; open failures return 404. |
| `StaticDir(path, desc, http.Dir(...))` | Appends `/<path:fileName>` and opens the matching file under that directory. An empty directory is treated as `.`. |
| `StaticFileFS(path, desc, fileFS, defaultPath)` | Maps a URL to `defaultPath` inside embed.FS. |
| `StaticDirFS(path, desc, dirFS, basePath)` | Appends `/<path:fileName>`, anchors the FS with `fs.Sub(basePath)`, and looks up `fileName`. Empty `basePath` means `.`. |

These methods register GET and HEAD. Directory routes require a nonempty `fileName`, so `/assets/` is not `/assets/index.html`. Local directory objects do not provide directory listings or automatic index files: opening a directory results in 404. Embedded responses delegate to `net/http.ServeFileFS`, including its redirects for `index.html` requests; they are not a SPA fallback.

For the following example, create `public/index.html` containing, for example, `<h1>Goi</h1>` and `public/hello.txt` containing `hello` alongside the `main.go` directory tree before building. Embedded paths use `/`, not operating-system absolute paths.

```go
package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/NeverStopDreamingWang/goi/v3"
)

//go:embed public
var site embed.FS

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()
	server.Router.StaticFileFS("", "home", site, "public/index.html")
	server.Router.StaticDirFS("assets", "embedded assets", site, "public")
	server.Router.StaticFile("download", "local file", "public/index.html")
	server.Router.StaticDir("files", "local directory", http.Dir("public"))

	for _, path := range []string{"/", "/assets/hello.txt", "/download", "/files/index.html"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		fmt.Println(path, recorder.Code)
	}
}
```

When composing views directly, `StaticFileFSView` accepts `fs.FS` plus a file name. `StaticDirFSView` accepts only one `fs.FS` and requires a `fileName` path parameter. Anchor the filesystem explicitly when using `fs.Sub` yourself. The response layer closes successfully returned local file objects; do not close them before returning from the view.

## Source Files

[router.go](../router.go) registers routes; [resolvers.go](../resolvers.go) compiles and matches paths; [converter.go](../converter.go) defines path converters; [view.go](../view.go) selects methods; [static.go](../static.go) and [context.go](../context.go) handle files and raw responses; [server.go](../server.go) connects the request flow.
