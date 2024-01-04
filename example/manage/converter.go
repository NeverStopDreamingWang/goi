package manage

import "github.com/NeverStopDreamingWang/hgee"

func RegisterMyConverter() {
	// 注册一个路由转换器

	// 手机号
	hgee.RegisterConverter("phone", `(1[3456789]\d{9})`)
}
