package user

import (
	"example/example"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 创建一个子路由
	userRouter := example.Server.Router.Include("/user")
	{
		// 添加路由
		userRouter.UrlPatterns("/test", goi.AsView{GET: UserTest})
		userRouter.UrlPatterns("/test_login", goi.AsView{GET: Testlogin})
		userRouter.UrlPatterns("/test_auth", goi.AsView{GET: TestAuth})
		userRouter.UrlPatterns("/user_model", goi.AsView{GET: TestModelList, POST: TestModelCreate})
		userRouter.UrlPatterns("/user_model/<int:user_id>", goi.AsView{GET: TestModelRetrieve, PUT: TestModelUpdate, DELETE: TestModelDelete})
		userRouter.UrlPatterns("/test_cache", goi.AsView{GET: TestCacheGet, POST: TestCacheSet, DELETE: TestCacheDel})
	}
}
