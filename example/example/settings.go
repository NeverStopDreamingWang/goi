package example

import (
	"github.com/NeverStopDreamingWang/goi"
	"os"
	"path"
)

// Http 服务
var Server *goi.Engine

func init() {
	// 创建 http 服务
	Server = goi.NewHttpServer()

	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)

	// 设置网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 运行地址
	Server.Settings.ADDRESS = "0.0.0.0"
	// 端口
	Server.Settings.PORT = 8080

	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()

	// 项目
	Server.Settings.SECRET_KEY = "xxxxxxxxxxxxxxxxx"

	// 设置 SSL
	Server.Settings.SSL = goi.MetaSSL{
		STATUS:    false, // SSL 开关
		CERT_PATH: path.Join(Server.Settings.BASE_DIR, "ssl/example.crt"),
		KEY_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/example.key"),
		CSR_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/example.csr"), // 可选
	}

	// 数据库配置
	Server.Settings.DATABASES["default"] = goi.MetaDataBase{
		ENGINE:   "mysql",
		NAME:     "test_goi",
		USER:     "root",
		PASSWORD: "123",
		HOST:     "127.0.0.1",
		PORT:     3306,
	}
	Server.Settings.DATABASES["sqlite_1"] = goi.MetaDataBase{
		ENGINE:   "sqlite3",
		NAME:     path.Join(Server.Settings.BASE_DIR, "data/test_goi.db"),
		USER:     "",
		PASSWORD: "",
		HOST:     "",
		PORT:     0,
	}

	// 设置时区
	Server.Settings.TIME_ZONE = "Asia/Shanghai" // 默认 Asia/Shanghai
	// Server.Settings.TIME_ZONE = "America/New_York"

	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用

	// 日志设置
	Server.Log.DEBUG = true
	Server.Log.INFO_OUT_PATH = "logs/info.log"      // 输出所有日志到文件
	Server.Log.ACCESS_OUT_PATH = "logs/asccess.log" // 输出访问日志文件
	Server.Log.ERROR_OUT_PATH = "logs/error.log"    // 输出错误日志文件
	// Server.Log.OUT_DATABASE = ""  // 输出到的数据库 例: default
	Server.Log.SplitSize = 1024 * 1024
	Server.Log.SplitTime = "2006-01-02"
	// 日志打印
	// Server.Log.Debug() = goi.Log.Debug() = goi.Log.Log(goi.DEBUG, "")
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()

	// Server.Settings.LOG_OUTPATH = "/log/server.log"
	// log_path := path.Join(BASE_DIR, LOG_OUTPATH)
	// log_file, err := os.OpenFile(log_path, os.O_CREATE|os.O_APPEND, 0777)
	// if err != nil {
	// 	panic(fmt.Sprintf("异常【日志创建】: %v\n", err))
	// 	return
	// }
	// log.SetOutput(log_file)
	// // 设置日志前缀为空
	// log.SetFlags(0)
	// // defer log_file.Close()

	// 设置自定义配置
	// redis配置
	Server.Settings.Set("REDIS_HOST", "127.0.0.1")
	Server.Settings.Set("REDIS_PORT", 6379)
	Server.Settings.Set("REDIS_PASSWORD", "123")
	Server.Settings.Set("REDIS_DB", 0)
}
