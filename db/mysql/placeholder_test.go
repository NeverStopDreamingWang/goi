package mysql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3/db"
)

type customPlaceholderParser struct{}

func (customPlaceholderParser) Placeholder() string {
	return ":v"
}

func (customPlaceholderParser) Parse(query string, args ...any) string {
	for index := 1; index <= len(args); index++ {
		query = strings.Replace(query, ":v", fmt.Sprintf("$%d", index), 1)
	}
	return query
}

func TestSetPlaceholderParser(t *testing.T) {
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(db.NewPlaceholderParser("?"))
	})
	SetPlaceholderParser(nil)

	engine, recorder := newEngineTestEngine(t)
	name := "admin"
	if _, err := engine.Fields("Name").Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatalf("Insert() returned an error: %v", err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` (`name`) VALUES ($1)" {
		t.Fatalf("unexpected parsed INSERT SQL: %s", recorder.execQuery)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != name {
		t.Fatalf("unexpected INSERT args: %#v", recorder.execArgs)
	}
}

func TestPlaceholderParserBuildsAndParsesWhereIn(t *testing.T) {
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(db.NewPlaceholderParser("?"))
	})

	engine, recorder := newEngineTestEngine(t)
	query := engine.Where("id = :v AND status IN :v", int64(1), []string{"enabled", "disabled"})
	if !reflect.DeepEqual(query.whereSQL, []string{"id = :v AND status IN (:v,:v)"}) {
		t.Fatalf("unexpected ORM WHERE SQL: %#v", query.whereSQL)
	}
	if !reflect.DeepEqual(query.args, []any{int64(1), "enabled", "disabled"}) {
		t.Fatalf("unexpected ORM WHERE args: %#v", query.args)
	}

	var roles []engineTestModel
	if err := query.Select(context.Background(), &roles); err != nil {
		t.Fatalf("Select() returned an error: %v", err)
	}
	wantSQL := "SELECT `id`,`name`,`status` FROM `test_schema`.`roles` WHERE id = $1 AND status IN ($2,$3)"
	if recorder.queryQuery != wantSQL {
		t.Fatalf("unexpected parsed SELECT SQL: %s", recorder.queryQuery)
	}
}

func TestPlaceholderParserParsesQueryRowContext(t *testing.T) {
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(db.NewPlaceholderParser("?"))
	})

	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"id"}
	recorder.rows = [][]driver.Value{{int64(1)}}

	var id int64
	if err := engine.QueryRowContext(context.Background(), "SELECT :v", int64(1)).Scan(&id); err != nil {
		t.Fatalf("QueryRowContext().Scan() returned an error: %v", err)
	}
	if recorder.queryQuery != "SELECT $1" {
		t.Fatalf("unexpected parsed QueryRow SQL: %s", recorder.queryQuery)
	}
}

func TestEnginePreservesPlaceholderParser(t *testing.T) {
	SetPlaceholderParser(db.NewPlaceholderParser("?"))
	t.Cleanup(func() {
		SetPlaceholderParser(db.NewPlaceholderParser("?"))
	})

	defaultEngine, defaultRecorder := newEngineTestEngine(t)
	SetPlaceholderParser(customPlaceholderParser{})
	customEngine, customRecorder := newEngineTestEngine(t)

	if _, err := defaultEngine.ExecuteContext(context.Background(), "UPDATE roles SET name = ?", "default"); err != nil {
		t.Fatalf("default Engine ExecuteContext() returned an error: %v", err)
	}
	if defaultRecorder.execQuery != "UPDATE roles SET name = ?" {
		t.Fatalf("existing Engine did not preserve its placeholder parser: %s", defaultRecorder.execQuery)
	}

	if _, err := customEngine.ExecuteContext(context.Background(), "UPDATE roles SET name = :v", "custom"); err != nil {
		t.Fatalf("custom Engine ExecuteContext() returned an error: %v", err)
	}
	if customRecorder.execQuery != "UPDATE roles SET name = $1" {
		t.Fatalf("new Engine did not use the updated placeholder parser: %s", customRecorder.execQuery)
	}
}
