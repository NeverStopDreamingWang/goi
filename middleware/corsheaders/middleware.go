package corsheaders

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

const (
	// HEADER_ORIGIN 请求头,标识请求的来源(协议+域名+端口)
	HEADER_ORIGIN = "Origin"

	// HEADER_VARY 响应头,告知缓存服务器响应内容会根据指定的请求头而变化
	HEADER_VARY = "Vary"

	// ACCESS_CONTROL_ALLOW_ORIGIN 响应头,指定允许访问资源的来源
	// 值可以是具体的来源(如 https://example.com)或 * (允许所有来源)
	ACCESS_CONTROL_ALLOW_ORIGIN = "Access-Control-Allow-Origin"

	// ACCESS_CONTROL_EXPOSE_HEADERS 响应头,指定浏览器可以访问的响应头列表
	// 默认情况下浏览器只能访问简单响应头,其他响应头需要明确暴露
	ACCESS_CONTROL_EXPOSE_HEADERS = "Access-Control-Expose-Headers"

	// ACCESS_CONTROL_ALLOW_CREDENTIALS 响应头,指定是否允许发送凭证(Cookie、Authorization 等)
	// 值为 true 时,浏览器会在跨域请求中携带凭证
	ACCESS_CONTROL_ALLOW_CREDENTIALS = "Access-Control-Allow-Credentials"

	// ACCESS_CONTROL_ALLOW_HEADERS 响应头,指定预检请求中允许的请求头列表
	// 用于响应预检请求,告知浏览器实际请求可以使用哪些自定义请求头
	ACCESS_CONTROL_ALLOW_HEADERS = "Access-Control-Allow-Headers"

	// ACCESS_CONTROL_ALLOW_METHODS 响应头,指定预检请求中允许的 HTTP 方法列表
	// 用于响应预检请求,告知浏览器实际请求可以使用哪些 HTTP 方法
	ACCESS_CONTROL_ALLOW_METHODS = "Access-Control-Allow-Methods"

	// ACCESS_CONTROL_MAX_AGE 响应头,指定预检请求结果的缓存时长(单位: 秒)
	// 在缓存期内,浏览器不会为同一请求再次发送预检请求
	ACCESS_CONTROL_MAX_AGE = "Access-Control-Max-Age"

	// ACCESS_CONTROL_REQUEST_METHOD 请求头,预检请求中使用,指定实际请求将使用的 HTTP 方法
	ACCESS_CONTROL_REQUEST_METHOD = "Access-Control-Request-Method"

	// ACCESS_CONTROL_REQUEST_PRIVATE_NETWORK 请求头,Chrome 实验性特性
	// 用于请求访问私有网络资源(如从公网访问内网)
	ACCESS_CONTROL_REQUEST_PRIVATE_NETWORK = "Access-Control-Request-Private-Network"

	// ACCESS_CONTROL_ALLOW_PRIVATE_NETWORK 响应头,Chrome 实验性特性
	// 用于响应私有网络访问请求,允许从公网访问内网资源
	ACCESS_CONTROL_ALLOW_PRIVATE_NETWORK = "Access-Control-Allow-Private-Network"
)

// Default 返回带默认配置的 CORS 中间件实例
//
// 默认配置采用安全策略:
//   - 不允许所有来源,需要明确配置白名单
//   - 允许标准的 HTTP 方法(DELETE/GET/OPTIONS/PATCH/POST/PUT)
//   - 仅允许简单请求头(Accept/Accept-Language/Content-Language/Content-Type)
//   - 预检缓存 24 小时,减少预检请求
//   - 默认匹配所有路径
func Default() CorsMiddleWare {
	return CorsMiddleWare{
		// 不允许所有来源(需要明确配置白名单,更安全)
		CORS_ALLOW_ALL_ORIGINS: false,

		// 允许的来源白名单为空(需要手动添加,如 []string{"https://example.com"})
		CORS_ALLOWED_ORIGINS: []string{},

		// 允许的来源正则白名单为空(支持动态匹配,如 []string{"^https://.*\\.example\\.com$"})
		CORS_ALLOWED_ORIGIN_REGEXES: []string{},

		// 不允许发送凭证(Cookie/Authorization 等,避免 CSRF 风险)
		CORS_ALLOW_CREDENTIALS: false,

		// 不暴露额外的响应头(仅暴露简单响应头)
		CORS_EXPOSE_HEADERS: []string{},

		// 允许的请求头列表(默认为 CORS 规范定义的简单请求头)
		// 简单请求头: Accept, Accept-Language, Content-Language, Content-Type
		// 注意: 默认不包含 Authorization,需要时请手动添加
		CORS_ALLOW_HEADERS: []string{"Accept", "Accept-Language", "Content-Language", "Content-Type"},

		// 允许的 HTTP 方法(覆盖标准 RESTful API 操作)
		CORS_ALLOW_METHODS: []string{http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut},

		// 预检缓存时长 86400 秒(24小时),减少预检请求频率
		CORS_PREFLIGHT_MAX_AGE: 86400,

		// 默认匹配所有路径(可根据需要调整为特定 API 路径,如 "^/api/.*$")
		CORS_URLS_REGEX: "^.*$",

		// 不允许访问私有网络(Chrome 实验性特性,默认关闭)
		CORS_ALLOW_PRIVATE_NETWORK: false,
	}
}

// CorsMiddleWare 提供跨域资源共享(CORS)支持的中间件,实现 goi.MiddleWare 接口
//
// 主要功能:
//   - 来源控制: 支持白名单(精确匹配)、正则匹配、或允许所有来源
//   - 预检处理: 自动处理 OPTIONS 预检请求,返回 200 和相应的 CORS 响应头
//   - 凭证支持: 可配置是否允许携带 Cookie/Authorization 等凭证
//   - 响应头控制: 灵活配置允许的请求头、方法、暴露的响应头
//   - 路径过滤: 通过正则表达式控制哪些路径启用 CORS
//   - 缓存优化: 支持设置预检结果缓存时长,减少预检请求
//   - 私网访问: 支持 Chrome 私有网络访问实验性特性
//
// 参考 django-cors-headers 的行为实现,兼容标准 CORS 规范
// 使用 Default() 获取推荐的默认配置,或根据需要自定义 CORS 策略
type CorsMiddleWare struct {
	// 是否允许所有来源
	// 启用后返回 Access-Control-Allow-Origin: *
	// 注意: 当 CORS_ALLOW_CREDENTIALS=true 时,不会返回 *,而是回显具体 Origin
	CORS_ALLOW_ALL_ORIGINS bool

	// 允许的来源白名单(精确匹配)
	// 示例: []string{"https://example.com", "https://app.example.com"}
	CORS_ALLOWED_ORIGINS []string

	// 允许的来源正则白名单(正则匹配)
	// 示例: []string{"^https://.*\\.example\\.com$"} 匹配所有 example.com 的子域名
	CORS_ALLOWED_ORIGIN_REGEXES []string

	// 是否允许发送凭证
	// 启用后返回 Access-Control-Allow-Credentials: true
	// 允许浏览器发送 Cookie、Authorization Header、TLS 客户端证书等
	CORS_ALLOW_CREDENTIALS bool

	// 暴露给浏览器的非简单响应头列表
	// 对应 Access-Control-Expose-Headers
	// 示例: []string{"X-Custom-Header", "X-Request-ID"}
	CORS_EXPOSE_HEADERS []string

	// 预检响应中允许的请求头列表
	// 对应 Access-Control-Allow-Headers
	// 示例: []string{"Content-Type", "Authorization", "X-Custom-Header"}
	CORS_ALLOW_HEADERS []string

	// 预检响应中允许的方法列表
	// 对应 Access-Control-Allow-Methods
	// 方法名应使用大写,如 GET/POST/PUT/DELETE
	CORS_ALLOW_METHODS []string

	// 预检结果缓存时长(单位: 秒)
	// 对应 Access-Control-Max-Age
	// 0 表示不设置,浏览器每次都需要发送预检请求
	// 建议设置为 3600(1小时)或 86400(24小时)以减少预检请求
	CORS_PREFLIGHT_MAX_AGE int

	// 启用 CORS 的 URL 路径正则
	// 仅匹配该正则的路径会附加 CORS 响应头
	// 示例: "^/api/.*$" 仅对 /api/ 开头的路径启用 CORS
	CORS_URLS_REGEX string

	// 是否允许访问私有网络(Chrome 实验性特性)
	// 当请求头携带 Access-Control-Request-Private-Network: true 时
	// 返回 Access-Control-Allow-Private-Network: true
	CORS_ALLOW_PRIVATE_NETWORK bool
}

// ProcessRequest 请求预处理,处理 CORS 预检请求
func (self CorsMiddleWare) ProcessRequest(request *goi.Request) interface{} {
	if !self.isCorsPathEnabled(request) {
		return nil
	}
	// 仅当带 Origin 头时才认为是 CORS 相关请求
	if request.Object.Header.Get(HEADER_ORIGIN) == "" {
		return nil
	}
	// 预检：OPTIONS 且带 Access-Control-Request-Method
	if isPreflight(request.Object) {
		resp := goi.Response{Status: http.StatusOK, Data: nil}
		resp.Header().Set("Content-Length", "0")
		return resp
	}
	return nil
}

// ProcessView 视图处理前(本中间件不处理)
func (self CorsMiddleWare) ProcessView(request *goi.Request) interface{} { return nil }

// ProcessException 异常处理(本中间件不处理)
func (self CorsMiddleWare) ProcessException(request *goi.Request, exception any) interface{} {
	return nil
}

// ProcessResponse 响应后处理,设置 CORS 相关的响应头
func (self CorsMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
	if !self.isCorsPathEnabled(request) {
		return
	}
	origin := request.Object.Header.Get(HEADER_ORIGIN)
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
	if self.CORS_ALLOW_ALL_ORIGINS && !self.CORS_ALLOW_CREDENTIALS {
		response.Header().Set(ACCESS_CONTROL_ALLOW_ORIGIN, "*")
	} else {
		response.Header().Set(ACCESS_CONTROL_ALLOW_ORIGIN, origin)
		if self.CORS_ALLOW_CREDENTIALS {
			response.Header().Set(ACCESS_CONTROL_ALLOW_CREDENTIALS, "true")
		}
	}

	// Expose-Headers（非预检也可返回）
	if len(self.CORS_EXPOSE_HEADERS) > 0 {
		response.Header().Set(ACCESS_CONTROL_EXPOSE_HEADERS, strings.Join(self.CORS_EXPOSE_HEADERS, ", "))
	}

	// 预检专用响应头
	if isPreflight(request.Object) {
		if len(self.CORS_ALLOW_HEADERS) > 0 {
			response.Header().Set(ACCESS_CONTROL_ALLOW_HEADERS, strings.Join(self.CORS_ALLOW_HEADERS, ", "))
		}
		if len(self.CORS_ALLOW_METHODS) > 0 {
			response.Header().Set(ACCESS_CONTROL_ALLOW_METHODS, strings.Join(self.CORS_ALLOW_METHODS, ", "))
		}
		if self.CORS_PREFLIGHT_MAX_AGE > 0 {
			response.Header().Set(ACCESS_CONTROL_MAX_AGE, strconv.Itoa(self.CORS_PREFLIGHT_MAX_AGE))
		}
	}

	// Private Network（Chrome 私网访问实验标头）
	if self.CORS_ALLOW_PRIVATE_NETWORK && strings.EqualFold(request.Object.Header.Get(ACCESS_CONTROL_REQUEST_PRIVATE_NETWORK), "true") {
		response.Header().Set(ACCESS_CONTROL_ALLOW_PRIVATE_NETWORK, "true")
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
func (self CorsMiddleWare) isCorsPathEnabled(request *goi.Request) bool {
	pattern := self.CORS_URLS_REGEX
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
//   - 方法为 OPTIONS
//   - 携带 Access-Control-Request-Method 请求头
//
// 参数:
//   - r *http.Request: HTTP 请求对象
//
// 返回:
//   - bool: 是否为预检请求
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get(ACCESS_CONTROL_REQUEST_METHOD) != ""
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
func (self CorsMiddleWare) originAllowed(origin string, originURL *url.URL) bool {
	// 1. 允许所有来源
	if self.CORS_ALLOW_ALL_ORIGINS {
		return true
	}

	// 2. 检查 "null" 来源(用于本地文件、沙箱 iframe 等场景)
	if origin == "null" {
		for _, o := range self.CORS_ALLOWED_ORIGINS {
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
	for _, pat := range self.CORS_ALLOWED_ORIGIN_REGEXES {
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
func (self CorsMiddleWare) urlInWhitelist(originURL *url.URL) bool {
	for _, allowedOrigin := range self.CORS_ALLOWED_ORIGINS {
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
	if existing := h.Get(HEADER_VARY); existing == "" {
		h.Set(HEADER_VARY, token)
	} else {
		h.Set(HEADER_VARY, existing+", "+token)
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
	for _, v := range h.Values(HEADER_VARY) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
