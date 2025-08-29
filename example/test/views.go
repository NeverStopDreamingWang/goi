package test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"example/user"

	"github.com/NeverStopDreamingWang/goi"
)

func Test1(request *goi.Request) interface{} {
	resp := map[string]interface{}{
		"status": http.StatusOK,
		"msg":    "test1 OK",
		"data":   "OK",
	}
	return resp
}

// query 查询字符串传参
func TestQueryParams(request *goi.Request) interface{} {
	var name string
	var queryParams goi.Params
	var validationErr goi.ValidationError
	queryParams = request.QueryParams()

	validationErr = queryParams.Get("name", name)
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Response{
		Status: http.StatusCreated, // 返回指定响应状态码 404
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: msg,
			Results: "",
		}, // 响应数据 ""
	}
}

// body 请求体传参
func TestBodyParams(request *goi.Request) interface{} {
	var name string
	var bodyParams goi.Params
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams()

	validationErr = bodyParams.Get("name", name)
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Response{
		Status: http.StatusCreated, // 返回指定响应状态码 404
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: msg,
			Results: "",
		}, // 响应数据 ""
	}
}

// 路由传参
func TestPathParams(request *goi.Request) interface{} {
	var name string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("name", &name) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Response{
		Status: http.StatusCreated, // 返回指定响应状态码 404
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: msg,
			Results: "",
		}, // 响应数据 ""
	}
}

// 参数验证
type testParamsValidParams struct {
	Username string            `name:"username" type:"string" required:"true"`
	Password string            `name:"password" type:"string" required:"true"`
	Age      string            `name:"age" type:"int" required:"true"`
	Phone    string            `name:"phone" type:"phone" required:"true"`
	Args     []string          `name:"args" type:"slice"`
	Kwargs   map[string]string `name:"kwargs" type:"map"`
}

// type:"name" 字段类型, name 验证器名称
// required:"bool" 字段是否必填，bool 布尔值 false 默认可选，true 必传参数
// allow_null:"bool" 字段值是否允许为空，bool 布尔值 false 默认不允许，true 允许为空。当 required != true 并且 allow_null 未设置时，参数值可以为 null
// 支持
// int *int []*int []... map[string]*int map[...]...
// ...

func TestParamsValid(request *goi.Request) interface{} {
	var params testParamsValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

		// 自定义返回
		// return goi.Response{
		// 	Status: http.StatusOK,
		// 	Data: goi.Data{
		// 		Code: http.StatusBadRequest,
		// 		Message:    "参数错误",
		// 		Results:   nil,
		// 	},
		// }
	}
	fmt.Println(params)
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: "ok",
			Results: nil,
		},
	}
}

// 测试手机号路由转换器
func TestPhone(request *goi.Request) interface{} {
	var phone string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("phone", &phone)
	if validationErr != nil {
		return validationErr.Response()
	}
	resp := map[string]interface{}{
		"status": http.StatusOK,
		"msg":    phone,
		"data":   "OK",
	}
	return resp
}

// 两个同名参数时
func TestParamsStrs(request *goi.Request) interface{} {
	var validationErr goi.ValidationError

	var nameSlice []string
	// 多个值
	validationErr = request.BodyParams().Get("name", nameSlice) // 返回一个 []interface{}
	if validationErr != nil {
		return validationErr.Response()
	}

	fmt.Printf("%v,%T\n", nameSlice, nameSlice)
	name1 := nameSlice[0]
	name2 := nameSlice[1]

	msg1 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name1, name1)
	msg2 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name2, name2)
	fmt.Println(msg1)
	fmt.Println(msg2)
	return goi.Data{
		Code:    http.StatusOK,
		Message: "ok",
		Results: []string{msg1, msg2},
	}
}

func TestConverterParamsStrs(request *goi.Request) interface{} {
	var name string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("name", &name)
	if validationErr != nil {
		return validationErr.Response()
	}

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Data{
		Code:    http.StatusOK,
		Message: "ok",
		Results: msg,
	}
}

// 上下文
func TestContext(request *goi.Request) interface{} {
	// 请求上下文
	requestID := request.Object.Context().Value("requestID")

	fmt.Println("requestID", requestID)

	// 请求参数
	userInfo := user.UserModel{}
	request.Params.Get("user", &userInfo)
	fmt.Println("user", userInfo)

	// fmt.Println("PathParams", request.PathParams)
	var name string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("name", &name)
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("test %s %T", name, name)

	type Student struct {
		Requestid interface{} `json:"requestid"`
		Name      string      `json:"name"`
		Age       int         `json:"age"`
	}
	student := &Student{
		Requestid: requestID,
		Name:      "asda",
		Age:       12,
	}
	return goi.Data{
		Code:    http.StatusOK,
		Message: msg,
		Results: student,
	}
}

// 返回文件
func TestFile(request *goi.Request) interface{} {
	absolutePath := filepath.Join(goi.Settings.BASE_DIR, "template/test.txt")
	file, err := os.Open(absolutePath)
	if err != nil {
		_ = file.Close()
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败",
		}
	}
	// return file // 返回文件对象
	return file // 返回文件对象
}

// 异常处理
func TestPanic(request *goi.Request) interface{} {
	var bodyParams goi.Params
	bodyParams = request.BodyParams()
	name := bodyParams["name"]

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)

	panic(name)

	return goi.Response{
		Status: http.StatusCreated, // 返回指定响应状态码 404
		Data: goi.Data{
			Code:    http.StatusOK,
			Message: msg,
			Results: "",
		}, // 响应数据 null
	}
}
