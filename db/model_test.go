package db

import (
	"reflect"
	"testing"
)

type modelTestModel struct {
	ID int64
}

type otherModelTestModel struct {
	ID int64
}

func TestModelStructValue(t *testing.T) {
	model := &modelTestModel{ID: 1}
	value, err := ModelStructValue(model)
	if err != nil {
		t.Fatalf("ModelStructValue() returned an error: %v", err)
	}
	if value.Type() != reflect.TypeOf(modelTestModel{}) || value.FieldByName("ID").Int() != 1 {
		t.Fatalf("unexpected model value: %#v", value.Interface())
	}

	var nilModel *modelTestModel
	if _, err = ModelStructValue(nilModel); err == nil {
		t.Fatal("ModelStructValue() accepted a nil model")
	}
	if _, err = ModelStructValue("model"); err == nil {
		t.Fatal("ModelStructValue() accepted a non-struct model")
	}
}

func TestValidateOperationModel(t *testing.T) {
	operation := &modelTestModel{ID: 1}
	value, err := ValidateOperationModel(modelTestModel{}, operation)
	if err != nil {
		t.Fatalf("ValidateOperationModel() returned an error: %v", err)
	}
	if value.Type() != reflect.TypeOf(modelTestModel{}) {
		t.Fatalf("unexpected operation model type: %v", value.Type())
	}

	if _, err = ValidateOperationModel(modelTestModel{}, otherModelTestModel{}); err == nil {
		t.Fatal("ValidateOperationModel() accepted a different model type")
	}
}
