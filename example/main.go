package main

import (
	"example/example"
	// 注册app
	_ "example/test"
	_ "example/user"
)

func main() {

	// 启动服务
	example.Server.RunServer()
}
