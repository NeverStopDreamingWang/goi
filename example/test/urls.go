package test

import (
	"example/example"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 创建一个子路由
	testRouter := example.Server.Router.Include("/test", "测试路由")
	{
		// 注册一个路径
		testRouter.UrlPatterns("/test1", "测试路由1", goi.AsView{GET: Test1})

		testRouter.UrlPatterns("/test2", "", goi.AsView{GET: Test2}) // 添加路由

		// 创建一个三级子路由
		test3Router := testRouter.Include("/test3", "测试3级路由")
		{
			test3Router.UrlPatterns("/test3", "", goi.AsView{GET: Test3}) // 嵌套子路由
		}

		// Path 路由传参
		testRouter.UrlPatterns("/path_params_int/<int:id>", "Path路由传参", goi.AsView{GET: TestPathParamsInt})
		// 字符串
		testRouter.UrlPatterns("/path_params_str/<string:name>", "路由字符串传参", goi.AsView{GET: TestPathParamsStr})

		// 两个同名参数时
		testRouter.UrlPatterns("/path_params_strs/<string:name>/<string:name>", "", goi.AsView{GET: TestPathParamsStrs})

		// Query 传参
		testRouter.UrlPatterns("/query_params", "Query 传参", goi.AsView{GET: TestQueryParams})

		// Body 传参
		testRouter.UrlPatterns("/body_params", "Body 传参", goi.AsView{GET: TestBodyParams})

		// 参数验证
		testRouter.UrlPatterns("/params_valid", "参数验证", goi.AsView{POST: TestParasmValid})

		// 使用自定义路由转换器获取参数
		testRouter.UrlPatterns("/converter_params/<string:name>", "自定义路由转换器获取参数", goi.AsView{GET: TestConverterParamsStrs})

		// 从上下文中获取数据
		testRouter.UrlPatterns("/test_context/<string:name>", "从上下文中获取数据", goi.AsView{GET: TestContext})
	}

	// 静态路由
	// 静态文件
	example.Server.Router.StaticFilePatterns("/index.html", "测试静态文件", "template/html/index.html")

	// 静态目录
	example.Server.Router.StaticDirPatterns("/static", "测试静态目录", "template")
	// testRouter.StaticDirPatterns("/static", "./template")

	// 自定义方法
	// 返回文件
	example.Server.Router.UrlPatterns("/test_file", "返回文件", goi.AsView{GET: TestFile})

	// 异常处理
	example.Server.Router.UrlPatterns("/test_panic", "异常处理", goi.AsView{GET: TestPanic})
}
