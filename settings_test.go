package goi

import (
	"fmt"
	"testing"
)

// 调试
func TestSettings(t *testing.T) {
	settings := newSettings()

	var name string
	var age int32
	var height float64
	var args []string
	var kwargs map[string]interface{}
	var err error

	name = "张三"
	settings.Set("name", name)
	age = 16
	settings.Set("age", age)
	height = 1.76
	settings.Set("height", height)
	args = []string{
		"参数1",
		"参数2",
	}
	settings.Set("args", args)
	kwargs = map[string]interface{}{
		"参数1": 1,
		"参数2": 1,
	}
	settings.Set("kwargs", kwargs)

	var read_name string
	var read_age int32
	// var read_age int 错误类型的变量
	var read_height float64
	var read_args []string
	var read_kwargs map[string]interface{}

	err = settings.Get("name", &read_name)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("name:", read_name)

	err = settings.Get("age", &read_age)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("age:", read_age)

	err = settings.Get("height", &read_height)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("height:", read_height)

	err = settings.Get("args", &read_args)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("args:", read_args)

	err = settings.Get("kwargs", &read_kwargs)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("kwargs:", read_kwargs)

}
