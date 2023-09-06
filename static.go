package hgee

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"time"
)

// 静态文件处理函数
func metaStaticHandler(request *Request) any {
	staticPattern := request.PathParams.Get("staticPattern").(string)
	staticPath := request.PathParams.Get("staticPath").(string)

	absolutePath := path.Join(staticPattern, staticPath)
	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		return Response{
			Status: http.StatusNotFound,
			Data:   nil,
		}
	}
	file, err := os.Open(absolutePath)
	if err != nil {
		file.Close()
		return Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	return file
}

// 返回响应文件内容
func metaResponseStatic(file *os.File, request *Request, response http.ResponseWriter) {
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		return
	}
	if fileInfo.IsDir() {
		response.WriteHeader(http.StatusNotFound)
		return
	}
	size := fileInfo.Size()
	http.ServeContent(response, request.Object, fileInfo.Name(), time.Now(), file)

	fmt.Printf("[%v] - %v - %v %v static %v byte status %v %v\n",
		time.Now().Format("2006-01-02 15:04:05"),
		request.Object.Host,
		request.Object.Method,
		request.Object.URL.Path,
		size,
		request.Object.Proto,
		http.StatusOK,
	)
}
