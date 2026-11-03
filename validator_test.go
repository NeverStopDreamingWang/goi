package goi_test

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
)

// phoneValidator 手机号验证器示例
type phoneValidator struct{}

// Parse 验证手机号，并将有效值转换为字符串。
func (validator phoneValidator) Parse(value any) (any, goi.ValidationError) {
	var valueStr string
	switch typeValue := value.(type) {
	case int:
		valueStr = strconv.Itoa(typeValue)
	case string:
		valueStr = typeValue
	default:
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
	re := regexp.MustCompile(`^(1[3456789]\d{9})$`)
	if !re.MatchString(valueStr) {
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return valueStr, nil
}

// emailValidator 邮箱验证器示例
type emailValidator struct{}

// Parse 验证邮箱，并返回字段需要的自定义类型。
func (validator emailValidator) Parse(value any) (any, goi.ValidationError) {
	switch v := value.(type) {
	case string:
		re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !re.MatchString(v) {
			return nil, goi.NewValidationError(http.StatusBadRequest, "invalid email format")
		}
		return Email{Address: v}, nil
	default:
		return nil, goi.NewValidationError(http.StatusBadRequest, "email must be string")
	}
}

type CustomValidator struct{}

// Parse 验证格式，并将有效字符串转换为 Custom。
func (validator CustomValidator) Parse(value any) (any, goi.ValidationError) {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\d{16})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
		return Custom(typeValue), nil
	default:
		return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
}

// 支持自定义类型
type Email struct {
	Address string `json:"address"`
	// 其它自定义字段
}

type Custom []byte

type testParamsValidParams struct {
	Phone  string `name:"phone" type:"phone" required:"true"`
	Email  *Email `name:"email" type:"email" required:"true"`
	Custom Custom `name:"custom" type:"custom" required:"true"`
}

// ExampleRegisterValidator 展示如何注册和使用自定义验证器
func ExampleRegisterValidator() {
	// 注册手机号验证器
	goi.RegisterValidator("phone", phoneValidator{})

	// 注册邮箱验证器
	goi.RegisterValidator("email", emailValidator{})

	// 注册自定义验证器
	goi.RegisterValidator("custom", CustomValidator{})

	var validationErr goi.ValidationError

	var params testParamsValidParams
	bodyParams := goi.Params{
		"phone":  "13800000000",
		"email":  "test@example.com",
		"custom": "1234567890123456",
	}
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return
	}
	fmt.Printf("Phone %+v %T\n", params.Phone, params.Phone)
	fmt.Printf("Email %+v %T\n", params.Email, params.Email)
	fmt.Printf("Custom %+v %T\n", params.Custom, params.Custom)

	// Output:
	// Phone 13800000000 string
	// Email &{Address:test@example.com} *goi_test.Email
	// Custom [49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54] goi_test.Custom
}

func TestBuiltinValidatorParse(t *testing.T) {
	sliceValue := []int{1, 2}
	arrayValue := [2]int{1, 2}
	mapValue := map[string]int{"count": 2}
	var nilSlice []int
	var nilMap map[string]int
	var nilSlicePointer *[]int
	var nilMapPointer *map[string]int

	type parseTest struct {
		name      string
		validator string
		input     any
		want      any
		wantError bool
	}
	tests := []parseTest{
		{name: "bool/native", validator: "bool", input: true, want: true},
		{name: "bool/true", validator: "bool", input: "true", want: true},
		{name: "bool/false", validator: "bool", input: "false", want: false},
		{name: "bool/uppercase_true", validator: "bool", input: "TRUE", want: true},
		{name: "bool/uppercase_false", validator: "bool", input: "FALSE", want: false},
		{name: "bool/title_case", validator: "bool", input: "True", want: true},
		{name: "bool/title_case_false", validator: "bool", input: "False", want: false},
		{name: "bool/mixed_case_false", validator: "bool", input: "fAlSe", wantError: true},
		{name: "bool/mixed_case_true", validator: "bool", input: "tRuE", wantError: true},
		{name: "bool/string_one", validator: "bool", input: "1", want: true},
		{name: "bool/string_zero", validator: "bool", input: "0", want: false},
		{name: "bool/lowercase_t", validator: "bool", input: "t", want: true},
		{name: "bool/lowercase_f", validator: "bool", input: "f", want: false},
		{name: "bool/uppercase_t", validator: "bool", input: "T", want: true},
		{name: "bool/uppercase_f", validator: "bool", input: "F", want: false},
		{name: "bool/empty", validator: "bool", input: "", wantError: true},
		{name: "bool/leading_whitespace", validator: "bool", input: " true", wantError: true},
		{name: "bool/trailing_whitespace", validator: "bool", input: "false ", wantError: true},
		{name: "bool/numeric", validator: "bool", input: 1, wantError: true},
		{name: "bool/invalid", validator: "bool", input: "yes", wantError: true},
		{name: "bool/nil", validator: "bool", wantError: true},
		{name: "int/string", validator: "int", input: "123", want: 123},
		{name: "int/leading_zero", validator: "int", input: "001", want: 1},
		{name: "int/negative_string", validator: "int", input: "-1", want: -1},
		{name: "int/positive_sign", validator: "int", input: "+1", want: 1},
		{name: "int/decimal_string", validator: "int", input: "1.5", wantError: true},
		{name: "int/empty", validator: "int", input: "", wantError: true},
		{name: "int/larger_than_int", validator: "int", input: "18446744073709551615", wantError: true},
		{name: "int/native_uint64_max", validator: "int", input: ^uint64(0), want: ^uint64(0)},
		{name: "int/bool", validator: "int", input: true, wantError: true},
		{name: "int/nil", validator: "int", wantError: true},
		{name: "string/native", validator: "string", input: "hello", want: "hello"},
		{name: "string/empty", validator: "string", input: "", want: ""},
		{name: "string/non_string", validator: "string", input: 123, want: "123"},
		{name: "string/bool", validator: "string", input: true, want: "true"},
		{name: "string/slice", validator: "string", input: []int{1, 2}, want: "[1 2]"},
		{name: "string/nil", validator: "string", want: "<nil>"},
		{name: "time/invalid", validator: "time", input: "2026-13-40", wantError: true},
		{name: "time/native_rejected", validator: "time", input: time.Time{}, wantError: true},
		{name: "time/nil", validator: "time", wantError: true},
		{name: "slice/native", validator: "slice", input: sliceValue, want: sliceValue},
		{name: "slice/array", validator: "slice", input: arrayValue, wantError: true},
		{name: "slice/pointer", validator: "slice", input: &sliceValue, want: &sliceValue},
		{name: "slice/array_pointer", validator: "slice", input: &arrayValue, wantError: true},
		{name: "slice/typed_nil", validator: "slice", input: nilSlice, want: nilSlice},
		{name: "slice/typed_nil_pointer", validator: "slice", input: nilSlicePointer, want: nilSlicePointer},
		{name: "slice/json", validator: "slice", input: `[1,"two"]`, want: []any{float64(1), "two"}},
		{name: "slice/empty_json", validator: "slice", input: `[]`, want: []any{}},
		{name: "slice/invalid_json", validator: "slice", input: `[1,]`, wantError: true},
		{name: "slice/json_object", validator: "slice", input: `{}`, wantError: true},
		{name: "slice/json_null", validator: "slice", input: `null`, want: []any(nil)},
		{name: "slice/outer_whitespace", validator: "slice", input: ` [1] `, want: []any{float64(1)}},
		{name: "slice/multiline", validator: "slice", input: "[\n1\n]", want: []any{float64(1)}},
		{name: "slice/json_bool", validator: "slice", input: `true`, wantError: true},
		{name: "slice/json_number", validator: "slice", input: `123`, wantError: true},
		{name: "slice/wrong_type", validator: "slice", input: 1, wantError: true},
		{name: "slice/nil", validator: "slice", wantError: true},
		{name: "map/native", validator: "map", input: mapValue, want: mapValue},
		{name: "map/pointer", validator: "map", input: &mapValue, want: &mapValue},
		{name: "map/typed_nil", validator: "map", input: nilMap, want: nilMap},
		{name: "map/typed_nil_pointer", validator: "map", input: nilMapPointer, want: nilMapPointer},
		{name: "map/json", validator: "map", input: `{"count":2}`, want: map[string]any{"count": float64(2)}},
		{name: "map/empty_json", validator: "map", input: `{}`, want: map[string]any{}},
		{name: "map/invalid_json", validator: "map", input: `{"count":}`, wantError: true},
		{name: "map/json_array", validator: "map", input: `[]`, wantError: true},
		{name: "map/json_null", validator: "map", input: `null`, want: map[string]any(nil)},
		{name: "map/outer_whitespace", validator: "map", input: ` {"count":2} `, want: map[string]any{"count": float64(2)}},
		{name: "map/multiline", validator: "map", input: "{\n\"count\":2\n}", want: map[string]any{"count": float64(2)}},
		{name: "map/json_bool", validator: "map", input: `true`, wantError: true},
		{name: "map/json_number", validator: "map", input: `123`, wantError: true},
		{name: "map/wrong_type", validator: "map", input: []int{1}, wantError: true},
		{name: "map/nil", validator: "map", wantError: true},
		{name: "slug/native", validator: "slug", input: "hello-world_123", want: "hello-world_123"},
		{name: "slug/trailing_path", validator: "slug", input: "hello/world", wantError: true},
		{name: "slug/trailing_newline", validator: "slug", input: "hello\n", wantError: true},
		{name: "slug/trailing_space", validator: "slug", input: "hello ", wantError: true},
		{name: "slug/trailing_punctuation", validator: "slug", input: "hello!", wantError: true},
		{name: "slug/invalid", validator: "slug", input: "/hello", wantError: true},
		{name: "slug/empty", validator: "slug", input: "", wantError: true},
		{name: "slug/non_string", validator: "slug", input: 123, wantError: true},
		{name: "slug/nil", validator: "slug", wantError: true},
		{name: "uuid/native", validator: "uuid", input: "123e4567-e89b-12d3-a456-426614174000", want: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "uuid/trailing_path", validator: "uuid", input: "123e4567-e89b-12d3-a456-426614174000/extra", wantError: true},
		{name: "uuid/trailing_newline", validator: "uuid", input: "123e4567-e89b-12d3-a456-426614174000\n", wantError: true},
		{name: "uuid/trailing_space", validator: "uuid", input: "123e4567-e89b-12d3-a456-426614174000 ", wantError: true},
		{name: "uuid/trailing_character", validator: "uuid", input: "123e4567-e89b-12d3-a456-426614174000a", wantError: true},
		{name: "uuid/uppercase", validator: "uuid", input: "123E4567-E89B-12D3-A456-426614174000", wantError: true},
		{name: "uuid/invalid", validator: "uuid", input: "invalid", wantError: true},
		{name: "uuid/non_string", validator: "uuid", input: 123, wantError: true},
		{name: "uuid/nil", validator: "uuid", wantError: true},
	}
	for _, input := range []any{int(-1), int8(1), int16(2), int32(3), int64(4), uint(5), uint8(6), uint16(7), uint32(8), uint64(9), float32(1.5), float64(2.5)} {
		tests = append(tests, parseTest{name: "int/" + reflect.TypeOf(input).String(), validator: "int", input: input, want: input})
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, time.RFC1123, time.RFC1123Z, time.DateTime, time.DateOnly} {
		input := time.Date(2026, time.November, 3, 12, 34, 56, 123456789, time.UTC).Format(layout)
		want, err := time.Parse(layout, input)
		if err != nil {
			t.Fatal(err)
		}
		tests = append(tests, parseTest{name: "time/" + layout, validator: "time", input: input, want: want})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, ok := goi.GetValidator(test.validator)
			if !ok {
				t.Fatalf("validator %q not registered", test.validator)
			}
			got, err := validator.Parse(test.input)
			if test.wantError {
				if err == nil {
					t.Fatalf("Parse(%#v) = %#v, expected validation error", test.input, got)
				}
				if got != nil || err.Response().Status != http.StatusBadRequest {
					t.Fatalf("Parse(%#v) = %#v, status %d; want nil, status 400", test.input, got, err.Response().Status)
				}
				return
			}
			if wantTime, ok := test.want.(time.Time); ok {
				gotTime, isTime := got.(time.Time)
				if err != nil || !isTime || !gotTime.Equal(wantTime) {
					t.Fatalf("Parse(%#v) = %#v, %v; want time %v", test.input, got, err, wantTime)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Parse(%#v) = %#v, %v; want %#v", test.input, got, err, test.want)
			}
		})
	}
}

func TestParseParamsUsesParsedValues(t *testing.T) {
	var params struct {
		Enabled bool           `name:"enabled" type:"bool" required:"true"`
		Count   int            `name:"count" type:"int" required:"true"`
		Name    string         `name:"name" type:"string" required:"true"`
		Time    time.Time      `name:"time" type:"time" required:"true"`
		Items   []int          `name:"items" type:"slice" required:"true"`
		Meta    map[string]any `name:"meta" type:"map" required:"true"`
		Slug    string         `name:"slug" type:"slug" required:"true"`
		UUID    string         `name:"uuid" type:"uuid" required:"true"`
	}
	values := goi.Params{
		"enabled": "true",
		"count":   "12",
		"name":    "demo",
		"time":    "2026-11-03",
		"items":   `[1,2]`,
		"meta":    `{"active":true}`,
		"slug":    "demo-slug",
		"uuid":    "123e4567-e89b-12d3-a456-426614174000",
	}
	if err := values.ParseParams(&params); err != nil {
		t.Fatal(err)
	}
	if !params.Enabled || params.Count != 12 || params.Name != "demo" ||
		!params.Time.Equal(time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC)) ||
		!reflect.DeepEqual(params.Items, []int{1, 2}) ||
		!reflect.DeepEqual(params.Meta, map[string]any{"active": true}) ||
		params.Slug != values["slug"] || params.UUID != values["uuid"] {
		t.Fatalf("unexpected parsed params: %+v", params)
	}
}

func TestParseParamsRejectsInvalidInput(t *testing.T) {
	var params struct {
		Enabled bool `name:"enabled" type:"bool" required:"true"`
	}
	err := (goi.Params{"enabled": "invalid"}).ParseParams(&params)
	if err == nil || err.Response().Status != http.StatusBadRequest {
		t.Fatalf("invalid boolean error = %v; want status 400", err)
	}
}

func TestParseParamsRejectsArrayForSlice(t *testing.T) {
	var params struct {
		Items []int `name:"items" type:"slice" required:"true"`
	}
	err := (goi.Params{"items": [2]int{1, 2}}).ParseParams(&params)
	if err == nil || err.Response().Status != http.StatusBadRequest {
		t.Fatalf("array input error = %v; want status 400", err)
	}
	if params.Items != nil {
		t.Fatalf("parsed items = %#v; want unchanged nil slice", params.Items)
	}
}

func TestParseParamsConvertsNumberToString(t *testing.T) {
	var params struct {
		Value string `name:"value" type:"string" required:"true"`
	}
	if err := (goi.Params{"value": 123}).ParseParams(&params); err != nil {
		t.Fatal(err)
	}
	if params.Value != "123" {
		t.Fatalf("parsed value = %q; want 123", params.Value)
	}
}

func TestParseParamsRejectsStringOutsideIntRange(t *testing.T) {
	var params struct {
		Value uint64 `name:"value" type:"int" required:"true"`
	}
	err := (goi.Params{"value": "18446744073709551615"}).ParseParams(&params)
	if err == nil || err.Response().Status != http.StatusBadRequest {
		t.Fatalf("out-of-range integer error = %v; want status 400", err)
	}
	if params.Value != 0 {
		t.Fatalf("parsed value = %d; want unchanged zero value", params.Value)
	}
}

type validatorParseFunc func(any) (any, goi.ValidationError)

func (parse validatorParseFunc) Parse(value any) (any, goi.ValidationError) {
	return parse(value)
}

func TestParseParamsCallsCustomParserOnce(t *testing.T) {
	original, ok := goi.GetValidator("string")
	if !ok {
		t.Fatal("string validator not registered")
	}
	t.Cleanup(func() { goi.RegisterValidator("string", original) })
	var calls int
	goi.RegisterValidator("string", validatorParseFunc(func(value any) (any, goi.ValidationError) {
		calls++
		if value != "raw" {
			t.Fatalf("parser input = %#v; want raw", value)
		}
		return "parsed", nil
	}))
	var params struct {
		Value string `name:"value" type:"string"`
	}
	if err := (goi.Params{"value": "raw"}).ParseParams(&params); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || params.Value != "parsed" {
		t.Fatalf("calls = %d, value = %q; want 1, parsed", calls, params.Value)
	}

	sentinel := goi.NewValidationError(http.StatusUnprocessableEntity, "custom validation failed")
	calls = 0
	goi.RegisterValidator("string", validatorParseFunc(func(any) (any, goi.ValidationError) {
		calls++
		return nil, sentinel
	}))
	err := (goi.Params{"value": "raw"}).ParseParams(&params)
	if calls != 1 || err != sentinel || params.Value != "parsed" {
		t.Fatalf("calls = %d, error = %v, value = %q; want one call, original error, unchanged value", calls, err, params.Value)
	}
}

func TestParseParamsRequiredAndNullRules(t *testing.T) {
	original, ok := goi.GetValidator("string")
	if !ok {
		t.Fatal("string validator not registered")
	}
	t.Cleanup(func() { goi.RegisterValidator("string", original) })
	var calls int
	goi.RegisterValidator("string", validatorParseFunc(func(value any) (any, goi.ValidationError) {
		calls++
		return fmt.Sprint(value), nil
	}))
	for _, test := range []struct {
		name      string
		tags      string
		values    goi.Params
		want      string
		wantError bool
		wantCalls int
	}{
		{name: "required_missing", tags: `required:"true"`, values: goi.Params{}, want: "unchanged", wantError: true},
		{name: "optional_missing", values: goi.Params{}, want: "unchanged"},
		{name: "required_null", tags: `required:"true"`, values: goi.Params{"value": nil}, want: "unchanged", wantError: true},
		{name: "required_null_allowed", tags: `required:"true" allow_null:"true"`, values: goi.Params{"value": nil}, want: "unchanged"},
		{name: "optional_null", values: goi.Params{"value": nil}, want: "unchanged"},
		{name: "optional_null_disallowed", tags: `allow_null:"false"`, values: goi.Params{"value": nil}, want: "unchanged", wantError: true},
		{name: "required_empty_string", tags: `required:"true"`, values: goi.Params{"value": ""}, want: "", wantCalls: 1},
		{name: "optional_number", values: goi.Params{"value": 12}, want: "12", wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls = 0
			paramsType := reflect.StructOf([]reflect.StructField{{
				Name: "Value", Type: reflect.TypeFor[string](),
				Tag: reflect.StructTag(`name:"value" type:"string" ` + test.tags),
			}})
			params := reflect.New(paramsType)
			params.Elem().Field(0).SetString("unchanged")
			err := test.values.ParseParams(params.Interface())
			if test.wantError {
				if err == nil || err.Response().Status != http.StatusBadRequest {
					t.Fatalf("ParseParams error = %v; want status 400", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if got := params.Elem().Field(0).String(); got != test.want || calls != test.wantCalls {
				t.Fatalf("value = %q, Parse calls = %d; want %q, %d", got, calls, test.want, test.wantCalls)
			}
		})
	}
}

func TestDefaultValidationError(t *testing.T) {
	original := goi.Validation
	t.Cleanup(func() { goi.Validation = original })
	for _, fallback := range []bool{false, true} {
		if fallback {
			goi.Validation = nil
		} else {
			goi.Validation = original
		}
		err := goi.NewValidationError(http.StatusBadRequest, "invalid parameter")
		response := err.Response()
		if err.Error() != "invalid parameter" || response.Status != http.StatusBadRequest || response.Data != "invalid parameter" {
			t.Fatalf("fallback=%t: error=%v, response=%+v", fallback, err, response)
		}
	}
}

type recordingValidationError struct {
	status  int
	message string
	args    []any
	calls   int
}

func (handler *recordingValidationError) NewValidationError(status int, message string, args ...any) goi.ValidationError {
	handler.calls++
	handler.status, handler.message, handler.args = status, message, args
	return &recordingValidationError{status: status, message: message, args: args}
}

func (validationErr *recordingValidationError) Error() string {
	return validationErr.message
}

func (validationErr *recordingValidationError) Response() goi.Response {
	return goi.Response{Status: http.StatusOK, Data: map[string]any{
		"code": validationErr.status, "message": validationErr.message,
	}}
}

func TestCustomValidationErrorHandler(t *testing.T) {
	original := goi.Validation
	if original == nil {
		t.Fatal("validation manager not initialized")
	}
	manager := *original
	goi.Validation = &manager
	t.Cleanup(func() { goi.Validation = original })
	handler := &recordingValidationError{}
	goi.Validation.SetValidationError(handler)

	err := goi.NewValidationError(http.StatusUnprocessableEntity, "custom message", "first", 42)
	if handler.calls != 1 || handler.status != http.StatusUnprocessableEntity ||
		handler.message != "custom message" || !reflect.DeepEqual(handler.args, []any{"first", 42}) {
		t.Fatalf("arguments not forwarded to custom handler: %+v", handler)
	}
	if _, ok := err.(*recordingValidationError); !ok {
		t.Fatalf("error type = %T; want custom error", err)
	}
	response := err.Response()
	if err.Error() != "custom message" || response.Status != http.StatusOK ||
		!reflect.DeepEqual(response.Data, map[string]any{"code": http.StatusUnprocessableEntity, "message": "custom message"}) {
		t.Fatalf("custom error response = %+v", response)
	}

	validator, _ := goi.GetValidator("bool")
	_, err = validator.Parse("invalid")
	if _, ok := err.(*recordingValidationError); !ok || handler.calls != 2 || handler.status != http.StatusBadRequest {
		t.Fatalf("validator did not use custom error handler: error=%v, handler=%+v", err, handler)
	}
	var params struct {
		Value string `name:"value" type:"string" required:"true"`
	}
	err = (goi.Params{}).ParseParams(&params)
	if _, ok := err.(*recordingValidationError); !ok || handler.calls != 3 || handler.status != http.StatusBadRequest {
		t.Fatalf("required parameter error did not use custom handler: error=%v, handler=%+v", err, handler)
	}
}
