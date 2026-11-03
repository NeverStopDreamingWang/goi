# Requests, parsers, and responses

[简体中文](http.md) | **English** · [Documentation](README.en.md) · [Parameter validation](validation.en.md)

This guide explains how HTTP data enters `Request`, when it is parsed and validated, and how view results become responses. A Goi view has the signature `func(*goi.Request) any`, rather than the standard library handler signature with an `http.ResponseWriter` argument.

Source: [context.go](../context.go), [params.go](../params.go), [view.go](../view.go), [parser/](../parser/), [response/](../response/).

## 1. Request objects and parameter sources

| Entry point | Contents and behavior |
| --- | --- |
| `request.Object` | The original `*http.Request`, including headers, cookies, Body, MultipartForm, and Context. |
| `request.PathParams` | Values captured by route patterns such as `<type:name>`, already converted by the route `Converter.ToGo`. This is separate from the parameter `Validator.Parse` interface. |
| `request.QueryParams()` | Creates a new `Params` from the URL on each call: one value becomes a `string`, one empty value becomes `nil`, and repeated keys become `[]string`. |
| `request.BodyParams()` | Selects a parser by Content-Type and reads the body. Form parsers may also merge URL query parameters, as described below. |
| `request.Params` | An initially empty, per-request `Params` for sharing application data between middleware and views. Query and body parameters are not copied into it automatically. |

For example, `?name=goi&empty=&tag=a&tag=b` produces `name: "goi"`, `empty: nil`, and `tag: []string{"a", "b"}`. Empty form strings have different behavior; do not assume all input sources produce the same types.

`Params.Get(key, &dest)` reads one key and delegates assignment to `SetValue`; it does not run a `type` tag validator. `Params.ParseParams(&struct)` selects `Validator.Parse` through field tags. Both return `ValidationError`: check it and return `err.Response()` from a view instead of returning the error object as successful data. See [Parameter validation](validation.en.md) for tags, `required`, and `allow_null`.

### Request context

Pass `request.Object.Context()` to the ORM, HTTP clients, or other context-aware operations. `request.WithContext(ctx)` replaces `Object` with the result of `Object.WithContext(ctx)`; it does not create new PathParams or Params maps.

Derive additional context from the original request context instead of replacing its cancellation chain with `context.Background()`. `ServeHTTP` adds the framework's `requestID` to that chain. Use an application-specific key type for your own context values to avoid collisions.

## 2. Body parsing

The parser contract and registration entry points are shown below. The registration key is the MIME string used for lookup, not the result of `Name()`:

```text
type Parser interface {
    Name() string
    Parse(*http.Request) (parser.Params, error)
}
parser.RegisterParser(mimeType, implementation)
parser.GetParser(contentType)
```

| Parser | Default matching Content-Type | Result or limitation |
| --- | --- | --- |
| `parser.JSON` | `application/json` | Decodes into `map[string]any`; ordinary numbers are `float64` by default. A top-level object is supported, `null` returns a nil map, and non-object arrays/scalars return errors. |
| `parser.YAML` | `application/x-yaml` | Uses the YAML decoder with `parser.Params` as its destination. |
| `parser.FormMultipart` | `multipart/form-data` | Parses multipart data with a 32 MiB memory threshold and returns ordinary `request.Form` fields, not uploaded files. |
| `parser.Form` | Fallback for unmatched MIME values | Parses URL query and form values, including URL-encoded and multipart forms. One value is a `string`, repeated keys are `[]string`, and empty strings are not changed to nil. |
| `parser.FormPost` | Not separately registered | Reads URL-encoded form values from `request.PostForm` only. Select it explicitly or register it yourself. |
| `parser.XML` | `application/xml`, `text/xml` | The current implementation passes a map destination to the standard XML decoder and is not a working general-purpose XML parameter parser. Implement a struct-oriented parser instead of relying on this entry point. |

### MIME lookup is exact

`GetParser` looks up the entire supplied string. It does not remove parameters such as charset or multipart boundary, or automatically recognize `+json`. Consequently, `application/json; charset=utf-8` does not directly match JSON. An unknown or empty Content-Type falls back to Form; it does not automatically produce 415.

To support MIME parameters, use `mime.ParseMediaType` first and then call `parser.GetParser(mediaType).Parse(request.Object)`. Keep the original request header so the multipart decoder can still read its boundary. If your application permits only particular formats, validate the media type and return 415 before selecting a parser. The complete example below follows this approach.

### Helpers, errors, and repeated reads

- `BodyParamsJSON()` and `BodyParamsYAML()` select their parsers explicitly; `BodyParamsParsing(p)` uses the supplied parser. `BodyParamsXML()` has the XML limitation described above.
- Every `BodyParams*` helper calls `panic(err)` when parsing fails; these helpers do not return an error. This is not an automatic 400 response. Panics not handled by exception middleware reach framework error handling, and an already-written response cannot be replaced in full.
- Call `Parser.Parse` directly and check its error when malformed input should explicitly produce 400. Neither a registered parser nor an argument to `BodyParamsParsing` should be nil.
- JSON/YAML body parsing is not cached. Repeated calls consume the same Body and may reach EOF. Do not read the body first and expect a helper to replay it.
- JSON currently performs one `Decode` call without checking subsequent input or enforcing a body size limit. `EnableDecoderUseNumber` preserves numbers as `json.Number`, which application validators must also support. `EnableDecoderDisallowUnknownFields` cannot provide struct-style unknown-field checking for the current map destination.

### Multipart files

Read ordinary fields separately from files. Use `request.Object.FormFile("file")` or `request.Object.MultipartForm.File` after parsing. Check errors, close opened files, and use `MultipartForm.RemoveAll()` when needed to remove temporary files. Do not trust a client-provided filename as a storage path.

32 MiB is the memory threshold of `ParseMultipartForm`, not a total upload size limit; file data beyond it may be stored on temporary disk. Apply total request limits at your HTTP layer or proxy before reading. The map returned by BodyParams is not a file collection.

## 3. Return values and response writing

Ordinary view values are converted to an HTTP 200 `goi.Response`. Existing `goi.Response` and `*goi.Response` values retain their configuration; a zero Status is not automatically changed to 200. Always set a valid status when constructing an explicit Response.

| `Response.Data` or direct return value | Output |
| --- | --- |
| `string` | Text, with `text/plain` unless explicitly overridden. |
| `[]byte` | Binary data, with `application/octet-stream` by default. |
| Structs, maps, `response.Data`, `response.Result`, etc. | `json.Marshal`, with `application/json` by default. Data uses `code/message/data`; Result uses `code/message/result`. These envelopes are not automatically applied to every value. |
| `*os.File`, `http.File` | `http.ServeContent`, followed by closing the file; missing files and directories produce 404. |
| `goi.FS`, `fs.FS` / `embed.FS` | `http.ServeFileFS`. `goi.FS.Name` specifies a filename; a directly returned filesystem uses the request URL path. |
| `io.Reader` | `io.Copy` with `Response.Status`. A reader that also implements `io.Closer` is closed afterward. |
| `goi.RawHandler` | The callback writes its own status, headers, and body. Goi does not JSON-encode it or automatically set Content-Length. |
| `nil`, nil `*goi.Response` | No body is written. The current `Data == nil` branch also does not call WriteHeader, so the standard library defaults to 200 unless other code writes a status. |

For an empty response with an explicit status, use `goi.Response{Status: http.StatusNoContent, Data: ""}`, or write the status in a RawHandler. Do not rely on `Response{Status: 204, Data: nil}` to send 204.

Add headers through `resp.Header().Set(...)`. Ordinary text, byte, and JSON branches fill Content-Length when absent. File and RawHandler branches have their own output/status behavior; Response.Status does not necessarily override the underlying standard library handler. Returning an error object has no automatic error-response meaning: convert it to a Response yourself.

The actual RawHandler signature is `func(http.ResponseWriter, *goi.Request) error`. Its callback executes during response writing, after route middleware has returned; errors or panics from that callback do not pass through the view's ProcessException chain. See [Middleware](middleware.en.md). A callback can use `http.Flusher`; the framework wrapper's Flush is a no-op when the underlying writer does not support it, so not every streaming feature is guaranteed. The wrapper also does not forward all optional interfaces, such as `http.Hijacker`. Returned errors enter response error handling, but an already-sent status or body cannot be rewritten in full.

## 4. Complete example

The following program demonstrates path/query values, MIME parameters, custom MIME registration, multipart files, and raw responses. Build it in a project that depends on Goi. Running it listens on local port 8080 and creates `server.log` in the current directory. The upload endpoint only reads file information; it does not save files.

```go
package main

import (
	"context"
	"io"
	"mime"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/parser"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

type requestKey struct{}
type vendorJSONParser struct{}

func (vendorJSONParser) Name() string { return "vendor-json" }
func (vendorJSONParser) Parse(r *http.Request) (parser.Params, error) {
	return parser.JSON.Parse(r)
}

func detail(request *goi.Request) any {
	var id int
	if err := request.PathParams.Get("id", &id); err != nil {
		return err.Response()
	}
	var query struct {
		Page int `name:"page" type:"int"`
	}
	if err := request.QueryParams().ParseParams(&query); err != nil {
		return err.Response()
	}
	request.Params.Set("entity_id", id)
	request.WithContext(context.WithValue(request.Object.Context(), requestKey{}, id))
	resp := goi.Response{Status: http.StatusOK, Data: response.Data{
		Code: 200, Message: "ok", Data: map[string]any{"id": id, "page": query.Page},
	}}
	resp.Header().Set("X-Example", "goi")
	return resp
}

func echo(request *goi.Request) any {
	mediaType, _, err := mime.ParseMediaType(request.Object.Header.Get(goi.ContentType))
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	switch mediaType {
	case parser.MIMEJSON, parser.MIMEYAML, "application/vnd.example+json":
	default:
		return goi.Response{Status: http.StatusUnsupportedMediaType, Data: "unsupported content type"}
	}
	params, err := parser.GetParser(mediaType).Parse(request.Object)
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	return goi.Response{Status: http.StatusCreated, Data: response.Result{
		Code: 201, Message: "ok", Result: params,
	}}
}

func upload(request *goi.Request) any {
	if err := request.Object.ParseMultipartForm(32 << 20); err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	if request.Object.MultipartForm != nil {
		defer request.Object.MultipartForm.RemoveAll()
	}
	file, header, err := request.Object.FormFile("file")
	if err != nil {
		return goi.Response{Status: http.StatusBadRequest, Data: err.Error()}
	}
	defer file.Close()
	return response.Data{Code: 200, Message: "ok", Data: map[string]any{
		"name": header.Filename, "size": header.Size,
	}}
}

func stream(request *goi.Request) any {
	return goi.RawHandler(func(w http.ResponseWriter, r *goi.Request) error {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "hello\n"); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	})
}

func main() {
	logger := goi.NewLogger("server.log")
	goi.Log = logger
	defer logger.Close()
	server := goi.NewHTTPServer()
	parser.RegisterParser("application/vnd.example+json", vendorJSONParser{})
	server.Router.Path("items/<int:id>/", "item", goi.ViewSet{GET: detail})
	server.Router.Path("echo/", "echo", goi.ViewSet{POST: echo})
	server.Router.Path("upload/", "upload", goi.ViewSet{POST: upload})
	server.Router.Path("stream/", "stream", goi.ViewSet{GET: stream})
	server.RunServer()
}
```

See [Server lifecycle](lifecycle.en.md) for shutdown and background tasks. Body parsing, parameter validation, and response encoding are separate stages: choosing a parser does not validate business fields, and successful validation does not choose a response envelope.
