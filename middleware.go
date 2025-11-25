package goi

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// getResponseFunc 定义“获取响应”的函数类型。
// 我们用该类型来承载“被中间件层层包装后的最终处理器”。
type getResponseFunc func(request *Request) (response *Response)

// 中间件
// MiddleWare 定义中间件接口：
// - ProcessRequest：请求阶段（顺序执行）；返回非 nil 则直接写入响应，内层不再进入
// - ProcessException：异常阶段（逆序执行）；
// - ProcessResponse：响应阶段（逆序执行）；常用于追加/修改响应头
type MiddleWare interface {
	ProcessRequest(request *Request) any                  // 请求中间件
	ProcessException(request *Request, exception any) any // 异常中间件
	ProcessResponse(request *Request, response *Response) // 响应中间件
}

type MiddleWares []MiddleWare

// loadMiddleware 构建“洋葱模型”中间件链，返回对 handlerFunc 的包装函数。
//
// 参数:
//   - handlerFunc HandlerFunc: 最内层的视图处理函数
//
// 返回:
//   - getResponseFunc: 包装后的函数
func (middlewares MiddleWares) loadMiddleware(handlerFunc HandlerFunc) getResponseFunc {
	getResponse := func(request *Request) (response *Response) {
		defer func() {
			// 异常捕获
			err := recover()
			if err != nil {
				for i := len(middlewares) - 1; i >= 0; i-- {
					result := middlewares[i].ProcessException(request, err)
					if result != nil {
						response = toResponse(result)
						break
					}
				}
				// 未被中间件处理的异常，继续 panic
				if response == nil {
					panic(err)
				}
			}
		}()

		// 视图处理
		content := handlerFunc(request)
		return toResponse(content)
	}

	// 由内向外包裹
	for i := len(middlewares) - 1; i >= 0; i-- {
		getResponse = wrapMiddleware(middlewares[i], getResponse)
	}
	return getResponse
}

// wrapMiddleware 包装单个中间件，返回对 getResponse 的包装函数。
//
// 参数:
//   - middleware MiddleWare: 单个中间件实例
//   - getResponse getResponseFunc: 内层的“获取响应”函数
//
// 返回:
//   - getResponseFunc: 包装后的函数
func wrapMiddleware(middleware MiddleWare, getResponse getResponseFunc) getResponseFunc {
	return func(request *Request) (response *Response) {
		defer func() {
			err := recover()
			if err != nil {
				response = convertExceptionToResponse(request, err)
				return
			}
		}()

		var result any
		// 请求阶段
		result = middleware.ProcessRequest(request)
		if result != nil {
			response = toResponse(result)
		} else {
			// 执行视图获取响应
			result = getResponse(request)
			response = toResponse(result)
		}

		// 响应阶段中间件
		middleware.ProcessResponse(request, response)
		return response
	}
}

// convertExceptionToResponse 将 panic 的错误转换为 Response（最常用映射）。
// - 已知 HttpError：按 Status 返回
// - 其它：统一 500 Internal Server Error（生产可扩展为 DEBUG 模式返回详细页）
func convertExceptionToResponse(request *Request, exc interface{}) *Response {
	switch err := exc.(type) {
	case error:
		Log.Error(fmt.Sprintf("%v", err))
		Log.Error(string(debug.Stack()))
		// 可补充对常见错误文本的分类映射；当前默认 500
		return &Response{Status: http.StatusInternalServerError, Data: err.Error()}
	default:
		Log.Error(fmt.Sprintf("%v", err))
		Log.Error(string(debug.Stack()))
		return &Response{Status: http.StatusInternalServerError, Data: "Internal Server Error"}
	}
}
