package example

import (
	"os"
	"path"

	"github.com/NeverStopDreamingWang/goi"
)

// Http 服务
var Server *goi.Engine

func init() {
	// 创建 http 服务
	Server = goi.NewHttpServer()

	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)

	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()

	// 设置网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 监听地址
	Server.Settings.BIND_ADDRESS = "0.0.0.0"
	// 端口
	Server.Settings.PORT = 8080
	// 绑定域名
	Server.Settings.BindDomain = ""

	// 项目 AES 密钥
	Server.Settings.SECRET_KEY = "xxxxxxxxxxxxxxxxx"

	// 项目 RSA 私钥
	Server.Settings.PRIVATE_KEY = "xxxxxxxxxxxxxxxxx"

	// 项目 RSA 公钥
	Server.Settings.PUBLIC_KEY = "xxxxxxxxxxxxxxxxx"

	// 设置 SSL
	Server.Settings.SSL = goi.MetaSSL{
		STATUS:    false,  // SSL 开关
		TYPE:      "自签证书", // 证书类型
		CERT_PATH: path.Join(Server.Settings.BASE_DIR, "ssl/example.crt"),
		KEY_PATH:  path.Join(Server.Settings.BASE_DIR, "ssl/example.key"),
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
	Server.Settings.TIME_ZONE = "Asia/Shanghai" // 默认 Asia/Shanghai，America/New_York

	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用

	// 日志设置
	Server.Log.DEBUG = true
	// 日志列表
	defaultLog := newDefaultLog()
	accessLog := newAccessLog()
	errorLog := newErrorLog()
	Server.Log.LOGGERS = []*goi.MetaLogger{
		defaultLog, // 默认日志
		accessLog,  // 访问日志
		errorLog,   // 错误日志
	}

	// 日志打印
	// Server.Log.Log() = goi.Log.Log()
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()

	// 设置自定义配置
	// redis配置
	Server.Settings.Set("REDIS_HOST", "127.0.0.1")
	Server.Settings.Set("REDIS_PORT", 6379)
	Server.Settings.Set("REDIS_PASSWORD", "123")
	Server.Settings.Set("REDIS_DB", 0)
}
