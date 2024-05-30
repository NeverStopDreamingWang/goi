package goi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// 静态文件处理函数
func metaStaticFileHandler(request *Request) interface{} {
	var staticFile string
	var validationErr ValidationError
	var err error
	validationErr = request.PathParams.Get("static_file", &staticFile)
	if validationErr != nil {
		return validationErr.Response()
	}
	if staticFile[0] != '/' {
		staticFile = filepath.Join(Settings.BASE_DIR, staticFile)
	}

	if _, err = os.Stat(staticFile); os.IsNotExist(err) {
		return Response{
			Status: http.StatusNotFound,
			Data:   "文件不存在！",
		}
	}
	file, err := os.Open(staticFile)
	if err != nil {
		_ = file.Close()
		return Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	return file
}

// 静态目录映射处理函数
func metaStaticDirHandler(request *Request) interface{} {
	const indexPage = "/index.html"
	var staticFile string
	var staticDir string
	var path string
	var validationErr ValidationError
	var err error
	validationErr = request.PathParams.Get("static_dir", &staticDir)
	if validationErr != nil {
		return validationErr.Response()
	}
	validationErr = request.PathParams.Get("path", &path)
	if validationErr != nil {
		return validationErr.Response()
	}
	if staticDir[0] != '/' {
		staticDir = filepath.Join(Settings.BASE_DIR, staticDir)
	}

	if path == "/" {
		path = indexPage
	}
	staticFile = filepath.Join(staticDir, path)

	if _, err = os.Stat(staticFile); os.IsNotExist(err) {
		return Response{
			Status: http.StatusNotFound,
			Data:   "文件不存在！",
		}
	}
	file, err := os.Open(staticFile)
	if err != nil {
		_ = file.Close()
		return Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	return file
}

// 返回响应文件内容
func metaResponseStatic(file *os.File, request *Request, response http.ResponseWriter) (int64, []byte) {
	defer file.Close()

	fileInfo, err := file.Stat()
	if os.IsNotExist(err) {
		response.WriteHeader(http.StatusNotFound)
		return 0, []byte("文件不存在")
	} else if err != nil {
		response.WriteHeader(http.StatusNotFound)
		return 0, []byte(fmt.Sprintf("获取文件状态失败: %v", err))
	}
	if fileInfo.IsDir() {
		response.WriteHeader(http.StatusNotFound)
		return 0, []byte("不是一个文件")
	}
	http.ServeContent(response, request.Object, fileInfo.Name(), time.Now().In(Settings.LOCATION), file)
	return fileInfo.Size(), nil
}
