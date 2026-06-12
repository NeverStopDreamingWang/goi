package corsheaders

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v2"
)

const (
	// HeaderOrigin 请求头,标识请求的来源(协议+域名+端口)
	HeaderOrigin = "Origin"

	// HeaderVary 响应头,告知缓存服务器响应内容会根据指定的请求头而变化
	HeaderVary = "Vary"

	// AccessControlAllowOrigin 响应头,指定允许访问资源的来源
	// 值可以是具体的来源(如 https://example.com)或 * (允许所有来源)
	AccessControlAllowOrigin = "Access-Control-Allow-Origin"

	// AccessControlExposeHeaders 响应头,指定浏览器可以访问的响应头列表
	// 默认情况下浏览器只能访问简单响应头,其他响应头需要明确暴露
	AccessControlExposeHeaders = "Access-Control-Expose-Headers"

	// AccessControlAllowCredentials 响应头,指定是否允许发送凭证(Cookie、Authorization 等)
	// 值为 true 时,浏览器会在跨域请求中携带凭证
	AccessControlAllowCredentials = "Access-Control-Allow-Credentials"

	// AccessControlAllowHeaders 响应头,指定预检请求中允许的请求头列表
	// 用于响应预检请求,告知浏览器实际请求可以使用哪些自定义请求头
	AccessControlAllowHeaders = "Access-Control-Allow-Headers"

	// AccessControlAllowMethods 响应头,指定预检请求中允许的 HTTP 方法列表
	// 用于响应预检请求,告知浏览器实际请求可以使用哪些 HTTP 方法
	AccessControlAllowMethods = "Access-Control-Allow-Methods"

	// AccessControlMaxAge 响应头,指定预检请求结果的缓存时长(单位: 秒)
	// 在缓存期内,浏览器不会为同一请求再次发送预检请求
	AccessControlMaxAge = "Access-Control-Max-Age"

	// AccessControlRequestMethod 请求头,预检请求中使用,指定实际请求将使用的 HTTP 方法
	AccessControlRequestMethod = "Access-Control-Request-Method"

	// AccessControlRequestPrivateNetwork 请求头,Chrome 实验性特性
	// 用于请求访问私有网络资源(如从公网访问内网)
	AccessControlRequestPrivateNetwork = "Access-Control-Request-Private-Network"

	// AccessControlAllowPrivateNetwork 响应头,Chrome 实验性特性
	// 用于响应私有网络访问请求,允许从公网访问内网资源
	AccessControlAllowPrivateNetwork = "Access-Control-Allow-Private-Network"
)

// Default 返回带默认配置的 CORS 中间件实例
//
// 默认配置采用安全策略:
//   - 不允许所有来源,需要明确配置白名单
//   - 允许标准的 HTTP 方法(DELETE/GET/Options/PATCH/POST/PUT)
//   - 仅允许简单请求头(Accept/Accept-Language/Content-Language/Content-Type)
//   - 预检缓存 24 小时,减少预检请求
//   - 默认匹配所有路径
func Default() CORSMiddleware {
	return CORSMiddleware{
		// 不允许所有来源(需要明确配置白名单,更安全)
		AllowAllOrigins: false,

		// 允许的来源白名单为空(需要手动添加,如 []string{"https://example.com"})
		AllowedOrigins: []string{},

		// 允许的来源正则白名单为空(支持动态匹配,如 []string{"^https://.*\\.example\\.com$"})
		AllowedOriginRegexes: []string{},

		// 不允许发送凭证(Cookie/Authorization 等,避免 CSRF 风险)
		AllowCredentials: false,

		// 不暴露额外的响应头(仅暴露简单响应头)
		ExposeHeaders: []string{},

		// 允许的请求头列表(默认为 CORS 规范定义的简单请求头)
		// 简单请求头: Accept, Accept-Language, Content-Language, Content-Type
		// 注意: 默认不包含 Authorization,需要时请手动添加
		AllowHeaders: []string{"Accept", "Accept-Language", "Content-Language", "Content-Type"},

		// 允许的 HTTP 方法(覆盖标准 RESTful API 操作)
		AllowMethods: []string{http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut},

		// 预检缓存时长 86400 秒(24小时),减少预检请求频率
		PreflightMaxAge: 86400,

		// 默认匹配所有路径(可根据需要调整为特定 API 路径,如 "^/api/.*$")
		URLRegex: "^.*$",

		// 不允许访问私有网络(Chrome 实验性特性,默认关闭)
		AllowPrivateNetwork: false,
	}
}

// CORSMiddleware 提供跨域资源共享(CORS)支持的中间件,实现 goi.Middleware 接口
//
// 主要功能:
//   - 来源控制: 支持白名单(精确匹配)、正则匹配、或允许所有来源
//   - 预检处理: 自动处理 Options 预检请求,返回 200 和相应的 CORS 响应头
//   - 凭证支持: 可配置是否允许携带 Cookie/Authorization 等凭证
//   - 响应头控制: 灵活配置允许的请求头、方法、暴露的响应头
//   - 路径过滤: 通过正则表达式控制哪些路径启用 CORS
//   - 缓存优化: 支持设置预检结果缓存时长,减少预检请求
//   - 私网访问: 支持 Chrome 私有网络访问实验性特性
//
// 参考 django-cors-headers 的行为实现,兼容标准 CORS 规范
// 使用 Default() 获取推荐的默认配置,或根据需要自定义 CORS 策略
type CORSMiddleware struct {
	// 是否允许所有来源
	// 启用后返回 Access-Control-Allow-Origin: *
	// 注意: 当 AllowCredentials=true 时,不会返回 *,而是回显具体 Origin
	AllowAllOrigins bool

	// 允许的来源白名单(精确匹配)
	// 示例: []string{"https://example.com", "https://app.example.com"}
	AllowedOrigins []string

	// 允许的来源正则白名单(正则匹配)
	// 示例: []string{"^https://.*\\.example\\.com$"} 匹配所有 example.com 的子域名
	AllowedOriginRegexes []string

	// 是否允许发送凭证
	// 启用后返回 Access-Control-Allow-Credentials: true
	// 允许浏览器发送 Cookie、Authorization Header、TLS 客户端证书等
	AllowCredentials bool

	// 暴露给浏览器的非简单响应头列表
	// 对应 Access-Control-Expose-Headers
	// 示例: []string{"X-Custom-Header", "X-Request-ID"}
	ExposeHeaders []string

	// 预检响应中允许的请求头列表
	// 对应 Access-Control-Allow-Headers
	// 示例: []string{"Content-Type", "Authorization", "X-Custom-Header"}
	AllowHeaders []string

	// 预检响应中允许的方法列表
	// 对应 Access-Control-Allow-Methods
	// 方法名应使用大写,如 GET/POST/PUT/DELETE
	AllowMethods []string

	// 预检结果缓存时长(单位: 秒)
	// 对应 Access-Control-Max-Age
	// 0 表示不设置,浏览器每次都需要发送预检请求
	// 建议设置为 3600(1小时)或 86400(24小时)以减少预检请求
	PreflightMaxAge int

	// 启用 CORS 的 URL 路径正则
	// 仅匹配该正则的路径会附加 CORS 响应头
	// 示例: "^/api/.*$" 仅对 /api/ 开头的路径启用 CORS
	URLRegex string

	// 是否允许访问私有网络(Chrome 实验性特性)
	// 当请求头携带 Access-Control-Request-Private-Network: true 时
	// 返回 Access-Control-Allow-Private-Network: true
	AllowPrivateNetwork bool
}

// ProcessRequest 请求预处理,处理 CORS 预检请求
func (self CORSMiddleware) ProcessRequest(request *goi.Request) any {
	if !self.isCorsPathEnabled(request) {
		return nil
	}
	// 仅当带 Origin 头时才认为是 CORS 相关请求
	if request.Object.Header.Get(HeaderOrigin) == "" {
		return nil
	}
	// 预检：Options 且带 Access-Control-Request-Method
	if isPreflight(request.Object) {
		resp := goi.Response{Status: http.StatusOK, Data: nil}
		resp.Header().Set("Content-Length", "0")
		return resp
	}
	return nil
}

// ProcessException 异常处理(本中间件不处理)
func (self CORSMiddleware) ProcessException(request *goi.Request, exception any) any {
	return nil
}

// ProcessResponse 响应后处理,设置 CORS 相关的响应头
func (self CORSMiddleware) ProcessResponse(request *goi.Request, response *goi.Response) {
	if !self.isCorsPathEnabled(request) {
		return
	}
	origin := request.Object.Header.Get(HeaderOrigin)
	if origin == "" {
		return
	}

	// 尝试解析 Origin URL(验证格式有效性)
	// 注意: "null" 是特殊值,不需要解析
	var originURL *url.URL
	var err error
	if origin != "null" {
		originURL, err = url.Parse(origin)
		if err != nil || originURL.Scheme == "" || originURL.Host == "" {
			// Origin 格式无效,跳过 CORS 处理
			return
		}
	}

	// 校验允许的来源
	if !self.originAllowed(origin, originURL) {
		return
	}

	// Vary: Origin（避免缓存污染）
	addVary(response.Header(), "Origin")

	// Allow-Origin / Credentials
	if self.AllowAllOrigins && !self.AllowCredentials {
		response.Header().Set(AccessControlAllowOrigin, "*")
	} else {
		response.Header().Set(AccessControlAllowOrigin, origin)
		if self.AllowCredentials {
			response.Header().Set(AccessControlAllowCredentials, "true")
		}
	}

	// Expose-Headers（非预检也可返回）
	if len(self.ExposeHeaders) > 0 {
		response.Header().Set(AccessControlExposeHeaders, strings.Join(self.ExposeHeaders, ", "))
	}

	// 预检专用响应头
	if isPreflight(request.Object) {
		if len(self.AllowHeaders) > 0 {
			response.Header().Set(AccessControlAllowHeaders, strings.Join(self.AllowHeaders, ", "))
		}
		if len(self.AllowMethods) > 0 {
			response.Header().Set(AccessControlAllowMethods, strings.Join(self.AllowMethods, ", "))
		}
		if self.PreflightMaxAge > 0 {
			response.Header().Set(AccessControlMaxAge, strconv.Itoa(self.PreflightMaxAge))
		}
	}

	// Private Network（Chrome 私网访问实验标头）
	if self.AllowPrivateNetwork && strings.EqualFold(request.Object.Header.Get(AccessControlRequestPrivateNetwork), "true") {
		response.Header().Set(AccessControlAllowPrivateNetwork, "true")
	}
}

// ---- 辅助方法 ----

// isCorsPathEnabled 判断当前请求路径是否启用 CORS
//
// 参数:
//   - request *goi.Request: 请求对象
//
// 返回:
//   - bool: 是否启用 CORS
func (self CORSMiddleware) isCorsPathEnabled(request *goi.Request) bool {
	pattern := self.URLRegex
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(request.Object.URL.Path)
}

// isPreflight 判断是否为 CORS 预检请求
//
// 预检请求的特征:
//   - 方法为 Options
//   - 携带 Access-Control-Request-Method 请求头
//
// 参数:
//   - r *http.Request: HTTP 请求对象
//
// 返回:
//   - bool: 是否为预检请求
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get(AccessControlRequestMethod) != ""
}

// originAllowed 判断请求来源是否被允许
//
// 检查顺序:
//  1. 如果允许所有来源,直接返回 true
//  2. 检查 "null" 来源(用于本地文件、沙箱 iframe)
//  3. 检查精确匹配白名单(比较 scheme + host)
//  4. 检查正则匹配白名单
//
// 参数:
//   - origin string: 请求的 Origin 头值
//   - originURL *url.URL: 解析后的 Origin URL(如果是 "null" 则为 nil)
//
// 返回:
//   - bool: 是否允许该来源
func (self CORSMiddleware) originAllowed(origin string, originURL *url.URL) bool {
	// 1. 允许所有来源
	if self.AllowAllOrigins {
		return true
	}

	// 2. 检查 "null" 来源(用于本地文件、沙箱 iframe 等场景)
	if origin == "null" {
		for _, o := range self.AllowedOrigins {
			if o == "null" {
				return true
			}
		}
		return false
	}

	// 3. 精确匹配白名单(比较 scheme + host)
	if originURL != nil && self.urlInWhitelist(originURL) {
		return true
	}

	// 4. 正则匹配白名单
	for _, pat := range self.AllowedOriginRegexes {
		if pat == "" {
			continue
		}
		if re, err := regexp.Compile(pat); err == nil && re.MatchString(origin) {
			return true
		}
	}

	return false
}

// urlInWhitelist 检查 URL 是否在白名单中(比较 scheme 和 host)
//
// 参数:
//   - originURL *url.URL: 请求的 Origin URL
//
// 返回:
//   - bool: 是否在白名单中
func (self CORSMiddleware) urlInWhitelist(originURL *url.URL) bool {
	for _, allowedOrigin := range self.AllowedOrigins {
		if allowedOrigin == "" || allowedOrigin == "null" {
			continue
		}
		allowedURL, err := url.Parse(allowedOrigin)
		if err != nil {
			continue
		}
		// 比较 scheme 和 host(包含端口)
		if allowedURL.Scheme == originURL.Scheme && allowedURL.Host == originURL.Host {
			return true
		}
	}
	return false
}

// addVary 向响应头添加 Vary 字段
//
// Vary 头用于告知缓存服务器,响应内容会根据指定的请求头而变化
// 对于 CORS,通常需要添加 "Vary: Origin" 以避免缓存污染
//
// 参数:
//   - h http.Header: 响应头对象
//   - token string: 要添加的 Vary 值(如 "Origin")
func addVary(h http.Header, token string) {
	if hasVaryToken(h, token) {
		return
	}
	if existing := h.Get(HeaderVary); existing == "" {
		h.Set(HeaderVary, token)
	} else {
		h.Set(HeaderVary, existing+", "+token)
	}
}

// hasVaryToken 检查响应头中是否已包含指定的 Vary 值
//
// 参数:
//   - h http.Header: 响应头对象
//   - token string: 要检查的 Vary 值
//
// 返回:
//   - bool: 是否已包含该值
func hasVaryToken(h http.Header, token string) bool {
	for _, v := range h.Values(HeaderVary) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
