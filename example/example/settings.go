package example

import (
	"database/sql"
	"os"
	"path"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// Http 服务
var Server *goi.Engine

func init() {
	var err error

	// 创建 http 服务
	Server = goi.NewHttpServer()

	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)

	// 项目路径
	Server.Settings.BASE_DIR, _ = os.Getwd()

	// 设置网络协议
	Server.Settings.NET_WORK = "tcp" // 默认 "tcp" 常用网络协议 "tcp"、"tcp4"、"tcp6"、"udp"、"udp4"、"udp6
	// 监听地址
	Server.Settings.BIND_ADDRESS = "0.0.0.0" // 默认 127.0.0.1
	// 端口
	Server.Settings.PORT = 8080
	// 绑定域名
	Server.Settings.BIND_DOMAIN = ""

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
		KEY_PATH: path.Join(Server.Settings.BASE_DIR, "ssl/example.ke"+
			"y"),
	}

	// 数据库配置
	Server.Settings.DATABASES["default"] = &goi.DataBase{
		ENGINE:         "mysql",
		DataSourceName: "root:123123@tcp(127.0.0.1:3306)/test_goi",
		Connect: func(ENGINE string, DataSourceName string) (*sql.DB, error) {
			mysqlDB, err := sql.Open(ENGINE, DataSourceName)
			if err != nil {
				panic(err)
			}
			// 设置连接池参数
			mysqlDB.SetMaxOpenConns(10)           // 设置最大打开连接数
			mysqlDB.SetMaxIdleConns(5)            // 设置最大空闲连接数
			mysqlDB.SetConnMaxLifetime(time.Hour) // 设置连接的最大存活时间
			return mysqlDB, nil
		},
	}
	Server.Settings.DATABASES["sqlite"] = &goi.DataBase{
		ENGINE:         "sqlite3",
		DataSourceName: path.Join(Server.Settings.BASE_DIR, "data/test_goi.db"),
		Connect: func(ENGINE string, DataSourceName string) (*sql.DB, error) {
			sqliteDB, err := sql.Open(ENGINE, DataSourceName)
			if err != nil {
				panic(err)
			}
			return sqliteDB, nil
		},
	}

	// 设置时区
	err = Server.Settings.SetTimeZone("Asia/Shanghai") // 默认为空字符串 ''，本地时间
	if err != nil {
		panic(err)
	}
	//  Server.Settings.GetLocation() 获取时区 Location

	// 设置框架语言
	err = Server.Settings.SetLanguage(goi.ZH_CN) // 默认 ZH_CN
	// err = Server.Settings.SetLanguage(goi.EN_US) // 默认 ZH_CN
	if err != nil {
		panic(err)
	}

	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用

	// 日志 DEBUG 设置
	Server.Log.DEBUG = true
	// 注册日志
	defaultLog := newDefaultLog() // 默认日志
	err = Server.Log.RegisterLogger(defaultLog)
	if err != nil {
		panic(err)
	}
	accessLog := newAccessLog() // 访问日志
	err = Server.Log.RegisterLogger(accessLog)
	if err != nil {
		panic(err)
	}
	errorLog := newErrorLog() // 错误日志
	err = Server.Log.RegisterLogger(errorLog)
	if err != nil {
		panic(err)
	}

	// 日志打印
	// Server.Log.Log() = goi.Log.Log()
	// Server.Log.Info() = goi.Log.Info()
	// Server.Log.Warning() = goi.Log.Warning()
	// Server.Log.Error() = goi.Log.Error()

	// 设置验证器错误处理，不指定则使用默认
	Server.Validator.SetValidationError(&exampleValidationError{})

	// 设置自定义配置
	// Server.Settings.Set(key string, value interface{})
	// Server.Settings.Get(key string, dest interface{})

}
