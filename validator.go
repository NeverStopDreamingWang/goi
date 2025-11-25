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

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
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
	if Validator == nil {
		return defaultValidationError{}.NewValidationError(status, message, args...)
	}
	return Validator.validation_error.NewValidationError(status, message, args...)
}

// newValidator 创建新的验证器管理器
//
// 返回:
//   - *validator: 验证器管理器实例
func newValidator() *validator {
	return &validator{
		validation_error: defaultValidationError{}, // 使用默认参数验证错误
	}
}

// validator 验证器管理结构
//
// 字段:
//   - validation_error ValidationError: 验证错误处理器
type validator struct {
	validation_error ValidationError
}

// SetValidationError 设置验证错误处理器
//
// 参数:
//   - validationError ValidationError: 自定义的验证错误处理器
func (metaValidator *validator) SetValidationError(validationError ValidationError) {
	metaValidator.validation_error = validationError
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

// Validate 验证器接口
//
// 方法:
//   - Validate: 验证方法
//   - ToGo: 将字符串转换为Go值，ToGo 的返回值要与读取字段类型一致
type Validate interface {
	Validate(value any) ValidationError
	ToGo(value any) (any, ValidationError)
}

// validateMu 保护 validates
var validateMu sync.RWMutex

// validates 存储验证器映射
// key: 验证器名称
// value: 验证器实例
var validates = map[string]Validate{
	"bool":   &boolValidator{},
	"int":    &intValidator{},
	"string": &stringValidator{},
	"slice":  &sliceValidator{},
	"map":    &mapValidator{},
	"slug":   &slugValidator{},
	"uuid":   &uuidValidator{},
}

// RegisterValidate 注册自定义验证器
//
// 参数:
//   - name string: 验证器名称
//   - validate Validate: 验证器实例
func RegisterValidate(name string, validate Validate) {
	validateMu.Lock()
	defer validateMu.Unlock()
	validates[name] = validate
}

// GetValidate 获取验证器
//
// 参数:
//   - name string: 验证器名称
//
// 返回:
//   - Validate: 验证器实例
//   - bool: 是否存在
func GetValidate(name string) (Validate, bool) {
	validateMu.RLock()
	defer validateMu.RUnlock()
	validate, ok := validates[name]
	return validate, ok
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
				"err": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"err": value,
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
				"err": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"err": value,
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
			"err": value,
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

// sliceValidator 切片类型验证器
type sliceValidator struct{}

func (validator sliceValidator) Validate(value any) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\[.\])$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"err": value,
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
			"err": value,
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
		var reStr = `^(\{.\})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
				"err": value,
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
			"err": value,
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
				"err": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"err": value,
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
				"err": value,
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := i18n.T("validator.params_error", map[string]any{
			"err": value,
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
