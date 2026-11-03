# Middleware Registration, Ordering, and Built-in Configuration

[简体中文](middleware.md) | **English** · [Documentation](README.en.md) · [Routing](routing.en.md)

Goi middleware belongs to the route tree. Root, group, and leaf middleware are combined along the matched route and wrap the selected view. This is not a global `net/http` filter independent of routing. Finish configuration and `Use` registration before handling requests.

## A Complete Example: Normal, Short-Circuited, Panic, and Preflight Requests

This program invokes the chain through `httptest`, without a listener or log files. The logger is assigned before `NewHTTPServer`; configure an appropriate output before calling `RunServer` in a real application.

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/clickjacking"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/common"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/corsheaders"
	"github.com/NeverStopDreamingWang/goi/v3/middleware/security"
)

type traceMiddleware struct{ name string }

func (m traceMiddleware) ProcessRequest(request *goi.Request) any {
	fmt.Println("request", m.name)
	if m.name == "group" && request.Object.Header.Get("X-Block") == "1" {
		return goi.Response{Status: http.StatusForbidden, Data: "blocked"}
	}
	return nil
}

func (m traceMiddleware) ProcessException(_ *goi.Request, exception any) any {
	fmt.Println("exception", m.name)
	if m.name == "group" {
		return goi.Response{Status: http.StatusInternalServerError, Data: "handled panic"}
	}
	return nil
}

func (m traceMiddleware) ProcessResponse(_ *goi.Request, response *goi.Response) {
	fmt.Println("response", m.name)
	response.Header().Add("X-Trace", m.name)
}

func main() {
	goi.Settings.Debug = false
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	defer goi.Log.Close()
	server := goi.NewHTTPServer()

	cors := corsheaders.Default()
	cors.AllowedOrigins = []string{"https://client.example"}
	cors.AllowHeaders = append(cors.AllowHeaders, "Authorization")
	cors.ExposeHeaders = []string{"X-Trace"}
	server.Router.Use(common.Default(), security.Default(), clickjacking.Default(), cors, traceMiddleware{"root"})
	api := server.Router.Include("api", "API").Use(traceMiddleware{"group"})
	api.Path("ping", "ping", goi.ViewSet{
		GET: func(request *goi.Request) any {
			if request.Object.Header.Get("X-Panic") == "1" {
				panic("example")
			}
			return "ok"
		},
	}).Use(traceMiddleware{"leaf"})

	for _, mode := range []string{"normal", "blocked", "panic", "preflight"} {
		method := http.MethodGet
		if mode == "preflight" {
			method = http.MethodOptions
		}
		request := httptest.NewRequest(method, "/api/ping", nil)
		request.Header.Set("Origin", "https://client.example")
		switch mode {
		case "blocked":
			request.Header.Set("X-Block", "1")
		case "panic":
			request.Header.Set("X-Panic", "1")
		case "preflight":
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		fmt.Println(mode, recorder.Code, recorder.Header().Values("X-Trace"))
	}
}
```

The four results are `normal 200 [leaf group root]`, `blocked 403 [group root]`, `panic 500 [leaf group root]`, and `preflight 200 []`. CORS answers preflights before later request middleware and the view run. Security-header middleware placed outside CORS can still process that response.

## Interface, Registration, and Lifetime

Custom types must implement all three `goi.Middleware` methods. Return nil or use an empty body for an unused phase.

| Method | Role |
| --- | --- |
| `ProcessRequest(*goi.Request) any` | Nil enters the next layer. A non-nil result becomes the response and skips inner middleware and the view. |
| `ProcessException(*goi.Request, any) any` | Handles a view panic. A non-nil result stops further outer exception hooks. |
| `ProcessResponse(*goi.Request, *goi.Response)` | Changes the response, such as its status, data, or headers. Normally do not write directly to the network here. |

`server.Router.Use(...)` participates in requests where a view is **successfully selected** under that root. `group.Use(...)` applies to that group, and `group.Path(...).Use(...)` applies only to that leaf. `Use` returns the current router. Leaves returned by `Path` and static registration methods are only for `Use`; do not recursively register more paths on them.

Instances are reused across requests. Store request-local data in `request.Params` or update the context with `request.WithContext`. Do not store request data in shared instance fields without synchronization. Route registration and middleware configuration do not promise concurrent runtime mutation. Built-in `Default()` functions return configuration values: configure them before passing them to `Use`, rather than expecting later changes to the original value to update the registered copy.

These options are middleware struct fields, not `Server.Settings` fields. `Server.Settings.SSL.Enabled` controls TLS listening; `SecurityMiddleware.SSLRedirect` only sends an HTTP redirect. They are separate settings.

## Execution Order and Exception Boundaries

Considering just the three trace layers in the example:

```text
root.ProcessRequest → group.ProcessRequest → leaf.ProcessRequest → view
root.ProcessResponse ← group.ProcessResponse ← leaf.ProcessResponse
```

If group short-circuits during its request phase, group and the already-entered root response phases still run, but leaf is not entered. Ordinary return values become `goi.Response`; return `goi.Response` or `*goi.Response` to control status and data directly.

For a view panic, `ProcessException` tries leaf → group → root until a hook returns a non-nil response. Once handled, response processing continues outward. **Panics in middleware's own `ProcessRequest` or `ProcessResponse` do not pass through these view exception hooks**; the wrapper attempts a 500 response. Unhandled panics, panicking exception hooks, or response-writing failures may also reach outer recovery. Do not assume every failure executes every `ProcessResponse`.

`ProcessException` handles panics, not ordinary returned `error` or `ValidationError` values. Return an appropriate response, such as `validationErr.Response()`. Produce safe public messages in your exception hook when internal details must remain hidden; default recovery does not guarantee that every error message is hidden. `RawHandler` executes during response writing, and its execution failures do not pass through these view exception hooks.

Default unmatched-route 404 and unsupported-method 405 responses are returned before constructing the middleware chain, so root `Use` does not process them. Explicit `NoRoute(ViewSet)` and `ViewSet.NoMethod` handlers let selected fallbacks pass through the route chain. CORS has no separate global interception of default 404/405 responses.

## common: General Request Handling

`common.Default()` enables neither filtering nor redirects.

| Field | Default and behavior |
| --- | --- |
| `PrependWWW` | false. When enabled, returns 301 for a Host not starting with `www.`, preserving RequestURI. |
| `DisallowedUserAgents` | Empty. Matches regexes against nonempty User-Agent values and returns 403 on a match. Empty and invalid patterns are ignored. |

Redirect schemes use `request.TLS` or `X-Forwarded-Proto: https`. The request Host is used directly; this is not a host allowlist. Constrain and sanitize incoming Host and forwarding headers at a trusted proxy boundary. User-Agent filtering is not authentication.

## security: Security Headers and HTTPS Redirects

| Field | `security.Default()` and behavior |
| --- | --- |
| `ContentTypeNoSniff` | true. Adds `X-Content-Type-Options: nosniff` only when absent. |
| `CrossOriginOpenerPolicy` | `same-origin`. Added when configured and not already set. |
| `ReferrerPolicy` | `[]string{"same-origin"}`. Joined with commas; an existing header is preserved. |
| `SSLRedirect` | false. When enabled, redirects nonsecure, nonexempt requests to HTTPS with 301. |
| `SSLHost` | Empty, retaining the original Host for redirects; a nonempty value replaces it. |
| `RedirectExempt` | Empty. Matches regexes against paths with leading `/` characters removed; invalid regexes fall back to prefix matching. |
| `HSTSSeconds` | 0. Writes HSTS `max-age` only when positive, the request is considered HTTPS, and the header is absent. |
| `HSTSIncludeSubdomains`, `HSTSPreload` | false. Append directives when HSTS is emitted; no external registration or validation is performed. |

HTTPS detection directly uses TLS state or the `https` value of `X-Forwarded-Proto`, without a trusted-proxy allowlist. Depend on that header only when your deployment controls and sanitizes it. Defaults do not force HTTPS or enable HSTS; these headers do not create certificates or start a TLS listener.

## corsheaders: CORS and Preflight

| Field | `corsheaders.Default()` and behavior |
| --- | --- |
| `AllowAllOrigins` | false. When enabled, allows every origin. With credentials, echoes the concrete Origin instead of `*`. |
| `AllowedOrigins` | Empty. Exact entries compare URL scheme and host, including port. Without `AllowAllOrigins`, the string `null` requires an explicit entry. |
| `AllowedOriginRegexes` | Empty. Regexes match the raw Origin; invalid rules are ignored. Add anchors when a whole-string match is intended. |
| `AllowCredentials` | false. Adds `Access-Control-Allow-Credentials: true` only for an allowed origin. |
| `ExposeHeaders` | Empty. Lists exposed response headers for an allowed origin. |
| `AllowHeaders` | `Accept`, `Accept-Language`, `Content-Language`, `Content-Type`; `Authorization` is not included by default. |
| `AllowMethods` | `DELETE`, `GET`, `OPTIONS`, `PATCH`, `POST`, `PUT`. This is an advertised list, not a replacement for `ViewSet` method registration. |
| `PreflightMaxAge` | 86400 seconds. Emitted for preflight when positive, omitted otherwise. |
| `URLRegex` | `^.*$`. Applies to matching URL.Path values; empty or invalid patterns disable processing. |
| `AllowPrivateNetwork` | false. Adds the corresponding allow header when enabled and requested with the true marker; it does not enforce network access control. |

An Origin header is required. Preflight additionally means OPTIONS with a nonempty `Access-Control-Request-Method`. On a route that has matched and entered the chain, preflight returns 200 before response-phase origin checks add allowed headers. **A disallowed origin may still receive 200 without an allow-origin header**; ordinary requests are not rejected by the server because of CORS. CORS is not authentication, CSRF protection, or server-side request-origin filtering.

Allowed-origin responses add `Vary: Origin`. `AllowHeaders` and `AllowMethods` advertise lists in preflight responses rather than validating each requested header or method. When authentication would block credential-free OPTIONS, place CORS before it. The application must choose which origins and credentials to allow.

## clickjacking: X-Frame-Options

`clickjacking.Default()` sets `XFrameOptions` to `DENY`. Empty values also use `DENY`, and configured values are uppercased. Existing `X-Frame-Options` headers are preserved. Common values are `DENY` and `SAMEORIGIN`; the implementation does not validate arbitrary strings or configure CSP.

The implementation also checks the response-header map for the **exact lowercase key** `xframe_options_exempt`, whose first value equals `true` case-insensitively. This convention requires direct access to `response.Header()["xframe_options_exempt"]`; do not assume a key canonicalized by standard `Header.Set` will match. The marker is not automatically removed and is not a privacy or authorization mechanism.

## Source Files

[middleware.go](../middleware.go) defines the interface and wrapping order. [server.go](../server.go), [recovery.go](../recovery.go), and [context.go](../context.go) handle route selection, recovery, and response writing. [settings.go](../settings.go) defines server settings. Built-ins: [common](../middleware/common/middleware.go), [security](../middleware/security/middleware.go), [corsheaders](../middleware/corsheaders/middleware.go), and [clickjacking](../middleware/clickjacking/middleware.go).
