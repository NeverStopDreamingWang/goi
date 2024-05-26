package goi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

type metaValues map[string][]string

// 根据键添加一个值
func (values metaValues) Add(key, value string) {
	values[key] = append(values[key], value)
}

// 设置一个键值，会覆盖原来的值
func (values metaValues) Set(key, value string) {
	values[key] = []string{value}
}

// 删除一个值
func (values metaValues) Del(key string) {
	delete(values, key)
}

// 查看值是否存在
func (values metaValues) Has(key string) bool {
	_, ok := values[key]
	return ok
}

// 根据键获取一个值, 获取不到返回 nil
func (values metaValues) Get(key string, dest interface{}) ValidationError {
	var validationErr ValidationError
	value_list, ok := values[key]
	if ok == false {
		return NewValidationError(http.StatusBadRequest, fmt.Sprintf("'%v' 缺少必填参数", key))
	}
	// 获取目标变量的反射值
	destValue := reflect.ValueOf(dest)
	// 检查目标变量是否为指针类型
	if destValue.Kind() != reflect.Ptr {
		return NewValidationError(http.StatusInternalServerError, "参数必须是指针类型")
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
func (values metaValues) ParseParams(paramsDest interface{}) ValidationError {
	var validationErr ValidationError
	// 使用反射获取参数结构体的值信息
	paramsValue := reflect.ValueOf(paramsDest)

	// 如果参数不是指针或者不是结构体类型，则返回错误
	if paramsValue.Kind() != reflect.Ptr {
		return NewValidationError(http.StatusInternalServerError, "参数必须是指针类型")
	}
	paramsValue = paramsValue.Elem()
	if paramsValue.Kind() != reflect.Struct {
		return NewValidationError(http.StatusInternalServerError, "参数必须是结构体指针类型")
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
			fieldName = field.Name // 字段名
		}

		value_list, ok := values[fieldName]
		if validator_name = field.Tag.Get("required"); validator_name != "" && ok == false {
			return NewValidationError(http.StatusBadRequest, fmt.Sprintf("'%v' 缺少必填参数！", fieldName))
		} else if validator_name = field.Tag.Get("optional"); validator_name == "" {
			return NewValidationError(http.StatusInternalServerError, fmt.Sprintf("'%v' 字段标签 required 与 optional 必须存在一个！", fieldName))
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
func (values metaValues) setFieldValue(field reflect.Value, value string) ValidationError {
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
		return NewValidationError(http.StatusInternalServerError, "dest 不可赋值的值")
	}

	fieldType := field.Type()
	// 尝试将值转换为目标变量的类型并赋值给目标变量
	switch fieldType.Kind() {
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
		return NewValidationError(http.StatusBadRequest, "不支持的目标变量类型")
	}
	return nil
}
