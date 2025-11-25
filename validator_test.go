package goi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi"
)

// phoneValidator 手机号验证器示例
type phoneValidator struct{}

func (validator phoneValidator) Validate(value any) goi.ValidationError {
	switch typeValue := value.(type) {
	case int:
		valueStr := strconv.Itoa(typeValue)
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if !re.MatchString(valueStr) {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	case string:
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if !re.MatchString(typeValue) {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	default:
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
	return nil
}

func (validator phoneValidator) ToGo(value any) (any, goi.ValidationError) {
	switch typeValue := value.(type) {
	case int:
		return strconv.Itoa(typeValue), nil
	case string:
		return typeValue, nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// emailValidator 邮箱验证器示例
type emailValidator struct{}

func (validator emailValidator) Validate(value any) goi.ValidationError {
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
}

// 当解析参数值类型与结构体字段类型不一致，且结构体字段类型为自定义类型时，需要实现 ToGo 方法
func (validator emailValidator) ToGo(value any) (any, goi.ValidationError) {
	switch v := value.(type) {
	case string:
		// 转换类型，并返回
		return Email{Address: v}, nil
	default:
		// Validate 会拦截返回
		return nil, nil
	}
}

type CustomValidator struct{}

func (validator CustomValidator) Validate(value any) goi.ValidationError {
	switch typeValue := value.(type) {
	case string:
		var reStr = `^(\d{16})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	default:
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
	return nil
}

func (validator CustomValidator) ToGo(value any) (any, goi.ValidationError) {
	switch typeValue := value.(type) {
	case string:
		// 转换为自定义类型，并返回
		return Custom(typeValue), nil
	default:
		// 尝试转换为字符串
		return fmt.Sprintf("%v", value), nil
	}
}

// 支持自定义类型
type Email struct {
	Address string `json:"address"`
	// 其它自定义字段
}

type Custom []byte

type testParamsValidParams struct {
	Phone  string `name:"phone" type:"phone" required:"true"`
	Email  *Email `name:"email" type:"email" required:"true"`
	Custom Custom `name:"custom" type:"custom" required:"true"`
}

// ExampleRegisterValidate 展示如何注册和使用自定义验证器
func ExampleRegisterValidate() {
	// 注册手机号验证器
	goi.RegisterValidate("phone", phoneValidator{})

	// 注册邮箱验证器
	goi.RegisterValidate("email", emailValidator{})

	// 注册自定义验证器
	goi.RegisterValidate("custom", CustomValidator{})

	var validationErr goi.ValidationError

	var params testParamsValidParams
	bodyParams := goi.Params{
		"phone":  "13800000000",
		"email":  "test@example.com",
		"custom": "1234567890123456",
	}
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return
	}
	fmt.Printf("Phone %+v %T\n", params.Phone, params.Phone)
	fmt.Printf("Email %+v %T\n", params.Email, params.Email)
	fmt.Printf("Custom %+v %T\n", params.Custom, params.Custom)

	// Output:
	// Phone 13800000000 string
	// Email &{Address:test@example.com} *goi_test.Email
	// Custom [49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54] goi_test.Custom
}
