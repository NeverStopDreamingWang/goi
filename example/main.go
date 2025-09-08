package main

import (
	"fmt"
	"reflect"
	"strings"

	"example/example"

	// 注册中间件
	_ "example/middleware"
	// 注册app
	_ "example/test"
	_ "example/user"

	"github.com/NeverStopDreamingWang/goi"
)

func main() {
	// 获取所有路径信息
	fmt.Println("Route:")
	route := example.Server.Router.GetRoute()
	printRouteInfo(route, 1)

	// 启动服务
	example.Server.RunServer()
}

var replaceStr = "  "

func printRouteInfo(route goi.Route, count int) {
	fmt.Printf("%vPath: %v Desc: %v Pattern: %v Regex:%v ParamInfos:%+v\n",
		strings.Repeat(replaceStr, count),
		route.Path,
		route.Desc,
		route.Pattern,
		route.Regex,
		route.ParamInfos,
	)
	var method_list []string
	// ViewSet
	ViewSetValue := reflect.ValueOf(route.ViewSet)
	ViewSetType := ViewSetValue.Type()
	// 获取字段
	for i := 0; i < ViewSetType.NumField(); i++ {
		method := ViewSetType.Field(i).Name
		if ViewSetValue.Field(i).IsZero() {
			continue
		}
		method_list = append(method_list, method)
	}
	fmt.Printf("%vViewSet: %v\n", strings.Repeat(replaceStr, count+1), strings.Join(method_list, ", "))
	if len(route.Children) > 0 {
		fmt.Printf("%vChildren:\n", strings.Repeat(replaceStr, count))
		for _, child := range route.Children {
			printRouteInfo(child, count+1)
		}
	}
}
