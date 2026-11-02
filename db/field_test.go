package db

import (
	"database/sql/driver"
	"reflect"
	"testing"
)

type fieldTestModel struct {
	ID      int64  `field_name:"id" field_type:"BIGINT"`
	Name    string `field_type:"VARCHAR(64)"`
	Ignored bool
	Skipped string `field_type:"-"`
	Empty   string `field_type:""`
	Blank   string `field_type:"   "`
}

type fieldValueTestModel struct {
	Value            int64
	Zero             int64
	Pointer          *int64
	NilPointer       *int64
	Valuer           *fieldTestValuer
	NilValuer        *fieldTestValuer
	hidden           int64
	hiddenPointer    *int64
	hiddenNilPointer *int64
}

type fieldTestValuer string

func (value *fieldTestValuer) Value() (driver.Value, error) {
	return "valuer:" + string(*value), nil
}

func TestModelFieldValue(t *testing.T) {
	number := int64(7)
	valuer := fieldTestValuer("custom")
	modelValue := reflect.ValueOf(fieldValueTestModel{
		Value:         1,
		Pointer:       &number,
		Valuer:        &valuer,
		hidden:        2,
		hiddenPointer: &number,
	})
	tests := []struct {
		name      string
		fieldName string
		want      any
		wantOK    bool
	}{
		{name: "ordinary value", fieldName: "Value", want: int64(1), wantOK: true},
		{name: "zero value", fieldName: "Zero", want: int64(0), wantOK: true},
		{name: "non nil pointer preserved", fieldName: "Pointer", want: &number, wantOK: true},
		{name: "nil pointer", fieldName: "NilPointer", want: nil, wantOK: true},
		{name: "valuer pointer preserved", fieldName: "Valuer", want: &valuer, wantOK: true},
		{name: "nil valuer pointer", fieldName: "NilValuer", want: nil, wantOK: true},
		{name: "missing field", fieldName: "Missing"},
		{name: "unexported value", fieldName: "hidden"},
		{name: "unexported non nil pointer", fieldName: "hiddenPointer"},
		{name: "unexported nil pointer", fieldName: "hiddenNilPointer"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, ok := ModelFieldValue(modelValue, test.fieldName)
			// These comparable values also verify pointer identity, not only pointed-to contents.
			if value != test.want || ok != test.wantOK {
				t.Fatalf("ModelFieldValue(%q) = (%#v, %t), want (%#v, %t)", test.fieldName, value, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestModelFieldValueSQLParameterConversion(t *testing.T) {
	number := int64(7)
	valuer := fieldTestValuer("custom")
	modelValue := reflect.ValueOf(fieldValueTestModel{Pointer: &number, Valuer: &valuer})
	tests := []struct {
		fieldName string
		want      driver.Value
	}{
		{fieldName: "Pointer", want: number},
		{fieldName: "NilPointer", want: nil},
		{fieldName: "Valuer", want: "valuer:custom"},
		{fieldName: "NilValuer", want: nil},
	}
	for _, test := range tests {
		t.Run(test.fieldName, func(t *testing.T) {
			value, ok := ModelFieldValue(modelValue, test.fieldName)
			if !ok {
				t.Fatalf("ModelFieldValue(%q) rejected a readable field", test.fieldName)
			}
			converted, err := driver.DefaultParameterConverter.ConvertValue(value)
			if err != nil {
				t.Fatalf("ConvertValue(%q) returned an error: %v", test.fieldName, err)
			}
			if converted != test.want {
				t.Fatalf("ConvertValue(%q) = %#v, want %#v", test.fieldName, converted, test.want)
			}
		})
	}
}

func TestModelFields(t *testing.T) {
	modelValue, err := ModelStructValue(fieldTestModel{})
	if err != nil {
		t.Fatalf("ModelStructValue() returned an error: %v", err)
	}
	fields, fieldSQL := ModelFields(modelValue, NewColumnParser("field_name", "field_type"))
	if !reflect.DeepEqual(fields, []string{"ID", "Name"}) {
		t.Fatalf("ModelFields() fields = %#v", fields)
	}
	if !reflect.DeepEqual(fieldSQL, []string{"id", "name"}) {
		t.Fatalf("ModelFields() fieldSQL = %#v", fieldSQL)
	}
}

type customColumnParser struct{}

func (customColumnParser) Parse(field reflect.StructField) (string, string, bool) {
	if field.Name == "Ignored" {
		return "", "", false
	}
	return "custom_" + field.Name, "TEXT", true
}

func TestModelFieldsUsesCustomParser(t *testing.T) {
	modelValue := reflect.ValueOf(struct {
		ID      int64
		Name    string
		Ignored bool
	}{})
	fields, fieldSQL := ModelFields(modelValue, customColumnParser{})
	if !reflect.DeepEqual(fields, []string{"ID", "Name"}) {
		t.Fatalf("ModelFields() fields = %#v", fields)
	}
	if !reflect.DeepEqual(fieldSQL, []string{"custom_ID", "custom_Name"}) {
		t.Fatalf("ModelFields() fieldSQL = %#v", fieldSQL)
	}
}
