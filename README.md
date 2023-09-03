# hgee

基于 `net/http` 进行开发的 Web 框架

示例：

```go
package main

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
	"os"
	"path"
)

func main() {
	server := hgee.NewHttpServer("0.0.0.0", 8989)
	
	// 注册一个路由转换器
	hgee.RegisterConverter("string", `([A-Za-z]+)`)
	
	// 注册一个路径
	server.Router.UrlPatterns("/test1", hgee.AsView{GET: Test1})
	
	// 创建一个二级子路由
	// var testRouter *hgee.Router = server.Router.Include("/test")
	testRouter := server.Router.Include("/test")
	testRouter.UrlPatterns("/test2", hgee.AsView{GET: Test2}) // 添加路由
	
	// 创建一个三级子路由
	test3Router := testRouter.Include("/test3")
	test3Router.UrlPatterns("/test3", hgee.AsView{GET: Test3}) // 嵌套子路由
	
	// Path 路由传参
	server.Router.UrlPatterns("/path_params_int/<int:id>", hgee.AsView{GET: TestPathParamsInt})
	// 字符串
	server.Router.UrlPatterns("/path_params_str/<str:name>", hgee.AsView{GET: TestPathParamsStr})
	
	// 两个同名参数时
	server.Router.UrlPatterns("/path_params_strs/<str:name>/<str:name>", hgee.AsView{GET: TestPathParamsStrs})
	
	// Query 传参
	server.Router.UrlPatterns("/query_params", hgee.AsView{GET: TestQueryParams})
	
	// Body 传参
	server.Router.UrlPatterns("/body_params", hgee.AsView{GET: TestBodyParams})
	
	// 使用自定义路由转换器获取参数
	server.Router.UrlPatterns("/converter_params/<string:name>", hgee.AsView{GET: TestConverterParamsStrs})
	
	// 从上下文中获取数据
	server.Router.UrlPatterns("/test_context/<string:name>", hgee.AsView{GET: TestContext})
	
	// 中间件
	// 注册请求中间件
	server.MiddleWares.BeforeRequest(RequestMiddleWare)
	// 注册视图中间件
	// server.MiddleWares.RegisterView()
	// 注册响应中间件
	// server.MiddleWares.RegisterResponse()
	
	// 静态文件
	server.Router.StaticUrlPatterns("/static_dir", "template")
	// testRouter.StaticUrlPatterns("/static_dir", "./template")
	
	// 返回文件
	server.Router.UrlPatterns("/test_file", hgee.AsView{GET: TestFile})
	
	err := server.RunServer()
	fmt.Println(err)
}

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
	
	msg := fmt.Sprintf("参数：%v 参数类型： %T", id, id)
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
	
	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
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
	
	msg1 := fmt.Sprintf("参数：%v 参数类型： %T\n", name1, name1)
	msg2 := fmt.Sprintf("参数：%v 参数类型： %T\n", name2, name2)
	fmt.Println(msg1)
	fmt.Println(msg2)
	return hgee.Data{http.StatusOK, "ok", []string{msg1, msg2}}
}

// 获取 Query 传参
func TestQueryParams(request *hgee.Request) any {
	name := request.QueryParams.Get("name")
	
	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
	fmt.Println(msg)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

// 获取Body传参
func TestBodyParams(request *hgee.Request) any {
	name := request.BodyParams["name"]
	
	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
	fmt.Println(msg)
	return hgee.Response{
		Status: http.StatusCreated,                // 返回指定响应状态码 404
		Data:   hgee.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

func TestConverterParamsStrs(request *hgee.Request) any {
	name := request.PathParams.Get("name")
	
	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
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
	return hgee.Response{
		Status: http.StatusOK,
		Data:   file, // 返回文件对象
	}
}

// 请求中间件
func RequestMiddleWare(request *hgee.Request) any {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}


```