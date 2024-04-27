package goi

import (
	"fmt"
	"net/http"
	"regexp"
)

// 验证器处理函数
type validateFunc func(value string) ValidationError

var metaValidator = map[string]validateFunc{
	"int":    intValidate,
	"string": stringValidate,
	"slice":  sliceValidate,
	"map":    mapValidate,
	"slug":   slugValidate,
	"uuid":   uuidValidate,
}

// 注册一个验证器
func RegisterValidator(name string, validate validateFunc) {
	metaValidator[name] = validate
}

func metaValidate(validator_name string, value string) ValidationError {
	validate, ok := metaValidator[validator_name]
	if ok == false {
		return NewValidationError(fmt.Sprintf("验证器 %v 没有 validateFunc 方法", validator_name), http.StatusInternalServerError)
	}
	return validate(value)
}

// 验证器错误类型
type ValidationError interface {
	Error() string
	Response() Response
}

type validationError struct {
	s      string
	status int
}

// 实现 error 接口的 Error 方法
func (validationErr *validationError) Error() string {
	return validationErr.s
}

// 实现 error 接口的 Resp 方法
func (validationErr *validationError) Response() Response {
	return Response{
		Status: validationErr.status,
		Data:   validationErr.Error(),
	}
}

func NewValidationError(text string, status int) ValidationError {
	return &validationError{
		s:      text,
		status: status,
	}
}

// int 类型
func intValidate(value string) ValidationError {
	var IntRe = `^([0-9]+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}

// string 类型
func stringValidate(value string) ValidationError {
	var IntRe = `^(.+)$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}

// slice 类型
func sliceValidate(value string) ValidationError {
	var IntRe = `^(\[.+\])$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}

// map 类型
func mapValidate(value string) ValidationError {
	var IntRe = `^(\{.+\})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}

// slug 类型
func slugValidate(value string) ValidationError {
	var IntRe = `^([-a-zA-Z0-9_]+)`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}

// uuid 类型
func uuidValidate(value string) ValidationError {
	var IntRe = `^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return NewValidationError(fmt.Sprintf("参数错误：%v", value), http.StatusBadRequest)
	}
	return nil
}
