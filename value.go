package goi

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

type AnyUnmarshaler interface {
	UnmarshalAny(value any) error
}

var (
	anyUnmarshalerType = reflect.TypeFor[AnyUnmarshaler]()
)

// SetValue 将 source 赋值给 destValue
//
// 参数:
//   - destValue reflect.Value: 目标值
//   - source any: 源值
func SetValue(destValue reflect.Value, source any) error {
	sourceValue := reflect.ValueOf(source)
	if !sourceValue.IsValid() {
		return nil
	}
	if sourceValue.Kind() == reflect.Ptr {
		if sourceValue.IsNil() {
			return nil
		}
		sourceValue = sourceValue.Elem()
	}

	// 处理指针类型
	if destValue.Kind() == reflect.Ptr {
		// 检查字段值是否是零值
		if destValue.IsNil() {
			// 如果字段值是零值，则创建一个新的指针对象并设置为该字段的值
			destValue.Set(reflect.New(destValue.Type().Elem()))
		}
		destValue = destValue.Elem()
	}

	if !destValue.CanSet() {
		paramsIsNotCanSetMsg := i18n.T("params.params_is_not_can_set", map[string]any{
			"name": destValue.Type().Name(),
		})
		return errors.New(paramsIsNotCanSetMsg)
	}

	// 自定义类型处理
	if destValue.CanInterface() && destValue.Type().Implements(anyUnmarshalerType) {
		return destValue.Interface().(AnyUnmarshaler).UnmarshalAny(source)
	}
	if destValue.CanAddr() {
		pv := destValue.Addr()
		if pv.CanInterface() && pv.Type().Implements(anyUnmarshalerType) {
			return pv.Interface().(AnyUnmarshaler).UnmarshalAny(source)
		}
	}

	sourceType := sourceValue.Type()
	fieldType := destValue.Type()

	// 类型匹配
	if sourceType.AssignableTo(fieldType) {
		destValue.Set(sourceValue)
		return nil
	}

	// 类型转换
	if sourceType.ConvertibleTo(fieldType) {
		destValue.Set(sourceValue.Convert(fieldType))
		return nil
	}

	valueInvalidMsg := i18n.T("params.value_invalid", map[string]any{
		"value": source,
		"type":  fieldType.String(),
	})
	valueInvalidErr := errors.New(valueInvalidMsg)

	// 类型转换处理
	switch fieldType.Kind() {
	case reflect.Bool:
		switch v := source.(type) {
		case bool:
			destValue.SetBool(v)
		case string:
			boolValue, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("%w\n%v", valueInvalidErr, err)
			}
			destValue.SetBool(boolValue)
		default:
			return valueInvalidErr
		}
	case reflect.String:
		switch v := source.(type) {
		case string:
			destValue.SetString(v)
		default:
			destValue.SetString(fmt.Sprint(source))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := source.(type) {
		case int, int8, int16, int32, int64:
			i := reflect.ValueOf(v).Int()
			if destValue.OverflowInt(i) {
				return valueInvalidErr
			}
			destValue.SetInt(i)
		case uint, uint8, uint16, uint32, uint64:
			u := reflect.ValueOf(v).Uint()
			if u > math.MaxInt64 {
				return valueInvalidErr
			}
			i := int64(u)
			if destValue.OverflowInt(i) {
				return valueInvalidErr
			}
			destValue.SetInt(i)
		case float32, float64:
			f := reflect.ValueOf(v).Float()
			if f < float64(math.MinInt64) || f > float64(math.MaxInt64) {
				return valueInvalidErr
			}
			i := int64(f)
			if destValue.OverflowInt(i) {
				return valueInvalidErr
			}
			destValue.SetInt(i)
		case string:
			i, err := strconv.ParseInt(v, 10, fieldType.Bits())
			if err != nil {
				return fmt.Errorf("%w\n%v", valueInvalidErr, err)
			}
			if destValue.OverflowInt(i) {
				return valueInvalidErr
			}
			destValue.SetInt(i)
		default:
			return valueInvalidErr
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := source.(type) {
		case uint, uint8, uint16, uint32, uint64:
			u := reflect.ValueOf(v).Uint()
			if destValue.OverflowUint(u) {
				return valueInvalidErr
			}
			destValue.SetUint(u)
		case int, int8, int16, int32, int64:
			i := reflect.ValueOf(v).Int()
			if i < 0 {
				return valueInvalidErr
			}
			u := uint64(i)
			if destValue.OverflowUint(u) {
				return valueInvalidErr
			}
			destValue.SetUint(u)
		case float32, float64:
			f := reflect.ValueOf(v).Float()
			if f < 0 || f > float64(math.MaxUint64) {
				return valueInvalidErr
			}
			u := uint64(f)
			if destValue.OverflowUint(u) {
				return valueInvalidErr
			}
			destValue.SetUint(u)
		case string:
			u, err := strconv.ParseUint(v, 10, fieldType.Bits())
			if err != nil {
				return fmt.Errorf("%w\n%v", valueInvalidErr, err)
			}
			if destValue.OverflowUint(u) {
				return valueInvalidErr
			}
			destValue.SetUint(u)
		default:
			return valueInvalidErr
		}
	case reflect.Float32, reflect.Float64:
		switch v := source.(type) {
		case float32, float64:
			destValue.SetFloat(reflect.ValueOf(v).Float())
		case int, int8, int16, int32, int64:
			destValue.SetFloat(float64(reflect.ValueOf(v).Int()))
		case uint, uint8, uint16, uint32, uint64:
			destValue.SetFloat(float64(reflect.ValueOf(v).Uint()))
		case string:
			floatValue, err := strconv.ParseFloat(v, fieldType.Bits())
			if err != nil {
				return fmt.Errorf("%w\n%v", valueInvalidErr, err)
			}
			destValue.SetFloat(floatValue)
		default:
			return valueInvalidErr
		}
	case reflect.Map:
		if sourceValue.Kind() != reflect.Map {
			return valueInvalidErr
		}

		if destValue.IsNil() {
			destValue.Set(reflect.MakeMap(fieldType))
		}
		keyType := fieldType.Key()
		valueTypeElem := fieldType.Elem()
		for _, key := range sourceValue.MapKeys() {
			// 确保键和值的类型兼容
			if key.Type().AssignableTo(keyType) == false {
				return valueInvalidErr
			}

			valueVal := sourceValue.MapIndex(key)
			if valueVal.Type().AssignableTo(valueTypeElem) {
				destValue.SetMapIndex(key, valueVal)
			} else {
				// 创建目标 Map 的值，并递归调用 setFieldValue 进行赋值
				val := reflect.New(valueTypeElem).Elem()
				err := SetValue(val, valueVal.Interface())
				if err != nil {
					return err
				}
				destValue.SetMapIndex(key, val)
			}
		}
		return nil
	case reflect.Slice:
		if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
			return valueInvalidErr
		}

		destLen := destValue.Len()
		sourceLen := sourceValue.Len()
		total := destLen + sourceLen
		if destValue.IsNil() {
			newSlice := reflect.MakeSlice(fieldType, sourceLen, sourceLen)
			destValue.Set(newSlice)
		} else if destValue.Cap() < total {
			// 如果目标切片不为空且容量不足，则创建一个新的切片并复制数据
			newSlice := reflect.MakeSlice(fieldType, total, total)
			reflect.Copy(newSlice, destValue)
			destValue.Set(newSlice)
		}
		// 如果目标切片的长度小于总长度，则调整其长度
		if destValue.Len() < total {
			destValue.Set(destValue.Slice(0, total))
		}

		for i := 0; i < sourceLen; i++ {
			err := SetValue(destValue.Index(destLen+i), sourceValue.Index(i).Interface())
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		// 处理结构体类型
		if sourceValue.Kind() != reflect.Map {
			return valueInvalidErr
		}

		// 处理 map[string]any 类型
		for i := 0; i < fieldType.NumField(); i++ {
			structField := destValue.Field(i)
			fieldName := fieldType.Field(i).Name
			// 先尝试使用 tag name 获取值，如果没有则尝试使用字段名
			tagName := fieldType.Field(i).Tag.Get("json")
			tagName, _, _ = strings.Cut(tagName, ",")
			if tagName == "-" {
				continue
			}
			var mapValue reflect.Value
			if tagName != "" {
				mapValue = sourceValue.MapIndex(reflect.ValueOf(tagName))
			}
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
		return valueInvalidErr
	}
	return nil
}
