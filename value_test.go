package goi_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
)

type testStruct struct {
	Username string   `name:"username"`
	Password string   `name:"password"`
	Age      int      `name:"age"`
	Custom   []string `name:"custom" type:"custom" required:"true"`
}

func ExampleSetValue() {
	source := map[string]any{
		"username": "test_user",
		"password": "test123456",
		"age":      18,
		"custom":   []any{111, "test1", "test2"},
	}
	dest := &testStruct{}

	destValue := reflect.ValueOf(dest)
	err := goi.SetValue(destValue, source)
	if err != nil {
		return
	}
	fmt.Printf("Username: %+v %T\n", dest.Username, dest.Username)
	fmt.Printf("Password: %+v %T\n", dest.Password, dest.Password)
	fmt.Printf("Age: %+v %T\n", dest.Age, dest.Age)
	fmt.Printf("Custom: %+v %T\n", dest.Custom, dest.Custom)

	// Output:
	// Username: test_user string
	// Password: test123456 string
	// Age: 18 int
	// Custom: [111 test1 test2] []string
}

func ExampleSetValue_manual() {
	source := map[string]any{
		"username": "test_user",
		"password": "test123456",
		"age":      18,
		"custom":   []any{111, "test1", "test2"},
	}
	dest := &testStruct{}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return
	}
	destValue = destValue.Elem()
	destType := destValue.Type()

	// 遍历参数结构体的字段
	for i := 0; i < destType.NumField(); i++ {
		var err error
		var fieldValue reflect.Value
		var fieldType reflect.StructField
		var fieldName string

		fieldValue = destValue.Field(i)
		fieldType = destType.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		fieldName = fieldType.Tag.Get("name")
		if fieldName == "" || fieldName == "-" {
			fieldName = strings.ToLower(fieldType.Name) // 字段名
		}

		value, ok := source[fieldName]
		if !ok {
			continue
		}

		err = goi.SetValue(fieldValue, value)
		if err != nil {
			return
		}
	}
	fmt.Printf("Username: %+v %T\n", dest.Username, dest.Username)
	fmt.Printf("Password: %+v %T\n", dest.Password, dest.Password)
	fmt.Printf("Age: %+v %T\n", dest.Age, dest.Age)
	fmt.Printf("Custom: %+v %T\n", dest.Custom, dest.Custom)

	// Output:
	// Username: test_user string
	// Password: test123456 string
	// Age: 18 int
	// Custom: [111 test1 test2] []string
}
