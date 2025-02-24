package goi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi"
)

// ExampleRegisterValidate 展示如何注册和使用自定义验证器
func ExampleRegisterValidate() {

	// 注册手机号验证器
	goi.RegisterValidate("phone", func(value interface{}) goi.ValidationError {
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
	})

	// 注册邮箱验证器
	goi.RegisterValidate("email", func(value interface{}) goi.ValidationError {
		switch v := value.(type) {
		case string:
			re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !re.MatchString(v) {
				return goi.NewValidationError(http.StatusBadRequest, "invalid email format")
			}
			return nil
		default:
			return goi.NewValidationError(http.StatusBadRequest, "email must be string")
		}
	})
}
