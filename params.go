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
		var field reflect.StructField
		var fieldName string
		var validator_name string
		var is_required bool

		field = paramsType.Field(i)

		fieldName = field.Tag.Get("name")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name) // 字段名
		}

		value, ok := values[fieldName]
		if validator_name = field.Tag.Get("required"); validator_name != "" {
			is_required = true
			if !ok {
				requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.required_params",
					TemplateData: map[string]interface{}{
						"name": fieldName,
					},
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
		} else if validator_name = field.Tag.Get("optional"); validator_name != "" {
			is_required = false
			if !ok {
				continue
			}
		} else {
			isNotRequiredOrOptionalMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.is_not_required_or_optional",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			return NewValidationError(http.StatusInternalServerError, isNotRequiredOrOptionalMsg)
		}

		// 获取该类型的零值
		zeroValue := reflect.Zero(reflect.ValueOf(value).Type()).Interface()

		// 是否零值
		if reflect.DeepEqual(value, zeroValue) {
			if is_required == false { // 非必填项跳过
				continue
			}
			requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.required_params",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
		}

		validationErr = validateValue(validator_name, value)
		if validationErr != nil {
			return validationErr
		}

		// 设置到参数结构体中
		err := SetValue(paramsValue.Field(i), value)
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
	}

	return nil
}
