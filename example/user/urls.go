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
		userRouter.Path("", "测试模型", goi.ViewSet{GET: listView, POST: createView})
		userRouter.Path("<int:user_id>", "测试模型查询", goi.ViewSet{GET: retrieveView, PUT: updateView, DELETE: deleteView})
		userRouter.Path("transaction", "测试模型查询", goi.ViewSet{POST: TestWithTransaction})
	}
}
