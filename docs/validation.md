# 参数验证与类型转换

**简体中文** | [English](validation.en.md) · [文档目录](README.md) · [项目主页](../README.md)

`Params.ParseParams` 按结构体标签选择验证器，每个非 nil 参数只调用一次 `Validator.Parse`，再通过 `SetValue` 写入字段。`Params.Get` 只按键读取并赋值，不调用注册的验证器。

## 基本用法

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

`ParseParams` 接收结构体指针，字段必须可写。字段按声明顺序处理；后续字段失败时，之前的赋值不会回滚。

## 字段标签

| 标签 | 规则 |
| --- | --- |
| `name` | 参数名；没有时依次使用 `json` 标签、Go 字段名的小写形式。标签中逗号后的选项不参与名称匹配；`-` 跳过字段。 |
| `type` | 验证器注册名；未设置、空字符串或 `-` 时跳过验证和赋值，不自动推断类型。 |
| `required:"true"` | 参数缺失时返回验证错误；参数为 nil 时，除非显式设置 `allow_null:"true"`，否则也返回错误。 |
| `allow_null` | `true` 允许 nil，`false` 拒绝 nil；未设置时，非必填字段允许 nil。允许 nil 时跳过赋值，不清空原字段，也不调用验证器。 |

`allow_null` 的前置判断是 `value == nil`。带动态类型的 nil 指针、nil 切片、nil map，以及字符串 `"null"` 都不满足这个判断，仍会进入对应验证器。

## 内置验证器

| `type` | 输入与转换规则 |
| --- | --- |
| `bool` | bool 原样返回；字符串使用 `strconv.ParseBool`，接受 `1/t/T/true/TRUE/True` 和 `0/f/F/false/FALSE/False`。不额外转换大小写或去除空白，`tRuE` 无效。 |
| `int` | 内置整数、无符号整数及浮点数原样返回；字符串使用 `strconv.Atoi` 转为 int，支持正负号，格式错误或超出当前平台 int 范围时返回验证错误。 |
| `string` | 使用 `fmt.Sprint` 转成字符串。切片、map 使用 Go 文本格式，不是 JSON；直接调用时，nil 得到 `"<nil>"`。 |
| `time` | 接收字符串，依次按 `time.RFC3339Nano`、`RFC3339`、`RFC1123`、`RFC1123Z`、`DateTime`、`DateOnly` 解析，返回 `time.Time`。 |
| `slice` | 接收切片及其单层指针，拒绝原生数组和数组指针；字符串直接通过 `json.Unmarshal` 解析为 `[]any`。 |
| `map` | 接收 map 及其单层指针；字符串直接通过 `json.Unmarshal` 解析为 `map[string]any`。 |
| `slug` | 完整匹配非空字符串，只允许 ASCII 字母、数字、下划线和连字符；拒绝尾随路径、空格、换行等内容。 |
| `uuid` | 完整匹配小写十六进制的 `8-4-4-4-12` 格式；不接受大写或额外字符，不校验 UUID 版本和变体位。 |

`int` 验证器不会拒绝原生浮点数，也不会把所有数值统一转换成 int；最终字段赋值由 `SetValue` 完成。需要更严格的整数、范围或业务规则时，可注册自定义验证器。

### JSON 集合与 null

`slice`、`map` 不再使用正则预检查：有效的外层空白和多行 JSON 均可解析，错误 JSON 或不匹配的 JSON 类型返回验证错误。JSON 数字在 `[]any`、`map[string]any` 中按 `encoding/json` 的默认规则解析为 float64。

字符串 `"null"` 可以成功解析，分别得到带类型的 nil 切片或 nil map。它不等同于原生 nil 参数，也不会触发前面的 `allow_null` 判断。最终字段是否保持 nil 取决于 `SetValue` 和目标类型，不能用 JSON `null` 保证清空字段；不同集合类型间赋值时，可能得到空集合而非 nil。

## 自定义验证器

只需实现一个方法，在其中完成验证和转换；失败返回 `nil, ValidationError`。成功值应能由 `SetValue` 写入目标字段。

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

通常在配置初始化时或 `RunServer` 前注册。同名注册会替换旧验证器；`GetValidator(name)` 可获取验证器直接调用 `Parse`。

参数验证器不再需要分别实现 `Validate` 和 `ToGo`。路由路径转换器的 `Converter.ToGo` 是独立接口，不受影响。

## 错误与响应定制

内置验证器解析失败，以及必填、非空或未知验证器错误，默认返回状态码 400。目标类型或赋值失败通常返回 500；`ParseParams` 原样返回自定义 `Parse` 产生的 `ValidationError`。

实现 `ValidationError` 的 `NewValidationError`、`Error` 和 `Response` 方法，再通过 `Server.Validation.SetValidationError` 设置错误处理器，可以定制消息和最终响应格式。应在处理请求前完成配置；CLI 生成的项目包含完整示例。

## SetValue 与数组

`SetValue` 可单独用于赋值。它保留已有的直接赋值、Go 类型转换和 `AnyUnmarshaler` 扩展入口；通用数值转换不等同于完整的范围或业务校验。

数组目标支持切片或数组来源，必要时递归转换每个元素，例如：

```go
var result [2]int
err := goi.SetValue(reflect.ValueOf(&result), []any{1, "2"})
// err == nil，result == [2]int{1, 2}
```

此片段需导入 `reflect` 和 Goi。调用方必须保证源与目标数组长度一致；框架不额外检查长度或捕获 panic。长度不足可能 panic，长度超出可能只取前 N 项，不要依赖不等长行为。元素赋值失败时，已经写入的元素不会回滚。

这不改变 `type:"slice"` 的输入规则：它仍拒绝原生数组；直接调用 `SetValue` 和参数验证是两个不同入口。

## 源码入口

| 文件 | 职责 |
| --- | --- |
| [params.go](../params.go) | 参数读取、标签处理以及验证与赋值流程 |
| [validator.go](../validator.go) | `Validator.Parse` 接口、注册表与内置验证器 |
| [validation.go](../validation.go) | 验证错误接口、默认实现和错误处理器配置 |
| [value.go](../value.go) | 反射赋值、类型转换与数组逐元素写入 |
| [validator_test.go](../validator_test.go)、[value_test.go](../value_test.go) | 自定义验证器示例与回归测试 |
