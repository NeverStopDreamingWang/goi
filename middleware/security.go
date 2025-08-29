package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// SecurityMiddleWare
// - 可选强制 HTTPS 重定向（SECURE_SSL_REDIRECT, SECURE_SSL_HOST, SECURE_REDIRECT_EXEMPT）
// - 安全响应头：X-Content-Type-Options、Referrer-Policy、Cross-Origin-Opener-Policy
// - HSTS：Strict-Transport-Security（仅在 HTTPS 且 SECURE_HSTS_SECONDS>0 时）

type SecurityMiddleWare struct{}

// isSecure 判断请求是否为 HTTPS（包含常见反代头 X-Forwarded-Proto=https）
func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	xfp := strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
	return xfp == "https"
}

// shouldExempt 判断是否命中跳过 HTTPS 重定向的路径规则
// 与 Django 行为一致：匹配对象为去除前导斜杠的 path
func shouldExempt(path string, patterns []string) bool {
	p0 := strings.TrimLeft(path, "/")
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		// 优先正则，再退化为前缀匹配，兼容简单写法
		if re, err := regexp.Compile(pat); err == nil {
			if re.MatchString(p0) {
				return true
			}
		} else if strings.HasPrefix(p0, pat) {
			return true
		}
	}
	return false
}

// setDefaultHeader 仅当未设置时再设置响应头（模拟 Django 的 setdefault 行为）
func setDefaultHeader(h http.Header, key, value string) {
	if h.Get(key) == "" {
		h.Set(key, value)
	}
}

func (SecurityMiddleWare) ProcessRequest(request *goi.Request) interface{} {
	// HTTPS 重定向（与 Django 逻辑对齐）
	if goi.Settings.SECURE_SSL_REDIRECT && !isSecure(request.Object) && !shouldExempt(request.Object.URL.Path, goi.Settings.SECURE_REDIRECT_EXEMPT) {
		host := request.Object.Host
		if goi.Settings.SECURE_SSL_HOST != "" {
			host = goi.Settings.SECURE_SSL_HOST
		}
		target := "https://" + host + request.Object.URL.RequestURI()
		// 构造 301 响应并设置 Location（Django: HttpResponsePermanentRedirect）
		response := goi.Response{Status: http.StatusMovedPermanently, Data: ""}
		response.Header().Set("Location", target)
		return response
	}
	return nil
}

func (SecurityMiddleWare) ProcessView(request *goi.Request) interface{} { return nil }

func (SecurityMiddleWare) ProcessException(request *goi.Request, exception any) interface{} {
	return nil
}

func (SecurityMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
	headers := response.Header()

	// HSTS：仅在 HTTPS 请求、且未设置该响应头时生效
	if goi.Settings.SECURE_HSTS_SECONDS > 0 && isSecure(request.Object) && headers.Get("Strict-Transport-Security") == "" {
		sts_header := fmt.Sprintf("max-age=%d", goi.Settings.SECURE_HSTS_SECONDS)
		if goi.Settings.SECURE_HSTS_INCLUDE_SUBDOMAINS {
			sts_header += "; includeSubDomains"
		}
		if goi.Settings.SECURE_HSTS_PRELOAD {
			sts_header += "; preload"
		}
		setDefaultHeader(headers, "Strict-Transport-Security", sts_header)
	}

	// X-Content-Type-Options: nosniff（未设置时才添加）
	if goi.Settings.SECURE_CONTENT_TYPE_NOSNIFF {
		setDefaultHeader(headers, "X-Content-Type-Options", "nosniff")
	}

	// Referrer-Policy：SECURE_REFERRER_POLICY 为 []string，按逗号拼接
	if len(goi.Settings.SECURE_REFERRER_POLICY) > 0 {
		setDefaultHeader(headers, "Referrer-Policy", strings.Join(goi.Settings.SECURE_REFERRER_POLICY, ","))
	}

	// Cross-Origin-Opener-Policy（未设置时才添加）
	if goi.Settings.SECURE_CROSS_ORIGIN_OPENER_POLICY != "" {
		setDefaultHeader(headers, "Cross-Origin-Opener-Policy", goi.Settings.SECURE_CROSS_ORIGIN_OPENER_POLICY)
	}
}
