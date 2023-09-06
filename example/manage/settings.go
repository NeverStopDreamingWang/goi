package manage

import (
	"example/middleware"
	"github.com/NeverStopDreamingWang/hgee"
	"os"
)

// Http 服务
var Server *hgee.Engine

func init() {
	// 创建 http 服务
	Server = hgee.NewHttpServer()

	// 运行地址
	Server.Settings.SERVER_ADDRESS = "0.0.0.0"
	// 端口
	Server.Settings.SERVER_PORT = 8989

	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()

	// 项目
	Server.Settings.SECRET_KEY = "xxxxxxxxxxxxxxxxx"

	// 数据库配置
	Server.Settings.DATABASES["default"] = hgee.DataBase{
		ENGINE:   "mysql",
		NAME:     "test_hgee",
		USER:     "root",
		PASSWORD: "admin",
		HOST:     "127.0.0.1",
		PORT:     3306,
	}

	// 日志设置
	Server.Log.DEBUG = false
	Server.Log.INFO_OUT_PATH = "logs/server.log" // 输出所有日志到文件
	// Server.Log.ACCESS_OUT_PATH = "" // 输出访问日志文件
	// Server.Log.ERROR_OUT_PATH = ""  // 输出错误日志文件
	// Server.Log.OUT_DATABASE = ""  // 输出到的数据库 例：default

	// 日志打印
	// Server.Log.Debug() = hgee.Debug()
	// Server.Log.Info() = hgee.Info()
	// Server.Log.Warning() = hgee.Warning()
	// Server.Log.Error() = hgee.Error()

	// Server.Settings.LOG_OUTPATH = "/log/server.log"
	// log_path := path.Join(BASE_DIR, LOG_OUTPATH)
	// log_file, err := os.OpenFile(log_path, os.O_CREATE|os.O_APPEND, 0777)
	// if err != nil {
	// 	panic(fmt.Sprintf("异常【日志创建】：%v\n", err))
	// 	return
	// }
	// log.SetOutput(log_file)
	// // 设置日志前缀为空
	// log.SetFlags(0)
	// // defer log_file.Close()

	// 注册中间件
	// 注册请求中间件
	Server.MiddleWares.BeforeRequest(middleware.RequestMiddleWare)
	// 注册视图中间件
	// Server.MiddleWares.BeforeView()
	// 注册响应中间件
	// Server.MiddleWares.BeforeResponse()

	// 设置自定义配置
	// redis配置
	Server.Settings.Set("REDIS_HOST", "127.0.0.1")
	Server.Settings.Set("REDIS_PORT", 6379)
	Server.Settings.Set("REDIS_PASSWORD", "123")
	Server.Settings.Set("REDIS_DB", 0)
}