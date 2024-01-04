package test

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
	"os"
	"path"
)

func Test1(request *hgee.Request) any {
	resp := map[string]any{
		"status": http.StatusOK,
		"msg":    "test1 OK",
		"data":   "OK",
	}
	return resp
}

func Test2(request *hgee.Request) any {
	resp := map[string]any{
		"status": http.StatusOK,
		"msg":    "test2 OK",
		"data":   "OK",
	}
	return resp
}

func Test3(request *hgee.Request) any {
	return hgee.Data{http.StatusOK, "test3 OK", "OK"}
}

func TestPathParamsInt(request *hgee.Request) any {
	// 路由传参 request.PathParams
	// ids, _ := request.PathParams["id"] // []any
	id := request.PathParams.Get("id").(int)

	if id == 0 {
		// 返回 hgee.Response
		return hgee.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", id, id)
	fmt.Println(msg)
	resp := map[string]any{
		"status": http.StatusOK,
		"msg":    msg,
		"data":   "OK",
	}
	return resp
}

func TestPathParamsStr(request *hgee.Request) any {
	name := request.PathParams.Get("name")

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

func TestPathParamsStrs(request *hgee.Request) any {
	// 多个值
	nameSlice := request.PathParams["name"] // 返回一个 []any
	fmt.Printf("%v,%T\n", nameSlice, nameSlice)
	name1 := nameSlice[0]
	name2 := nameSlice[1]

	msg1 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name1, name1)
	msg2 := fmt.Sprintf("参数: %v 参数类型:  %T\n", name2, name2)
	fmt.Println(msg1)
	fmt.Println(msg2)
	return hgee.Data{http.StatusOK, "ok", []string{msg1, msg2}}
}

// 获取 Query 传参
func TestQueryParams(request *hgee.Request) any {
	name := request.QueryParams.Get("name")

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

// 获取Body传参
func TestBodyParams(request *hgee.Request) any {
	name := request.BodyParams["name"]

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

func TestConverterParamsStrs(request *hgee.Request) any {
	name := request.PathParams.Get("name")

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return hgee.Data{http.StatusOK, "ok", msg}
}

// 上下文
func TestContext(request *hgee.Request) any {

	// 请求上下文
	// request.Object.Context() == request.Context
	// fmt.Println(request.Object.Context())
	// fmt.Println(request.Context)
	// fmt.Println("requestID", request.Object.Context().Value("requestID"))
	// fmt.Println("requestID", request.Context.Value("requestID"))
	requestID := request.Object.Context().Value("requestID")

	// fmt.Println("PathParams", request.PathParams)
	name := request.PathParams.Get("name")
	msg := fmt.Sprintf("test %s %T", name, name)

	type Student struct {
		Requestid any    `json:"requestid"`
		Name      string `json:"name"`
		Age       int    `json:"age"`
	}
	student := &Student{
		Requestid: requestID,
		Name:      "asda",
		Age:       12,
	}
	return hgee.Data{http.StatusOK, msg, student}
}

// 返回文件
func TestFile(request *hgee.Request) any {
	baseDir, err := os.Getwd()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   "获取目录失败！",
		}
	}
	absolutePath := path.Join(baseDir, "template/test.txt")
	file, err := os.Open(absolutePath)
	if err != nil {
		file.Close()
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	// return file // 返回文件对象
	return file // 返回文件对象
}

// 异常处理
func TestPanic(request *hgee.Request) any {

	name := request.BodyParams["name"]

	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)

	name_tmp := name[100]
	fmt.Println(name_tmp)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}
