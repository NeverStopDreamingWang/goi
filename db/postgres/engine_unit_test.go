package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
)

type connectionConfigProvider struct {
	config Config
	err    error
	calls  int
	target any
}

func (provider *connectionConfigProvider) GetConfig(target any) error {
	provider.calls++
	provider.target = target
	if provider.err != nil {
		return provider.err
	}
	return provider.config.GetConfig(target)
}

func configuredTestEntry(t *testing.T, entry string, manager *goi.Database) *Engine {
	t.Helper()
	name := t.Name()
	if entry == "factory" {
		return db.ConnectDatabase[*Engine](name, manager)
	}
	old, existed := goi.Settings.Databases[name]
	goi.Settings.Databases[name] = manager
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[name] = old
		} else {
			delete(goi.Settings.Databases, name)
		}
	})
	return Connect(name)
}

func TestConnectionConfigFormsUseBothEntrypoints(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	defaults := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	quoter := db.NewIdentifierQuoter("<", ">")
	parser := db.NewPlaceholderParser(":v")
	column := db.NewColumnParser("column", "sql_type")
	custom := Config{EngineOptions: db.EngineOptions{
		PlaceholderParser: &parser,
		IdentifierQuoter:  &quoter,
		ColumnParser:      &column,
	}}
	var typedNil *Config
	tests := []struct {
		name   string
		config goi.DatabaseConfig
		want   db.EngineOptions
	}{
		{"nil", nil, defaults},
		{"typed_nil", typedNil, defaults},
		{"empty_value", Config{}, defaults},
		{"empty_pointer", &Config{}, defaults},
		{"value", custom, custom.EngineOptions},
		{"pointer", &custom, custom.EngineOptions},
	}
	for _, entry := range []string{"package", "factory"} {
		for _, test := range tests {
			t.Run(entry+"/"+test.name, func(t *testing.T) {
				poolCalls := 0
				manager := &goi.Database{
					Engine: driverName,
					Config: test.config,
					Connect: func(string) *sql.DB {
						poolCalls++
						return base.DB
					},
				}
				engine := configuredTestEntry(t, entry, manager)
				if engine.DB != base.DB || poolCalls != 1 ||
					!reflect.DeepEqual(engine.placeholderParser, test.want.PlaceholderParser) ||
					!reflect.DeepEqual(engine.identifierQuoter, test.want.IdentifierQuoter) ||
					!reflect.DeepEqual(engine.columnParser, test.want.ColumnParser) {
					t.Fatalf("configuration not applied: pool calls=%d", poolCalls)
				}
			})
		}
	}
}

func TestConnectionConfigProviderRunsBeforePool(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	sentinel := errors.New("config unavailable")
	providerError := fmt.Errorf("provider: %w", sentinel)
	for _, entry := range []string{"package", "factory"} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fail_%v", entry, fail), func(t *testing.T) {
				parser := db.NewPlaceholderParser(":config")
				provider := &connectionConfigProvider{config: Config{EngineOptions: db.EngineOptions{PlaceholderParser: &parser}}}
				if fail {
					provider.err = providerError
				}
				poolCalls := 0
				manager := &goi.Database{
					Engine: driverName,
					Config: provider,
					Connect: func(string) *sql.DB {
						poolCalls++
						if provider.calls != 1 {
							t.Fatalf("pool created before reading config exactly once: %d", provider.calls)
						}
						return base.DB
					},
				}
				if fail {
					var recovered any
					func() {
						defer func() { recovered = recover() }()
						configuredTestEntry(t, entry, manager)
					}()
					got, ok := recovered.(error)
					if !ok || got != providerError || !errors.Is(got, sentinel) || poolCalls != 0 {
						t.Fatalf("configuration error=%v, pool calls=%d", recovered, poolCalls)
					}
				} else {
					engine := configuredTestEntry(t, entry, manager)
					if engine.placeholderParser != &parser || poolCalls != 1 {
						t.Fatal("custom provider configuration was ignored")
					}
				}
				if provider.calls != 1 {
					t.Fatalf("provider calls=%d, want 1", provider.calls)
				}
				if target, ok := provider.target.(*Config); !ok || target == nil {
					t.Fatalf("provider target=%T, want non-nil *Config", provider.target)
				}
			})
		}
	}
}

func TestConnectionConfigPriorityAndSnapshots(t *testing.T) {
	for _, entry := range []string{"package", "factory"} {
		t.Run(entry, func(t *testing.T) {
			oldOverride, oldShared := columnParser, db.GetColumnParser()
			oldQuoter, oldParser := GetIdentifierQuoter(), GetPlaceholderParser()
			t.Cleanup(func() {
				columnParser = oldOverride
				db.SetColumnParser(oldShared)
				SetIdentifierQuoter(oldQuoter)
				SetPlaceholderParser(oldParser)
			})
			base, _ := newEngineTestEngine(t)
			shared := &precedenceColumnParser{columnName: "shared_name"}
			packageColumn := &precedenceColumnParser{columnName: "package_name"}
			connectionColumn := &precedenceColumnParser{columnName: "connection_name"}
			packageQuoter := db.NewIdentifierQuoter("{", "}")
			packageParser := db.NewPlaceholderParser(":package")
			connectionQuoter := db.NewIdentifierQuoter("<", ">")
			connectionParser := db.NewPlaceholderParser(":connection")
			db.SetColumnParser(shared)
			SetColumnParser(packageColumn)
			SetIdentifierQuoter(&packageQuoter)
			SetPlaceholderParser(&packageParser)
			config := &Config{EngineOptions: db.EngineOptions{
				PlaceholderParser: &connectionParser,
				IdentifierQuoter:  &connectionQuoter,
				ColumnParser:      connectionColumn,
			}}
			manager := &goi.Database{
				Engine:  driverName,
				Config:  config,
				Connect: func(string) *sql.DB { return base.DB },
			}
			connectionEngine := configuredTestEntry(t, entry, manager)
			if connectionEngine.placeholderParser != &connectionParser ||
				connectionEngine.identifierQuoter != &connectionQuoter ||
				connectionEngine.columnParser != connectionColumn {
				t.Fatal("connection strategies did not override package strategies")
			}
			if GetIdentifierQuoter() != &packageQuoter || GetPlaceholderParser() != &packageParser ||
				columnParser != packageColumn || db.GetColumnParser() != shared {
				t.Fatal("connection config polluted global strategies")
			}

			config.EngineOptions = db.EngineOptions{}
			packageEngine := configuredTestEntry(t, entry, manager)
			if packageEngine.placeholderParser != &packageParser || packageEngine.identifierQuoter != &packageQuoter ||
				packageEngine.columnParser != packageColumn {
				t.Fatal("nil connection strategies did not inherit package strategies")
			}
			columnParser = nil
			sharedEngine := configuredTestEntry(t, entry, manager)
			if sharedEngine.columnParser != shared {
				t.Fatal("unset package parser did not inherit db default")
			}
			db.SetColumnParser(&precedenceColumnParser{columnName: "next_shared"})
			SetColumnParser(&precedenceColumnParser{columnName: "next_package"})
			SetIdentifierQuoter(db.NewIdentifierQuoter("[", "]"))
			SetPlaceholderParser(db.NewPlaceholderParser(":next"))
			for _, snapshot := range []struct {
				engine *Engine
				column string
			}{
				{connectionEngine, "connection_name"},
				{packageEngine, "package_name"},
				{sharedEngine, "shared_name"},
			} {
				snapshot.engine.SetModel(engineTestModel{})
				if !reflect.DeepEqual(snapshot.engine.fieldSQL, []string{snapshot.column}) {
					t.Fatalf("snapshot columns=%v, want %q", snapshot.engine.fieldSQL, snapshot.column)
				}
			}
			if connectionEngine.identifierQuoter != &connectionQuoter || connectionEngine.placeholderParser != &connectionParser ||
				packageEngine.identifierQuoter != &packageQuoter || packageEngine.placeholderParser != &packageParser {
				t.Fatal("created engines did not retain quoter/parser snapshots")
			}
		})
	}
}

func TestLowLevelConnectDatabaseDoesNotReadConfig(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	provider := &connectionConfigProvider{err: errors.New("must not read config")}
	poolCalls := 0
	manager := &goi.Database{
		Engine: "other-dialect",
		Config: provider,
		Connect: func(string) *sql.DB {
			poolCalls++
			return base.DB
		},
	}
	engine := ConnectDatabase("low-level", manager)
	if engine.DB != base.DB || provider.calls != 0 || poolCalls != 1 {
		t.Fatalf("low-level config calls=%d, pool calls=%d", provider.calls, poolCalls)
	}
}

func TestConfigGetConfigValidatesTargetAndShallowCopies(t *testing.T) {
	quoter := db.NewIdentifierQuoter("<", ">")
	parser := db.NewPlaceholderParser(":config")
	column := db.NewColumnParser("column", "sql_type")
	config := Config{EngineOptions: db.EngineOptions{
		PlaceholderParser: &parser,
		IdentifierQuoter:  &quoter,
		ColumnParser:      &column,
	}}
	for _, target := range []any{nil, (*Config)(nil), Config{}, new(int)} {
		if err := config.GetConfig(target); err == nil {
			t.Fatalf("GetConfig(%T) accepted an invalid target", target)
		}
	}
	var target Config
	if err := config.GetConfig(&target); err != nil {
		t.Fatal(err)
	}
	if target.EngineOptions.PlaceholderParser != &parser || target.EngineOptions.IdentifierQuoter != &quoter ||
		target.EngineOptions.ColumnParser != &column {
		t.Fatal("GetConfig did not preserve strategy object references")
	}
	target.EngineOptions = db.EngineOptions{}
	if config.EngineOptions.PlaceholderParser != &parser || config.EngineOptions.IdentifierQuoter != &quoter ||
		config.EngineOptions.ColumnParser != &column {
		t.Fatal("changing target config changed source config")
	}
}

type precedenceColumnParser struct{ columnName string }

func (parser precedenceColumnParser) Parse(field reflect.StructField) (string, string, bool) {
	if field.Name != "Name" {
		return "", "", false
	}
	return parser.columnName, "VARCHAR(32) NOT NULL", true
}

func TestColumnParserPrecedenceAndSnapshots(t *testing.T) {
	originalOverride, originalDefault := columnParser, db.GetColumnParser()
	t.Cleanup(func() {
		columnParser = originalOverride
		db.SetColumnParser(originalDefault)
	})
	base, _ := newEngineTestEngine(t)
	manager := &goi.Database{
		Engine:  driverName,
		Connect: func(string) *sql.DB { return base.DB },
	}
	shared := precedenceColumnParser{columnName: "shared_name"}
	packageOverride := precedenceColumnParser{columnName: "package_name"}
	instanceOverride := precedenceColumnParser{columnName: "instance_name"}
	nextShared := precedenceColumnParser{columnName: "next_shared_name"}
	assertColumn := func(engine *Engine, want string) {
		t.Helper()
		engine.SetModel(engineTestModel{})
		if !reflect.DeepEqual(engine.fieldSQL, []string{want}) {
			t.Fatalf("columns = %v, want [%s]", engine.fieldSQL, want)
		}
	}

	db.SetColumnParser(shared)
	columnParser = nil
	if !reflect.DeepEqual(GetColumnParser(), shared) {
		t.Fatal("package did not inherit the shared db parser")
	}
	inherited := ConnectDatabase("shared", manager)
	assertColumn(inherited, "shared_name")
	if err := Validate(engineTestModel{}, false); err == nil || !strings.Contains(err.Error(), "shared_name") {
		t.Fatalf("Validate did not use shared parser: %v", err)
	}

	SetColumnParser(packageOverride)
	packageEngine := ConnectDatabase("package", manager)
	assertColumn(packageEngine, "package_name")
	instanceEngine := ConnectDatabase("instance", manager, db.EngineOptions{ColumnParser: instanceOverride})
	assertColumn(instanceEngine, "instance_name")
	if !reflect.DeepEqual(GetColumnParser(), packageOverride) {
		t.Fatal("instance options changed the package override")
	}

	SetColumnParser(nil)
	if !reflect.DeepEqual(columnParser, packageOverride) || !reflect.DeepEqual(GetColumnParser(), packageOverride) {
		t.Fatal("SetColumnParser(nil) changed the package override")
	}
	columnParser = nil
	if !reflect.DeepEqual(GetColumnParser(), shared) {
		t.Fatal("unset package override did not inherit the shared db parser")
	}
	fallback := ConnectDatabase("fallback", manager)
	assertColumn(fallback, "shared_name")
	db.SetColumnParser(nextShared)
	assertColumn(inherited, "shared_name")
	assertColumn(fallback, "shared_name")
	assertColumn(packageEngine, "package_name")
	assertColumn(instanceEngine, "instance_name")
	assertColumn(ConnectDatabase("new_shared", manager), "next_shared_name")
	if err := Validate(engineTestModel{}, false); err == nil || !strings.Contains(err.Error(), "next_shared_name") {
		t.Fatalf("Validate did not obtain the updated shared parser: %v", err)
	}
}

type engineTestModel struct {
	ID     *int64  `field_name:"id" field_type:"BIGSERIAL PRIMARY KEY"`
	Name   *string `field_name:"name" field_type:"VARCHAR(64) NOT NULL"`
	Status *string `field_name:"status" field_type:"VARCHAR(32) DEFAULT 'enabled'"`
}

type namedPlaceholderParser struct{}

func (namedPlaceholderParser) Placeholder() string { return ":v" }

func (namedPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	for index := range args {
		query = strings.Replace(query, ":v", fmt.Sprintf("$%d", index+1), 1)
	}
	return query, args
}

func TestCustomPlaceholderBuildsAndParsesEveryOperation(t *testing.T) {
	old := GetPlaceholderParser()
	t.Cleanup(func() { SetPlaceholderParser(old) })
	SetPlaceholderParser(namedPlaceholderParser{})
	engine, recorder := newEngineTestEngine(t)
	name := "admin"
	if _, err := engine.Fields("Name").Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" ("+quoteTestName("name")+") VALUES ($1)" {
		t.Fatalf("custom INSERT = %q", recorder.execQuery)
	}
	query := engine.Where("id = :v AND status IN :v", int64(1), []string{"enabled", "disabled"})
	if !reflect.DeepEqual(query.args, []any{int64(1), "enabled", "disabled"}) {
		t.Fatalf("custom WHERE args = %v", query.args)
	}
	var results []engineTestModel
	if err := query.Select(context.Background(), &results); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(recorder.queryQuery, " WHERE id = $1 AND status IN ($2,$3)") {
		t.Fatalf("custom SELECT = %q", recorder.queryQuery)
	}
	if _, err := query.Fields("Name").Update(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(recorder.execQuery, " SET "+quoteTestName("name")+" = $1 WHERE id = $2 AND status IN ($3,$4)") {
		t.Fatalf("custom UPDATE = %q", recorder.execQuery)
	}
	if _, err := query.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(recorder.execQuery, " WHERE id = $1 AND status IN ($2,$3)") {
		t.Fatalf("custom DELETE = %q", recorder.execQuery)
	}
}

type migrationLifecycleModel struct {
	Name   *string `field_name:"name" field_type:"VARCHAR(64)"`
	before func() error
	after  func() error
}

func (model migrationLifecycleModel) ModelSet() *Settings {
	return &Settings{
		Schema:            "test_schema",
		TableName:         "roles",
		MigrationsHandler: MigrationsHandler{BeforeHandler: model.before, AfterHandler: model.after},
	}
}

func TestMigrationHooksKeepTheirLifecycle(t *testing.T) {
	engine, recorder := prepareMigrationTest(t)
	var events []string
	model := migrationLifecycleModel{
		before: func() error {
			if recorder.execCount != 0 {
				t.Fatal("before callback ran after CREATE TABLE")
			}
			events = append(events, "before")
			return nil
		},
		after: func() error {
			if recorder.execCount != 1 {
				t.Fatal("after callback ran before CREATE TABLE")
			}
			events = append(events, "after")
			return nil
		},
	}
	engine.Migrate(context.Background(), model)
	if !reflect.DeepEqual(events, []string{"before", "after"}) {
		t.Fatalf("migration callbacks = %v", events)
	}
	recorder.rows = [][]driver.Value{{int64(1)}}
	events = nil
	engine.Migrate(context.Background(), model)
	if len(events) != 0 || recorder.execCount != 1 {
		t.Fatal("existing table executed callbacks or CREATE TABLE")
	}
}

func (engineTestModel) ModelSet() *Settings {
	return &Settings{
		Schema:    "test_schema",
		TableName: "roles",
	}
}

type engineUnexportedModel struct {
	name *string `field_name:"name" field_type:"VARCHAR(64) NOT NULL"`
}

func (engineUnexportedModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

type engineTestPlaceholderParser struct{}

func (engineTestPlaceholderParser) Placeholder() string {
	return "?"
}

func (engineTestPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	return "parsed: " + query, args
}

type engineTestRecorder struct {
	mu sync.Mutex

	execCount int
	execQuery string
	execArgs  []driver.NamedValue
	execCtx   context.Context

	queryCount int
	queryQuery string
	queryArgs  []driver.NamedValue

	beginCount    int
	commitCount   int
	rollbackCount int

	columns []string
	rows    [][]driver.Value
}

func (recorder *engineTestRecorder) recordExec(ctx context.Context, query string, args []driver.NamedValue) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.execCount++
	recorder.execQuery = query
	recorder.execArgs = append([]driver.NamedValue(nil), args...)
	recorder.execCtx = ctx
}

func (recorder *engineTestRecorder) recordQuery(query string, args []driver.NamedValue) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.queryCount++
	recorder.queryQuery = query
	recorder.queryArgs = append([]driver.NamedValue(nil), args...)
}

type engineTestConnector struct {
	recorder *engineTestRecorder
}

func (connector *engineTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &engineTestConn{recorder: connector.recorder}, nil
}

func (*engineTestConnector) Driver() driver.Driver {
	return engineTestDriver{}
}

type engineTestDriver struct{}

func (engineTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("engine test driver requires OpenDB")
}

type engineTestConn struct {
	recorder *engineTestRecorder
}

func (*engineTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (*engineTestConn) Close() error { return nil }

func (conn *engineTestConn) Begin() (driver.Tx, error) {
	return conn.BeginTx(context.Background(), driver.TxOptions{})
}

func (conn *engineTestConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn.recorder.mu.Lock()
	conn.recorder.beginCount++
	conn.recorder.mu.Unlock()
	return &engineTestTx{recorder: conn.recorder}, nil
}

func (conn *engineTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn.recorder.recordExec(ctx, query, args)
	return driver.RowsAffected(1), nil
}

func (conn *engineTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn.recorder.recordQuery(query, args)

	conn.recorder.mu.Lock()
	defer conn.recorder.mu.Unlock()
	rows := make([][]driver.Value, len(conn.recorder.rows))
	for i := range conn.recorder.rows {
		rows[i] = append([]driver.Value(nil), conn.recorder.rows[i]...)
	}
	return &engineTestRows{
		columns: append([]string(nil), conn.recorder.columns...),
		rows:    rows,
	}, nil
}

type engineTestTx struct {
	recorder *engineTestRecorder
}

func (tx *engineTestTx) Commit() error {
	tx.recorder.mu.Lock()
	defer tx.recorder.mu.Unlock()
	tx.recorder.commitCount++
	return nil
}

func (tx *engineTestTx) Rollback() error {
	tx.recorder.mu.Lock()
	defer tx.recorder.mu.Unlock()
	tx.recorder.rollbackCount++
	return nil
}

type engineTestRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (rows *engineTestRows) Columns() []string { return rows.columns }

func (*engineTestRows) Close() error { return nil }

func (rows *engineTestRows) Next(dest []driver.Value) error {
	if rows.index >= len(rows.rows) {
		return io.EOF
	}
	copy(dest, rows.rows[rows.index])
	rows.index++
	return nil
}

func newEngineTestEngine(t *testing.T) (*Engine, *engineTestRecorder) {
	t.Helper()
	recorder := &engineTestRecorder{
		columns: []string{"id", "name", "status"},
		rows: [][]driver.Value{
			{int64(1), "admin", "enabled"},
		},
	}
	database := sql.OpenDB(&engineTestConnector{recorder: recorder})
	t.Cleanup(func() {
		_ = database.Close()
	})
	engine := &Engine{
		name:              "test",
		DB:                database,
		identifierQuoter:  GetIdentifierQuoter(),
		placeholderParser: GetPlaceholderParser(),
		columnParser:      GetColumnParser(),
	}
	engine.SetModel(engineTestModel{})
	return engine, recorder
}

const placeholderPrefix = "$"

func wantDriverSQL(query string) string {
	if placeholderPrefix == "" {
		return query
	}
	for index := 1; strings.Contains(query, "?"); index++ {
		query = strings.Replace(query, "?", fmt.Sprintf("%s%d", placeholderPrefix, index), 1)
	}
	return query
}

func quoteTestName(name string) string {
	return "\"" + name + "\""
}

func TestDefaultDialectStrategies(t *testing.T) {
	if got := GetIdentifierQuoter().QuoteIdentifier("Role"); got != quoteTestName("Role") {
		t.Fatalf("identifier = %q", got)
	}
	parser := GetPlaceholderParser()
	if parser.Placeholder() != "?" {
		t.Fatalf("ORM placeholder = %q", parser.Placeholder())
	}
	const query = "SELECT ? FROM roles WHERE id = ?"
	if got, _ := parser.Parse(query, "admin", int64(1)); got != wantDriverSQL(query) {
		t.Fatalf("driver SQL = %q, want %q", got, wantDriverSQL(query))
	}
	native := wantDriverSQL("SELECT ?")
	if got, _ := parser.Parse(native, int64(1)); got != native {
		t.Fatalf("native SQL changed: %q", got)
	}
	expandedSQL, expandedArgs := parser.Parse("SELECT ? FROM roles WHERE id IN ? AND status = ?", "admin", []int64{7, 8}, "enabled")
	wantSQL := wantDriverSQL("SELECT ? FROM roles WHERE id IN (?,?) AND status = ?")
	if expandedSQL != wantSQL || !reflect.DeepEqual(expandedArgs, []any{"admin", int64(7), int64(8), "enabled"}) {
		t.Fatalf("expanded SQL=%q args=%#v", expandedSQL, expandedArgs)
	}
}

func TestDefaultPlaceholderParserWritesDriverPlaceholders(t *testing.T) {
	parser := defaultPlaceholderParser{}
	placeholder := parser.Placeholder()
	query := "tenant_id = " + placeholder + " AND id IN " + placeholder + " AND status NOT IN " + placeholder + " AND name = " + placeholder
	args := []any{7, []int64{1, 2}, [2]string{"disabled", "deleted"}, "admin"}
	query, args = parser.Parse(query, args...)
	wantSQL := "tenant_id = $1 AND id IN ($2,$3) AND status NOT IN ($4,$5) AND name = $6"
	wantArgs := []any{7, int64(1), int64(2), "disabled", "deleted", "admin"}
	if query != wantSQL || !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", query, args, wantSQL, wantArgs)
	}
}

func TestDefaultPlaceholderParserExpansionBoundaries(t *testing.T) {
	bytes := []byte{1, 2}
	tests := []struct {
		name     string
		query    string
		args     []any
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "empty collection", query: "id IN ? AND name = ?", args: []any{[]int64{}, "admin"},
			wantSQL: "id IN () AND name = $1", wantArgs: []any{"admin"},
		},
		{
			name: "multiple collections", query: "id IN ? AND status NOT IN ? AND active = ?",
			args:    []any{[]int64{7, 8}, [2]string{"disabled", "deleted"}, true},
			wantSQL: "id IN ($1,$2) AND status NOT IN ($3,$4) AND active = $5", wantArgs: []any{int64(7), int64(8), "disabled", "deleted", true},
		},
		{
			name: "placeholder text in values", query: "id = ? AND name IN ? AND active = ?",
			args:    []any{7, []string{"?", "$1"}, true},
			wantSQL: "id = $1 AND name IN ($2,$3) AND active = $4", wantArgs: []any{7, "?", "$1", true},
		},
		{
			name: "empty collection between scalars", query: "name = ? AND id IN ? AND age = ?", args: []any{"admin", []int64{}, 18},
			wantSQL: "name = $1 AND id IN () AND age = $2", wantArgs: []any{"admin", 18},
		},
		{
			name: "all empty collections", query: "id IN ? OR status NOT IN ?", args: []any{[]int64{}, [0]string{}},
			wantSQL: "id IN () OR status NOT IN ()", wantArgs: []any{},
		},
		{
			name: "manual parentheses", query: "id IN (?)", args: []any{[]int64{7, 8}},
			wantSQL: "id IN ($1)", wantArgs: []any{[]int64{7, 8}},
		},
		{
			name: "bytes", query: "data IN ?", args: []any{bytes},
			wantSQL: "data IN $1", wantArgs: []any{bytes},
		},
		{
			name: "extra argument", query: "id IN ?", args: []any{[]int64{7, 8}, "extra"},
			wantSQL: "id IN ($1,$2)", wantArgs: []any{int64(7), int64(8), "extra"},
		},
		{
			name: "extra placeholder", query: "id IN ? AND name = ?", args: []any{[]int64{7, 8}},
			wantSQL: "id IN ($1,$2) AND name = ?", wantArgs: []any{int64(7), int64(8)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, args := (defaultPlaceholderParser{}).Parse(test.query, test.args...)
			if query != test.wantSQL || !reflect.DeepEqual(args, test.wantArgs) {
				t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", query, args, test.wantSQL, test.wantArgs)
			}
		})
	}
}

func TestExecutionEntrypointsUseParserAndContext(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	const update = "UPDATE roles SET name = ? WHERE id = ?"
	if _, err := engine.ExecuteContext(ctx, update, "admin", int64(1)); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != wantDriverSQL(update) || engine.GetSQL() != recorder.execQuery || recorder.execCtx != ctx {
		t.Fatalf("ExecuteContext SQL=%q, context=%v", recorder.execQuery, recorder.execCtx)
	}
	recorder.columns = []string{"value"}
	recorder.rows = [][]driver.Value{{int64(1)}}
	var value int64
	if err := engine.QueryRow("SELECT ?", int64(1)).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if recorder.queryQuery != wantDriverSQL("SELECT ?") || engine.GetSQL() != recorder.queryQuery {
		t.Fatalf("QueryRow SQL=%q", recorder.queryQuery)
	}
	rows, err := engine.Query("SELECT ? FROM roles", int64(1))
	if err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	if recorder.queryQuery != wantDriverSQL("SELECT ? FROM roles") || engine.GetSQL() != recorder.queryQuery {
		t.Fatalf("Query SQL=%q", recorder.queryQuery)
	}
	if _, err := engine.Execute(update, "admin", int64(1)); err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := engine.ExecuteContext(canceled, update, "admin", int64(1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteContext cancellation: %v", err)
	}
	if err := engine.QueryRowContext(canceled, "SELECT ?", int64(1)).Scan(&value); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueryRowContext cancellation: %v", err)
	}
	if _, err := engine.QueryContext(canceled, "SELECT ?", int64(1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueryContext cancellation: %v", err)
	}
}

type alternateTagModel struct {
	ID   *int64  `field_name:"id" field_type:"BIGINT" column:"role_id" sql_type:"BIGINT"`
	Name *string `field_name:"name" field_type:"VARCHAR(64)" column:"display_name" sql_type:"VARCHAR(64)"`
	Skip *string `field_type:"-" sql_type:"-"`
}

func (alternateTagModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestStrategySettersIgnoreNil(t *testing.T) {
	oldQuoter, oldParser := GetIdentifierQuoter(), GetPlaceholderParser()
	t.Cleanup(func() {
		SetIdentifierQuoter(oldQuoter)
		SetPlaceholderParser(oldParser)
	})
	quoter := db.NewIdentifierQuoter("<", ">")
	parser := db.NewPlaceholderParser(":v")
	SetIdentifierQuoter(&quoter)
	SetPlaceholderParser(&parser)
	SetIdentifierQuoter(nil)
	SetPlaceholderParser(nil)
	if GetIdentifierQuoter() != &quoter {
		t.Fatal("nil replaced the configured identifier quoter")
	}
	if GetPlaceholderParser() != &parser {
		t.Fatal("nil replaced the configured placeholder parser")
	}
}

func TestFactorySnapshotsStrategiesAndSharesPool(t *testing.T) {
	oldIdentifier, oldPlaceholder, oldColumn := GetIdentifierQuoter(), GetPlaceholderParser(), GetColumnParser()
	t.Cleanup(func() {
		SetIdentifierQuoter(oldIdentifier)
		SetPlaceholderParser(oldPlaceholder)
		SetColumnParser(oldColumn)
	})
	base, _ := newEngineTestEngine(t)
	calls := 0
	manager := &goi.Database{
		Engine: driverName,
		Connect: func(string) *sql.DB {
			calls++
			return base.DB
		},
	}
	first := db.ConnectDatabase[*Engine]("first", manager)
	SetIdentifierQuoter(db.NewIdentifierQuoter("<", ">"))
	SetPlaceholderParser(engineTestPlaceholderParser{})
	SetColumnParser(db.NewColumnParser("column", "sql_type"))
	second := db.ConnectDatabase[*Engine]("second", manager)
	first.SetModel(alternateTagModel{})
	second.SetModel(alternateTagModel{})
	if first == second || first.DB != second.DB || calls != 1 || first.Name() != "first" || second.Name() != "second" {
		t.Fatal("factory did not create independent engines sharing one pool")
	}
	if first.tableName != quoteTestName("test_schema")+"."+quoteTestName("roles") || second.tableName != "<test_schema>.<roles>" {
		t.Fatalf("quoter snapshots: %q, %q", first.tableName, second.tableName)
	}
	if !reflect.DeepEqual(first.fieldSQL, []string{"id", "name"}) || !reflect.DeepEqual(second.fieldSQL, []string{"role_id", "display_name"}) {
		t.Fatalf("column snapshots: %v, %v", first.fieldSQL, second.fieldSQL)
	}
	firstSQL, _ := first.placeholderParser.Parse("SELECT ?", 1)
	secondSQL, _ := second.placeholderParser.Parse("SELECT ?", 1)
	if firstSQL != wantDriverSQL("SELECT ?") || secondSQL != "parsed: SELECT ?" {
		t.Fatal("placeholder parser snapshot changed")
	}
}

type engineTestDBProvider struct {
	pool  *sql.DB
	calls int
}

func (provider *engineTestDBProvider) DB() *sql.DB {
	provider.calls++
	return provider.pool
}

func TestConnectDatabaseAcceptsProviderAndOrderedOptions(t *testing.T) {
	defaults := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	t.Cleanup(func() {
		SetPlaceholderParser(defaults.PlaceholderParser)
		SetIdentifierQuoter(defaults.IdentifierQuoter)
		SetColumnParser(defaults.ColumnParser)
	})
	base, _ := newEngineTestEngine(t)
	provider := &engineTestDBProvider{pool: base.DB}
	placeholder := db.EngineOptions{PlaceholderParser: namedPlaceholderParser{}}
	identifier := db.EngineOptions{IdentifierQuoter: db.NewIdentifierQuoter("[", "]")}
	column := db.EngineOptions{ColumnParser: db.NewColumnParser("column", "sql_type")}
	last := db.EngineOptions{PlaceholderParser: engineTestPlaceholderParser{}}
	engine := ConnectDatabase("bare-provider", provider, placeholder, identifier, column, last, db.EngineOptions{})
	if engine.DB != base.DB || engine.Name() != "bare-provider" || provider.calls != 1 {
		t.Fatal("ConnectDatabase did not use the bare provider")
	}
	engine.SetModel(alternateTagModel{})
	if engine.tableName != "[test_schema].[roles]" || !reflect.DeepEqual(engine.fieldSQL, []string{"role_id", "display_name"}) {
		t.Fatalf("local strategies = %q, %v", engine.tableName, engine.fieldSQL)
	}
	if got, _ := engine.placeholderParser.Parse("SELECT ?", 1); got != "parsed: SELECT ?" {
		t.Fatalf("ordered placeholder override = %q", got)
	}
	if !reflect.DeepEqual(GetPlaceholderParser(), defaults.PlaceholderParser) ||
		!reflect.DeepEqual(GetIdentifierQuoter(), defaults.IdentifierQuoter) ||
		!reflect.DeepEqual(GetColumnParser(), defaults.ColumnParser) {
		t.Fatal("instance options changed package defaults")
	}
	plain := ConnectDatabase("default-strategies", provider)
	if !reflect.DeepEqual(plain.placeholderParser, defaults.PlaceholderParser) ||
		!reflect.DeepEqual(plain.identifierQuoter, defaults.IdentifierQuoter) ||
		!reflect.DeepEqual(plain.columnParser, defaults.ColumnParser) {
		t.Fatal("omitted strategies did not inherit package defaults")
	}

	identifier.IdentifierQuoter = db.NewIdentifierQuoter("{", "}")
	column.ColumnParser = defaults.ColumnParser
	last.PlaceholderParser = defaults.PlaceholderParser
	SetIdentifierQuoter(db.NewIdentifierQuoter("<", ">"))
	SetColumnParser(defaults.ColumnParser)
	SetPlaceholderParser(namedPlaceholderParser{})
	engine.SetModel(alternateTagModel{})
	if query, _ := engine.placeholderParser.Parse("SELECT ?", 1); engine.tableName != "[test_schema].[roles]" ||
		!reflect.DeepEqual(engine.fieldSQL, []string{"role_id", "display_name"}) || query != "parsed: SELECT ?" {
		t.Fatal("existing engine strategies changed after configuration replacement")
	}
	later := ConnectDatabase("later", provider)
	later.SetModel(alternateTagModel{})
	if later.tableName != "<test_schema>.<roles>" || !reflect.DeepEqual(later.fieldSQL, []string{"id", "name"}) ||
		later.placeholderParser.Placeholder() != ":v" {
		t.Fatal("new engine did not capture current package defaults")
	}
}

func TestConnectChecksConfigurationBeforeCreatingEngine(t *testing.T) {
	name := t.Name()
	original, existed := goi.Settings.Databases[name]
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[name] = original
		} else {
			delete(goi.Settings.Databases, name)
		}
	})
	base, _ := newEngineTestEngine(t)
	calls := 0
	manager := &goi.Database{
		Engine: driverName,
		Connect: func(string) *sql.DB {
			calls++
			return base.DB
		},
	}
	goi.Settings.Databases[name] = manager
	if engine := Connect(name); engine.DB != base.DB || engine.Name() != name || calls != 1 {
		t.Fatal("Connect did not use its matching configuration")
	}
	expectPanic := func(label string, call func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s did not panic", label)
			}
		}()
		call()
	}
	delete(goi.Settings.Databases, name)
	expectPanic("missing configuration", func() { Connect(name) })
	mismatched := &goi.Database{
		Engine: "kingbase",
		Connect: func(string) *sql.DB {
			calls++
			return base.DB
		},
	}
	goi.Settings.Databases[name] = mismatched
	expectPanic("mismatched engine", func() { Connect(name) })
	if calls != 1 {
		t.Fatal("Connect opened a pool before checking the engine name")
	}
	if engine := factory(name, mismatched).(*Engine); engine.DB != base.DB || calls != 2 {
		t.Fatal("factory repeated the Connect-only engine-name check")
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
		{name: "bytes", leftPart: "payload IN ", placeholder: "?", param: bytes},
		{name: "bytes pointer", leftPart: "payload IN ", placeholder: "?", param: &bytes},
		{name: "typed nil bytes", leftPart: "payload IN ", placeholder: "?", param: []byte(nil)},
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

func TestWhereExpansionAndBranches(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	engine.whereSQL = make([]string, 1, 4)
	engine.whereSQL[0] = "status = ?"
	engine.args = make([]any, 1, 4)
	engine.args[0] = "enabled"
	left := engine.Where("id IN ?", []int64{1, 2})
	right := engine.Where("id = ?", int64(3))
	if !reflect.DeepEqual(left.whereSQL, []string{"status = ?", "id IN (?,?)"}) ||
		!reflect.DeepEqual(left.args, []any{"enabled", int64(1), int64(2)}) {
		t.Fatalf("left query changed: %v %v", left.whereSQL, left.args)
	}
	if !reflect.DeepEqual(right.args, []any{"enabled", int64(3)}) || len(engine.args) != 1 {
		t.Fatalf("branch arguments shared: %v", right.args)
	}
	if got := engine.Where("id NOT IN ?", [0]int{}).whereSQL[1]; got != "id NOT IN ()" {
		t.Fatalf("empty IN = %q", got)
	}
	bytes := []byte{1, 2}
	if got := engine.Where("payload = ?", bytes).args[1]; !reflect.DeepEqual(got, bytes) {
		t.Fatalf("byte parameter expanded: %#v", got)
	}
}

func TestSelectFirstAndPaginationUseDialect(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	query := engine.Where("id IN ?", []int64{1, 2}).GroupBy("id", "name", "status").OrderBy("-id")
	recorder.columns = []string{"count"}
	recorder.rows = [][]driver.Value{{int64(3)}}
	total, pages, err := query.Page(context.Background(), 2, 2)
	if err != nil || total != 3 || pages != 2 {
		t.Fatalf("Page = %d, %d, %v", total, pages, err)
	}
	if recorder.queryQuery != wantDriverSQL("SELECT count(*) FROM "+engine.tableName+" WHERE id IN (?,?)") {
		t.Fatalf("Count query = %q", recorder.queryQuery)
	}
	recorder.columns = []string{"id", "name", "status"}
	recorder.rows = [][]driver.Value{{int64(1), "admin", "enabled"}}
	var results []map[string]any
	if err := query.Select(context.Background(), &results); err != nil {
		t.Fatal(err)
	}
	columns := quoteTestName("id") + "," + quoteTestName("name") + "," + quoteTestName("status")
	want := wantDriverSQL("SELECT " + columns + " FROM " + engine.tableName + " WHERE id IN (?,?) GROUP BY " + columns +
		" ORDER BY " + quoteTestName("id") + " DESC LIMIT 2 OFFSET 2")
	if recorder.queryQuery != want {
		t.Fatalf("Select SQL=%q, want %q", recorder.queryQuery, want)
	}
	if len(results) != 1 || *results[0]["Name"].(*string) != "admin" {
		t.Fatalf("map result = %#v", results)
	}
	if _, exists := results[0]["name"]; exists {
		t.Fatal("map key must be the Go field name")
	}
	recorder.columns = []string{"name"}
	recorder.rows = [][]driver.Value{{"admin"}}
	var result struct{ Name *string }
	if err := engine.Fields("Name").Where("id = ?", int64(1)).First(context.Background(), &result); err != nil ||
		result.Name == nil || *result.Name != "admin" {
		t.Fatalf("First result = %#v, error = %v", result, err)
	}
	recorder.rows = nil
	if err := engine.Fields("Name").First(context.Background(), &result); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("First no rows = %v", err)
	}
	var list []engineTestModel
	if err := engine.Select(context.Background(), &list); err != nil || len(list) != 0 {
		t.Fatalf("empty Select = %v, %v", list, err)
	}
}

func TestWritesSelectedFieldsWithAndWithoutConditions(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	name := "renamed"
	if _, err := engine.Fields("Name").Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + ") VALUES (?)")
	if recorder.execQuery != want || len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != name {
		t.Fatalf("Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	// The mock driver checks forwarded SQL, not whether the database accepts this syntax.
	if _, err := engine.Fields().Insert(context.Background(), engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" () VALUES ()" {
		t.Fatalf("empty-field Insert = %q", recorder.execQuery)
	}
	if len(recorder.execArgs) != 0 || recorder.execCount != 2 {
		t.Fatalf("empty-field Insert args=%#v, total executions=%d; want no args and one execution per Insert", recorder.execArgs, recorder.execCount)
	}
	if _, err := engine.Update(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
		quoteTestName("name") + " = ?," + quoteTestName("status") + " = ?")
	if recorder.execQuery != want || !reflect.DeepEqual(writeArgumentValues(recorder), []any{nil, name, nil}) {
		t.Fatalf("global Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "DELETE FROM "+engine.tableName || len(recorder.execArgs) != 0 {
		t.Fatalf("global Delete = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	query := engine.Where("id = ?", int64(7))
	if _, err := query.Fields().Update(context.Background(), engineTestModel{}, false); err == nil {
		t.Fatal("empty Update was allowed")
	}
	if _, err := query.Fields("Name").Update(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 2 ||
		recorder.execArgs[0].Value != name || recorder.execArgs[1].Value != int64(7) {
		t.Fatalf("Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := query.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != wantDriverSQL("DELETE FROM "+engine.tableName+" WHERE id = ?") ||
		!reflect.DeepEqual(writeArgumentValues(recorder), []any{int64(7)}) {
		t.Fatalf("Delete = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Insert(context.Background(), alternateTagModel{}, false); err == nil {
		t.Fatal("different operation model was accepted")
	}
}

func TestWritesSelectedNilAndDefaultAllFields(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	if _, err := engine.Fields("Name").Insert(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + ") VALUES (?)")
	if recorder.execQuery != want || len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("selected NULL Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Fields("Name").Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 2 ||
		recorder.execArgs[0].Value != nil || recorder.execArgs[1].Value != int64(7) {
		t.Fatalf("selected NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Insert(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	columns := quoteTestName("id") + "," + quoteTestName("name") + "," + quoteTestName("status")
	want = wantDriverSQL("INSERT INTO " + engine.tableName + " (" + columns + ") VALUES (?,?,?)")
	if recorder.execQuery != want || len(recorder.execArgs) != 3 {
		t.Fatalf("all-fields NULL Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	for _, arg := range recorder.execArgs {
		if arg.Value != nil {
			t.Fatalf("nil pointer was not sent as NULL: %#v", recorder.execArgs)
		}
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
		quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 4 || recorder.execArgs[3].Value != int64(7) {
		t.Fatalf("all-fields NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	for _, arg := range recorder.execArgs[:3] {
		if arg.Value != nil {
			t.Fatalf("nil pointer was omitted from Update: %#v", recorder.execArgs)
		}
	}
}

type engineWriteScalarModel struct {
	Count   int64  `field_name:"count" field_type:"BIGINT"`
	Enabled bool   `field_name:"enabled" field_type:"BOOLEAN"`
	Label   string `field_name:"label" field_type:"TEXT"`
}

func (engineWriteScalarModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestWritesKeepSelectedZeroValues(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineWriteScalarModel{})
	if _, err := engine.Insert(context.Background(), engineWriteScalarModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 3 || recorder.execArgs[0].Value != int64(0) ||
		recorder.execArgs[1].Value != false || recorder.execArgs[2].Value != "" {
		t.Fatalf("zero-value Insert args = %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(context.Background(), engineWriteScalarModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 4 || recorder.execArgs[0].Value != int64(0) ||
		recorder.execArgs[1].Value != false || recorder.execArgs[2].Value != "" ||
		recorder.execArgs[3].Value != int64(7) {
		t.Fatalf("zero-value Update args = %#v", recorder.execArgs)
	}
}

type engineWriteValuer string

func (value *engineWriteValuer) Value() (driver.Value, error) {
	if value == nil {
		return "unexpected nil Valuer call", nil
	}
	return "encoded:" + string(*value), nil
}

type engineWriteValuerModel struct {
	Value *engineWriteValuer `field_name:"value" field_type:"TEXT"`
}

func (engineWriteValuerModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestWritesPreservePointerValuerAndNormalizeNil(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineWriteValuerModel{})
	value := engineWriteValuer("payload")
	if _, err := engine.Insert(context.Background(), engineWriteValuerModel{Value: &value}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != "encoded:payload" {
		t.Fatalf("pointer Valuer was lost: %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(context.Background(), engineWriteValuerModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != nil {
		t.Fatalf("nil pointer Valuer was not written as NULL: %#v", recorder.execArgs)
	}
}

func TestTransactionsReuseOuterTransaction(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	err := engine.WithTransaction(ctx, func(outer *Engine, _ ...any) error {
		return outer.WithTransaction(ctx, func(inner *Engine, _ ...any) error {
			if inner.tx != outer.tx || inner.tx == nil {
				t.Fatal("nested transaction did not reuse outer transaction")
			}
			_, err := inner.ExecuteContext(ctx, "UPDATE roles SET name = ?", "admin")
			return err
		})
	})
	if err != nil || recorder.beginCount != 1 || recorder.commitCount != 1 || recorder.rollbackCount != 0 || engine.tx != nil {
		t.Fatalf("transaction = %v, counts = %d/%d/%d", err, recorder.beginCount, recorder.commitCount, recorder.rollbackCount)
	}
	sentinel := errors.New("rollback")
	err = engine.WithTransaction(ctx, func(tx *Engine, _ ...any) error {
		return tx.WithTransaction(ctx, func(*Engine, ...any) error { return sentinel })
	})
	if !errors.Is(err, sentinel) || recorder.rollbackCount != 1 {
		t.Fatalf("nested error = %v, rollbacks = %d", err, recorder.rollbackCount)
	}
	func() {
		defer func() {
			if got := recover(); got != sentinel {
				t.Fatalf("panic = %v", got)
			}
		}()
		_ = engine.WithTransaction(ctx, func(*Engine, ...any) error { panic(sentinel) })
	}()
	if recorder.rollbackCount != 2 {
		t.Fatalf("panic rollbacks = %d", recorder.rollbackCount)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := engine.WithTransaction(canceled, func(*Engine, ...any) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction = %v", err)
	}
}

func prepareMigrationTest(t *testing.T) (*Engine, *engineTestRecorder) {
	t.Helper()
	old := goi.Log
	goi.Log = &goi.Logger{Logger: log.New(io.Discard, "", 0)}
	t.Cleanup(func() { goi.Log = old })
	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"exists"}
	recorder.rows = nil
	return engine, recorder
}

func TestMigrateUsesDialectMetadataAndCapturedColumnParser(t *testing.T) {
	old := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(old) })
	SetColumnParser(db.NewColumnParser("column", "sql_type"))
	engine, recorder := prepareMigrationTest(t)
	SetColumnParser(old)
	engine.Migrate(context.Background(), alternateTagModel{})
	want := "CREATE TABLE " + engine.tableName + " (\n  " + quoteTestName("role_id") + " BIGINT,\n  " +
		quoteTestName("display_name") + " VARCHAR(64)\n)"
	if recorder.execQuery != want {
		t.Fatalf("Migrate = %q, want %q", recorder.execQuery, want)
	}
	if recorder.queryQuery != wantDriverSQL("SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?;") {
		t.Fatalf("metadata SQL = %q", recorder.queryQuery)
	}
	if len(recorder.queryArgs) != 2 || recorder.queryArgs[len(recorder.queryArgs)-1].Value != "roles" {
		t.Fatalf("metadata args = %#v", recorder.queryArgs)
	}
	recorder.execCount = 0
	recorder.rows = [][]driver.Value{{int64(1)}}
	engine.Migrate(context.Background(), alternateTagModel{})
	if recorder.execCount != 0 {
		t.Fatal("existing table was recreated")
	}
}

type defaultSchemaModel struct {
	Name *string `field_name:"name" field_type:"VARCHAR(64)"`
}

func (defaultSchemaModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

func TestDefaultSchemaMatchesMigrationsAndQueries(t *testing.T) {
	engine, recorder := prepareMigrationTest(t)
	engine.SetModel(defaultSchemaModel{})
	if engine.tableName != quoteTestName("public")+"."+quoteTestName("roles") {
		t.Fatalf("default table = %q", engine.tableName)
	}
	engine.Migrate(context.Background(), defaultSchemaModel{})
	if recorder.queryQuery != wantDriverSQL("SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?;") {
		t.Fatalf("default schema metadata = %q", recorder.queryQuery)
	}
}

type validationColumnParser struct{}

func (validationColumnParser) Parse(field reflect.StructField) (string, string, bool) {
	if field.Name == "Name" {
		return "custom_name", "VARCHAR(64) NOT NULL", true
	}
	return "", "", false
}

func TestColumnParserControlsValidationAndFields(t *testing.T) {
	old := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(old) })
	SetColumnParser(validationColumnParser{})
	engine, _ := newEngineTestEngine(t)
	if !reflect.DeepEqual(engine.fields, []string{"Name"}) || !reflect.DeepEqual(engine.fieldSQL, []string{"custom_name"}) {
		t.Fatalf("fields = %v, columns = %v", engine.fields, engine.fieldSQL)
	}
	if err := Validate(engineTestModel{}, false); err == nil || !strings.Contains(err.Error(), "custom_name") {
		t.Fatalf("custom validation = %v", err)
	}
	if err := Validate(engineTestModel{}, true); err != nil {
		t.Fatalf("partial validation = %v", err)
	}
}

func writeArgumentValues(recorder *engineTestRecorder) []any {
	values := make([]any, len(recorder.execArgs))
	for i, arg := range recorder.execArgs {
		values[i] = arg.Value
	}
	return values
}

func TestSkipNilControlsSelectedWriteValues(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			ctx := context.Background()
			name := "kept"
			model := engineTestModel{Name: &name}
			if _, err := engine.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			columns := quoteTestName("id") + "," + quoteTestName("name") + "," + quoteTestName("status")
			wantSQL := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + columns + ") VALUES (?,?,?)")
			wantArgs := []any{nil, name, nil}
			if skipNil {
				wantSQL = wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + ") VALUES (?)")
				wantArgs = []any{name}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), wantArgs) {
				t.Fatalf("Insert = %q, %#v; want %q, %#v", recorder.execQuery, recorder.execArgs, wantSQL, wantArgs)
			}

			query := engine.Where("id = ?", int64(7))
			if _, err := query.Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
				quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?")
			wantArgs = []any{nil, name, nil, int64(7)}
			if skipNil {
				wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?")
				wantArgs = []any{name, int64(7)}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), wantArgs) {
				t.Fatalf("Update = %q, %#v; want %q, %#v", recorder.execQuery, recorder.execArgs, wantSQL, wantArgs)
			}

			status := "selected"
			model = engineTestModel{Status: &status}
			selected := engine.Fields("Name", "Status")
			if _, err := selected.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL = wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + "," + quoteTestName("status") + ") VALUES (?,?)")
			wantArgs = []any{nil, status}
			if skipNil {
				wantSQL = wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("status") + ") VALUES (?)")
				wantArgs = []any{status}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), wantArgs) {
				t.Fatalf("Fields + Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
			}
			if _, err := selected.Where("id = ?", int64(7)).Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?")
			wantArgs = []any{nil, status, int64(7)}
			if skipNil {
				wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("status") + " = ? WHERE id = ?")
				wantArgs = []any{status, int64(7)}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), wantArgs) {
				t.Fatalf("Fields + Update = %q, %#v", recorder.execQuery, recorder.execArgs)
			}
		})
	}
}

func TestSkipNilDoesNotAddEmptyWriteFallback(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	if _, err := engine.Insert(ctx, engineTestModel{}, true); err != nil {
		t.Fatal(err)
	}
	// The mock records the empty SQL form; the database decides its validity.
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("all-nil Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	count := recorder.execCount
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineTestModel{}, true); err == nil {
		t.Fatal("all-nil Update did not reject an empty update set")
	}
	if recorder.execCount != count {
		t.Fatal("all-nil Update reached the driver")
	}
}

type engineConvertedNullValuer struct {
	calls *int
}

func (value *engineConvertedNullValuer) Value() (driver.Value, error) {
	*value.calls = *value.calls + 1
	return nil, nil
}

type engineConvertedNullModel struct {
	Nullable sql.NullString             `field_name:"nullable" field_type:"TEXT"`
	Custom   *engineConvertedNullValuer `field_name:"custom" field_type:"TEXT"`
}

func (engineConvertedNullModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestSkipNilKeepsZeroValuesAndConvertedNull(t *testing.T) {
	ctx := context.Background()
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineWriteScalarModel{})
	if _, err := engine.Insert(ctx, engineWriteScalarModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(writeArgumentValues(recorder), []any{int64(0), false, ""}) {
		t.Fatalf("skipNil dropped scalar zero values: %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineWriteScalarModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(writeArgumentValues(recorder), []any{int64(0), false, "", int64(7)}) {
		t.Fatalf("skipNil dropped zero update values: %#v", recorder.execArgs)
	}

	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("converted NULL skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			engine.SetModel(engineConvertedNullModel{})
			calls := 0
			model := engineConvertedNullModel{Custom: &engineConvertedNullValuer{calls: &calls}}
			if _, err := engine.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("nullable") + "," + quoteTestName("custom") + ") VALUES (?,?)")
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), []any{nil, nil}) || calls != 1 {
				t.Fatalf("converted NULL Insert = %q, %#v, Valuer calls=%d", recorder.execQuery, recorder.execArgs, calls)
			}
			if _, err := engine.Where("id = ?", int64(7)).Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("nullable") + " = ?," + quoteTestName("custom") + " = ? WHERE id = ?")
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), []any{nil, nil, int64(7)}) || calls != 2 {
				t.Fatalf("converted NULL Update = %q, %#v, Valuer calls=%d", recorder.execQuery, recorder.execArgs, calls)
			}
		})
	}
}

func TestGlobalUpdateRespectsFieldsAndSkipNil(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			ctx := context.Background()
			name := "all rows"
			query := engine.Fields("Name", "Status")
			if _, err := query.Update(ctx, engineTestModel{Name: &name}, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL := wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?," + quoteTestName("status") + " = ?")
			wantArgs := []any{name, nil}
			if skipNil {
				wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?")
				wantArgs = []any{name}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), wantArgs) {
				t.Fatalf("global Update = %q, %#v; want %q, %#v", recorder.execQuery, recorder.execArgs, wantSQL, wantArgs)
			}

			count := recorder.execCount
			_, err := engine.Fields("Name").Update(ctx, engineTestModel{}, skipNil)
			if skipNil {
				if err == nil || recorder.execCount != count {
					t.Fatal("all-nil global Update did not reject an empty update set before execution")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				wantSQL = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?")
				if recorder.execQuery != wantSQL || !reflect.DeepEqual(writeArgumentValues(recorder), []any{nil}) {
					t.Fatalf("global NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
				}
			}
			count = recorder.execCount
			if _, err := engine.Fields().Update(ctx, engineTestModel{}, skipNil); err == nil || recorder.execCount != count {
				t.Fatal("explicitly empty global Update reached the driver")
			}
		})
	}
}
