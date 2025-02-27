package parse

import (
	"errors"
	"net/http"
)

const MIME_MultipartPOSTForm = "multipart/form-data"

var Form formParsing
var FormPost formPostParsing
var FormMultipart formMultipartParsing

// defaultMultipartMemory 定义了处理multipart/form-data请求的默认内存限制(32MB)
const defaultMultipartMemory = 32 << 20

// formParsing 用于解析所有类型的表单数据，包括URL查询参数和POST表单数据
type formParsing struct{}

// formPostParsing 专门用于解析application/x-www-form-urlencoded类型的POST表单数据
type formPostParsing struct{}

// formMultipartParsing 专门用于解析multipart/form-data类型的表单数据
type formMultipartParsing struct{}

// Name 返回解析器的名称
func (formParsing) Name() string {
	return "form"
}

// Parse 解析HTTP请求中的所有表单数据
//
// 参数:
//   - request *http.Request: HTTP请求对象
//
// 返回:
//   - Params: 解析后的参数映射，键为参数名，值为参数值的切片
//   - error: 解析过程中的错误信息
//
// 注意：
//   - 会同时解析URL查询参数和POST表单数据
//   - 支持multipart/form-data和application/x-www-form-urlencoded格式
func (formParsing) Parse(request *http.Request) (Params, error) {
	var params Params
	var err error

	err = request.ParseForm()
	if err != nil {
		return nil, err
	}
	err = request.ParseMultipartForm(defaultMultipartMemory)
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return nil, err
	}

	for name, values := range request.Form {
		if len(values) == 1 {
			params[name] = values[0]
		} else {
			params[name] = values
		}
	}
	return params, nil
}

// Name 返回POST表单解析器的名称
func (formPostParsing) Name() string {
	return "form-urlencoded"
}

// Parse 解析application/x-www-form-urlencoded类型的POST表单数据
//
// 参数:
//   - request *http.Request: HTTP请求对象
//
// 返回:
//   - Params: 解析后的参数映射，键为参数名，值为参数值的切片
//   - error: 解析过程中的错误信息
func (formPostParsing) Parse(request *http.Request) (Params, error) {
	var params Params
	var err error

	err = request.ParseForm()
	if err != nil {
		return nil, err
	}

	for name, values := range request.PostForm {
		if len(values) == 1 {
			params[name] = values[0]
		} else {
			params[name] = values
		}
	}
	return params, nil
}

// Name 返回multipart表单解析器的名称
func (formMultipartParsing) Name() string {
	return "multipart/form-data"
}

// Parse 解析multipart/form-data类型的表单数据
//
// 参数:
//   - request *http.Request: HTTP请求对象
//
// 返回:
//   - Params: 解析后的参数映射，键为参数名，值为参数值的切片
//   - error: 解析过程中的错误信息
//
// 注意：
//   - 默认最大内存限制为32MB
//   - 超过内存限制的文件会被临时存储到磁盘
func (formMultipartParsing) Parse(request *http.Request) (Params, error) {
	var params Params
	var err error

	err = request.ParseMultipartForm(defaultMultipartMemory)
	if err != nil {
		return nil, err
	}

	for name, values := range request.Form {
		if len(values) == 1 {
			params[name] = values[0]
		} else {
			params[name] = values
		}
	}
	return params, nil
}
