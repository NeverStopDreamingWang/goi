package goi

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

type Params map[string]interface{}

// 设置一个键值，会覆盖原来的值
func (values Params) Set(key string, value interface{}) {
	values[key] = value
}

// 删除一个值
func (values Params) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values Params) Has(key string) bool {
	_, ok := values[key]
	return ok
}

// 根据键读取一个值到 dest
func (values Params) Get(key string, dest interface{}) ValidationError {
	var err error
	value, ok := values[key]
	if !ok {
		requiredParamsMsg := i18n.T("params.required_params", map[string]interface{}{
			"name": key,
		})
		return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
	}
	// 获取目标变量的反射值
	destValue := reflect.ValueOf(dest)
	// 检查目标变量是否为指针类型
	if destValue.Kind() != reflect.Ptr {
		paramsIsNotPtrMsg := i18n.T("params.params_is_not_ptr", map[string]interface{}{
			"name": "dest",
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotPtrMsg)
	}

	err = SetValue(destValue, value)
	if err != nil {
		return NewValidationError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// 解析参数到指定结构体
func (values Params) ParseParams(paramsDest interface{}) ValidationError {
	var validationErr ValidationError
	// 使用反射获取参数结构体的值信息
	paramsValue := reflect.ValueOf(paramsDest)

	// 如果参数不是指针或者不是结构体类型，则返回错误
	if paramsValue.Kind() != reflect.Ptr {
		paramsIsNotPtrMsg := i18n.T("params.params_is_not_ptr", map[string]interface{}{
			"name": "paramsDest",
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotPtrMsg)
	}
	paramsValue = paramsValue.Elem()
	if paramsValue.Kind() != reflect.Struct {
		paramsIsNotStructPtrMsg := i18n.T("params.params_is_not_struct_ptr", map[string]interface{}{
			"name": "paramsDest",
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotStructPtrMsg)
	}

	paramsType := paramsValue.Type()

	// 遍历参数结构体的字段
	for i := 0; i < paramsType.NumField(); i++ {
		var err error
		var fieldValue reflect.Value
		var fieldType reflect.StructField
		var fieldName string
		var validator_name string

		fieldValue = paramsValue.Field(i)
		fieldType = paramsType.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		fieldName = fieldType.Tag.Get("name")
		if fieldName == "" || fieldName == "-" {
			// 无 name 标签则尝试获取 json 标签
			fieldName = fieldType.Tag.Get("json")
			if fieldName == "" || fieldName == "-" {
				// 无 json 标签则使用字段名小写
				fieldName = strings.ToLower(fieldType.Name) // 字段名
			}
		}
		validator_name = fieldType.Tag.Get("type") // 类型
		if validator_name == "" || validator_name == "-" {
			// 无 type 标签，跳过验证读取
			continue
		}

		value, ok := values[fieldName]

		required := fieldType.Tag.Get("required")
		if !ok {
			if required == "true" { // 必填项返回错误
				requiredParamsMsg := i18n.T("params.required_params", map[string]interface{}{
					"name": fieldName,
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
			continue
		}

		// 判断是否允许空值
		if value == nil {
			allow_null := fieldType.Tag.Get("allow_null")
			if allow_null == "false" || (required == "true" && allow_null != "true") {
				requiredParamsMsg := i18n.T("params.params_is_not_null", map[string]interface{}{
					"name": fieldName,
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
			continue
		}

		// 获取验证器
		validate, ok := GetValidate(validator_name)
		if !ok {
			validatorNotExistsMsg := i18n.T("validator.validator_not_exists", map[string]interface{}{
				"name": validator_name,
			})
			return NewValidationError(http.StatusBadRequest, validatorNotExistsMsg)
		}

		// 执行验证
		validationErr = validate.Validate(value)
		if validationErr != nil {
			return validationErr
		}
		// 转换为Go值
		value, validationErr = validate.ToGo(value)
		if validationErr != nil {
			return validationErr
		}
		// 设置到参数结构体中
		err = SetValue(fieldValue, value)
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
	}

	return nil
}
