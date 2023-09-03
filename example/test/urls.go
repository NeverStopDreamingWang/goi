package user

import (
	"example/manage"
	"github.com/NeverStopDreamingWang/hgee"
)

func init() {
	// 创建一个子路由
	testRouter := manage.Server.Router.Include("/test")

	// 注册一个路由转换器
	hgee.RegisterConverter("string", `([A-Za-z]+)`)

	// 注册一个路径
	testRouter.UrlPatterns("/test1", hgee.AsView{GET: Test1})

	testRouter.UrlPatterns("/test2", hgee.AsView{GET: Test2}) // 添加路由

	// 创建一个三级子路由
	test3Router := testRouter.Include("/test3")
	test3Router.UrlPatterns("/test3", hgee.AsView{GET: Test3}) // 嵌套子路由

	// Path 路由传参
	testRouter.UrlPatterns("/path_params_int/<int:id>", hgee.AsView{GET: TestPathParamsInt})
	// 字符串
	testRouter.UrlPatterns("/path_params_str/<str:name>", hgee.AsView{GET: TestPathParamsStr})

	// 两个同名参数时
	testRouter.UrlPatterns("/path_params_strs/<str:name>/<str:name>", hgee.AsView{GET: TestPathParamsStrs})

	// Query 传参
	testRouter.UrlPatterns("/query_params", hgee.AsView{GET: TestQueryParams})

	// Body 传参
	testRouter.UrlPatterns("/body_params", hgee.AsView{GET: TestBodyParams})

	// 使用自定义路由转换器获取参数
	testRouter.UrlPatterns("/converter_params/<string:name>", hgee.AsView{GET: TestConverterParamsStrs})

	// 从上下文中获取数据
	testRouter.UrlPatterns("/test_context/<string:name>", hgee.AsView{GET: TestContext})

	// 静态文件
	testRouter.StaticUrlPatterns("/static_dir", "template")
	// testRouter.StaticUrlPatterns("/static_dir", "./template")

	// 返回文件
	testRouter.UrlPatterns("/test_file", hgee.AsView{GET: TestFile})

	// 异常处理
	testRouter.UrlPatterns("/test_panic", hgee.AsView{GET: TestPanic})
}
