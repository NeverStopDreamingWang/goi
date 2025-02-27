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
		testRouter.UrlPatterns("test1", "测试路由1", goi.AsView{GET: Test1})

		// 创建一个三级子路由
		paramsRouter := testRouter.Include("params/", "传参")
		{
			paramsRouter.UrlPatterns("path/<string:name>", "路由传参", goi.AsView{GET: TestPathParams})

			paramsRouter.UrlPatterns("query", "query 查询字符串传参", goi.AsView{GET: TestQueryParams})

			paramsRouter.UrlPatterns("body", "body 请求体传参", goi.AsView{GET: TestBodyParams})

			testRouter.UrlPatterns("valid", "参数验证", goi.AsView{POST: TestParamsValid})

			testRouter.UrlPatterns("test_phone/<phone:phone>", "测试路由转换器", goi.AsView{GET: TestPhone})

			testRouter.UrlPatterns("params_strs", "两个同名参数时", goi.AsView{GET: TestParamsStrs})

			testRouter.UrlPatterns("converter/<string:name>", "自定义路由转换器获取参数", goi.AsView{GET: TestConverterParamsStrs})

			testRouter.UrlPatterns("context/<string:name>", "从上下文中获取数据", goi.AsView{GET: TestContext})
		}
	}

	// 静态路由
	// 静态文件
	example.Server.Router.StaticFilePatterns("index.html", "测试静态文件", "template/html/index.html")

	// 静态目录
	testRouter.StaticDirPatterns("static", "测试静态目录", "template")

	// embed.FS 静态文件
	testRouter.StaticFilePatternsFS("index1.html", "测试静态FS文件", template.IndexHtml)

	// embed.FS 静态目录
	testRouter.StaticDirPatternsFS("static1", "测试静态FS目录", template.Html)

	// 自定义方法
	// 返回文件
	testRouter.UrlPatterns("test_file", "返回文件", goi.AsView{GET: TestFile})

	// 异常处理
	testRouter.UrlPatterns("test_panic", "异常处理", goi.AsView{GET: TestPanic})

	// 缓存
	testRouter.UrlPatterns("cache", "测试缓存", goi.AsView{GET: TestCacheGet, POST: TestCacheSet, DELETE: TestCacheDel})
}
