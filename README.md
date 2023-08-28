# hgee
基于 `net/http` 进行开发的 Web 框架

示例：
```go
package main

import (
	"fmt"
	"hgee"
	"hgee/urls"
	"net/http"
)

func main() {
	server := hgee.NewHttpServer()

	// hgee.RegisterConverter()

	// 注册一个路由转换器
	urls.RegisterConverter("string", `([A-Za-z]+)`)

	// 注册一个路径
	server.Router.UrlPatterns("/test1", hgee.AsView{GET: Test1})

	// 创建一个二级子路由
	// var testRouter *hgee.Router = server.Router.Include("/test")
	testRouter := server.Router.Include("/test")
	testRouter.UrlPatterns("/test2", hgee.AsView{GET: Test2}) // 添加路由

	// 创建一个三级子路由
	test3Router := testRouter.Include("/test3")
	test3Router.UrlPatterns("/test3", hgee.AsView{GET: Test3}) // 嵌套子路由

	// 添加一个带参数的路由
	server.Router.UrlPatterns("/params_int/<int:id>", hgee.AsView{GET: TestParamsInt})
	// 字符串
	server.Router.UrlPatterns("/params_str/<str:name>", hgee.AsView{GET: TestParamsStrs})

	// 两个同名参数时
	server.Router.UrlPatterns("/params_strs/<str:name>/<str:name>", hgee.AsView{GET: TestParamsStrs})

	// 使用自定义路由转换器获取参数
	server.Router.UrlPatterns("/converter_params/<string:name>", hgee.AsView{GET: TestConverterParamsStrs})

	// 从上下文中获取数据
	server.Router.UrlPatterns("/test_context/<string:name>", hgee.AsView{GET: TestContext})

	// 注册请求中间件
	server.MiddleWares.RegisterRequest(RequestMiddleWare)
	// 注册视图中间件
	// server.MiddleWares.RegisterView()
	// 注册响应中间件
	// server.MiddleWares.RegisterResponse()

	err := server.RunServer("0.0.0.0:8989")
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
	resp := map[string]any{
		"status": http.StatusOK,
		"msg":    "test3 OK",
		"data":   "OK",
	}
	return resp
}

func TestParamsInt(request *hgee.Request) any {
	// 路由传参 request.PathParams
	// ids, _ := request.PathParams["id"] // []any
	id := request.PathParams.Get("id").(string)
	msg := fmt.Sprintf("参数：%v 参数类型： %T", id, id)
	fmt.Println(msg)
	resp := map[string]any{
		"status": http.StatusOK,
		"msg":    msg,
		"data":   "OK",
	}
	return resp
}

func TestParamsStr(request *hgee.Request) any {
	name := request.PathParams.Get("name")

	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
	fmt.Println(msg)
	return hgee.Response{http.StatusOK, msg, ""}
}

func TestParamsStrs(request *hgee.Request) any {
	// 多个值
	nameSlice := request.PathParams["name"] // 返回一个 []any
	fmt.Printf("%v,%T\n", nameSlice, nameSlice)
	name1 := nameSlice[0]
	name2 := nameSlice[1]

	msg1 := fmt.Sprintf("参数：%v 参数类型： %T\n", name1, name1)
	msg2 := fmt.Sprintf("参数：%v 参数类型： %T\n", name2, name2)
	fmt.Println(msg1)
	fmt.Println(msg2)
	return hgee.Response{http.StatusOK, "ok", []string{msg1, msg2}}
}

func TestConverterParamsStrs(request *hgee.Request) any {
	name := request.PathParams.Get("name")

	msg := fmt.Sprintf("参数：%v 参数类型： %T", name, name)
	fmt.Println(msg)
	return hgee.Response{http.StatusOK, "ok", msg}
}

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
	return hgee.Response{http.StatusOK, msg, student}
}

// 请求中间件
func RequestMiddleWare(request *hgee.Request) any {
	// fmt.Println("请求中间件", request.Object.URL)
	return nil
}

```