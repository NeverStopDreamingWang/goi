package goi

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"
)

func (engine *Engine) recovery(request *Request, responseWriter *ResponseWriter, elapsed time.Duration) {
	err := recover()
	if err != nil {
		// 如果没有写入响应，则写入 500 错误
		if responseWriter.Status == 0 && responseWriter.Bytes == 0 {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			_, _ = responseWriter.Write([]byte("Internal Server Error"))
		}
		engine.Log.Error(fmt.Sprintf("%v", err))
		engine.Log.Error(string(debug.Stack()))
	}

	// 格式化时间单位（统一为毫秒，保留2位小数）
	timeMs := float64(elapsed) / float64(time.Millisecond)

	log := fmt.Sprintf("- %v - %v %v => generated %d bytes in %.2f msecs (%s %d) %d headers",
		request.Object.Host,
		request.Object.Method,
		request.Object.URL.Path,
		responseWriter.Bytes,
		timeMs,
		request.Object.Proto,
		responseWriter.Status,
		len(responseWriter.Header()),
	)
	engine.Log.Info(log)
}
