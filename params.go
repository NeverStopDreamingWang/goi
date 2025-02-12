package goi

import (
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type ParamsValues map[string][]string

// 根据键添加一个值
func (values ParamsValues) Add(key, value string) {
	values[key] = append(values[key], value)
}

// 设置一个键值，会覆盖原来的值
func (values ParamsValues) Set(key, value string) {
	values[key] = []string{value}
}

// 删除一个值
func (values ParamsValues) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values ParamsValues) Has(key string) bool {
	_, ok := values[key]
	return ok
}

// 根据键读取一个值到 dest
func (values ParamsValues) Get(key string, dest interface{}) ValidationError {
	var validationErr ValidationError
	value_list, ok := values[key]
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
	for _, value := range value_list {
		// 设置到参数结构体中
		validationErr = values.setFieldValue(destValue, value)
		if validationErr != nil {
			return validationErr
		}
	}
	return nil
}

// 解析参数到指定结构体
func (values ParamsValues) ParseParams(paramsDest interface{}) ValidationError {
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

		value_list, ok := values[fieldName]
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

		var flag bool
		for _, value := range value_list {
			if value == "" {
				continue
			}
			validationErr = validateValue(validator_name, value)
			if validationErr != nil {
				return validationErr
			}
			// 设置到参数结构体中
			validationErr = values.setFieldValue(paramsValue.Field(i), value)
			if validationErr != nil {
				return validationErr
			}
			flag = true
			break
		}

		if is_required == true && flag == false {
			requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.required_params",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
		}
	}

	return nil
}

// 将值设置到结构体字段中
func (values ParamsValues) setFieldValue(field reflect.Value, value string) ValidationError {
	// 处理指针类型
	if field.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if field.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	if !field.CanSet() {
		paramsIsNotCanSetMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_can_set",
			TemplateData: map[string]interface{}{
				"name": "dest",
			},
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotCanSetMsg)
	}

	fieldType := field.Type()
	valueInterface := reflect.ValueOf(value)

	// 直接类型匹配
	if valueInterface.Type().AssignableTo(fieldType) {
		field.Set(valueInterface)
		return nil
	}

	// 类型转换处理
	switch fieldType.Kind() {
	case reflect.Interface:
		field.Set(reflect.ValueOf(value))
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
		field.SetBool(boolValue)
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, fieldType.Bits())
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, fieldType.Bits())
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, fieldType.Bits())
		if err != nil {
			return NewValidationError(http.StatusInternalServerError, err.Error())
		}
		field.SetFloat(floatValue)
	default:
		paramsTypeIsNotMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_type_is_unsupported",
		})
		return NewValidationError(http.StatusBadRequest, paramsTypeIsNotMsg)
	}
	return nil
}

// body 参数
type BodyParamsValues map[string][]interface{}

// 根据键添加一个值
func (values BodyParamsValues) Add(key string, value interface{}) {
	values[key] = append(values[key], value)
}

// 设置一个键值，会覆盖原来的值
func (values BodyParamsValues) Set(key string, value interface{}) {
	values[key] = []interface{}{value}
}

// 删除一个值
func (values BodyParamsValues) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values BodyParamsValues) Has(key string) bool {
	_, ok := values[key]
	return ok
}

// 根据键读取一个值到 dest
func (values BodyParamsValues) Get(key string, dest interface{}) ValidationError {
	var validationErr ValidationError
	value_list, ok := values[key]
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
	for _, value := range value_list {
		// 设置到参数结构体中
		validationErr = values.setFieldValue(destValue, value)
		if validationErr != nil {
			return validationErr
		}
	}
	return nil
}

// 解析参数到指定结构体
func (values BodyParamsValues) ParseParams(paramsDest interface{}) ValidationError {
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

		value_list, ok := values[fieldName]
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

		var flag bool
		for _, value := range value_list {
			if value == nil {
				continue
			}
			validationErr = validateValue(validator_name, value)
			if validationErr != nil {
				return validationErr
			}
			// 设置到参数结构体中
			validationErr = values.setFieldValue(paramsValue.Field(i), value)
			if validationErr != nil {
				return validationErr
			}
			flag = true
			break
		}

		if is_required == true && flag == false {
			requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.required_params",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
		}
	}

	return nil
}

// 将值设置到结构体字段中
func (values BodyParamsValues) setFieldValue(field reflect.Value, value interface{}) ValidationError {
	// 处理指针类型
	if field.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if field.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	if !field.CanSet() {
		paramsIsNotCanSetMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_can_set",
			TemplateData: map[string]interface{}{
				"name": "dest",
			},
		})
		return NewValidationError(http.StatusInternalServerError, paramsIsNotCanSetMsg)
	}

	fieldType := field.Type()
	valueInterface := reflect.ValueOf(value)

	// 处理 nil 值
	if value == nil {
		field.Set(reflect.Zero(fieldType))
		return nil
	}

	// 直接类型匹配
	if valueInterface.Type().AssignableTo(fieldType) {
		field.Set(valueInterface)
		return nil
	}

	// 类型转换处理
	switch fieldType.Kind() {
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case string:
			boolValue, err := strconv.ParseBool(v)
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetBool(boolValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.String:
		switch v := value.(type) {
		case string:
			field.SetString(v)
		default:
			field.SetString(fmt.Sprint(value))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int, int8, int16, int32, int64:
			field.SetInt(reflect.ValueOf(v).Int())
		case uint, uint8, uint16, uint32, uint64:
			field.SetInt(int64(reflect.ValueOf(v).Uint()))
		case float32, float64:
			floatVal := reflect.ValueOf(v).Float()
			if floatVal > float64(math.MaxInt64) || floatVal < float64(math.MinInt64) {
				valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.value_invalid",
					TemplateData: map[string]interface{}{
						"name": value,
					},
				})
				return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
			}
			field.SetInt(int64(floatVal))
		case string:
			intValue, err := strconv.ParseInt(v, 10, fieldType.Bits())
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetInt(intValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case uint, uint8, uint16, uint32, uint64:
			field.SetUint(reflect.ValueOf(v).Uint())
		case int, int8, int16, int32, int64:
			field.SetUint(uint64(reflect.ValueOf(v).Int()))
		case float32, float64:
			field.SetUint(uint64(reflect.ValueOf(v).Float()))
		case string:
			uintValue, err := strconv.ParseUint(v, 10, fieldType.Bits())
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetUint(uintValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float32, float64:
			field.SetFloat(reflect.ValueOf(v).Float())
		case int, int8, int16, int32, int64:
			field.SetFloat(float64(reflect.ValueOf(v).Int()))
		case uint, uint8, uint16, uint32, uint64:
			field.SetFloat(float64(reflect.ValueOf(v).Uint()))
		case string:
			floatValue, err := strconv.ParseFloat(v, fieldType.Bits())
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetFloat(floatValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Map:
		if valueInterface.Kind() != reflect.Map {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}

		if field.IsNil() {
			field.Set(reflect.MakeMap(fieldType))
		}
		keyType := fieldType.Key()
		valueTypeElem := fieldType.Elem()
		for _, key := range valueInterface.MapKeys() {
			// 确保键和值的类型兼容
			if key.Type().AssignableTo(keyType) == false {
				valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.value_invalid",
					TemplateData: map[string]interface{}{
						"name": value,
					},
				})
				return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
			}

			valueVal := valueInterface.MapIndex(key)
			if valueVal.Type().AssignableTo(valueTypeElem) {
				field.SetMapIndex(key, valueVal)
			} else {
				// 创建目标 Map 的值，并递归调用 setFieldValue 进行赋值
				val := reflect.New(valueTypeElem).Elem()
				err := values.setFieldValue(val, valueVal.Interface())
				if err != nil {
					return err
				}
				field.SetMapIndex(key, val)
			}
		}
		return nil
	case reflect.Slice:
		if valueInterface.Kind() != reflect.Slice {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}

		// 如果目标切片的长度小于源切片的长度，扩展目标切片
		if field.Len() < valueInterface.Len() {
			// 创建一个新的切片并设置给 field
			newSlice := reflect.MakeSlice(fieldType, valueInterface.Len(), valueInterface.Len())
			field.Set(newSlice)
		}

		for i := 0; i < valueInterface.Len(); i++ {
			err := values.setFieldValue(field.Index(i), valueInterface.Index(i).Interface())
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		// 处理结构体类型
		if valueInterface.Kind() != reflect.Map {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": value,
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}

		// 处理 map[string]interface{} 类型
		for i := 0; i < fieldType.NumField(); i++ {
			structField := field.Field(i)
			fieldName := fieldType.Field(i).Name
			// 尝试获取 tag 中的 name，如果没有则使用字段名
			tagName := fieldType.Field(i).Tag.Get("json")
			if tagName == "" {
				tagName = strings.ToLower(fieldName)
			}
			// 先尝试使用 tag name 获取值，如果没有则尝试使用字段名
			mapValue := valueInterface.MapIndex(reflect.ValueOf(tagName))
			if !mapValue.IsValid() {
				mapValue = valueInterface.MapIndex(reflect.ValueOf(fieldName))
			}

			if mapValue.IsValid() && structField.CanSet() {
				err := values.setFieldValue(structField, mapValue.Interface())
				if err != nil {
					return err
				}
			}
		}
		return nil
	default:
		valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.value_invalid",
			TemplateData: map[string]interface{}{
				"name": value,
			},
		})
		return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
	}
	return nil
}
