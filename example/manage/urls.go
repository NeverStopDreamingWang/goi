package manage

import (
	"github.com/NeverStopDreamingWang/hgee"
	"net/http"
	"os"
	"path"
)

func init() {
	// 注册路由
	// Server.Router.UrlPatterns("/test", hgee.AsView{GET: TestFunc})

	// 注册静态路径
	Server.Router.StaticUrlPatterns("/static", "template")
	Server.Router.StaticUrlPatterns("/static_2", "template")
	Server.Router.UrlPatterns("/static_file", hgee.AsView{GET: TestFile})
}

// 返回文件
func TestFile(request *hgee.Request) any {
	baseDir, err := os.Getwd()
	if err != nil {
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   "获取目录失败！",
		}
	}
	absolutePath := path.Join(baseDir, "template/test.txt")
	file, err := os.Open(absolutePath)
	if err != nil {
		file.Close()
		return hgee.Response{
			Status: http.StatusInternalServerError,
			Data:   "读取文件失败！",
		}
	}
	// return file // 返回文件对象
	return file // 返回文件对象
}
