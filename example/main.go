package main

import (
	"example/manage"
	"fmt"

	// 注册路由
	_ "example/test"
	_ "example/user"
)

func main() {
	err := manage.Server.RunServer()
	if err != nil {
		fmt.Printf("服务已停止！")
		// panic(fmt.Sprintf("停止：%v\n", err))
	}
}
