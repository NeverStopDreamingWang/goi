package goi

import (
	"net/http"
	"reflect"
	"regexp"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 参数验证器返回错误
type ValidationError interface {
	// 创建参数验证错误方法
	NewValidationError(status int, message string, args ...interface{}) ValidationError
	// 参数验证错误响应格式方法
	Response() Response
}

// 框架全局创建参数验证错误
func NewValidationError(status int, message string, args ...interface{}) ValidationError {
	return Validator.validation_error.NewValidationError(status, message, args...)
}

type metaValidator struct {
	validation_error ValidationError
}

func newValidator() *metaValidator {
	return &metaValidator{
		validation_error: &defaultValidationError{}, // 使用默认参数验证错误
	}
}
func (metaValidator *metaValidator) SetValidationError(validationError ValidationError) {
	metaValidator.validation_error = validationError
}

// 默认参数验证错误
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

// 验证器处理函数
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

// 注册一个验证器
func RegisterValidate(name string, validate validateFunc) {
	metaValidate[name] = validate
}

func validateValue(validator_name string, value interface{}) ValidationError {
	validate, ok := metaValidate[validator_name]
	if ok == false {
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

// bool 类型
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

// int 类型
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

// string 类型
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

// slice 类型
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
		// 判断是否为切片类型
		if valueType.Kind() == reflect.Slice {
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

// map 类型
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

// slug 类型
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

// uuid 类型
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
