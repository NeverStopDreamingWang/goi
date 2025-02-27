package goi

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func SetValue(targetValue reflect.Value, source interface{}) error {
	// 处理指针类型
	if targetValue.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if targetValue.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			targetValue.Set(reflect.New(targetValue.Type().Elem()))
		}
		targetValue = targetValue.Elem()
	}

	if !targetValue.CanSet() {
		paramsIsNotCanSetMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "params.params_is_not_can_set",
			TemplateData: map[string]interface{}{
				"name": targetValue.Type().Name(),
			},
		})
		return errors.New(paramsIsNotCanSetMsg)
	}

	fieldType := targetValue.Type()
	// 处理 nil 值
	if source == nil {
		targetValue.Set(reflect.Zero(fieldType))
		return nil
	}

	sourceValue := reflect.ValueOf(source)
	if targetValue.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if targetValue.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			targetValue.Set(reflect.New(targetValue.Type().Elem()))
		}
		targetValue = targetValue.Elem()
	}

	sourceType := sourceValue.Type()
	// 直接类型匹配
	if sourceType.AssignableTo(fieldType) {
		targetValue.Set(sourceValue)
		return nil
	}

	// 类型转换处理
	switch fieldType.Kind() {
	case reflect.Bool:
		switch v := source.(type) {
		case bool:
			targetValue.SetBool(v)
		case string:
			boolValue, err := strconv.ParseBool(v)
			if err != nil {
				return errors.New(err.Error())
			}
			targetValue.SetBool(boolValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}
	case reflect.String:
		switch v := source.(type) {
		case string:
			targetValue.SetString(v)
		default:
			targetValue.SetString(fmt.Sprint(source))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := source.(type) {
		case int, int8, int16, int32, int64:
			targetValue.SetInt(reflect.ValueOf(v).Int())
		case uint, uint8, uint16, uint32, uint64:
			targetValue.SetInt(int64(reflect.ValueOf(v).Uint()))
		case float32, float64:
			floatVal := reflect.ValueOf(v).Float()
			if floatVal > float64(math.MaxInt64) || floatVal < float64(math.MinInt64) {
				valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.value_invalid",
					TemplateData: map[string]interface{}{
						"name": source,
					},
				})
				return errors.New(valueInvalidMsg)
			}
			targetValue.SetInt(int64(floatVal))
		case string:
			intValue, err := strconv.ParseInt(v, 10, fieldType.Bits())
			if err != nil {
				return errors.New(err.Error())
			}
			targetValue.SetInt(intValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := source.(type) {
		case uint, uint8, uint16, uint32, uint64:
			targetValue.SetUint(reflect.ValueOf(v).Uint())
		case int, int8, int16, int32, int64:
			targetValue.SetUint(uint64(reflect.ValueOf(v).Int()))
		case float32, float64:
			targetValue.SetUint(uint64(reflect.ValueOf(v).Float()))
		case string:
			uintValue, err := strconv.ParseUint(v, 10, fieldType.Bits())
			if err != nil {
				return errors.New(err.Error())
			}
			targetValue.SetUint(uintValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}
	case reflect.Float32, reflect.Float64:
		switch v := source.(type) {
		case float32, float64:
			targetValue.SetFloat(reflect.ValueOf(v).Float())
		case int, int8, int16, int32, int64:
			targetValue.SetFloat(float64(reflect.ValueOf(v).Int()))
		case uint, uint8, uint16, uint32, uint64:
			targetValue.SetFloat(float64(reflect.ValueOf(v).Uint()))
		case string:
			floatValue, err := strconv.ParseFloat(v, fieldType.Bits())
			if err != nil {
				return errors.New(err.Error())
			}
			targetValue.SetFloat(floatValue)
		default:
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}
	case reflect.Map:
		if sourceValue.Kind() != reflect.Map {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}

		if targetValue.IsNil() {
			targetValue.Set(reflect.MakeMap(fieldType))
		}
		keyType := fieldType.Key()
		valueTypeElem := fieldType.Elem()
		for _, key := range sourceValue.MapKeys() {
			// 确保键和值的类型兼容
			if key.Type().AssignableTo(keyType) == false {
				valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "params.value_invalid",
					TemplateData: map[string]interface{}{
						"name": source,
					},
				})
				return errors.New(valueInvalidMsg)
			}

			valueVal := sourceValue.MapIndex(key)
			if valueVal.Type().AssignableTo(valueTypeElem) {
				targetValue.SetMapIndex(key, valueVal)
			} else {
				// 创建目标 Map 的值，并递归调用 setFieldValue 进行赋值
				val := reflect.New(valueTypeElem).Elem()
				err := SetValue(val, valueVal.Interface())
				if err != nil {
					return err
				}
				targetValue.SetMapIndex(key, val)
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}

		// 如果目标切片的长度小于源切片的长度，扩展目标切片
		if targetValue.Len() < sourceValue.Len() {
			// 创建一个新的切片并设置给 targetValue
			newSlice := reflect.MakeSlice(fieldType, sourceValue.Len(), sourceValue.Len())
			targetValue.Set(newSlice)
		}

		for i := 0; i < sourceValue.Len(); i++ {
			err := SetValue(targetValue.Index(i), sourceValue.Index(i).Interface())
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		// 处理结构体类型
		if sourceValue.Kind() != reflect.Map {
			valueInvalidMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "params.value_invalid",
				TemplateData: map[string]interface{}{
					"name": source,
				},
			})
			return errors.New(valueInvalidMsg)
		}

		// 处理 map[string]interface{} 类型
		for i := 0; i < fieldType.NumField(); i++ {
			structField := targetValue.Field(i)
			fieldName := fieldType.Field(i).Name
			// 尝试获取 tag 中的 name，如果没有则使用字段名
			tagName := fieldType.Field(i).Tag.Get("json")
			if tagName == "" {
				tagName = strings.ToLower(fieldName)
			}
			// 先尝试使用 tag name 获取值，如果没有则尝试使用字段名
			mapValue := sourceValue.MapIndex(reflect.ValueOf(tagName))
			if !mapValue.IsValid() {
				mapValue = sourceValue.MapIndex(reflect.ValueOf(fieldName))
			}

			if mapValue.IsValid() && structField.CanSet() {
				err := SetValue(structField, mapValue.Interface())
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
				"name": source,
			},
		})
		return errors.New(valueInvalidMsg)
	}
	return nil
}
