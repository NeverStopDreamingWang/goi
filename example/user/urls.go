package user

import (
	"example/manage"
	"github.com/NeverStopDreamingWang/hgee"
)

func init() {
	// 创建一个子路由
	userRouter := manage.Server.Router.Include("/user")

	// 添加路由
	userRouter.UrlPatterns("/test", hgee.AsView{GET: UserTest})
	userRouter.UrlPatterns("/test_login", hgee.AsView{GET: Testlogin})
	userRouter.UrlPatterns("/test_auth", hgee.AsView{GET: TestAuth})
	userRouter.UrlPatterns("/user_model", hgee.AsView{GET: TestModelList, POST: TestModelCreate})
	userRouter.UrlPatterns("/user_model/<int:user_id>", hgee.AsView{GET: TestModelRetrieve, PUT: TestModelUpdate, DELETE: TestModelDelete})
	// userRouter.UrlPatterns("/test_model_first", hgee.AsView{GET: TestModelFirst})
	// userRouter.UrlPatterns("/test_model_select", hgee.AsView{GET: TestModelSelect})
	// userRouter.UrlPatterns("/test_model_insert", hgee.AsView{GET: TestModelInsert})
	// userRouter.UrlPatterns("/test_model_update", hgee.AsView{GET: TestModelUpdate})
	// userRouter.UrlPatterns("/test_model_delete", hgee.AsView{GET: TestModelDelete})
	userRouter.UrlPatterns("/test_cache_set", hgee.AsView{GET: TestCacheSet})
	userRouter.UrlPatterns("/test_cache_get", hgee.AsView{GET: TestCacheGet})
	userRouter.UrlPatterns("/test_cache_del", hgee.AsView{GET: TestCacheDel})
}
