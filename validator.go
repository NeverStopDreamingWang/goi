package goi

import (
	"net/http"
	"reflect"
	"regexp"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// ValidationError 参数验证错误接口
//
// 方法:
//   - NewValidationError: 创建参数验证错误
//   - Response: 生成错误响应
type ValidationError interface {
	// NewValidationError 创建参数验证错误
	//
	// 参数:
	//   - status int: HTTP状态码
	//   - message string: 错误消息
	//   - args ...interface{}: 可选参数
	//
	// 返回:
	//   - ValidationError: 验证错误实例
	NewValidationError(status int, message string, args ...interface{}) ValidationError

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
//   - args ...interface{}: 可选参数
//
// 返回:
//   - ValidationError: 验证错误实例
func NewValidationError(status int, message string, args ...interface{}) ValidationError {
	return Validator.validation_error.NewValidationError(status, message, args...)
}

// newValidator 创建新的验证器管理器
//
// 返回:
//   - *metaValidator: 验证器管理器实例
func newValidator() *metaValidator {
	return &metaValidator{
		validation_error: &defaultValidationError{}, // 使用默认参数验证错误
	}
}

// metaValidator 验证器管理结构
//
// 字段:
//   - validation_error ValidationError: 验证错误处理器
type metaValidator struct {
	validation_error ValidationError
}

// SetValidationError 设置验证错误处理器
//
// 参数:
//   - validationError ValidationError: 自定义的验证错误处理器
func (metaValidator *metaValidator) SetValidationError(validationError ValidationError) {
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
func (validationErr *defaultValidationError) NewValidationError(status int, message string, args ...interface{}) ValidationError {
	return &defaultValidationError{
		Status:  status,
		Message: message,
	}
}

// 默认参数验证错误响应格式方法
func (validationErr *defaultValidationError) Response() Response {
	return Response{
		Status: validationErr.Status,
		Data:   validationErr.Message,
	}
}

// validateFunc 验证器处理函数类型
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
type validateFunc func(value interface{}) ValidationError

var metaValidate = map[string]validateFunc{
	"bool":   boolValidate,
	"int":    intValidate,
	"string": stringValidate,
	"slice":  sliceValidate,
	"map":    mapValidate,
	"slug":   slugValidate,
	"uuid":   uuidValidate,
}

// RegisterValidate 注册自定义验证器
//
// 参数:
//   - name string: 验证器名称
//   - validate validateFunc: 验证处理函数
func RegisterValidate(name string, validate validateFunc) {
	metaValidate[name] = validate
}

// validateValue 执行验证
//
// 参数:
//   - validator_name string: 验证器名称
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func validateValue(validator_name string, value interface{}) ValidationError {
	validate, ok := metaValidate[validator_name]
	if !ok {
		validatorIsNotValidateFuncMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.validator_is_not_validateFunc",
			TemplateData: map[string]interface{}{
				"name": validator_name,
			},
		})
		return NewValidationError(http.StatusBadRequest, validatorIsNotValidateFuncMsg)
	}
	return validate(value)
}

// boolValidate 布尔类型验证器
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func boolValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case bool:
		return nil
	case string:
		var reStr = `^(true|false)$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// intValidate 整数类型验证器
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func intValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return nil
	case string:
		var reStr = `^([0-9]+)$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// stringValidate 字符串类型验证器
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func stringValidate(value interface{}) ValidationError {
	switch value.(type) {
	case string:
		return nil
	default:
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// sliceValidate 切片类型验证器
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func sliceValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\[.*\])$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
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
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// mapValidate 映射类型验证器
//
// 参数:
//   - value interface{}: 待验证的值
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func mapValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\{.*\})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
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
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// slugValidate slug格式验证器
//
// 参数:
//   - value interface{}: 待验证的值，必须是字符串且只包含字母、数字、下划线和连字符
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func slugValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^([-a-zA-Z0-9_]+)`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}

// uuidValidate UUID格式验证器
//
// 参数:
//   - value interface{}: 待验证的值，必须是符合UUID格式的字符串
//
// 返回:
//   - ValidationError: 验证错误，如果验证通过则返回nil
func uuidValidate(value interface{}) ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.params_error",
				TemplateData: map[string]interface{}{
					"err": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
		}
		return nil
	default:
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
}
