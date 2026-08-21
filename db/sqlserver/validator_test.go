package sqlserver

import (
	"reflect"
	"testing"
)

type validatorNamedString string

type validatorNilModel struct{}

func (*validatorNilModel) ModelSet() *Settings { return &Settings{} }

func TestValidateFieldTypeNamedString(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		value     validatorNamedString
		wantError bool
	}{
		{name: "nvarchar characters", fieldType: "NVARCHAR(2)", value: "中文", wantError: false},
		{name: "nvarchar too long", fieldType: "NVARCHAR(1)", value: "中文", wantError: true},
		{name: "nvarchar surrogate pair", fieldType: "NVARCHAR(1)", value: "😀", wantError: true},
		{name: "varchar bytes", fieldType: "VARCHAR(6)", value: "中文", wantError: false},
		{name: "varchar bytes too long", fieldType: "VARCHAR(5)", value: "中文", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFieldType(test.fieldType, reflect.ValueOf(test.value))
			if (err != nil) != test.wantError {
				t.Fatalf("validateFieldType() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestValidateNilModel(t *testing.T) {
	var model *validatorNilModel
	if err := Validate(model, false); err == nil {
		t.Fatal("Validate() did not reject a nil model")
	}
}
