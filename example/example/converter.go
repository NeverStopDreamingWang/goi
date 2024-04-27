package example

import "github.com/NeverStopDreamingWang/goi"

func init() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter("phone", `(1[3456789]\d{9})`)
}
