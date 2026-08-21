package sqlite3

import "testing"

type validatorNilModel struct{}

func (*validatorNilModel) ModelSet() *Settings { return &Settings{} }

func TestValidateNilModel(t *testing.T) {
	var model *validatorNilModel
	if err := Validate(model, false); err == nil {
		t.Fatal("Validate() did not reject a nil model")
	}
}
