package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// CommonMiddleWare 对应 django.middleware.common.CommonMiddleware（去除 APPEND_SLASH 相关逻辑）
// 功能：
// - 拦截被禁用的 User-Agent（DISALLOWED_USER_AGENTS）
// - 当启用 PREPEND_WWW 时，将非 www 主机重定向到 www 前缀
type CommonMiddleWare struct{}

func (CommonMiddleWare) ProcessRequest(request *goi.Request) interface{} {
	// 1) 禁止 User-Agent
	if ua := request.Object.Header.Get("User-Agent"); ua != "" {
		for _, pat := range goi.Settings.DISALLOWED_USER_AGENTS {
			if pat == "" { // 跳过空模式
				continue
			}
			if re, err := regexp.Compile(pat); err == nil {
				if re.MatchString(ua) {
					return goi.Response{Status: http.StatusForbidden, Data: "Forbidden user agent"}
				}
			}
		}
	}

	// 2) PREPEND_WWW 重定向
	if goi.Settings.PREPEND_WWW {
		host := request.Object.Host
		if host != "" && !strings.HasPrefix(strings.ToLower(host), "www.") {
			// 根据是否安全连接决定 scheme
			scheme := "http"
			if isSecure(request.Object) { // 使用 security.go 中的辅助方法
				scheme = "https"
			}
			target := scheme + "://www." + host + request.Object.URL.RequestURI()
			response := goi.Response{Status: http.StatusMovedPermanently, Data: ""}
			response.Header().Set("Location", target)
			return response
		}
	}
	return nil
}

func (CommonMiddleWare) ProcessView(request *goi.Request) interface{} { return nil }

func (CommonMiddleWare) ProcessException(request *goi.Request, exception any) interface{} { return nil }

func (CommonMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
}
