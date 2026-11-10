package mysql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultPlaceholderParserExpandsInParameters(t *testing.T) {
	parser := GetPlaceholderParser()
	if parser.Placeholder() != "?" {
		t.Fatalf("default Placeholder() = %q, want ?", parser.Placeholder())
	}
	query, args := parser.Parse("id IN ? AND name = ?", []int64{1, 2}, "admin")
	if query != "id IN (?,?) AND name = ?" || !reflect.DeepEqual(args, []any{int64(1), int64(2), "admin"}) {
		t.Fatalf("default Parse() = (%q, %#v)", query, args)
	}
}

func TestDefaultPlaceholderParserWritesMySQLPlaceholders(t *testing.T) {
	parser := defaultPlaceholderParser{}
	placeholder := parser.Placeholder()
	tests := []struct {
		name     string
		query    string
		args     []any
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "scalar and collection", query: "tenant_id = " + placeholder + " AND id IN " + placeholder + " AND name = " + placeholder,
			args: []any{7, []int64{1, 2}, "admin"}, wantSQL: "tenant_id = ? AND id IN (?,?) AND name = ?", wantArgs: []any{7, int64(1), int64(2), "admin"},
		},
		{
			name: "empty collection", query: "id IN " + placeholder + " AND name = " + placeholder,
			args: []any{[]int64{}, "admin"}, wantSQL: "id IN () AND name = ?", wantArgs: []any{"admin"},
		},
		{
			name: "manual parentheses", query: "id IN (" + placeholder + "," + placeholder + ")",
			args: []any{1, 2}, wantSQL: "id IN (?,?)", wantArgs: []any{1, 2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, args := parser.Parse(test.query, test.args...)
			if query != test.wantSQL || !reflect.DeepEqual(args, test.wantArgs) {
				t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", query, args, test.wantSQL, test.wantArgs)
			}
		})
	}
}

type customPlaceholderParser struct{}

func (customPlaceholderParser) Placeholder() string {
	return ":v"
}

func (customPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	query = strings.ReplaceAll(query, ":v", "?")
	query, args = (defaultPlaceholderParser{}).Parse(query, args...)
	for index := 1; index <= len(args); index++ {
		query = strings.Replace(query, "?", fmt.Sprintf("$%d", index), 1)
	}
	return query, args
}

func TestSetPlaceholderParser(t *testing.T) {
	original := GetPlaceholderParser()
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(original)
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
	original := GetPlaceholderParser()
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(original)
	})

	engine, recorder := newEngineTestEngine(t)
	query := engine.Where("id = :v AND status IN :v", int64(1), []string{"enabled", "disabled"})
	if !reflect.DeepEqual(query.whereSQL, []string{"id = :v AND status IN :v"}) {
		t.Fatalf("unexpected ORM WHERE SQL: %#v", query.whereSQL)
	}
	if !reflect.DeepEqual(query.args, []any{int64(1), []string{"enabled", "disabled"}}) {
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
	if query.GetSQL() != wantSQL {
		t.Fatalf("GetSQL() = %q, want %q", query.GetSQL(), wantSQL)
	}
}

type recordingPlaceholderParser struct {
	query string
	args  []any
}

func (*recordingPlaceholderParser) Placeholder() string { return ":v" }

func (parser *recordingPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	parser.query = query
	parser.args = append([]any(nil), args...)
	return customPlaceholderParser{}.Parse(query, args...)
}

func TestExecutionMethodsUseParserResults(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		execute bool
		run     func(*Engine, string, ...any) error
	}{
		{name: "ExecuteContext", execute: true, run: func(engine *Engine, query string, args ...any) error {
			_, err := engine.ExecuteContext(ctx, query, args...)
			return err
		}},
		{name: "Execute", execute: true, run: func(engine *Engine, query string, args ...any) error {
			_, err := engine.Execute(query, args...)
			return err
		}},
		{name: "QueryContext", run: func(engine *Engine, query string, args ...any) error {
			rows, err := engine.QueryContext(ctx, query, args...)
			if err != nil {
				return err
			}
			return rows.Close()
		}},
		{name: "Query", run: func(engine *Engine, query string, args ...any) error {
			rows, err := engine.Query(query, args...)
			if err != nil {
				return err
			}
			return rows.Close()
		}},
		{name: "QueryRowContext", run: func(engine *Engine, query string, args ...any) error {
			var id int64
			return engine.QueryRowContext(ctx, query, args...).Scan(&id)
		}},
		{name: "QueryRow", run: func(engine *Engine, query string, args ...any) error {
			var id int64
			return engine.QueryRow(query, args...).Scan(&id)
		}},
	}

	const query = "SELECT id FROM roles WHERE tenant_id = :v AND id IN :v AND status NOT IN :v AND name = :v"
	const expandedQuery = "SELECT id FROM roles WHERE tenant_id = :v AND id IN (:v,:v) AND status NOT IN (:v,:v) AND name = :v"
	const wantQuery = "SELECT id FROM roles WHERE tenant_id = $1 AND id IN ($2,$3) AND status NOT IN ($4,$5) AND name = $6"
	args := []any{int64(7), []int64{1, 2}, [2]string{"disabled", "deleted"}, "admin"}
	wantArgs := []any{int64(7), int64(1), int64(2), "disabled", "deleted", "admin"}

	for _, test := range tests {
		for _, transaction := range []bool{false, true} {
			for _, custom := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/tx=%t/custom=%t", test.name, transaction, custom), func(t *testing.T) {
					engine, recorder := newEngineTestEngine(t)
					recorder.columns = []string{"id"}
					recorder.rows = [][]driver.Value{{int64(1)}}
					parser := &recordingPlaceholderParser{}
					inputQuery, expectedQuery := query, wantQuery
					if custom {
						engine.placeholderParser = parser
					} else {
						engine.placeholderParser = defaultPlaceholderParser{}
						inputQuery = strings.ReplaceAll(query, ":v", "?")
						expectedQuery = strings.ReplaceAll(expandedQuery, ":v", "?")
					}
					var gotSQL string
					run := func(current *Engine, _ ...any) error {
						err := test.run(current, inputQuery, args...)
						gotSQL = current.GetSQL()
						return err
					}
					var err error
					if transaction {
						err = engine.WithTransaction(ctx, run)
					} else {
						err = run(engine)
					}
					if err != nil {
						t.Fatalf("execution returned an error: %v", err)
					}
					if custom && (parser.query != query || !reflect.DeepEqual(parser.args, args)) {
						t.Fatalf("Parse received SQL=%q args=%#v, want original SQL=%q args=%#v", parser.query, parser.args, query, args)
					}
					gotQuery, gotArgs := recorder.queryQuery, recorder.queryArgs
					if test.execute {
						gotQuery, gotArgs = recorder.execQuery, recorder.execArgs
					}
					if gotSQL != expectedQuery || gotQuery != expectedQuery {
						t.Fatalf("GetSQL=%q driver SQL=%q, want %q", gotSQL, gotQuery, expectedQuery)
					}
					if len(gotArgs) != len(wantArgs) {
						t.Fatalf("driver args=%#v, want %#v", gotArgs, wantArgs)
					}
					for index, arg := range gotArgs {
						if arg.Ordinal != index+1 || !reflect.DeepEqual(arg.Value, wantArgs[index]) {
							t.Fatalf("driver argument %d = %#v, want %#v", index, arg, wantArgs[index])
						}
					}
					if transaction && (recorder.beginCount != 1 || recorder.commitCount != 1 || recorder.rollbackCount != 0) {
						t.Fatalf("unexpected transaction counts: begin=%d commit=%d rollback=%d", recorder.beginCount, recorder.commitCount, recorder.rollbackCount)
					}
				})
			}
		}
	}
}

func TestPlaceholderParserParsesQueryRowContext(t *testing.T) {
	original := GetPlaceholderParser()
	SetPlaceholderParser(customPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(original)
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
	original := GetPlaceholderParser()
	SetPlaceholderParser(defaultPlaceholderParser{})
	t.Cleanup(func() {
		SetPlaceholderParser(original)
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

func TestExpandInParameterReportsCollectionExpansion(t *testing.T) {
	values := []int64{1, 2}
	array := [2]string{"enabled", "disabled"}
	valuesPointer := &values
	var nilSlice []int64
	var nilPointer *[]int64
	bytes := []byte{1, 2}

	tests := []struct {
		name        string
		leftPart    string
		placeholder string
		param       any
		wantSQL     string
		wantArgs    []any
		wantMatched bool
	}{
		{name: "IN slice", leftPart: "id IN ", placeholder: "?", param: values, wantSQL: "(?,?)", wantArgs: []any{int64(1), int64(2)}, wantMatched: true},
		{name: "NOT IN array", leftPart: "status NOT IN ", placeholder: "?", param: array, wantSQL: "(?,?)", wantArgs: []any{"enabled", "disabled"}, wantMatched: true},
		{name: "case and whitespace", leftPart: " id in \t", placeholder: "?", param: values, wantSQL: "(?,?)", wantArgs: []any{int64(1), int64(2)}, wantMatched: true},
		{name: "custom placeholder", leftPart: "id IN ", placeholder: ":v", param: values, wantSQL: "(:v,:v)", wantArgs: []any{int64(1), int64(2)}, wantMatched: true},
		{name: "empty slice", leftPart: "id IN ", placeholder: "?", param: []int64{}, wantSQL: "()", wantArgs: []any{}, wantMatched: true},
		{name: "empty array", leftPart: "id NOT IN ", placeholder: "?", param: [0]int64{}, wantSQL: "()", wantArgs: []any{}, wantMatched: true},
		{name: "typed nil slice", leftPart: "id IN ", placeholder: "?", param: nilSlice, wantSQL: "()", wantArgs: []any{}, wantMatched: true},
		{name: "slice pointer", leftPart: "id IN ", placeholder: "?", param: &values, wantSQL: "(?,?)", wantArgs: []any{int64(1), int64(2)}, wantMatched: true},
		{name: "array pointer", leftPart: "status NOT IN ", placeholder: "?", param: &array, wantSQL: "(?,?)", wantArgs: []any{"enabled", "disabled"}, wantMatched: true},
		{name: "nested slice pointer", leftPart: "id IN ", placeholder: "?", param: &valuesPointer, wantSQL: "(?,?)", wantArgs: []any{int64(1), int64(2)}, wantMatched: true},
		{name: "nil slice pointer", leftPart: "id IN ", placeholder: "?", param: &nilSlice, wantSQL: "()", wantArgs: []any{}, wantMatched: true},
		{name: "scalar", leftPart: "id IN ", placeholder: "?", param: int64(1)},
		{name: "non IN slice", leftPart: "payload = ", placeholder: "?", param: values},
		{name: "wrapped IN placeholder", leftPart: "id IN (", placeholder: "?", param: values},
		{name: "bytes", leftPart: "payload IN ", placeholder: "?", param: bytes, wantSQL: "(?,?)", wantArgs: []any{byte(1), byte(2)}, wantMatched: true},
		{name: "bytes pointer", leftPart: "payload IN ", placeholder: "?", param: &bytes, wantSQL: "(?,?)", wantArgs: []any{byte(1), byte(2)}, wantMatched: true},
		{name: "typed nil bytes", leftPart: "payload IN ", placeholder: "?", param: []byte(nil), wantSQL: "()", wantArgs: []any{}, wantMatched: true},
		{name: "untyped nil", leftPart: "id IN ", placeholder: "?", param: nil},
		{name: "typed nil pointer", leftPart: "id IN ", placeholder: "?", param: nilPointer},
		{name: "short left part", leftPart: "i", placeholder: "?", param: values},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSQL, gotArgs, gotMatched := expandInParameter(test.leftPart, test.placeholder, test.param)
			if gotSQL != test.wantSQL || !reflect.DeepEqual(gotArgs, test.wantArgs) || gotMatched != test.wantMatched {
				t.Fatalf("expandInParameter() = (%q, %#v, %t), want (%q, %#v, %t)", gotSQL, gotArgs, gotMatched, test.wantSQL, test.wantArgs, test.wantMatched)
			}
		})
	}
}

func TestDefaultPlaceholderParserParsesParameters(t *testing.T) {
	bytes := []byte{1, 2}
	tests := []struct {
		name     string
		query    string
		args     []any
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "multiple collections", query: "id IN ? AND status NOT IN ? AND name = ?",
			args:    []any{[]int{1, 2}, [2]string{"disabled", "pending"}, "admin"},
			wantSQL: "id IN (?,?) AND status NOT IN (?,?) AND name = ?", wantArgs: []any{1, 2, "disabled", "pending", "admin"},
		},
		{
			name: "empty collection followed by scalar", query: "id IN ? AND name = ?",
			args: []any{[]int{}, "admin"}, wantSQL: "id IN () AND name = ?", wantArgs: []any{"admin"},
		},
		{
			name: "manual parentheses", query: "id IN (?,?)",
			args: []any{1, 2}, wantSQL: "id IN (?,?)", wantArgs: []any{1, 2},
		},
		{
			name: "unmatched collection and bytes in IN", query: "payload = ? AND data IN ?",
			args: []any{[]string{"a", "b"}, bytes}, wantSQL: "payload = ? AND data IN (?,?)", wantArgs: []any{[]string{"a", "b"}, byte(1), byte(2)},
		},
		{
			name: "native numbered placeholder", query: "id = $1",
			args: []any{1}, wantSQL: "id = $1", wantArgs: []any{1},
		},
		{
			name: "extra argument", query: "id IN ?",
			args: []any{[]int{1, 2}, "extra"}, wantSQL: "id IN (?,?)", wantArgs: []any{1, 2, "extra"},
		},
		{
			name: "extra placeholder", query: "id IN ? AND name = ?",
			args: []any{[]int{1, 2}}, wantSQL: "id IN (?,?) AND name = ?", wantArgs: []any{1, 2},
		},
		{name: "no arguments", query: "SELECT ?", wantSQL: "SELECT ?"},
		{name: "empty query", args: []any{1}, wantArgs: []any{1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalArgs := append([]any(nil), test.args...)
			query, args := (defaultPlaceholderParser{}).Parse(test.query, test.args...)
			if query != test.wantSQL || !reflect.DeepEqual(args, test.wantArgs) {
				t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", query, args, test.wantSQL, test.wantArgs)
			}
			if !reflect.DeepEqual(test.args, originalArgs) {
				t.Fatalf("input arguments were modified: %#v", test.args)
			}
		})
	}
}
