package security

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// Default 返回带默认配置的安全中间件实例
//
// 默认配置采用保守策略,仅启用基础安全响应头,不强制 HTTPS 重定向和 HSTS
// 适用于开发环境或需要逐步启用安全策略的场景
func Default() SecurityMiddleWare {
	return SecurityMiddleWare{
		// 启用 MIME 类型嗅探防护,防止浏览器将非可执行 MIME 类型解释为可执行内容
		SECURE_CONTENT_TYPE_NOSNIFF: true,

		// 设置同源窗口隔离策略,防止跨站点通过 window.opener 访问当前页面
		SECURE_CROSS_ORIGIN_OPENER_POLICY: "same-origin",

		// 不强制子域名使用 HTTPS(需配合 SECURE_HSTS_SECONDS > 0 才生效)
		SECURE_HSTS_INCLUDE_SUBDOMAINS: false,

		// 不启用 HSTS 预加载(需配合 SECURE_HSTS_SECONDS > 0 才生效)
		SECURE_HSTS_PRELOAD: false,

		// HSTS 未启用(设为 0 表示不下发 Strict-Transport-Security 响应头)
		SECURE_HSTS_SECONDS: 0,

		// 不设置 HTTPS 重定向豁免路径(空列表表示无豁免)
		SECURE_REDIRECT_EXEMPT: []string{},

		// 仅在同源请求时发送完整 Referer,跨域请求不发送
		SECURE_REFERRER_POLICY: []string{"same-origin"},

		// 不指定 HTTPS 重定向目标主机(留空则使用请求原始 Host)
		SECURE_SSL_HOST: "",

		// 不强制 HTTP 到 HTTPS 重定向(生产环境建议启用)
		SECURE_SSL_REDIRECT: false,
	}
}

// SecurityMiddleWare 提供 Web 应用安全防护的中间件,实现 goi.MiddleWare 接口
//
// 主要功能:
//   - HTTPS 强制重定向: 支持 HTTP 到 HTTPS 的 301 重定向,可配置目标主机和豁免路径(支持正则)
//   - HSTS 策略: 在 HTTPS 下自动添加 Strict-Transport-Security 响应头,支持 includeSubDomains 和 preload
//   - 安全响应头: 自动设置 X-Content-Type-Options: nosniff、Referrer-Policy、Cross-Origin-Opener-Policy
//   - 智能检测: 自动识别反向代理场景(X-Forwarded-Proto)
//
// 使用 Default() 获取推荐的默认配置,或根据需要自定义各项安全策略
type SecurityMiddleWare struct {
	// 是否在响应中添加 X-Content-Type-Options: nosniff，防止浏览器进行 MIME 嗅探
	SECURE_CONTENT_TYPE_NOSNIFF bool

	// 设置 Cross-Origin-Opener-Policy（例如 "same-origin"、"same-origin-allow-popups"、"unsafe-none"）
	// 用于隔离顶级浏览上下文，减少跨站泄露窗口引用的风险
	SECURE_CROSS_ORIGIN_OPENER_POLICY string

	// 当启用 HSTS 时，是否附加 includeSubDomains 指令，要求子域名也必须使用 HTTPS
	SECURE_HSTS_INCLUDE_SUBDOMAINS bool

	// 当启用 HSTS 时，是否附加 preload 指令，用于浏览器维护的 HSTS 预加载列表
	SECURE_HSTS_PRELOAD bool

	// Strict-Transport-Security 的 max-age（单位：秒）。仅当值 > 0 且本次请求是 HTTPS 时才会下发该响应头
	SECURE_HSTS_SECONDS int

	// 跳过 HTTPS 重定向的路径规则。支持正则；若正则编译失败则退化为前缀匹配
	SECURE_REDIRECT_EXEMPT []string

	// 设置 Referrer-Policy（多个策略按逗号拼接）。示例："no-referrer"、"same-origin"、"strict-origin-when-cross-origin" 等
	SECURE_REFERRER_POLICY []string

	// 启用 HTTPS 重定向时使用的目标 Host（留空则沿用请求 Host）
	SECURE_SSL_HOST string

	// 是否强制将 HTTP 请求 301 重定向到 HTTPS
	SECURE_SSL_REDIRECT bool
}

// isSecure 判断请求是否为 HTTPS（包含常见反代头 X-Forwarded-Proto=https）
//
// 参数:
//   - r *http.Request: HTTP请求对象
//
// 返回:
//   - bool: 是否为 HTTPS 请求
func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	xfp := strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
	return xfp == "https"
}

// shouldExempt 判断是否命中跳过 HTTPS 重定向的路径规则
//
// 参数:
//   - path string: 待匹配的 URL 路径
//   - patterns []string: 跳过规则列表
//
// 返回:
//   - bool: 是否命中跳过规则
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

// setDefaultHeader 仅当未设置时再设置响应头
//
// 参数:
//   - h http.Header: 响应头对象
//   - key string: 待设置的响应头键
//   - value string: 待设置的响应头值
func setDefaultHeader(h http.Header, key, value string) {
	if h.Get(key) == "" {
		h.Set(key, value)
	}
}

// ProcessRequest 请求预处理,执行 HTTPS 强制重定向
func (self SecurityMiddleWare) ProcessRequest(request *goi.Request) any {
	// HTTPS 重定向
	if self.SECURE_SSL_REDIRECT && !isSecure(request.Object) && !shouldExempt(request.Object.URL.Path, self.SECURE_REDIRECT_EXEMPT) {
		host := request.Object.Host
		if self.SECURE_SSL_HOST != "" {
			host = self.SECURE_SSL_HOST
		}
		target := "https://" + host + request.Object.URL.RequestURI()
		// 构造 301 响应并设置 Location
		response := goi.Response{Status: http.StatusMovedPermanently, Data: ""}
		response.Header().Set("Location", target)
		return response
	}
	return nil
}

// ProcessException 异常处理(本中间件不处理)
func (self SecurityMiddleWare) ProcessException(request *goi.Request, exception any) any {
	return nil
}

// ProcessResponse 响应后处理,设置安全相关的响应头
func (self SecurityMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
	headers := response.Header()

	// HSTS：仅在 HTTPS 请求、且未设置该响应头时生效
	if self.SECURE_HSTS_SECONDS > 0 && isSecure(request.Object) && headers.Get("Strict-Transport-Security") == "" {
		sts_header := fmt.Sprintf("max-age=%d", self.SECURE_HSTS_SECONDS)
		if self.SECURE_HSTS_INCLUDE_SUBDOMAINS {
			sts_header += "; includeSubDomains"
		}
		if self.SECURE_HSTS_PRELOAD {
			sts_header += "; preload"
		}
		setDefaultHeader(headers, "Strict-Transport-Security", sts_header)
	}

	// X-Content-Type-Options: nosniff（未设置时才添加）
	if self.SECURE_CONTENT_TYPE_NOSNIFF {
		setDefaultHeader(headers, "X-Content-Type-Options", "nosniff")
	}

	// Referrer-Policy：SECURE_REFERRER_POLICY 为 []string，按逗号拼接
	if len(self.SECURE_REFERRER_POLICY) > 0 {
		setDefaultHeader(headers, "Referrer-Policy", strings.Join(self.SECURE_REFERRER_POLICY, ","))
	}

	// Cross-Origin-Opener-Policy（未设置时才添加）
	if self.SECURE_CROSS_ORIGIN_OPENER_POLICY != "" {
		setDefaultHeader(headers, "Cross-Origin-Opener-Policy", self.SECURE_CROSS_ORIGIN_OPENER_POLICY)
	}
}
