package goi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// Validator 参数验证与转换接口。
type Validator interface {
	// Parse 验证输入并转换为可写入目标字段的 Go 值。
	//
	// 参数:
	//   - value any: 待验证和转换的参数值
	//
	// 返回:
	//   - any: 验证通过的值，由 SetValue 写入目标字段；无需转换时返回原值
	//   - ValidationError: 验证或转换失败时返回错误，此时第一个返回值为 nil
	Parse(value any) (any, ValidationError)
}

// validatorMu 保护 validators
var validatorMu sync.RWMutex

// validators 存储验证器映射
// key: 验证器名称
// value: 验证器实例
var validators = map[string]Validator{
	"bool":   &boolValidator{},
	"int":    &intValidator{},
	"string": &stringValidator{},
	"time":   &timeValidator{},
	"slice":  &sliceValidator{},
	"map":    &mapValidator{},
	"slug":   &slugValidator{},
	"uuid":   &uuidValidator{},
}

// RegisterValidator 注册自定义验证器
//
// 参数:
//   - name string: 验证器名称
//   - v Validator: 实现 Parse 的验证器实例
func RegisterValidator(name string, v Validator) {
	validatorMu.Lock()
	defer validatorMu.Unlock()
	validators[name] = v
}

// GetValidator 获取验证器
//
// 参数:
//   - name string: 验证器名称
//
// 返回:
//   - Validator: 验证器实例
//   - bool: 是否存在
func GetValidator(name string) (Validator, bool) {
	validatorMu.RLock()
	defer validatorMu.RUnlock()
	v, ok := validators[name]
	return v, ok
}

// boolValidator 支持 bool，字符串按 strconv.ParseBool 的规则解析。
type boolValidator struct{}

func (validator boolValidator) Parse(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case bool:
		return typeValue, nil
	case string:
		if boolValue, err := strconv.ParseBool(typeValue); err == nil {
			return boolValue, nil
		}
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}

// intValidator 保留数值类型，字符串按 strconv.Atoi 的规则转换为 int。
type intValidator struct{}

func (validator intValidator) Parse(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return typeValue, nil
	case string:
		if val, err := strconv.Atoi(typeValue); err == nil {
			return val, nil
		}
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}

// stringValidator 使用 fmt.Sprint 将输入转换为字符串。
type stringValidator struct{}

func (validator stringValidator) Parse(value any) (any, ValidationError) {
	return fmt.Sprint(value), nil
}

// timeValidator 将支持的日期时间字符串解析为 time.Time。
type timeValidator struct{}

var supportedTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	time.RFC1123,
	time.RFC1123Z,
	time.DateTime,
	time.DateOnly,
}

func parseTime(value string) (time.Time, bool) {
	for _, layout := range supportedTimeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (validator timeValidator) Parse(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		if t, ok := parseTime(typeValue); ok {
			return t, nil
		}
		timeFormatErrorMsg := i18n.T("validator.time_format_error", map[string]any{
			"value": value,
		})
		return nil, NewValidationError(http.StatusBadRequest, timeFormatErrorMsg)
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
		return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// sliceValidator 接受切片及其单层指针，将 JSON 数组字符串解析为 []any。
// JSON null 解析为 nil 切片。
type sliceValidator struct{}

func (validator sliceValidator) Parse(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		var sliceValue []any
		if err := json.Unmarshal([]byte(typeValue), &sliceValue); err != nil {
			return nil, NewValidationError(http.StatusBadRequest, err.Error())
		}
		return sliceValue, nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType != nil {
			if valueType.Kind() == reflect.Ptr {
				valueType = valueType.Elem()
			}
			if valueType.Kind() == reflect.Slice {
				return value, nil
			}
		}
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}

// mapValidator 接受 map 及其单层指针，将 JSON 对象字符串解析为 map[string]any。
// JSON null 解析为 nil map。
type mapValidator struct{}

func (validator mapValidator) Parse(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		var mapValue map[string]any
		if err := json.Unmarshal([]byte(typeValue), &mapValue); err != nil {
			return nil, NewValidationError(http.StatusBadRequest, err.Error())
		}
		return mapValue, nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType != nil {
			if valueType.Kind() == reflect.Ptr {
				valueType = valueType.Elem()
			}
			if valueType.Kind() == reflect.Map {
				return value, nil
			}
		}
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}

// slugValidator 完整匹配 ASCII 字母、数字、下划线和连字符组成的非空字符串。
type slugValidator struct{}

var slugValidatorRe = regexp.MustCompile(`^([-a-zA-Z0-9_]+)$`)

func (validator slugValidator) Parse(value any) (any, ValidationError) {
	if typeValue, ok := value.(string); ok && slugValidatorRe.MatchString(typeValue) {
		return typeValue, nil
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}

// uuidValidator 完整匹配小写十六进制的 8-4-4-4-12 格式，不校验版本或变体位。
type uuidValidator struct{}

var uuidValidatorRe = regexp.MustCompile(`^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

func (validator uuidValidator) Parse(value any) (any, ValidationError) {
	if typeValue, ok := value.(string); ok && uuidValidatorRe.MatchString(typeValue) {
		return typeValue, nil
	}
	paramsErrorMsg := i18n.T("validator.params_error", map[string]any{"value": value})
	return nil, NewValidationError(http.StatusBadRequest, paramsErrorMsg)
}
