package oracle

import (
	"reflect"
	"testing"
)

type validatorNamedString string

type validatorNilModel struct{}

func (*validatorNilModel) ModelSet() *Settings { return &Settings{} }

func TestValidateFieldTypeNamedString(t *testing.T) {
	value := reflect.ValueOf(validatorNamedString("中文"))
	tests := []struct {
		fieldType string
		wantError bool
	}{
		{fieldType: "VARCHAR2(6 BYTE)", wantError: false},
		{fieldType: "VARCHAR2(5 BYTE)", wantError: true},
		{fieldType: "VARCHAR2(2 CHAR)", wantError: false},
		{fieldType: "VARCHAR2(1 CHAR)", wantError: true},
		{fieldType: "NVARCHAR2(2)", wantError: false},
		{fieldType: "NVARCHAR2(1)", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.fieldType, func(t *testing.T) {
			err := validateFieldType(test.fieldType, value)
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
