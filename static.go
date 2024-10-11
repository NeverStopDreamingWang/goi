package goi

import (
	"net/http"
	"os"
	"time"
)

// 静态文件处理函数
func metaStaticFileHandler(staticFile string, request *Request) interface{} {
	var err error
	file, err := os.Open(staticFile)
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return Response{
			Status: http.StatusNotFound,
			Data:   "",
		}
	}
	return file
}

// 静态目录映射处理函数
func metaStaticDirHandler(staticDir http.Dir, request *Request) interface{} {
	var fileName string
	var validationErr ValidationError
	var err error
	validationErr = request.PathParams.Get("fileName", &fileName)
	if validationErr != nil {
		return validationErr.Response()
	}
	file, err := staticDir.Open(fileName)
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return Response{
			Status: http.StatusNotFound,
			Data:   "",
		}
	}
	return file
}

// 返回响应文件内容
func metaResponseStatic(file os.File, request *Request, response http.ResponseWriter) {
	defer file.Close()

	fileInfo, err := file.Stat()
	if os.IsNotExist(err) {
		response.WriteHeader(http.StatusNotFound)
	} else if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
	}
	if fileInfo.IsDir() {
		response.WriteHeader(http.StatusNotFound)
	} else {
		response.WriteHeader(http.StatusOK)
		http.ServeContent(response, request.Object, fileInfo.Name(), time.Now().In(Settings.LOCATION), &file)
	}
}
