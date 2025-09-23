package goi

import (
	"io/fs"
	"net/http"
	"os"
	"path"
)

// StaticFileView 静态文件视图
//
// 参数:
//   - filePath string: 文件路径
//
// 返回:
//   - HandlerFunc: 处理函数
//
// 说明:
//   - 生成静态路由的通用 View ，由上层统一处理 Content-Type
func StaticFileView(filePath string) HandlerFunc {
	return func(request *Request) interface{} {
		var err error

		file, err := os.Open(filePath)
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
}

// StaticDirView 静态目录视图
//
// 参数:
//   - dirPath http.Dir: 目录路径
//
// 返回:
//   - HandlerFunc: 处理函数
//
// 说明:
//   - 生成静态路由的通用 View ，由上层统一处理 Content-Type
func StaticDirView(dirPath http.Dir) HandlerFunc {
	return func(request *Request) interface{} {
		if dirPath == "" {
			dirPath = "."
		}
		var fileName string
		var validationErr ValidationError
		validationErr = request.PathParams.Get("fileName", &fileName)
		if validationErr != nil {
			return validationErr.Response()
		}
		file, err := dirPath.Open(fileName)
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
}

type FS struct {
	FS   fs.FS
	Name string
}

// StaticFileFSView fs.FS 静态文件嵌入式文件视图
//
// 参数:
//   - fileFS fs.FS: 嵌入式文件系统
//   - defaultPath string: 默认响应嵌入式文件的路径
//
// 说明:
//   - 生成静态路由的通用 View ，由上层统一处理 Content-Type
func StaticFileFSView(fileFS fs.FS, defaultPath string) HandlerFunc {
	return func(request *Request) interface{} {
		return FS{FS: fileFS, Name: defaultPath}
	}
}

// StaticDirFSView fs.FS 静态目录嵌入式文件视图
//
// 参数:
//   - dirFS fs.FS: 嵌入式文件系统
//   - basePath string: 嵌入式文件系统基础路径
//
// 说明:
//   - 需要在路由中包含 <path:fileName> 参数
//   - 会自动拼接 basePath + fileName 为嵌入文件查找路径
//   - 生成静态路由的通用 View ，由上层统一处理 Content-Type
func StaticDirFSView(dirFS fs.FS, basePath string) HandlerFunc {
	return func(request *Request) interface{} {
		var fileName string
		var validationErr ValidationError
		validationErr = request.PathParams.Get("fileName", &fileName)
		if validationErr != nil {
			return validationErr.Response()
		}
		return FS{FS: dirFS, Name: path.Join(basePath, fileName)}
	}
}
