package goi_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi/v3"
)

type testStruct struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Age      int      `json:"age"`
	Custom   []string `json:"custom"`
}

func ExampleSetValue() {
	source := map[string]any{
		"username": "test_user",
		"password": "test123456",
		"age":      18,
		"custom":   []any{true, "test1", "test2"},
	}
	dest := &testStruct{}

	destValue := reflect.ValueOf(dest)
	err := goi.SetValue(destValue, source)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Username: %+v %T\n", dest.Username, dest.Username)
	fmt.Printf("Password: %+v %T\n", dest.Password, dest.Password)
	fmt.Printf("Age: %+v %T\n", dest.Age, dest.Age)
	fmt.Printf("Custom: %+v %T\n", dest.Custom, dest.Custom)

	// Output:
	// Username: test_user string
	// Password: test123456 string
	// Age: 18 int
	// Custom: [true test1 test2] []string
}

func ExampleSetValue_manual() {
	source := map[string]any{
		"username": "test_user",
		"password": "test123456",
		"age":      18,
		"custom":   []any{true, "test1", "test2"},
	}
	dest := &testStruct{}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		panic("dest must be a pointer")
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

		fieldName = fieldType.Tag.Get("json")
		fieldName, _, _ = strings.Cut(fieldName, ",")
		if fieldName == "-" {
			continue
		}

		var value any
		var ok bool
		if fieldName != "" {
			value, ok = source[fieldName]
		}
		if !ok {
			value, ok = source[fieldType.Name]
		}
		if !ok {
			continue
		}

		err = goi.SetValue(fieldValue, value)
		if err != nil {
			panic(err)
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
	// Custom: [true test1 test2] []string
}
