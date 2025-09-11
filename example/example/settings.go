package example

import (
	"os"
	"path"
	"path/filepath"

	"example/utils"
	"example/utils/mongo_db"
	"example/utils/mysql_db"
	"example/utils/redis_db"
	"example/utils/sqlite3_db"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/middleware"
)

// Http 服务
var Server *goi.Engine
var Config *ConfigModel

func init() {
	var err error

	// 创建 http 服务
	Server = goi.NewHttpServer()

	// version := goi.Version() // 获取版本信息
	// fmt.Println("goi 版本", version)

	// 注册中间件
	Server.MiddleWare = []goi.MiddleWare{
		&middleware.SecurityMiddleWare{},
		&middleware.CommonMiddleWare{},
		&middleware.XFrameMiddleWare{},
	}

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

	// 密钥
	Server.Settings.SECRET_KEY = "goi-insecure-mk*!m()hh&l^otrxt6*hl)2tfqxnx2cf5*=ndwe4e8z4wl&one"

	// RSA 私钥
	Server.Settings.PRIVATE_KEY = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyU5tYS7kvKP8QM5vjUQGpOXMKLeU62AbgwMX7nsDtIBQwvFU
kporZNWld8GR4mLCcch7zZWeGd8FnxhUpxk2N9N/Mtm3GAsGfbRE69wBgbxFWkdO
o0qG9LZakodhGFBmww6YKtIdGaxhHWWXnaVCmKj2F/aG0sGGZJRvn7rMWPtlu5+d
l8oxHeq9n0ACDrhabPr9zrxLfRj/srdDy4KMl4u50nDGuCsACmWwWgr9sZswtfjn
zonz4qBPkyuSURBExuqKeangGvAnOKGs4mvCnJk7hUT3zHnPyEU2vMr1VBGo1Qaa
tGrIoCU0OonRgrDczwu38qt4ZleeebyLtbQKhwIDAQABAoIBAHynSefx58ZALUXc
DwuE4jBd8+wKsfaGjsKzua/9ELBG/LuaQOp++Pv5p/reLH3o9csLgE4vpbUTeyGn
KVRHsmEjYBKW6l/DBAP3Cu6aT3yMns1mdnV7AtKp0LAHkMJDlz6V3Pg3H7n0Gjbf
3+DIotJxXeI80APVvmit2eko/Lzj5ZQAJT65kkSzgUejTGznvi6Y6Fk0gHQa+R5X
w5TGyMKPKWsATvZ1wft+AE0HR9s277nsoW/pSCO19GcC8Xt+XGLWDViA+dEWnifO
nRTR33aSYPE0eFNFnrjPrx5ybOoANofiJyOg9aEwriunVxfFhbn53a47XGhNc/hE
pXSYm/ECgYEA9GHcNFXeXkERpUUGNw/Kw19qw09bjxliD1BrsbqWowKIM9TbgJbO
NK1uabYlx6uUKeO6cUqAUBbHaDAtu8tef3mkuGdfZXRQDphW54PT5dxMdd+divBO
l7mxy0/pCNwvk7FgTQEHNOR+ilUwXMrezYUl/VBx2Z+6lWdmed/4fvsCgYEA0uBU
+MBVKiQ1PuNwO4IDAYIepkD4vaW2+1OkVxsp7NQiXudhQbJ1oiMrj0qv7R9oHeQr
cP+8utCkFAFBlcjaSGd8poQaCuggjIFggOQel2VWH7fp53fX4zIzvK/aeK9DHLib
hct142EtONAOVT7jX2S/akQza+iT5kCFZUjlHOUCgYBJIa/gCYJN+nLpXkqJ17P5
22f7opfnrtTleE+CFDBX/736pMw7IuX6ZZwRDm0n33SrRHbayEy1qttplmFZPXa9
9w0QEf8+QRxkAbqf9ZdHxjErZQukNF2QkgVerj8yY5HpRL9oy9H4RhiIFQ4v9pXO
MvY3ZAdt9JrFcvf7qMaYWQKBgQCoArHQirSP4c9GbsEBuIEal4hB36wOtVRHg4mB
GRRbK1zDDkhPppbQeoL/JHtsSkSS5DK0Uh0VHpxLkACoDSHU5BbNOJzjKbSdHYs8
xgOVjdiDZu2GTNaFnn7YC6fd1Y17+Z13iPZBFjCIfkOdKYDQhR141iO+Csyje66M
Vvqr/QKBgQDwUUunJ0I7RB7bCBUZ6pI074agkPt3FtUQ9pdY0y6m4va14fLthvvr
cveCPilBKyJP+r39FjpFQr0QVCC/PA7FPGhLcObegvJKQTB9cRnDf8JfgfDswStJ
1tXmFFJ++yCVcZdFBMD2jolCuEDD+eXidBmy+Og4shUpQM3GgkgtOA==
-----END RSA PRIVATE KEY-----
`

	// RSA 公钥
	Server.Settings.PUBLIC_KEY = `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyU5tYS7kvKP8QM5vjUQG
pOXMKLeU62AbgwMX7nsDtIBQwvFUkporZNWld8GR4mLCcch7zZWeGd8FnxhUpxk2
N9N/Mtm3GAsGfbRE69wBgbxFWkdOo0qG9LZakodhGFBmww6YKtIdGaxhHWWXnaVC
mKj2F/aG0sGGZJRvn7rMWPtlu5+dl8oxHeq9n0ACDrhabPr9zrxLfRj/srdDy4KM
l4u50nDGuCsACmWwWgr9sZswtfjnzonz4qBPkyuSURBExuqKeangGvAnOKGs4mvC
nJk7hUT3zHnPyEU2vMr1VBGo1QaatGrIoCU0OonRgrDczwu38qt4ZleeebyLtbQK
hwIDAQAB
-----END RSA PUBLIC KEY-----
`

	// 设置 SSL
	Server.Settings.SSL = goi.MetaSSL{
		STATUS:    false,  // SSL 开关
		TYPE:      "自签证书", // 证书类型
		CERT_PATH: filepath.Join(Server.Settings.BASE_DIR, "ssl", "example.crt"),
		KEY_PATH:  filepath.Join(Server.Settings.BASE_DIR, "ssl", "example.key"),
	}

	// 加载 Config 配置
	configPath := path.Join(Server.Settings.BASE_DIR, "config.yaml")

	Config = &ConfigModel{
		MySQLConfig:   mysql_db.Config,
		RedisConfig:   redis_db.Config,
		MongoDBConfig: mongo_db.Config,
	}
	err = utils.LoadYaml(configPath, Config)
	if err != nil {
		panic(err)
	}

	// 数据库配置
	Server.Settings.DATABASES["default"] = &goi.DataBase{
		ENGINE:  "sqlite3",
		Connect: sqlite3_db.Connect,
	}
	Server.Settings.DATABASES["mysql"] = &goi.DataBase{
		ENGINE:  "mysql",
		Connect: mysql_db.Connect,
	}

	Server.Settings.USE_TZ = false
	// 设置时区
	err = Server.Settings.SetTimeZone("Asia/Shanghai") // 默认为空字符串 ''，本地时间
	if err != nil {
		panic(err)
	}
	//  goi.GetLocation() 获取时区 Location
	//  goi.GetTime() 获取当前时区的时间

	// 设置框架语言
	Server.Settings.SetLanguage(goi.ZH_CN) // 默认 ZH_CN
	// Server.Settings.SetLanguage(goi.EN_US) // 默认 ZH_CN

	// 设置最大缓存大小
	Server.Cache.EVICT_POLICY = goi.ALLKEYS_LRU   // 缓存淘汰策略
	Server.Cache.EXPIRATION_POLICY = goi.PERIODIC // 过期策略
	Server.Cache.MAX_SIZE = 100                   // 单位为字节，0 为不限制使用

	// 日志设置 DEBUG = true 输出到控制台
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

	// 连接其它数据库
	// redis_db.Connect()
	// mongo_db.Connect()

	// 注册关闭回调处理程序
	Server.RegisterShutdownHandler("清理连接", Shutdown)
}

func Shutdown(engine *goi.Engine) error {
	// var err error
	//
	// 关闭连接
	// err = redis_db.Close()
	// if err != nil {
	// 	return err
	// }
	//
	// err = mongo_db.Close()
	// if err != nil {
	// 	return err
	// }
	return nil
}
