package main

import (
	"example/example"
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"reflect"

	// 注册app
	_ "example/test"
	_ "example/user"
)

func main() {
	go routerInfo()

	// 启动服务
	example.Server.RunServer()
}

func routerInfo() {
	// 获取所有路径信息
	var routerInfoChan = make(chan goi.RouteInfo)
	go func() {
		defer close(routerInfoChan)
		example.Server.Router.Next("/", routerInfoChan)
	}()

	fmt.Println("Router List:")
	for RouterInfo := range routerInfoChan {
		fmt.Println(fmt.Sprintf("  Uri:%v,Desc:%v,IsParent:%v,ParentRouter:%v", RouterInfo.Uri, RouterInfo.Desc, RouterInfo.IsParent, RouterInfo.ParentRouter))
		fmt.Println("    ViewSet:")
		// ViewSet
		ViewSetValue := reflect.ValueOf(RouterInfo.ViewSet)
		ViewSetType := ViewSetValue.Type()
		// 获取字段
		for i := 0; i < ViewSetType.NumField(); i++ {
			method := ViewSetType.Field(i).Name
			if ViewSetValue.Field(i).IsZero() {
				continue
			}
			fmt.Println("      Method:", method)
		}
	}
}
