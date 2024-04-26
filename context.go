package goi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
)

type Request struct {
	Object      *http.Request
	Context     context.Context // 请求上下文
	PathParams  metaValues      // 路由参数
	QueryParams metaValues      // 查询字符串参数
	BodyParams  metaValues      // Body 传参
}

// 解析请求参数
func (request *Request) parseRequestParams() {
	var err error
	// 解析 Query 参数
	for name, values := range request.Object.URL.Query() {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := parseValue(paramType, value)
			request.QueryParams[name] = append(request.QueryParams[name], data)
		}
	}
	// 解析 Body 参数
	err = request.Object.ParseMultipartForm(32 << 20)
	if err != nil {
		panic(fmt.Sprintf("解析 Body 错误: %v\n", err))
	}
	for name, values := range request.Object.Form {
		for _, value := range values {
			paramType := reflect.TypeOf(value).String()
			data := parseValue(paramType, value)
			request.BodyParams[name] = append(request.BodyParams[name], data)
		}
	}

	// 解析 json 数据
	body, err := io.ReadAll(request.Object.Body)
	if err != nil {
		panic(fmt.Sprintf("读取 Body 错误: %v\n", err))
	}
	if len(body) != 0 {
		jsonData := make(map[string]interface{})
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			panic(fmt.Sprintf("解析 json 错误: %v\n", err))
		}
		for name, value := range jsonData {
			request.BodyParams[name] = append(request.BodyParams[name], value)
		}
	}
}

// 解析参数到指定结构体
func (request Request) ParseParams(params interface{}) error {
	// 使用反射获取参数结构体的类型信息
	paramsType := reflect.TypeOf(params)

	// 如果参数不是指针或者不是结构体类型，则返回错误
	if paramsType.Kind() != reflect.Ptr || paramsType.Elem().Kind() != reflect.Struct {
		return errors.New("参数必须是结构体指针类型")
	}
	// 使用反射获取参数结构体的值信息
	paramsValue := reflect.ValueOf(params).Elem()

	// 遍历参数结构体的字段
	for i := 0; i < paramsType.Elem().NumField(); i++ {
		field := paramsType.Elem().Field(i)
		fieldName := field.Name // 字段名
		fieldType := field.Type // 字段类型

		// 根据字段名从请求的路径参数、查询参数和Body参数中获取对应的值
		var value interface{}
		var found bool

		// 先从路径参数中查找
		if value, found = request.PathParams[fieldName]; found {
			// 如果找到了值，则将其转换为字段类型，并设置到参数结构体中
			if err := setFieldValue(paramsValue.Field(i), fieldType, value); err != nil {
				return err
			}
			continue
		}

		// 然后从查询参数中查找
		if value, found = request.QueryParams[fieldName]; found {
			// 如果找到了值，则将其转换为字段类型，并设置到参数结构体中
			if err := setFieldValue(paramsValue.Field(i), fieldType, value); err != nil {
				return err
			}
			continue
		}

		// 最后从Body参数中查找
		if value, found = request.BodyParams[fieldName]; found {
			// 如果找到了值，则将其转换为字段类型，并设置到参数结构体中
			if err := setFieldValue(paramsValue.Field(i), fieldType, value); err != nil {
				return err
			}
			continue
		}

		// 如果都没有找到对应的值，则根据字段的标签判断是否为必填字段，如果是则返回错误
		if field.Tag.Get("valid") == "required" {
			return fmt.Errorf("缺少必填参数: %s", fieldName)
		}
	}

	return nil
}

// 将值设置到结构体字段中
func setFieldValue(field reflect.Value, fieldType reflect.Type, value interface{}) error {
	// 如果字段是字符串类型，则直接设置值
	if fieldType.Kind() == reflect.String {
		field.SetString(value.(string))
	} else if fieldType.Kind() == reflect.Int { // 如果字段是整数类型，则将值转换为整数类型
		intValue, err := strconv.Atoi(value.(string))
		if err != nil {
			return fmt.Errorf("字段类型不匹配，期望类型: %s, 实际类型: %T", fieldType.Name(), value)
		}
		field.SetInt(int64(intValue))
	} else if fieldType.Kind() == reflect.Float64 { // 如果字段是浮点数类型，则将值转换为浮点数类型
		floatValue, err := strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return fmt.Errorf("字段类型不匹配，期望类型: %s, 实际类型: %T", fieldType.Name(), value)
		}
		field.SetFloat(floatValue)
	} else {
		// 如果字段类型不是字符串、整数或浮点数，则返回错误
		return fmt.Errorf("不支持的字段类型: %s", fieldType.Name())
	}

	return nil
}

// 请求响应数据
type Response struct {
	Status int
	Data interface{}
}

// 内置响应数据格式
type Data struct {
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

type metaValues map[string][]interface{}

// 根据键获取一个值, 获取不到返回 nil
func (values metaValues) Get(key string, dest interface{}) error {
	value := values[key]
	if len(value) == 0 {
		return errors.New(fmt.Sprintf("%v is null", key))
	}
	// 获取目标变量的反射值
	destValue := reflect.ValueOf(dest)
	// 检查目标变量是否为指针类型
	if destValue.Kind() != reflect.Ptr {
		return errors.New("目标变量必须是指针类型")
	}
	// 尝试将值转换为目标变量的类型并赋值给目标变量
	// 尝试将值转换为目标变量的类型并赋值给目标变量
	switch destValue.Elem().Interface().(type) {
	case string:
		*dest.(*string) = value[0].(string)
	case int, int8, int16, int32, int64:
		intValue, err := strconv.ParseInt(value[0].(string), 10, destValue.Elem().Type().Bits())
		if err != nil {
			return err
		}
		destValue.Elem().SetInt(intValue)
	case uint, uint8, uint16, uint32, uint64:
		uintValue, err := strconv.ParseUint(value[0].(string), 10, destValue.Elem().Type().Bits())
		if err != nil {
			return err
		}
		destValue.Elem().SetUint(uintValue)
	case float32, float64:
		floatValue, err := strconv.ParseFloat(value[0].(string), destValue.Elem().Type().Bits())
		if err != nil {
			return err
		}
		destValue.Elem().SetFloat(floatValue)
	// case interface{}:
	// 	if destValue.Elem().Type() != reflect.TypeOf(value[0]) {
	// 		return fmt.Errorf("类型不匹配，期望类型：%v，实际类型：%T", destValue.Elem().Type(), value[0])
	// 	}
	// 	destValue.Elem().Set(reflect.ValueOf(value[0]))
	// case struct{}:
	// 	jsonData, err := json.Marshal(value[0])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	err = json.Unmarshal(jsonData, dest)
	// 	if err != nil {
	// 		return err
	// 	}
	default:
		return errors.New("不支持的目标变量类型")
	}
	return nil
}

// 设置一个键值，会覆盖原来的值
func (values metaValues) Set(key, value string) {
	values[key] = []interface{}{value}
}

// 根据键添加一个值
func (values metaValues) Add(key, value string) {
	values[key] = append(values[key], value)
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
