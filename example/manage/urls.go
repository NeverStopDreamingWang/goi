package manage

func init() {
	// 注册路由
	// Server.Router.UrlPatterns("/test", hgee.AsView{GET: TestFunc})

	// 注册静态路径
	Server.Router.StaticUrlPatterns("/static", "template")
}
