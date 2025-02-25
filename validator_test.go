package goi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi"
)

// ExampleRegisterValidate 展示如何注册和使用自定义验证器
//
// 本示例展示了如何：
//  1. 注册手机号验证器，支持字符串和整数类型输入
//  2. 注册邮箱验证器，验证邮箱格式
//
// 验证器可以：
//   - 处理多种输入类型
//   - 返回自定义错误信息
//   - 使用正则表达式进行格式验证
func ExampleRegisterValidate() {

	// 注册手机号验证器
	goi.RegisterValidate("phone", func(value interface{}) goi.ValidationError {
		switch typeValue := value.(type) {
		case int:
			// 处理整数类型的手机号
			valueStr := strconv.Itoa(typeValue)
			var reStr = `^(1[3456789]\d{9})$`
			re := regexp.MustCompile(reStr)
			if re.MatchString(valueStr) == false {
				return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
			}
		case string:
			// 处理字符串类型的手机号
			var reStr = `^(1[3456789]\d{9})$`
			re := regexp.MustCompile(reStr)
			if re.MatchString(typeValue) == false {
				return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
			}
		default:
			// 处理不支持的类型
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
		}

		return nil
	})

	// 注册邮箱验证器
	goi.RegisterValidate("email", func(value interface{}) goi.ValidationError {
		switch v := value.(type) {
		case string:
			// 验证邮箱格式
			re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !re.MatchString(v) {
				return goi.NewValidationError(http.StatusBadRequest, "invalid email format")
			}
			return nil
		default:
			// 邮箱必须是字符串类型
			return goi.NewValidationError(http.StatusBadRequest, "email must be string")
		}
	})

	// 使用示例:
	// goi.validateValue("phone", "13812345678") // 验证成功
	// goi.validateValue("phone", 13812345678)   // 验证成功
	// goi.validateValue("phone", "1381234567")  // 验证失败：格式错误
	// goi.validateValue("email", "test@example.com") // 验证成功
	// goi.validateValue("email", "invalid-email")    // 验证失败：格式错误
}
