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

// 请求响应数据
type Response struct {
	Status int
	Data   interface{}
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
