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
		userRouter.UrlPatterns("", "测试模型", goi.AsView{GET: listView, POST: createView})
		userRouter.UrlPatterns("<int:user_id>", "测试模型查询", goi.AsView{GET: retrieveView, PUT: updateView, DELETE: deleteView})
		userRouter.UrlPatterns("transaction", "测试模型查询", goi.AsView{POST: TestWithTransaction})
	}
}
