package mysql

import (
	"context"
	"database/sql"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
)

type columnParserTestModel struct {
	ID      *int64  `field_name:"id" field_type:"BIGINT" column:"role_id" sql_type:"BIGINT"`
	Name    *string `field_name:"name" field_type:"VARCHAR(64)" column:"display_name" sql_type:"VARCHAR(2) COMMENT '显示名称'"`
	Ignored *string
	Skipped *string `field_type:"-" sql_type:"-"`
	Empty   *string `field_type:"" sql_type:""`
	Blank   *string `field_type:"   " sql_type:"   "`
}

func (columnParserTestModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "field_parser_roles"}
}

type customColumnParser struct{}

func (customColumnParser) Parse(field reflect.StructField) (string, string, bool) {
	if field.Name != "Name" {
		return "", "", false
	}
	return "custom_name", "VARCHAR(1)", true
}

func TestGetSetColumnParser(t *testing.T) {
	original := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(original) })

	SetColumnParser(customColumnParser{})
	field, _ := reflect.TypeOf(engineTestModel{}).FieldByName("Name")
	columnName, columnType, ok := GetColumnParser().Parse(field)
	if columnName != "custom_name" || columnType != "VARCHAR(1)" || !ok {
		t.Fatalf("GetColumnParser().Parse() = (%q, %q, %v)", columnName, columnType, ok)
	}
}

func TestEnginePreservesColumnParser(t *testing.T) {
	original := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(original) })

	SetColumnParser(db.NewColumnParser("field_name", "field_type"))
	base, recorder := newEngineTestEngine(t)
	database := &goi.Database{
		Engine: driverName,
		Connect: func(string) *sql.DB {
			return base.DB
		},
	}
	defaultEngine := ConnectDatabase("test", database)
	SetColumnParser(db.NewColumnParser("column", "sql_type"))
	customEngine := ConnectDatabase("test", database)

	// 在全局配置切换后才设置模型，验证旧 Engine 仍使用创建时保存的解析器。
	defaultEngine.SetModel(columnParserTestModel{})
	customEngine.SetModel(columnParserTestModel{})
	if !reflect.DeepEqual(defaultEngine.fieldSQL, []string{"id", "name"}) {
		t.Fatalf("existing Engine changed its parser: %#v", defaultEngine.fieldSQL)
	}
	if !reflect.DeepEqual(customEngine.fieldSQL, []string{"role_id", "display_name"}) {
		t.Fatalf("new Engine did not use custom tags: %#v", customEngine.fieldSQL)
	}

	SetColumnParser(db.NewColumnParser("field_name", "field_type"))
	name := "admin"
	model := columnParserTestModel{Name: &name, Skipped: &name, Empty: &name, Blank: &name}
	if _, err := defaultEngine.Fields("Name", "Ignored", "Skipped", "Empty", "Blank").Insert(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`field_parser_roles` (`name`) VALUES (?)" {
		t.Fatalf("default Engine generated %q", recorder.execQuery)
	}
	if _, err := customEngine.Fields("Name", "Ignored", "Skipped", "Empty", "Blank").Insert(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`field_parser_roles` (`display_name`) VALUES (?)" {
		t.Fatalf("custom Engine did not preserve its parser: %q", recorder.execQuery)
	}
}

func TestCustomColumnParserControlsFieldsAndValidation(t *testing.T) {
	original := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(original) })
	SetColumnParser(customColumnParser{})

	engine, recorder := newEngineTestEngine(t)
	if !reflect.DeepEqual(engine.fields, []string{"Name"}) ||
		!reflect.DeepEqual(engine.fieldSQL, []string{"custom_name"}) {
		t.Fatalf("SetModel() did not use the custom parser: fields=%#v columns=%#v", engine.fields, engine.fieldSQL)
	}
	name := "ab"
	if _, err := engine.Fields("ID", "Name", "Status").Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` (`custom_name`) VALUES (?)" {
		t.Fatalf("Fields() did not use the custom parser: %q", recorder.execQuery)
	}
	if err := Validate(engineTestModel{Name: &name}, true); err == nil || !strings.Contains(err.Error(), `"custom_name"`) {
		t.Fatalf("Validate() did not use the custom column name and type: %v", err)
	}

	SetColumnParser(db.NewColumnParser("field_name", "field_type"))
	if err := Validate(engineTestModel{Name: &name}, true); err != nil {
		t.Fatalf("Validate() did not obtain the current package parser: %v", err)
	}
}

func TestMigrateUsesColumnParser(t *testing.T) {
	original := GetColumnParser()
	originalLog := goi.Log
	t.Cleanup(func() {
		SetColumnParser(original)
		goi.Log = originalLog
	})
	SetColumnParser(db.NewColumnParser("column", "sql_type"))
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}

	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"exists"}
	recorder.rows = nil
	SetColumnParser(db.NewColumnParser("field_name", "field_type"))
	engine.Migrate(context.Background(), columnParserTestModel{})
	wantSQL := "CREATE TABLE `test_schema`.`field_parser_roles` (\n" +
		"  `role_id` BIGINT,\n" +
		"  `display_name` VARCHAR(2) COMMENT '显示名称'\n)"
	if recorder.execQuery != wantSQL {
		t.Fatalf("Migrate() generated %q, want %q", recorder.execQuery, wantSQL)
	}
}

func TestValidateUsesCustomTypeComment(t *testing.T) {
	original := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(original) })
	SetColumnParser(db.NewColumnParser("column", "sql_type"))

	name := "过长名称"
	if err := Validate(columnParserTestModel{Name: &name}, true); err == nil || !strings.Contains(err.Error(), `"显示名称"`) {
		t.Fatalf("Validate() did not extract COMMENT from the parsed column type: %v", err)
	}
	name = "名称"
	if err := Validate(columnParserTestModel{Name: &name}, true); err != nil {
		t.Fatalf("Validate() rejected a valid value: %v", err)
	}
}

func TestGetCommentPrefersCommentTag(t *testing.T) {
	field := reflect.StructField{Name: "Name", Tag: `comment:"标签注释"`}
	if got := getComment(field, "VARCHAR(2) COMMENT '类型注释'"); got != "标签注释" {
		t.Fatalf("getComment() = %q, want the comment tag", got)
	}
}
