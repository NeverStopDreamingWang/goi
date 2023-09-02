package manage

import (
	"crypto/rand"
	"encoding/hex"
	"example/middleware"
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"os"
)

var (
	Server         *hgee.Engine
	SERVER_ADDRESS string // 服务地址
	SERVER_PORT    uint16 // 服务端口
	BASE_DIR       string // 项目根路径
	SECRET_KEY     string // 项目密钥
	// 数据库配置
	DATABASES map[string]map[string]interface{}

	// 自定义
	REDIS_HOST     string
	REDIS_PORT     uint16
	REDIS_PASSWORD string
	REDIS_DB       uint8
)

func init() {
	// 运行地址
	SERVER_ADDRESS = "0.0.0.0"
	// 端口
	SERVER_PORT = 8989
	// 创建 http 服务
	Server = hgee.NewHttpServer(SERVER_ADDRESS, SERVER_PORT)

	// 注册中间件
	// 注册请求中间件
	Server.MiddleWares.BeforeRequest(middleware.RequestMiddleWare)
	// 注册视图中间件
	// Server.MiddleWares.BeforeView()
	// 注册响应中间件
	// Server.MiddleWares.BeforeResponse()

	var err error
	// 获取项目根路径
	BASE_DIR, err = os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("异常【获取项目根路径】：%v\n", err))
		return
	}

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("异常【密钥生成】：%v\n", err))
		return
	}
	// 将随机字节串转换为十六进制字符串
	SECRET_KEY = hex.EncodeToString(b)

	// 数据库配置
	DATABASES = map[string]map[string]interface{}{
		"default": {
			"ENGINE":   "mysql",
			"NAME":     "h_wiki",
			"USER":     "root",
			"PASSWORD": "admin",
			"HOST":     "127.0.0.1",
			"PORT":     "3306",
		},
	}

	// 自定义配置
	// redis配置
	REDIS_HOST = "127.0.0.1"
	REDIS_PORT = 6379
	REDIS_PASSWORD = "123"
	REDIS_DB = 0
}
