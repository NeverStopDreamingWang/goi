package goi

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
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
		requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.required_params",
			TemplateData: map[string]interface{}{
				"name": key,
			},
		})
		return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
	}
	// 获取目标变量的反射值
	destValue := reflect.ValueOf(dest)
	// 检查目标变量是否为指针类型
	if destValue.Kind() != reflect.Ptr {
		paramsIsNotPtrMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "dest",
			},
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
		paramsIsNotPtrMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "paramsDest",
			},
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotPtrMsg)
	}
	paramsValue = paramsValue.Elem()
	if paramsValue.Kind() != reflect.Struct {
		paramsIsNotStructPtrMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_struct_ptr",
			TemplateData: map[string]interface{}{
				"name": "paramsDest",
			},
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
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name) // 字段名
		}
		validator_name = fieldType.Tag.Get("type") // 类型
		if validator_name == "" {
			isNotTypeMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.is_not_type",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			return NewValidationError(http.StatusInternalServerError, isNotTypeMsg)
		}

		value, ok := values[fieldName]

		required := fieldType.Tag.Get("required")
		if !ok {
			if required == "true" { // 必填项返回错误
				requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.required_params",
					TemplateData: map[string]interface{}{
						"name": fieldName,
					},
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
			continue
		}

		// 判断是否允许空值
		if value == nil {
			allow_null := fieldType.Tag.Get("allow_null")
			if allow_null == "true" {
				continue
			} else if allow_null == "" && required != "true" { // 可选参数允许空值
				continue
			} else {
				requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.params_is_not_null",
					TemplateData: map[string]interface{}{
						"name": fieldName,
					},
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
		}

		// 获取验证器
		validate, ok := GetValidate(validator_name)
		if !ok {
			validatorNotExistsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "validator.validator_not_exists",
				TemplateData: map[string]interface{}{
					"name": validator_name,
				},
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
