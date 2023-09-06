package hgee

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

func metaRecovery(request *Request, response http.ResponseWriter) {
	if err := recover(); err != nil {
		message := fmt.Sprintf("%v", err)
		response.WriteHeader(http.StatusInternalServerError)
		write, _ := response.Write([]byte("Internal Server Error"))
		log := fmt.Sprintf("- %v - %v %v %v byte status %v %v \nerr: %v",
			request.Object.Host,
			request.Object.Method,
			request.Object.URL.Path,
			write,
			request.Object.Proto,
			http.StatusInternalServerError,
			trace(message),
		)
		engine.Log.Error(log)
	}
}
