package user

import (
	"example/example"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 创建一个子路由
	userRouter := example.Server.Router.Include("user/", "用户路由")
	{
		// 添加路由
		userRouter.UrlPatterns("test", "", goi.AsView{GET: UserTest})
		userRouter.UrlPatterns("test_login", "测试登录", goi.AsView{GET: Testlogin})
		userRouter.UrlPatterns("test_auth", "", goi.AsView{GET: TestAuth})

		userRouter.UrlPatterns("user_model", "测试模型", goi.AsView{GET: listView, POST: createView})
		userRouter.UrlPatterns("user_model/<int:user_id>", "测试模型查询", goi.AsView{GET: retrieveView, PUT: updateView, DELETE: deleteView})

		userRouter.UrlPatterns("test_cache", "测试缓存", goi.AsView{GET: TestCacheGet, POST: TestCacheSet, DELETE: TestCacheDel})
	}
}
