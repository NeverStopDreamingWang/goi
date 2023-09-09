package main

import (
	"example/manage"
	"fmt"

	// 注册app
	_ "example/test"
	_ "example/user"
)

func main() {

	// 启动服务
	err := manage.Server.RunServer()
	if err != nil {
		fmt.Printf("服务已停止！")
		// panic(fmt.Sprintf("停止：%v\n", err))
	}
}
