package test

import (
	"fmt"
	"net/http"
	"os"
	"path"

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

func Test3(request *goi.Request) interface{} {
	return goi.Data{http.StatusOK, "test3 OK", "OK"}
}

// 请求传参
func TestParams(request *goi.Request) interface{} {
	var name string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("name", &name) // 路由传参
	// validationErr = request.QueryParams.Get("name", &name) // Query 传参
	// validationErr = request.BodyParams.Get("name", &name) // Body 传参
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Response{
		Status: http.StatusCreated,               // 返回指定响应状态码 404
		Data:   goi.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

// 参数验证
type testParamsValidParams struct {
	Username string            `name:"username" required:"string"`
	Password string            `name:"password" required:"string"`
	Age      string            `name:"age" required:"int"`
	Phone    string            `name:"phone" required:"phone"`
	Args     []string          `name:"args" optional:"slice"`
	Kwargs   map[string]string `name:"kwargs" optional:"map"`
}

// required 必传参数
// optional 可选
// 支持
// int *int []*int []... map[string]*int map[...]...
// ...

func TestParamsValid(request *goi.Request) interface{} {
	var params testParamsValidParams
	var validationErr goi.ValidationError
	// validationErr = request.PathParams.ParseParams(&params) // 路由传参
	// validationErr = request.QueryParams.ParseParams(&params) // Query 传参
	validationErr = request.BodyParams.ParseParams(&params) // Body 传参
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

		// 自定义返回
		// return goi.Response{
		// 	Status: http.StatusOK,
		// 	Data: goi.Data{
		// 		Status: http.StatusBadRequest,
		// 		Message:    "参数错误",
		// 		Data:   nil,
		// 	},
		// }
	}
	fmt.Println(params)
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  http.StatusOK,
			Message: "ok",
			Data:    nil,
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
func TestPathParamsStrs(request *goi.Request) interface{} {
	// 多个值
	nameSlice := request.PathParams["name"] // 返回一个 []interface{}
	fmt.Printf("%v,%T\n", nameSlice, nameSlice)
	name1 := nameSlice[0]
	name2 := nameSlice[1]

	msg1 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name1, name1)
	msg2 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name2, name2)
	fmt.Println(msg1)
	fmt.Println(msg2)
	return goi.Data{http.StatusOK, "ok", []string{msg1, msg2}}
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
	return goi.Data{http.StatusOK, "ok", msg}
}

// 上下文
func TestContext(request *goi.Request) interface{} {

	// 请求上下文
	// request.Object.Context() == request.Context
	// fmt.Println(request.Object.Context())
	// fmt.Println(request.Context)
	// fmt.Println("requestID", request.Object.Context().Value("requestID"))
	// fmt.Println("requestID", request.Context.Value("requestID"))
	requestID := request.Object.Context().Value("requestID")

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
	return goi.Data{http.StatusOK, msg, student}
}

// 返回文件
func TestFile(request *goi.Request) interface{} {
	absolutePath := path.Join(goi.Settings.BASE_DIR, "template/test.txt")
	file, err := os.Open(absolutePath)
	if err != nil {
		_ = file.Close()
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	// return file // 返回文件对象
	return file // 返回文件对象
}

// 异常处理
func TestPanic(request *goi.Request) interface{} {

	name := request.BodyParams["name"]

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)

	name_tmp := name[100]
	fmt.Println(name_tmp)
	return goi.Response{
		Status: http.StatusCreated,               // 返回指定响应状态码 404
		Data:   goi.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}
