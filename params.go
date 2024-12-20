package goi

import (
	"encoding/json"
	"fmt"
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
	if ok == false {
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

		field = paramsType.Field(i)
		fieldName = field.Tag.Get("name")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name) // 字段名
		}

		value_list, ok := values[fieldName]
		if validator_name = field.Tag.Get("required"); validator_name != "" {
			if ok == false {
				requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.required_params",
					TemplateData: map[string]interface{}{
						"name": fieldName,
					},
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
		} else if validator_name = field.Tag.Get("optional"); validator_name != "" {
			if ok == false {
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

		for _, value := range value_list {
			validationErr = validateValue(validator_name, value)
			if validationErr != nil {
				return validationErr
			}
			// 设置到参数结构体中
			validationErr = values.setFieldValue(paramsValue.Field(i), value)
			if validationErr != nil {
				return validationErr
			}
		}
	}

	return nil
}

// 将值设置到结构体字段中
func (values ParamsValues) setFieldValue(field reflect.Value, value string) ValidationError {
	var validationErr ValidationError
	// 检查字段是否为指针类型
	if field.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if field.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	// 检查是否为可分配值
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
	// 尝试将值转换为目标变量的类型并赋值给目标变量
	switch fieldType.Kind() {
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
	case reflect.Slice:
		jsonData := make([]interface{}, 0, 0)
		err := json.Unmarshal([]byte(value), &jsonData)
		if err != nil {
			return NewValidationError(http.StatusBadRequest, err.Error())
		}
		// 获取切片元素的类型
		elementType := fieldType.Elem()
		// 创建一个新的切片，长度为0，容量为0
		sliceValue := reflect.MakeSlice(fieldType, 0, len(jsonData))
		for _, itemValue := range jsonData {
			elemValue := reflect.New(elementType).Elem()
			stringItemValue, ok := itemValue.(string)
			if !ok {
				// 如果无法转换为字符串，则使用 fmt.Sprint 将其转换为字符串
				stringItemValue = fmt.Sprint(itemValue)
			}
			validationErr = values.setFieldValue(elemValue, stringItemValue)
			if validationErr != nil {
				return validationErr
			}
			sliceValue = reflect.Append(sliceValue, elemValue)
		}
		field.Set(sliceValue)
	case reflect.Map:
		jsonData := make(map[string]interface{})
		err := json.Unmarshal([]byte(value), &jsonData)
		if err != nil {
			return NewValidationError(http.StatusBadRequest, err.Error())
		}

		// 获取映射键和值的类型
		keyType := fieldType.Key()
		valueType := fieldType.Elem()

		// 创建一个新的映射
		mapValue := reflect.MakeMap(field.Type())

		for itemName, itemValue := range jsonData {
			keyValue := reflect.New(keyType).Elem()
			valueValue := reflect.New(valueType).Elem()
			validationErr = values.setFieldValue(keyValue, itemName)
			if validationErr != nil {
				return validationErr
			}
			stringItemValue, ok := itemValue.(string)
			if !ok {
				// 如果无法转换为字符串，则使用 fmt.Sprint 将其转换为字符串
				stringItemValue = fmt.Sprint(itemValue)
			}
			validationErr = values.setFieldValue(valueValue, stringItemValue)
			if validationErr != nil {
				return validationErr
			}
			mapValue.SetMapIndex(keyValue, valueValue)
		}
		field.Set(mapValue)
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
	if ok == false {
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

		field = paramsType.Field(i)

		fieldName = field.Tag.Get("name")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name) // 字段名
		}

		value_list, ok := values[fieldName]
		if validator_name = field.Tag.Get("required"); validator_name != "" {
			if ok == false {
				requiredParamsMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.required_params",
					TemplateData: map[string]interface{}{
						"name": fieldName,
					},
				})
				return NewValidationError(http.StatusBadRequest, requiredParamsMsg)
			}
		} else if validator_name = field.Tag.Get("optional"); validator_name != "" {
			if ok == false {
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

		for _, value := range value_list {
			if value == nil {
				continue
			}
			stringValue := fmt.Sprint(value)
			validationErr = validateValue(validator_name, stringValue)
			if validationErr != nil {
				return validationErr
			}
			// 设置到参数结构体中
			validationErr = values.setFieldValue(paramsValue.Field(i), value)
			if validationErr != nil {
				return validationErr
			}
		}
	}

	return nil
}

// 将值设置到结构体字段中
func (values BodyParamsValues) setFieldValue(field reflect.Value, value interface{}) ValidationError {
	var validationErr ValidationError
	// 检查字段是否为指针类型
	if field.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if field.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	// 检查是否为可分配值
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
	// 尝试将值转换为目标变量的类型并赋值给目标变量
	switch fieldType.Kind() {
	case reflect.Bool:
		var boolValue bool
		switch v := value.(type) {
		case bool:
			boolValue = v
		case string:
			var err error
			boolValue, err = strconv.ParseBool(v)
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": fieldType.Name(),
					"type": "bool",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
		field.SetBool(boolValue)
	case reflect.String:
		if strValue, ok := value.(string); ok {
			field.SetString(strValue)
		} else {
			field.SetString(fmt.Sprint(value))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case string:
			intValue, err := strconv.ParseInt(v, 10, fieldType.Bits())
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetInt(intValue)
		case int64:
			field.SetInt(v)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": fieldType.Name(),
					"type": "int int8 int16 int32 int64",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case string:
			uintValue, err := strconv.ParseUint(v, 10, fieldType.Bits())
			if err != nil {
				return NewValidationError(http.StatusInternalServerError, err.Error())
			}
			field.SetUint(uintValue)
		case uint64:
			field.SetUint(v)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": fieldType.Name(),
					"type": "uint uint8 uint16 uint32 uint64",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float32:
			field.SetFloat(float64(v))
		case float64:
			field.SetFloat(v)
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
					"name": fieldType.Name(),
					"type": "float32 float64",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}
	case reflect.Slice:
		jsonData := make([]interface{}, 0, 0)
		switch v := value.(type) {
		case []interface{}:
			jsonData = v
		case string:
			err := json.Unmarshal([]byte(v), &jsonData)
			if err != nil {
				return NewValidationError(http.StatusBadRequest, err.Error())
			}
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": fieldType.Name(),
					"type": "slice",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}

		// 获取切片元素的类型
		elementType := fieldType.Elem()
		// 创建一个新的切片，长度为0，容量为0
		sliceValue := reflect.MakeSlice(fieldType, 0, len(jsonData))
		for _, itemValue := range jsonData {
			elemValue := reflect.New(elementType).Elem()
			stringItemValue, ok := itemValue.(string)
			if !ok {
				// 如果无法转换为字符串，则使用 fmt.Sprint 将其转换为字符串
				stringItemValue = fmt.Sprint(itemValue)
			}
			validationErr = values.setFieldValue(elemValue, stringItemValue)
			if validationErr != nil {
				return validationErr
			}
			sliceValue = reflect.Append(sliceValue, elemValue)
		}
		field.Set(sliceValue)
	case reflect.Map:
		jsonData := make(map[string]interface{})
		switch v := value.(type) {
		case map[string]interface{}:
			jsonData = v
		case string:
			err := json.Unmarshal([]byte(v), &jsonData)
			if err != nil {
				return NewValidationError(http.StatusBadRequest, err.Error())
			}
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": fieldType.Name(),
					"type": "map",
				},
			})
			return NewValidationError(http.StatusBadRequest, valueInvalidMsg)
		}

		// 获取映射键和值的类型
		keyType := fieldType.Key()
		valueType := fieldType.Elem()

		// 创建一个新的映射
		mapValue := reflect.MakeMap(field.Type())

		for itemName, itemValue := range jsonData {
			keyValue := reflect.New(keyType).Elem()
			valueValue := reflect.New(valueType).Elem()
			validationErr = values.setFieldValue(keyValue, itemName)
			if validationErr != nil {
				return validationErr
			}
			stringItemValue, ok := itemValue.(string)
			if !ok {
				// 如果无法转换为字符串，则使用 fmt.Sprint 将其转换为字符串
				stringItemValue = fmt.Sprint(itemValue)
			}
			validationErr = values.setFieldValue(valueValue, stringItemValue)
			if validationErr != nil {
				return validationErr
			}
			mapValue.SetMapIndex(keyValue, valueValue)
		}
		field.Set(mapValue)
	default:
		paramsTypeIsNotMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_type_is_unsupported",
		})
		return NewValidationError(http.StatusBadRequest, paramsTypeIsNotMsg)
	}
	return nil
}
