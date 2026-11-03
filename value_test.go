package goi_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

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

func TestSetValueArray(t *testing.T) {
	sliceSource := []any{1, "2"}
	arraySource := [2]string{"1", "2"}
	for _, test := range []struct {
		name   string
		source any
	}{
		{name: "slice", source: sliceSource},
		{name: "array", source: arraySource},
		{name: "same_type", source: [2]int{1, 2}},
		{name: "slice_pointer", source: &sliceSource},
		{name: "array_pointer", source: &arraySource},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got [2]int
			if err := goi.SetValue(reflect.ValueOf(&got), test.source); err != nil {
				t.Fatal(err)
			}
			if want := [2]int{1, 2}; got != want {
				t.Fatalf("SetValue result = %v; want %v", got, want)
			}
		})
	}
}

func TestSetValueArrayPointerTarget(t *testing.T) {
	var got *[2]int
	if err := goi.SetValue(reflect.ValueOf(&got).Elem(), []any{1, "2"}); err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != [2]int{1, 2} {
		t.Fatalf("SetValue result = %v; want pointer to [1 2]", got)
	}
}

func TestSetValueNestedArray(t *testing.T) {
	var got [2][2]int
	source := [][]any{{1, "2"}, {"3", 4}}
	if err := goi.SetValue(reflect.ValueOf(&got), source); err != nil {
		t.Fatal(err)
	}
	if want := [2][2]int{{1, 2}, {3, 4}}; got != want {
		t.Fatalf("SetValue result = %v; want %v", got, want)
	}
}

func TestSetValueArrayRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name   string
		source any
	}{
		{name: "invalid_element", source: []any{1, "invalid"}},
		{name: "number", source: 12},
		{name: "map", source: map[string]int{"value": 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got [2]int
			if err := goi.SetValue(reflect.ValueOf(&got), test.source); err == nil {
				t.Fatalf("SetValue(%#v) must return an error", test.source)
			}
		})
	}
}
