package example

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 注册验证器
	// 手机号
	goi.RegisterValidate("phone", phoneValidate)
}

// phone 类型
func phoneValidate(value string) goi.ValidationError {
	var IntRe = `^(1[3456789]\d{9})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}

// 自定义参数验证错误
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

// 参数验证错误响应格式
func (validationErr *exampleValidationError) Response() goi.Response {
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  validationErr.Status,
			Message: validationErr.Message,
			Data:    nil,
		},
	}
}
