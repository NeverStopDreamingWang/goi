package db

import (
	"errors"
	"reflect"

	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// RowScanner 扫描一行查询结果，*sql.Row 和 *sql.Rows 均实现此接口。
type RowScanner interface {
	// Scan 按查询字段顺序将当前行数据写入目标指针。
	Scan(dest ...any) error
}

// ResultWriter 扫描一行并写入结果；切片仅在扫描成功后追加，单行结果直接写入现有目标。
type ResultWriter func(scanner RowScanner) error

// PrepareSliceResult 检查切片目标并创建逐行写入回调
//
// 参数:
//   - model: any 查询模型，支持结构体或非 nil 的结构体指针
//   - queryResult: any 非 nil 的切片指针，元素为结构体、结构体指针或 map
//   - fields: []string 按查询列顺序排列的模型字段名，也用作 map 的键
//
// 返回:
//   - ResultWriter: rows.Next() 成功后传入 rows，扫描并追加一行结果
//   - error: 模型或结果目标无效时的错误
//
// 说明:
//   - 支持 nil 切片，不支持数组；追加结果，不清空已有内容
//   - 目标结构体中缺少或不可写的字段会被忽略
//   - map 以结构体字段名为键、模型字段类型为值；键值须可直接赋给目标 map，不做类型转换
func PrepareSliceResult(model any, queryResult any, fields []string) (ResultWriter, error) {
	modelValue, err := ModelStructValue(model)
	if err != nil {
		return nil, err
	}
	modelType := modelValue.Type()

	result := reflect.ValueOf(queryResult)
	if !result.IsValid() || result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := i18n.T("db.is_not_ptr", map[string]any{"name": "queryResult"})
		return nil, errors.New(isNotPtrErrorMsg)
	}
	if result.IsNil() {
		queryResultIsNilErrorMsg := i18n.T("db.query_result_is_nil", map[string]any{"name": "queryResult"})
		return nil, errors.New(queryResultIsNilErrorMsg)
	}
	result = result.Elem()
	if result.Kind() != reflect.Slice {
		isNotSlicePtrErrorMsg := i18n.T("db.is_not_slice", map[string]any{"name": "queryResult"})
		return nil, errors.New(isNotSlicePtrErrorMsg)
	}

	itemType := result.Type().Elem()
	itemIsPointer := false
	if itemType.Kind() == reflect.Ptr && itemType.Elem().Kind() == reflect.Struct {
		itemIsPointer = true
		itemType = itemType.Elem()
	}

	switch itemType.Kind() {
	case reflect.Struct:
		return func(scanner RowScanner) error {
			values := make([]any, len(fields))
			item := reflect.New(itemType).Elem()
			for i, fieldName := range fields {
				fieldValue := item.FieldByName(fieldName)
				if !fieldValue.IsValid() || !fieldValue.CanAddr() || !fieldValue.Addr().CanInterface() {
					values[i] = new(any)
				} else {
					values[i] = fieldValue.Addr().Interface()
				}
			}
			if err := scanner.Scan(values...); err != nil {
				return err
			}
			if itemIsPointer {
				result.Set(reflect.Append(result, item.Addr()))
			} else {
				result.Set(reflect.Append(result, item))
			}
			return nil
		}, nil
	case reflect.Map:
		return func(scanner RowScanner) error {
			values := make([]any, len(fields))
			item := reflect.MakeMap(itemType)
			for i, fieldName := range fields {
				field, _ := modelType.FieldByName(fieldName)
				values[i] = reflect.New(field.Type).Interface()
			}
			if err := scanner.Scan(values...); err != nil {
				return err
			}
			for i, fieldName := range fields {
				value := reflect.ValueOf(values[i]).Elem()
				item.SetMapIndex(reflect.ValueOf(fieldName), value)
			}
			result.Set(reflect.Append(result, item))
			return nil
		}, nil
	default:
		isNotStructPtrErrorMsg := i18n.T("db.is_not_slice_struct_ptr_or_map", map[string]any{"name": "queryResult"})
		return nil, errors.New(isNotStructPtrErrorMsg)
	}
}

// PrepareSingleResult 检查单行目标并创建写入回调
//
// 参数:
//   - model: any 查询模型，支持结构体或非 nil 的结构体指针
//   - queryResult: any 非 nil 的结构体指针，或已初始化的 map
//   - fields: []string 按查询列顺序排列的模型字段名，也用作 map 的键
//
// 返回:
//   - ResultWriter: 传入 row 或 rows，扫描并写入当前行
//   - error: 模型或结果目标无效时的错误
//
// 说明:
//   - 只写入指定字段，不清空目标；结构体中缺少或不可写的字段会被忽略
//   - map 以结构体字段名为键、模型字段类型为值；键值须可直接赋给目标 map，不做类型转换
func PrepareSingleResult(model any, queryResult any, fields []string) (ResultWriter, error) {
	result := reflect.ValueOf(queryResult)
	if !result.IsValid() {
		isNotPtrErrorMsg := i18n.T("db.is_not_ptr", map[string]any{"name": "queryResult"})
		return nil, errors.New(isNotPtrErrorMsg)
	}
	if result.Kind() == reflect.Map {
		if result.IsNil() {
			queryResultIsNilErrorMsg := i18n.T("db.query_result_is_nil", map[string]any{"name": "queryResult"})
			return nil, errors.New(queryResultIsNilErrorMsg)
		}
	} else {
		if result.Kind() != reflect.Ptr {
			isNotPtrErrorMsg := i18n.T("db.is_not_ptr", map[string]any{"name": "queryResult"})
			return nil, errors.New(isNotPtrErrorMsg)
		}
		if result.IsNil() {
			queryResultIsNilErrorMsg := i18n.T("db.query_result_is_nil", map[string]any{"name": "queryResult"})
			return nil, errors.New(queryResultIsNilErrorMsg)
		}
		result = result.Elem()
		if result.Kind() != reflect.Struct {
			isNotStructPtrErrorMsg := i18n.T("db.is_not_struct_ptr_or_map", map[string]any{"name": "queryResult"})
			return nil, errors.New(isNotStructPtrErrorMsg)
		}
	}

	modelValue, err := ModelStructValue(model)
	if err != nil {
		return nil, err
	}
	modelType := modelValue.Type()

	switch result.Kind() {
	case reflect.Struct:
		return func(scanner RowScanner) error {
			values := make([]any, len(fields))
			for i, fieldName := range fields {
				fieldValue := result.FieldByName(fieldName)
				if !fieldValue.IsValid() || !fieldValue.CanAddr() || !fieldValue.Addr().CanInterface() {
					values[i] = new(any)
				} else {
					values[i] = fieldValue.Addr().Interface()
				}
			}
			return scanner.Scan(values...)
		}, nil
	case reflect.Map:
		return func(scanner RowScanner) error {
			values := make([]any, len(fields))
			for i, fieldName := range fields {
				field, _ := modelType.FieldByName(fieldName)
				values[i] = reflect.New(field.Type).Interface()
			}
			if err := scanner.Scan(values...); err != nil {
				return err
			}
			for i, fieldName := range fields {
				value := reflect.ValueOf(values[i]).Elem()
				result.SetMapIndex(reflect.ValueOf(fieldName), value)
			}
			return nil
		}, nil
	default:
		isNotStructPtrErrorMsg := i18n.T("db.is_not_struct_ptr_or_map", map[string]any{"name": "queryResult"})
		return nil, errors.New(isNotStructPtrErrorMsg)
	}
}
