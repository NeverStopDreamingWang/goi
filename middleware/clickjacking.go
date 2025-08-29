package middleware

import (
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

// XFrameMiddleWare
// 作用：为响应设置 X-Frame-Options 头，防止点击劫持。
// 逻辑：
// - 若响应已包含该头，则不覆盖
// - 若响应标记了 xframe_options_exempt 则跳过
// - 否则按设置项 X_FRAME_OPTIONS（"DENY"/"SAMEORIGIN"）设置，默认 "DENY"

type XFrameMiddleWare struct{}

func (XFrameMiddleWare) ProcessRequest(request *goi.Request) interface{}                  { return nil }
func (XFrameMiddleWare) ProcessView(request *goi.Request) interface{}                     { return nil }
func (XFrameMiddleWare) ProcessException(request *goi.Request, exception any) interface{} { return nil }

func (XFrameMiddleWare) ProcessResponse(request *goi.Request, response *goi.Response) {
	headers := response.Header()
	if headers.Get("X-Frame-Options") != "" {
		return
	}
	// 跳过豁免响应
	if exempt, ok := response.Header()["xframe_options_exempt"]; ok {
		if len(exempt) > 0 && strings.ToLower(exempt[0]) == "true" {
			return
		}
	}
	value := goi.Settings.X_FRAME_OPTIONS
	if value == "" {
		value = "DENY"
	}
	headers.Set("X-Frame-Options", strings.ToUpper(value))
}
