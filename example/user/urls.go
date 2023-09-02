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
}
