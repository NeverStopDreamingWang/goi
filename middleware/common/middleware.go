package common

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// Default 返回带默认配置的通用中间件实例
//
// 默认配置不启用任何功能,需要根据实际需求手动配置
// 适用于需要精细控制站点行为的场景
func Default() CommonMiddleWare {
	return CommonMiddleWare{
		// 不启用 www 前缀重定向(保持原始域名访问)
		PREPEND_WWW: false,

		// 不禁止任何 User-Agent(允许所有客户端访问)
		DISALLOWED_USER_AGENTS: []string{},
	}
}

// CommonMiddleWare 提供通用站点行为控制的中间件,实现 goi.MiddleWare 接口
//
// 主要功能:
//   - User-Agent 过滤: 支持正则表达式匹配,拦截恶意爬虫或特定客户端(返回 403)
//   - WWW 前缀重定向: 统一域名访问方式,将 example.com 重定向到 www.example.com
//   - 协议感知: 自动识别 HTTPS 和反向代理场景,确保重定向 URL 正确
//
// 使用 Default() 获取默认配置,或根据需要自定义过滤规则和重定向策略
type CommonMiddleWare struct {
	// 是否启用 www 前缀重定向
	// 启用后,将非 www 前缀的主机名 301 重定向到带 www 的主机名
	// 例如: example.com -> www.example.com
	// 注意: 自动保持原始协议(HTTP/HTTPS)
	PREPEND_WWW bool

	// 被禁止的 User-Agent 正则表达式列表
	// 匹配成功后直接返回 403 Forbidden
	// 示例: []string{"(?i)bot", "(?i)spider"} 可拦截常见爬虫
	DISALLOWED_USER_AGENTS []string
}

// isSecure 判断请求是否为 HTTPS(包含常见反代头 X-Forwarded-Proto=https)
//
// 参数:
//   - r *http.Request: HTTP 请求对象
//
// 返回:
//   - bool: 是否为 HTTPS 请求
func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// ProcessRequest 请求预处理,执行 User-Agent 过滤和 WWW 前缀重定向
func (self CommonMiddleWare) ProcessRequest(request *goi.Request) any {
	// 1) 禁止 User-Agent
	if ua := request.Object.Header.Get("User-Agent"); ua != "" {
		for _, pat := range self.DISALLOWED_USER_AGENTS {
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
	if self.PREPEND_WWW {
		host := request.Object.Host
		if host != "" && !strings.HasPrefix(strings.ToLower(host), "www.") {
			// 根据是否安全连接决定 scheme
			scheme := "http"
			if isSecure(request.Object) {
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

// ProcessException 异常处理(本中间件不处理)
func (self CommonMiddleWare) ProcessException(request *goi.Request, exception any) any {
	return nil
}

// ProcessResponse 响应后处理(本中间件不处理)
func (self CommonMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
}
