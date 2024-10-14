package goi

import (
	"fmt"
	"net/http"
	"regexp"
)

// 参数验证器返回错误
type ValidationError interface {
	// 创建参数验证错误方法
	NewValidationError(status int, message string, args ...interface{}) ValidationError
	// 参数验证错误响应格式方法
	Response() Response
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

type metaValidator struct {
	VALIDATION_ERROR ValidationError
}

func newValidator() *metaValidator {
	return &metaValidator{
		VALIDATION_ERROR: &defaultValidationError{}, // 使用默认参数验证错误
	}
}

// 框架全局创建参数验证错误
func NewValidationError(status int, message string, args ...interface{}) ValidationError {
	return Validator.VALIDATION_ERROR.NewValidationError(status, message, args...)
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
		return NewValidationError(http.StatusInternalServerError, fmt.Sprintf("验证器 %v 没有 validateFunc 方法", validator_name))
	}
	return validate(value)
}

// bool 类型
func boolValidate(value string) ValidationError {
	var BoolRe = `^(true|false)$`
	re := regexp.MustCompile(BoolRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// int 类型
func intValidate(value string) ValidationError {
	var IntRe = `^([0-9]+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// string 类型
func stringValidate(value string) ValidationError {
	var IntRe = `^(.+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// slice 类型
func sliceValidate(value string) ValidationError {
	var IntRe = `^(\[.+\])$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// map 类型
func mapValidate(value string) ValidationError {
	var IntRe = `^(\{.+\})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// slug 类型
func slugValidate(value string) ValidationError {
	var IntRe = `^([-a-zA-Z0-9_]+)`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// uuid 类型
func uuidValidate(value string) ValidationError {
	var IntRe = `^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}
