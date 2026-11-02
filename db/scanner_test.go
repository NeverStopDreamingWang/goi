package db

import (
	"errors"
	"reflect"
	"testing"
)

type resultTestModel struct {
	ID   int64  `field_name:"id" field_type:"BIGINT"`
	Name string `field_type:"VARCHAR(64)"`
}

type resultTestRowScanner struct {
	values []any
	err    error
}

func (scanner resultTestRowScanner) Scan(dest ...any) error {
	if scanner.err != nil {
		return scanner.err
	}
	if len(dest) != len(scanner.values) {
		return errors.New("scan destination count does not match values")
	}
	for i, value := range scanner.values {
		target := reflect.ValueOf(dest[i])
		if !target.IsValid() || target.Kind() != reflect.Ptr || target.IsNil() {
			return errors.New("scan destination is not a non-nil pointer")
		}
		target = target.Elem()
		source := reflect.ValueOf(value)
		if !source.IsValid() {
			target.Set(reflect.Zero(target.Type()))
			continue
		}
		if !source.Type().AssignableTo(target.Type()) {
			return errors.New("scan value cannot be assigned to destination")
		}
		target.Set(source)
	}
	return nil
}

func TestPrepareSliceResult(t *testing.T) {
	fields := []string{"ID", "Name"}

	models := []resultTestModel{{ID: 1, Name: "existing"}}
	writeModel, err := PrepareSliceResult(resultTestModel{}, &models, fields)
	if err != nil {
		t.Fatalf("PrepareSliceResult(struct) returned an error: %v", err)
	}
	if err = writeModel(resultTestRowScanner{values: []any{int64(2), "second"}}); err != nil {
		t.Fatalf("ResultWriter(struct) returned an error: %v", err)
	}
	if err = writeModel(resultTestRowScanner{values: []any{int64(3), "third"}}); err != nil {
		t.Fatalf("ResultWriter(struct) returned an error: %v", err)
	}
	if len(models) != 3 || models[0].Name != "existing" || models[1].ID != 2 || models[2].Name != "third" {
		t.Fatalf("unexpected struct results: %#v", models)
	}

	var modelPointers []*resultTestModel
	writeModelPointer, err := PrepareSliceResult(resultTestModel{}, &modelPointers, fields)
	if err != nil {
		t.Fatalf("PrepareSliceResult(*struct) returned an error: %v", err)
	}
	if err = writeModelPointer(resultTestRowScanner{values: []any{int64(4), "pointer"}}); err != nil {
		t.Fatalf("ResultWriter(*struct) returned an error: %v", err)
	}
	if len(modelPointers) != 1 || modelPointers[0] == nil || modelPointers[0].Name != "pointer" {
		t.Fatalf("unexpected struct pointer results: %#v", modelPointers)
	}

	var maps []map[string]any
	writeMap, err := PrepareSliceResult(resultTestModel{}, &maps, fields)
	if err != nil {
		t.Fatalf("PrepareSliceResult(map) returned an error: %v", err)
	}
	if err = writeMap(resultTestRowScanner{values: []any{int64(5), "map"}}); err != nil {
		t.Fatalf("ResultWriter(map) returned an error: %v", err)
	}
	if len(maps) != 1 || maps[0]["ID"] != int64(5) || maps[0]["Name"] != "map" {
		t.Fatalf("unexpected map results: %#v", maps)
	}
}

func TestPrepareSliceResultRejectsInvalidTargets(t *testing.T) {
	fields := []string{"ID", "Name"}
	if _, err := PrepareSliceResult(resultTestModel{}, nil, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted nil")
	}
	directSlice := []resultTestModel{}
	if _, err := PrepareSliceResult(resultTestModel{}, directSlice, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted a direct slice")
	}
	var nilSlicePointer *[]resultTestModel
	if _, err := PrepareSliceResult(resultTestModel{}, nilSlicePointer, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted a nil slice pointer")
	}
	array := [1]resultTestModel{}
	if _, err := PrepareSliceResult(resultTestModel{}, &array, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted an array")
	}
	var values []string
	if _, err := PrepareSliceResult(resultTestModel{}, &values, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted a basic element type")
	}
	var doublePointers []**resultTestModel
	if _, err := PrepareSliceResult(resultTestModel{}, &doublePointers, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted a double pointer element")
	}
	var mapPointers []*map[string]any
	if _, err := PrepareSliceResult(resultTestModel{}, &mapPointers, fields); err == nil {
		t.Fatal("PrepareSliceResult() accepted a map pointer element")
	}
}

func TestPrepareSingleResult(t *testing.T) {
	fields := []string{"ID", "Name"}

	var model resultTestModel
	writeModel, err := PrepareSingleResult(resultTestModel{}, &model, fields)
	if err != nil {
		t.Fatalf("PrepareSingleResult(struct) returned an error: %v", err)
	}
	if err = writeModel(resultTestRowScanner{values: []any{int64(1), "struct"}}); err != nil {
		t.Fatalf("ResultWriter(struct) returned an error: %v", err)
	}
	if model.ID != 1 || model.Name != "struct" {
		t.Fatalf("unexpected struct result: %#v", model)
	}

	result := map[string]any{}
	writeMap, err := PrepareSingleResult(resultTestModel{}, result, fields)
	if err != nil {
		t.Fatalf("PrepareSingleResult(map) returned an error: %v", err)
	}
	if err = writeMap(resultTestRowScanner{values: []any{int64(2), "map"}}); err != nil {
		t.Fatalf("ResultWriter(map) returned an error: %v", err)
	}
	if result["ID"] != int64(2) || result["Name"] != "map" {
		t.Fatalf("unexpected map result: %#v", result)
	}
}

func TestPrepareSingleResultRejectsInvalidTargets(t *testing.T) {
	fields := []string{"ID", "Name"}

	if _, err := PrepareSingleResult(resultTestModel{}, nil, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted nil")
	}
	if _, err := PrepareSingleResult(resultTestModel{}, resultTestModel{}, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted a struct value")
	}
	var nilModel *resultTestModel
	if _, err := PrepareSingleResult(resultTestModel{}, nilModel, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted a nil struct pointer")
	}
	var nilMap map[string]any
	if _, err := PrepareSingleResult(resultTestModel{}, nilMap, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted a direct nil map")
	}
	pointerMap := map[string]any{}
	if _, err := PrepareSingleResult(resultTestModel{}, &pointerMap, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted a map pointer")
	}
	model := &resultTestModel{}
	if _, err := PrepareSingleResult(resultTestModel{}, &model, fields); err == nil {
		t.Fatal("PrepareSingleResult() accepted a double struct pointer")
	}
}

func TestResultWriterReturnsScanError(t *testing.T) {
	var models []resultTestModel
	writeResult, err := PrepareSliceResult(resultTestModel{}, &models, []string{"ID"})
	if err != nil {
		t.Fatalf("PrepareSliceResult() returned an error: %v", err)
	}
	if err = writeResult(resultTestRowScanner{values: []any{"invalid"}}); err == nil {
		t.Fatal("ResultWriter() accepted a value that cannot be scanned into the struct field")
	}
	if len(models) != 0 {
		t.Fatalf("ResultWriter() appended an invalid result: %#v", models)
	}
}

func TestResultWriterPreservesTargetOnScanError(t *testing.T) {
	fields := []string{"ID", "Name"}
	wantErr := errors.New("scan error")

	var models []resultTestModel
	writeModels, err := PrepareSliceResult(resultTestModel{}, &models, fields)
	if err != nil {
		t.Fatalf("PrepareSliceResult() returned an error: %v", err)
	}
	if err = writeModels(resultTestRowScanner{err: wantErr}); !errors.Is(err, wantErr) {
		t.Fatalf("ResultWriter() error = %v, want %v", err, wantErr)
	}
	if len(models) != 0 {
		t.Fatalf("ResultWriter() appended a value after scan failure: %#v", models)
	}

	result := resultTestModel{ID: 1, Name: "existing"}
	writeFirst, err := PrepareSingleResult(resultTestModel{}, &result, fields)
	if err != nil {
		t.Fatalf("PrepareSingleResult() returned an error: %v", err)
	}
	if err = writeFirst(resultTestRowScanner{err: wantErr}); !errors.Is(err, wantErr) {
		t.Fatalf("ResultWriter() error = %v, want %v", err, wantErr)
	}
	if result.ID != 1 || result.Name != "existing" {
		t.Fatalf("ResultWriter() modified the struct after scan failure: %#v", result)
	}
}
