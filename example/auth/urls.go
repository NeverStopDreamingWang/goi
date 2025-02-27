package auth

import (
	"cloud_review/backend/cloud_review"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 子路由
	authRouter := cloud_review.ApiRouter.Include("auth/", "认证模块")
	{
		authRouter.UrlPatterns("captcha", "获取图片验证码", goi.AsView{GET: captchaView})
		authRouter.UrlPatterns("login", "用户登录", goi.AsView{POST: loginView})
	}
}
