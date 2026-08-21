package kingbase

import (
	"reflect"
	"testing"
)

type validatorNamedString string

type validatorNilModel struct{}

func (*validatorNilModel) ModelSet() *Settings { return &Settings{} }

func TestValidateFieldTypeNamedString(t *testing.T) {
	if err := validateFieldType("VARCHAR(2)", reflect.ValueOf(validatorNamedString("待定"))); err != nil {
		t.Fatalf("validateFieldType() returned an error for a named string: %v", err)
	}
	if err := validateFieldType("VARCHAR(1)", reflect.ValueOf(validatorNamedString("待定"))); err == nil {
		t.Fatal("validateFieldType() did not enforce the VARCHAR character limit")
	}
}

func TestValidateNilModel(t *testing.T) {
	var model *validatorNilModel
	if err := Validate(model, false); err == nil {
		t.Fatal("Validate() did not reject a nil model")
	}
}
