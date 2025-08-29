package goi

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// 中间件
// MiddleWare 定义四个阶段钩子：
// - ProcessRequest：请求阶段（顺序执行）；返回非 nil 则短路，内层不再进入
// - ProcessView：路由解析后、视图执行前（顺序执行）；返回非 nil 则短路，但响应阶段仍会全链执行
// - ProcessException：异常阶段（逆序执行）；
// - ProcessResponse：响应阶段（逆序执行、全链）；常用于追加/修改响应头
type MiddleWare interface {
	ProcessRequest(*Request) interface{}                // 请求中间件
	ProcessView(*Request) interface{}                   // 视图中间件
	ProcessException(*Request, interface{}) interface{} // 异常中间件
	ProcessResponse(*Request, *Response)                // 响应中间件
}

// processExceptionByMiddleware 逆序调用异常中间件，返回首个非 nil 的结果；若均不处理则返回 nil。
func (engine *Engine) processExceptionByMiddleware(request *Request, err interface{}) interface{} {
	for i := len(engine.MiddleWare) - 1; i >= 0; i-- {
		result := engine.MiddleWare[i].ProcessException(request, err)
		if result != nil {
			return toResponse(result)
		}
	}
	return nil
}

// loadMiddleware 构建“洋葱模型”中间件链，返回对 final 的包装函数。
// 执行语义（对齐 Django）：
// - 进入每层时先执行 ProcessRequest，可短路返回 Response；短路则内层不再进入
// - 调用内层 next 获取 Response
// - 退出每层时先进行异常捕获（若 panic，按逆序调用 ProcessException，首个非 nil 即接住）
// - 最后无论正/异常都调用该层的 ProcessResponse（逆序、全链，不短路）
func (engine *Engine) loadMiddleware(get_response getResponseFunc) getResponseFunc {
	handler := get_response

	// 由内向外包裹
	for i := len(engine.MiddleWare) - 1; i >= 0; i-- {
		handler = engine.wrapMiddleware(engine.MiddleWare[i], handler)
	}
	return handler
}
func (engine *Engine) wrapMiddleware(middleware MiddleWare, get_response getResponseFunc) getResponseFunc {
	return func(request *Request) (response Response) {
		defer func() {
			err := recover()
			if err != nil {
				response = engine.convertExceptionToResponse(request, err)
				return
			}
		}()

		var result interface{}
		// 请求阶段
		result = middleware.ProcessRequest(request)
		if result != nil {
			response = toResponse(result)
		} else {
			// 执行视图获取响应
			response = get_response(request)
		}

		// 响应阶段中间件
		middleware.ProcessResponse(request, &response)
		return response
	}

}

// convertExceptionToResponse 将 panic 的错误转换为 Response（最常用映射）。
// - 已知 HttpError：按 Status 返回
// - 其它：统一 500 Internal Server Error（生产可扩展为 DEBUG 模式返回详细页）
func (engine *Engine) convertExceptionToResponse(request *Request, exc interface{}) Response {
	switch err := exc.(type) {
	case error:
		engine.Log.Error(fmt.Sprintf("%v", err))
		engine.Log.Error(string(debug.Stack()))
		// 可补充对常见错误文本的分类映射；当前默认 500
		return Response{Status: http.StatusInternalServerError, Data: err.Error()}
	default:
		engine.Log.Error(fmt.Sprintf("%v", err))
		engine.Log.Error(string(debug.Stack()))
		return Response{Status: http.StatusInternalServerError, Data: "Internal Server Error"}
	}
}
