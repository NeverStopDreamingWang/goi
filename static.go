package goi

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
)

// 生成静态路由的通用 View
// 单文件：返回文件对象，由上层统一写回
//
// 参数:
//   - filePath string: 文件路径
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

// 目录：根据路由参数 fileName 返回目标文件
//
// 参数:
//   - dirPath http.Dir: 目录路径
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

// embed.FS 单文件：直接返回 FS，由上层 FileServer 处理
//
// 参数:
//   - fileFS embed.FS: 嵌入文件系统
//   - defaultPath string: 嵌入文件默认路径
func StaticFileFSView(fileFS embed.FS, defaultPath string) HandlerFunc {
	return func(request *Request) interface{} {
		return FS{FS: fileFS, Name: defaultPath}
	}
}

// embed.FS 目录：直接返回 FS，由上层 FileServer 处理
//
// 参数:
//   - dirFS embed.FS: 嵌入文件系统
//   - basePath string: 嵌入文件基础路径
//
// 说明:
//   - 需要在路由中包含 <path:fileName> 参数
//   - 会自动拼接 basePath + fileName 为嵌入文件查找路径
func StaticDirFSView(dirFS embed.FS, basePath string) HandlerFunc {
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
