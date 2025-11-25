package clickjacking

import (
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// Default 返回带默认配置的点击劫持防护中间件实例
//
// 默认配置采用最严格的 DENY 策略,完全禁止页面被嵌入到 iframe 中
// 适用于大多数不需要被嵌入的 Web 应用
func Default() XFrameMiddleWare {
	return XFrameMiddleWare{
		// 使用最严格的 DENY 策略,禁止页面被任何站点通过 iframe/frame 嵌入
		X_FRAME_OPTIONS: "DENY",
	}
}

// XFrameMiddleWare 提供点击劫持防护的中间件,实现 goi.MiddleWare 接口
//
// 主要功能:
//   - 自动设置 X-Frame-Options 响应头,防止页面被恶意站点嵌入
//   - 支持 DENY(完全禁止)和 SAMEORIGIN(仅允许同源)两种策略
//   - 智能跳过: 已设置该头或标记了 xframe_options_exempt 的响应不会被覆盖
//
// 使用 Default() 获取推荐的默认配置(DENY),或根据需要设置为 SAMEORIGIN
type XFrameMiddleWare struct {
	// X-Frame-Options 策略
	// - "DENY": 禁止被任意页面通过 <frame>/<iframe> 嵌入(推荐)
	// - "SAMEORIGIN": 仅允许同源页面嵌入
	// 默认值: "DENY"
	X_FRAME_OPTIONS string
}

// ProcessRequest 请求预处理(本中间件不处理)
func (self XFrameMiddleWare) ProcessRequest(request *goi.Request) any { return nil }

// ProcessException 异常处理(本中间件不处理)
func (self XFrameMiddleWare) ProcessException(request *goi.Request, exception any) any {
	return nil
}

// ProcessResponse 响应后处理,设置 X-Frame-Options 响应头
func (self XFrameMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
	headers := response.Header()
	// 如果已设置 X-Frame-Options,则不覆盖
	if headers.Get("X-Frame-Options") != "" {
		return
	}
	// 跳过标记为豁免的响应
	if exempt, ok := response.Header()["xframe_options_exempt"]; ok {
		if len(exempt) > 0 && strings.ToLower(exempt[0]) == "true" {
			return
		}
	}
	// 设置 X-Frame-Options 值
	value := self.X_FRAME_OPTIONS
	if value == "" {
		value = "DENY"
	}
	headers.Set("X-Frame-Options", strings.ToUpper(value))
}
