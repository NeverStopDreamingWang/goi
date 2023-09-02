package main

import (
	"example/manage"
	_ "example/user" // 注册路由
	"fmt"
)

func main() {
	err := manage.Server.RunServer()
	if err != nil {
		fmt.Printf("服务已停止！")
		// panic(fmt.Sprintf("停止：%v\n", err))
	}
}
