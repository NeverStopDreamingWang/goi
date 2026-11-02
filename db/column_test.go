package db

import (
	"reflect"
	"testing"
)

func TestGlobalColumnParserIgnoresNil(t *testing.T) {
	old := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(old) })
	SetColumnParser(NewColumnParser("field_name", "field_type"))
	SetColumnParser(nil)
	field := reflect.StructField{
		Name: "ID",
		Tag:  `field_name:"role_id" field_type:"BIGINT" column:"custom_id" sql_type:"TEXT"`,
	}
	checkDefault := func() {
		t.Helper()
		parser := GetColumnParser()
		if parser == nil {
			t.Fatal("default column parser is nil")
		}
		name, columnType, ok := parser.Parse(field)
		if name != "role_id" || columnType != "BIGINT" || !ok {
			t.Fatalf("default Parse() = (%q, %q, %v)", name, columnType, ok)
		}
	}
	checkDefault()
	custom := NewColumnParser("column", "sql_type")
	SetColumnParser(&custom)
	SetColumnParser(nil)
	if GetColumnParser() != &custom {
		t.Fatal("SetColumnParser(nil) changed the configured parser")
	}
	name, columnType, ok := GetColumnParser().Parse(field)
	if name != "custom_id" || columnType != "TEXT" || !ok {
		t.Fatalf("Parse() after nil = (%q, %q, %v)", name, columnType, ok)
	}
}

func TestGlobalColumnParserUsesProvidedObject(t *testing.T) {
	old := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(old) })
	custom := NewColumnParser("column", "sql_type")
	SetColumnParser(&custom)
	if GetColumnParser() != &custom {
		t.Fatal("SetColumnParser copied or replaced the configured object")
	}
	field := reflect.StructField{
		Name: "ID",
		Tag:  `field_name:"role_id" field_type:"BIGINT" column:"custom_id" sql_type:"TEXT"`,
	}
	name, columnType, ok := GetColumnParser().Parse(field)
	if name != "custom_id" || columnType != "TEXT" || !ok {
		t.Fatalf("custom Parse() = (%q, %q, %v)", name, columnType, ok)
	}
}

func TestDefaultColumnParser(t *testing.T) {
	defaultParser := NewColumnParser("field_name", "field_type")
	tests := []struct {
		name       string
		parser     ColumnParser
		field      reflect.StructField
		columnName string
		columnType string
		ok         bool
	}{
		{
			name:       "explicit column name",
			parser:     defaultParser,
			field:      reflect.StructField{Name: "ID", Tag: `field_name:"role_id" field_type:"BIGINT"`},
			columnName: "role_id",
			columnType: "BIGINT",
			ok:         true,
		},
		{
			name:       "lowercase field name",
			parser:     defaultParser,
			field:      reflect.StructField{Name: "DisplayName", Tag: `field_type:"VARCHAR(64)"`},
			columnName: "displayname",
			columnType: "VARCHAR(64)",
			ok:         true,
		},
		{
			name:   "missing column type tag",
			parser: defaultParser,
			field:  reflect.StructField{Name: "Ignored", Tag: `field_name:"ignored"`},
		},
		{
			name:   "empty column type is ignored",
			parser: defaultParser,
			field:  reflect.StructField{Name: "Name", Tag: `field_type:""`},
		},
		{
			name:   "blank column type is ignored",
			parser: defaultParser,
			field:  reflect.StructField{Name: "Name", Tag: `field_type:" \t\n "`},
		},
		{
			name:   "dash column type is ignored",
			parser: defaultParser,
			field:  reflect.StructField{Name: "Name", Tag: `field_type:"-"`},
		},
		{
			name:   "dash surrounded by whitespace is ignored",
			parser: defaultParser,
			field:  reflect.StructField{Name: "Name", Tag: `field_type:" \t- "`},
		},
		{
			name:       "valid column type preserves whitespace",
			parser:     defaultParser,
			field:      reflect.StructField{Name: "Name", Tag: `field_type:"  VARCHAR(64) NOT NULL  "`},
			columnName: "name",
			columnType: "  VARCHAR(64) NOT NULL  ",
			ok:         true,
		},
		{
			name:       "empty column name tag is preserved",
			parser:     defaultParser,
			field:      reflect.StructField{Name: "Name", Tag: `field_name:"" field_type:"TEXT"`},
			columnType: "TEXT",
			ok:         true,
		},
		{
			name:       "custom tag names",
			parser:     NewColumnParser("column", "sql_type"),
			field:      reflect.StructField{Name: "ID", Tag: `field_name:"legacy" field_type:"INT" column:"custom_id" sql_type:"BIGINT"`},
			columnName: "custom_id",
			columnType: "BIGINT",
			ok:         true,
		},
		{
			name:   "custom parser ignores default tags",
			parser: NewColumnParser("column", "sql_type"),
			field:  reflect.StructField{Name: "ID", Tag: `field_name:"id" field_type:"INT"`},
		},
		{
			name:   "custom empty column type is ignored",
			parser: NewColumnParser("column", "sql_type"),
			field:  reflect.StructField{Name: "Name", Tag: `column:"name" sql_type:""`},
		},
		{
			name:   "custom dash column type is ignored",
			parser: NewColumnParser("column", "sql_type"),
			field:  reflect.StructField{Name: "Name", Tag: `column:"name" sql_type:"-"`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			columnName, columnType, ok := test.parser.Parse(test.field)
			if columnName != test.columnName || columnType != test.columnType || ok != test.ok {
				t.Fatalf("Parse() = (%q, %q, %v), want (%q, %q, %v)",
					columnName, columnType, ok, test.columnName, test.columnType, test.ok)
			}
		})
	}
}
