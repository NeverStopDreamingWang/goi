package auth

import (
	"example/example"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 子路由
	authRouter := example.Server.Router.Include("auth/", "认证模块")
	{
		authRouter.Path("login", "用户登录", goi.ViewSet{POST: loginView})
	}
}
