package goi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// ValidationError 参数验证错误处理器接口
//
// 方法:
//   - NewValidationError: 创建参数验证错误
//   - Error: 实现 error 接口，返回错误消息
//   - Response: 生成错误响应
type ValidationError interface {
	// NewValidationError 创建参数验证错误
	//
	// 参数:
	//   - status int: HTTP状态码
	//   - message string: 错误消息
	//   - args ...any: 可选参数
	//
	// 返回:
	//   - ValidationError: 验证错误实例
	NewValidationError(status int, message string, args ...any) ValidationError

	// Error 返回错误消息
	Error() string

	// Response 生成错误响应
	//
	// 返回:
	//   - Response: 标准响应格式
	Response() Response
}

// NewValidationError 创建全局参数验证错误
//
// 参数:
//   - status int: HTTP状态码
//   - message string: 错误消息
//   - args ...any: 可选参数
//
// 返回:
//   - ValidationError: 验证错误实例
func NewValidationError(status int, message string, args ...any) ValidationError {
	if Validation == nil {
		return defaultValidationError{}.NewValidationError(status, message, args...)
	}
	return Validation.validationError.NewValidationError(status, message, args...)
}

// newValidation 创建新的验证管理器
//
// 返回:
//   - *validation: 验证管理器实例
func newValidation() *validation {
	return &validation{
		validationError: defaultValidationError{}, // 使用默认参数验证错误
	}
}

// validation 验证管理器结构
//
// 字段:
//   - validationError ValidationError: 验证错误处理器
type validation struct {
	validationError ValidationError
}

// SetValidationError 设置验证错误处理器
//
// 参数:
//   - validationError ValidationError: 自定义的验证错误处理器
func (v *validation) SetValidationError(validationError ValidationError) {
	v.validationError = validationError
}

// defaultValidationError 默认验证错误实现
//
// 字段:
//   - Status int: HTTP状态码
//   - Message string: 错误消息
type defaultValidationError struct {
	Status  int
	Message string
}

// 默认创建参数验证错误方法
func (validationErr defaultValidationError) NewValidationError(status int, message string, args ...any) ValidationError {
	return &defaultValidationError{
		Status:  status,
		Message: message,
	}
}

// Error 返回错误消息
func (validationErr defaultValidationError) Error() string {
	return validationErr.Message
}

// 默认参数验证错误响应格式方法
func (validationErr defaultValidationError) Response() Response {
	return Response{
		Status: validationErr.Status,
		Data:   validationErr.Message,
	}
}

// Validator 验证器接口
//
// 方法:
//   - Validator: 验证方法
//   - ToGo: 将字符串转换为Go值，ToGo 的返回值要与读取字段类型一致
type Validator interface {
	Validate(value any) ValidationError
	ToGo(value any) (any, ValidationError)
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
//   - validator Validator: 验证器实例
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

// boolValidator 布尔类型验证器
type boolValidator struct{}

func (validator boolValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case bool:
		return nil
	case string:
		var reStr = `^(true|false)$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator boolValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case bool:
		return typeValue, nil
	case string:
		if strings.ToLower(typeValue) == "true" {
			return true, nil
		} else if strings.ToLower(typeValue) == "false" {
			return false, nil
		}
	}
	return value, nil
}

// intValidator 整数类型验证器
type intValidator struct{}

func (validator intValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return nil
	case string:
		var reStr = `^([0-9]+)$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator intValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return typeValue, nil
	case string:
		// 尝试转换字符串为整数
		if val, err := strconv.Atoi(typeValue); err == nil {
			return val, nil
		}
	}
	return value, nil
}

// stringValidator 字符串类型验证器
type stringValidator struct{}

func (validator stringValidator) Validate(value any) ValidationError {
	switch value.(type) {
	case string:
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator stringValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		return typeValue, nil
	default:
		// 尝试转换为字符串
		return fmt.Sprintf("%v", value), nil
	}
}

// time.Time 类型
// 支持格式: "2006-01-02 15:04:05" 和 ISO8601 (time.RFC3339)
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
func (validator timeValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		if _, ok := parseTime(typeValue); !ok {
			timeFormatErrorMsg := i18n.T("validator.time_format_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, timeFormatErrorMsg)
		}
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

func (validator timeValidator) ToGo(value any) (any, ValidationError) {
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
		return typeValue, nil
	}
}

// sliceValidator 切片类型验证器
type sliceValidator struct{}

func (validator sliceValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\[.*\])$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		kind := valueType.Kind()
		if kind == reflect.Slice || kind == reflect.Array {
			return nil
		}
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator sliceValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		var sliceValue []any
		// 对于字符串形式的切片，这里可以根据需要进行JSON解析等
		err := json.Unmarshal([]byte(typeValue), &sliceValue)
		if err != nil {
			return nil, NewValidationError(http.StatusBadRequest, err.Error())
		}
		return sliceValue, nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		kind := valueType.Kind()
		if kind == reflect.Slice || kind == reflect.Array {
			return value, nil
		}
		return value, nil
	}
}

// mapValidator 映射类型验证器
type mapValidator struct{}

func (validator mapValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\{.*\})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		// 判断是否为切片类型
		if valueType.Kind() == reflect.Map {
			return nil
		}
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator mapValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		// 对于字符串形式的映射，这里可以根据需要进行JSON解析等
		var mapValue map[string]any
		// 对于字符串形式的切片，这里可以根据需要进行JSON解析等
		err := json.Unmarshal([]byte(typeValue), &mapValue)
		if err != nil {
			return nil, NewValidationError(http.StatusBadRequest, err.Error())
		}
		return mapValue, nil
	default:
		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		kind := valueType.Kind()
		if kind == reflect.Map {
			return value, nil
		}
		return value, nil
	}
}

// slugValidator slug格式验证器
type slugValidator struct{}

func (validator slugValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^([-a-zA-Z0-9_]+)`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator slugValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		return typeValue, nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// uuidValidator UUID格式验证器
type uuidValidator struct{}

func (validator uuidValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"value": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"value": value,
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

func (validator uuidValidator) ToGo(value any) (any, ValidationError) {
	switch typeValue := value.(type) {
	case string:
		return typeValue, nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}
