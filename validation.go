package goi

// ValidationError 参数验证错误处理器接口
//
// 方法:
//   - NewValidationError: 创建参数验证错误
//   - Error: 实现 error 接口，返回错误消息
//   - Response: 生成错误响应
type ValidationError interface {
	// NewValidationError 创建参数验证错误
	//
	// 参数:
	//   - status int: HTTP状态码
	//   - message string: 错误消息
	//   - args ...any: 可选参数
	//
	// 返回:
	//   - ValidationError: 验证错误实例
	NewValidationError(status int, message string, args ...any) ValidationError

	// Error 返回错误消息
	Error() string

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
//   - args ...any: 可选参数
//
// 返回:
//   - ValidationError: 验证错误实例
func NewValidationError(status int, message string, args ...any) ValidationError {
	if Validation == nil {
		return defaultValidationError{}.NewValidationError(status, message, args...)
	}
	return Validation.validationError.NewValidationError(status, message, args...)
}

// newValidation 创建新的验证管理器
//
// 返回:
//   - *validation: 验证管理器实例
func newValidation() *validation {
	return &validation{
		validationError: defaultValidationError{}, // 使用默认参数验证错误
	}
}

// validation 验证管理器结构
//
// 字段:
//   - validationError ValidationError: 验证错误处理器
type validation struct {
	validationError ValidationError
}

// SetValidationError 设置验证错误处理器
//
// 参数:
//   - validationError ValidationError: 自定义的验证错误处理器
func (v *validation) SetValidationError(validationError ValidationError) {
	v.validationError = validationError
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
func (validationErr defaultValidationError) NewValidationError(status int, message string, args ...any) ValidationError {
	return &defaultValidationError{
		Status:  status,
		Message: message,
	}
}

// Error 返回错误消息
func (validationErr defaultValidationError) Error() string {
	return validationErr.Message
}

// 默认参数验证错误响应格式方法
func (validationErr defaultValidationError) Response() Response {
	return Response{
		Status: validationErr.Status,
		Data:   validationErr.Message,
	}
}
