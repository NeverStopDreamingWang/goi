package hgee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
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

func Recovery(request *Request, response http.ResponseWriter) {
	if err := recover(); err != nil {
		message := fmt.Sprintf("%s", err)
		log.Printf("%s\n\n", trace(message))
		response.WriteHeader(http.StatusInternalServerError)
		write, _ := response.Write([]byte("Internal Server Error"))
		log.Printf(" - %v - %v %v %v byte status %v %v\n",
			time.Now().Format("2006-01-02 15:04:05"),
			request.Object.Host,
			request.Object.Method,
			request.Object.URL.Path,
			write,
			request.Object.Proto,
			http.StatusInternalServerError,
		)
	}
}
