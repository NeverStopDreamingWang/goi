# Parameter Validation and Type Conversion

[简体中文](validation.md) | **English** · [Project Home](../README.en.md)

`Params.ParseParams` selects validators through struct tags, calls `Validator.Parse` once for each non-nil parameter, and assigns the result through `SetValue`. `Params.Get` only retrieves and assigns a value; it does not invoke registered validators.

## Basic Usage

```go
package main

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func main() {
	var params struct {
		Enabled bool           `name:"enabled" type:"bool" required:"true"`
		Page    int            `name:"page" type:"int"`
		Keyword string         `name:"keyword" type:"string"`
		IDs     []int          `name:"ids" type:"slice"`
		Meta    map[string]any `name:"meta" type:"map"`
	}
	values := goi.Params{
		"enabled": "TRUE",
		"page":    "-2",
		"keyword": 123,
		"ids":     `[1, 2]`,
		"meta":    `{"active": true}`,
	}
	if err := values.ParseParams(&params); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(params.Enabled, params.Page, params.Keyword, params.IDs, params.Meta["active"])
	// true -2 123 [1 2] true
}
```

`ParseParams` requires a struct pointer with writable fields. Fields are processed in declaration order; an error in a later field does not roll back earlier assignments.

## Field Tags

| Tag | Rule |
| --- | --- |
| `name` | Parameter name; falls back to the `json` tag, then the lowercase Go field name. Options after a comma are ignored when matching names. `-` skips the field. |
| `type` | Registered validator name. An absent, empty, or `-` value skips validation and assignment; the type is not inferred automatically. |
| `required:"true"` | A missing parameter is an error. A nil parameter is also an error unless `allow_null:"true"` is explicitly set. |
| `allow_null` | `true` allows nil; `false` rejects it. If unset, optional fields allow nil. Allowed nil values skip assignment without clearing the existing field or invoking the validator. |

The early `allow_null` check is `value == nil`. Typed nil pointers, slices, and maps, as well as the string `"null"`, do not satisfy this check and still enter the selected validator.

## Built-in Validators

| `type` | Accepted input and conversion |
| --- | --- |
| `bool` | Returns bool inputs unchanged. Strings use `strconv.ParseBool`, accepting `1/t/T/true/TRUE/True` and `0/f/F/false/FALSE/False`. No additional case folding or whitespace trimming is applied; `tRuE` is invalid. |
| `int` | Returns built-in signed integers, unsigned integers, and floating-point values unchanged. Strings use `strconv.Atoi`, including an optional sign. Invalid strings and values outside the platform's int range produce validation errors. |
| `string` | Uses `fmt.Sprint`. Slices and maps use Go's text representation, not JSON. A direct call with nil returns `"<nil>"`. |
| `time` | Accepts strings and tries `time.RFC3339Nano`, `RFC3339`, `RFC1123`, `RFC1123Z`, `DateTime`, and `DateOnly` in that order, returning `time.Time`. |
| `slice` | Accepts slices and single-level pointers to slices; rejects native arrays and array pointers. Strings are unmarshaled directly into `[]any`. |
| `map` | Accepts maps and single-level pointers to maps. Strings are unmarshaled directly into `map[string]any`. |
| `slug` | Matches the entire nonempty string, allowing only ASCII letters, digits, underscores, and hyphens. Trailing paths, spaces, and newlines are rejected. |
| `uuid` | Matches the entire lowercase hexadecimal `8-4-4-4-12` format. Uppercase and extra characters are rejected. UUID version and variant bits are not checked. |

The `int` validator does not reject native floating-point values or normalize every numeric input to int. `SetValue` performs the final assignment. Register a custom validator when stricter integer, range, or business rules are required.

### JSON Collections and null

`slice` and `map` do not use regular expressions before parsing. Valid surrounding whitespace and multiline JSON are accepted. Invalid JSON or an incompatible JSON type produces a validation error. JSON numbers inside `[]any` and `map[string]any` become float64 values under the default `encoding/json` rules.

The string `"null"` parses successfully into a typed nil slice or map. It is not a native nil parameter and does not trigger the early `allow_null` check. The final field value depends on `SetValue` and the destination type; JSON `null` does not guarantee that a field is cleared. Assignment between different collection types may produce an empty collection rather than nil.

## Custom Validators

Implement one method to validate and convert the input. Return `nil, ValidationError` on failure. A successful value must be assignable through `SetValue` to the destination field.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
)

type Status string
type statusValidator struct{}

func (statusValidator) Parse(value any) (any, goi.ValidationError) {
	status, ok := value.(string)
	if !ok || (status != "pending" && status != "done") {
		return nil, goi.NewValidationError(http.StatusBadRequest, "invalid status")
	}
	return Status(status), nil
}

func main() {
	goi.RegisterValidator("status", statusValidator{})
	var params struct {
		Status Status `name:"status" type:"status" required:"true"`
	}
	if err := (goi.Params{"status": "pending"}).ParseParams(&params); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s (%T)\n", params.Status, params.Status)
	// pending (main.Status)
}
```

Register validators during configuration initialization or before `RunServer`. Registering the same name replaces its previous validator. Use `GetValidator(name)` to obtain a validator and call `Parse` directly.

Parameter validators no longer need separate `Validate` and `ToGo` methods. The path converter's `Converter.ToGo` belongs to a separate API and is unchanged.

## Errors and Custom Responses

Built-in parsing failures, missing required values, disallowed nil values, and unknown validators use status 400 by default. Invalid destinations or assignment failures generally use status 500. `ParseParams` returns a custom parser's `ValidationError` unchanged.

Implement `NewValidationError`, `Error`, and `Response` from `ValidationError`, then configure the handler with `Server.Validation.SetValidationError` to customize messages and the final response format. Configure it before serving requests. CLI-generated projects include a complete example.

## SetValue and Arrays

`SetValue` can be called independently. It retains direct assignment, Go type conversion, and the `AnyUnmarshaler` extension point. General numeric conversion is not complete range or business validation.

Array destinations accept slices or arrays, recursively converting elements when necessary:

```go
var result [2]int
err := goi.SetValue(reflect.ValueOf(&result), []any{1, "2"})
// err == nil, result == [2]int{1, 2}
```

This fragment requires imports for `reflect` and Goi. Callers must ensure matching source and destination lengths. The framework does not add length checks or recover panics. Insufficient input may panic; longer input may use only the first N elements. Do not rely on mismatched-length behavior. Element errors do not roll back earlier writes.

This does not change `type:"slice"`: it still rejects native arrays. Direct assignment through `SetValue` and parameter validation are separate entry points.

## Source Files

| File | Responsibility |
| --- | --- |
| [params.go](../params.go) | Parameter lookup, tags, validation, and assignment flow |
| [validator.go](../validator.go) | `Validator.Parse`, registry, and built-in validators |
| [validation.go](../validation.go) | Validation error interface, default implementation, and handler configuration |
| [value.go](../value.go) | Reflection-based assignment, conversions, and array element writes |
| [validator_test.go](../validator_test.go), [value_test.go](../value_test.go) | Custom validator examples and regression tests |
