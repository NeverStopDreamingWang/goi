package test

import (
	"example/example"
	"example/template"

	"github.com/NeverStopDreamingWang/goi"
)

var testRouter *goi.MetaRouter

func init() {
	// 创建一个子路由
	testRouter = example.Server.Router.Include("test/", "测试路由")
	{
		// 注册一个路径
		testRouter.Path("test1", "测试路由1", goi.ViewSet{GET: Test1})

		// 创建一个三级子路由
		paramsRouter := testRouter.Include("params/", "传参")
		{
			paramsRouter.Path("path/<string:name>", "路由传参", goi.ViewSet{GET: TestPathParams})

			paramsRouter.Path("query", "query 查询字符串传参", goi.ViewSet{GET: TestQueryParams})

			paramsRouter.Path("body", "body 请求体传参", goi.ViewSet{GET: TestBodyParams})

			testRouter.Path("valid", "参数验证", goi.ViewSet{POST: TestParamsValid})

			testRouter.Path("test_phone/<phone:phone>", "测试路由转换器", goi.ViewSet{GET: TestPhone})

			testRouter.Path("params_strs", "两个同名参数时", goi.ViewSet{GET: TestParamsStrs})

			testRouter.Path("converter/<string:name>", "自定义路由转换器获取参数", goi.ViewSet{GET: TestConverterParamsStrs})

			testRouter.Path("context/<string:name>", "从上下文中获取数据", goi.ViewSet{GET: TestContext})
		}
	}

	// 静态路由
	// 静态文件
	example.Server.Router.StaticFile("index.html", "测试静态文件", "template/html/index.html")

	// 静态目录
	testRouter.StaticDir("static", "测试静态目录", "template")

	// embed.FS 静态文件
	testRouter.StaticFileFS("index1.html", "测试静态FS文件", template.IndexHtml)

	// embed.FS 静态目录
	testRouter.StaticDirFS("static1", "测试静态FS目录", template.Html)

	// 自定义方法
	// 返回文件
	testRouter.Path("test_file", "返回文件", goi.ViewSet{GET: TestFile})

	// 异常处理
	testRouter.Path("test_panic", "异常处理", goi.ViewSet{GET: TestPanic})

	// 缓存
	testRouter.Path("cache", "测试缓存", goi.ViewSet{GET: TestCacheGet, POST: TestCacheSet, DELETE: TestCacheDel})
}
