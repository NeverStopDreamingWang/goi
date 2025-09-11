package example

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 注册验证器
	// 手机号
	goi.RegisterValidate("phone", phoneValidator{})
}

// 自定义参数验证错误处理器
type exampleValidationError struct {
	Status  int
	Message string
}

// 创建参数验证错误方法
func (validationErr *exampleValidationError) NewValidationError(status int, message string, args ...interface{}) goi.ValidationError {
	return &exampleValidationError{
		Status:  status,
		Message: message,
	}
}

// 实现 error 接口，返回错误消息
func (validationErr *exampleValidationError) Error() string {
	return validationErr.Message
}

// 参数验证错误响应格式
func (validationErr *exampleValidationError) Response() goi.Response {
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Code:    validationErr.Status,
			Message: validationErr.Message,
			Results: nil,
		},
	}
}

type phoneValidator struct{}

func (validator phoneValidator) Validate(value interface{}) goi.ValidationError {
	switch typeValue := value.(type) {
	case int:
		valueStr := strconv.Itoa(typeValue)
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(valueStr) == false {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	case string:
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	default:
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
	return nil
}

func (validator phoneValidator) ToGo(value interface{}) (interface{}, goi.ValidationError) {
	switch typeValue := value.(type) {
	case int:
		return typeValue, nil
	case string:
		intValue, err := strconv.Atoi(typeValue)
		if err != nil {
			return nil, goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
		}
		return intValue, nil
	default:
		// 尝试转换为字符串
		return fmt.Sprintf("%v", value), nil
	}
}
