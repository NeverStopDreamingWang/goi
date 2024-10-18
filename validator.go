package goi

import (
	"net/http"
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
type validateFunc func(value string) ValidationError

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

func validateValue(validator_name string, value string) ValidationError {
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
func boolValidate(value string) ValidationError {
	var BoolRe = `^(true|false)$`
	re := regexp.MustCompile(BoolRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// int 类型
func intValidate(value string) ValidationError {
	var IntRe = `^([0-9]+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// string 类型
func stringValidate(value string) ValidationError {
	var IntRe = `^(.+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// slice 类型
func sliceValidate(value string) ValidationError {
	var IntRe = `^(\[.+\])$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// map 类型
func mapValidate(value string) ValidationError {
	var IntRe = `^(\{.+\})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// slug 类型
func slugValidate(value string) ValidationError {
	var IntRe = `^([-a-zA-Z0-9_]+)`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}

// uuid 类型
func uuidValidate(value string) ValidationError {
	var IntRe = `^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		paramsErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "validator.params_error",
			TemplateData: map[string]interface{}{
				"err": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, paramsErrorMsg)
	}
	return nil
}
