package sqlserver

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
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

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

func TestStrategySettersIgnoreNil(t *testing.T) {
	originalQuoter, originalParser := GetIdentifierQuoter(), GetPlaceholderParser()
	t.Cleanup(func() {
		identifierQuoter = originalQuoter
		placeholderParser = originalParser
	})
	quoter := db.NewIdentifierQuoter("<", ">")
	parser := &engineTestPlaceholderParser{}
	SetIdentifierQuoter(&quoter)
	SetPlaceholderParser(parser)
	SetIdentifierQuoter(nil)
	SetPlaceholderParser(nil)
	if GetIdentifierQuoter() != &quoter {
		t.Fatal("SetIdentifierQuoter(nil) changed the configured object")
	}
	if GetPlaceholderParser() != parser {
		t.Fatal("SetPlaceholderParser(nil) changed the configured object")
	}
}

type connectionConfigProvider func(target any) error

func (provider connectionConfigProvider) GetConfig(target any) error { return provider(target) }

type configuredTestEntrypoint struct {
	name string
	open func(string, *goi.Database) *Engine
}

func configuredTestEntrypoints() []configuredTestEntrypoint {
	return []configuredTestEntrypoint{
		{"package", func(name string, _ *goi.Database) *Engine { return Connect(name) }},
		{"factory", func(name string, database *goi.Database) *Engine {
			return db.ConnectDatabase[*Engine](name, database)
		}},
	}
}

func newConfiguredTestManager(t *testing.T, config goi.DatabaseConfig) (*goi.Database, *engineTestRecorder, *int) {
	t.Helper()
	base, recorder := newEngineTestEngine(t)
	poolCalls := 0
	manager := &goi.Database{
		Engine: driverName,
		Config: config,
		Connect: func(string) *sql.DB {
			poolCalls++
			return base.DB
		},
	}
	name := t.Name()
	original, existed := goi.Settings.Databases[name]
	goi.Settings.Databases[name] = manager
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[name] = original
		} else {
			delete(goi.Settings.Databases, name)
		}
	})
	return manager, recorder, &poolCalls
}

func TestConnectionConfigFormsAndEntrypoints(t *testing.T) {
	localParser := &engineTestPlaceholderParser{}
	localConfig := Config{EngineOptions: db.EngineOptions{PlaceholderParser: localParser}}
	var typedNil *Config
	for _, entry := range configuredTestEntrypoints() {
		t.Run(entry.name, func(t *testing.T) {
			for _, test := range []struct {
				name   string
				config goi.DatabaseConfig
				parser db.PlaceholderParser
			}{
				{"nil", nil, GetPlaceholderParser()},
				{"value", localConfig, localParser},
				{"pointer", &localConfig, localParser},
				{"typed_nil", typedNil, GetPlaceholderParser()},
			} {
				t.Run(test.name, func(t *testing.T) {
					manager, _, calls := newConfiguredTestManager(t, test.config)
					engine := entry.open(t.Name(), manager)
					if engine.Name() != t.Name() || engine.DB != manager.DB() || *calls != 1 ||
						!reflect.DeepEqual(engine.placeholderParser, test.parser) {
						t.Fatalf("configured entrypoint did not preserve pool, name or parser: name=%q calls=%d",
							engine.Name(), *calls)
					}
				})
			}
		})
	}
}

func TestCustomConnectionConfigProviderIsReadOnce(t *testing.T) {
	for _, entry := range configuredTestEntrypoints() {
		t.Run(entry.name, func(t *testing.T) {
			reads := 0
			parser := &engineTestPlaceholderParser{}
			provider := connectionConfigProvider(func(target any) error {
				reads++
				config, ok := target.(*Config)
				if !ok || config == nil {
					return fmt.Errorf("unexpected config target %T", target)
				}
				*config = Config{EngineOptions: db.EngineOptions{PlaceholderParser: parser}}
				return nil
			})
			manager, _, poolCalls := newConfiguredTestManager(t, provider)
			engine := entry.open(t.Name(), manager)
			if reads != 1 || *poolCalls != 1 || engine.placeholderParser != parser {
				t.Fatalf("provider reads=%d, pool calls=%d, parser=%T", reads, *poolCalls, engine.placeholderParser)
			}
		})
	}
}

func TestConnectionConfigPrecedenceAndSnapshots(t *testing.T) {
	oldQuoter, oldPlaceholder, oldColumn := identifierQuoter, placeholderParser, columnParser
	oldShared := db.GetColumnParser()
	t.Cleanup(func() {
		identifierQuoter, placeholderParser, columnParser = oldQuoter, oldPlaceholder, oldColumn
		db.SetColumnParser(oldShared)
	})
	shared := precedenceColumnParser{columnName: "shared_name"}
	packageColumn := precedenceColumnParser{columnName: "package_name"}
	connectionColumn := precedenceColumnParser{columnName: "connection_name"}
	packageQuoter := db.NewIdentifierQuoter("[", "]")
	packagePlaceholder := db.NewPlaceholderParser(":package")
	connectionQuoter := db.NewIdentifierQuoter("<", ">")
	connectionPlaceholder := &engineTestPlaceholderParser{}
	db.SetColumnParser(shared)
	SetIdentifierQuoter(&packageQuoter)
	SetPlaceholderParser(&packagePlaceholder)
	SetColumnParser(packageColumn)
	config := &Config{EngineOptions: db.EngineOptions{
		PlaceholderParser: connectionPlaceholder,
		IdentifierQuoter:  &connectionQuoter,
		ColumnParser:      connectionColumn,
	}}
	manager, recorder, poolCalls := newConfiguredTestManager(t, config)
	first := Connect(t.Name())
	second := db.ConnectDatabase[*Engine](t.Name(), manager)
	for _, engine := range []*Engine{first, second} {
		engine.SetModel(engineTestModel{})
		if engine.identifierQuoter != &connectionQuoter || engine.placeholderParser != connectionPlaceholder ||
			!reflect.DeepEqual(engine.fieldSQL, []string{"connection_name"}) {
			t.Fatalf("connection strategies did not override package strategies: %v", engine.fieldSQL)
		}
	}
	if GetIdentifierQuoter() != &packageQuoter || GetPlaceholderParser() != &packagePlaceholder ||
		!reflect.DeepEqual(GetColumnParser(), packageColumn) || !reflect.DeepEqual(db.GetColumnParser(), shared) {
		t.Fatal("connection config changed package or db defaults")
	}

	config.EngineOptions = db.EngineOptions{}
	packageEngine := Connect(t.Name()).SetModel(engineTestModel{})
	if !reflect.DeepEqual(packageEngine.fieldSQL, []string{"package_name"}) {
		t.Fatalf("empty connection config did not inherit package parser: %v", packageEngine.fieldSQL)
	}
	columnParser = nil
	sharedEngine := db.ConnectDatabase[*Engine](t.Name(), manager).SetModel(engineTestModel{})
	if !reflect.DeepEqual(sharedEngine.fieldSQL, []string{"shared_name"}) {
		t.Fatalf("empty connection and package config did not inherit db parser: %v", sharedEngine.fieldSQL)
	}
	db.SetColumnParser(precedenceColumnParser{columnName: "next_shared_name"})
	nextEngine := Connect(t.Name()).SetModel(engineTestModel{})
	sharedEngine.SetModel(engineTestModel{})
	if !reflect.DeepEqual(sharedEngine.fieldSQL, []string{"shared_name"}) ||
		!reflect.DeepEqual(nextEngine.fieldSQL, []string{"next_shared_name"}) {
		t.Fatal("configured entrypoint did not snapshot the shared parser")
	}
	SetIdentifierQuoter(db.NewIdentifierQuoter("{", "}"))
	SetPlaceholderParser(db.NewPlaceholderParser(":next"))
	name := "admin"
	if _, err := first.SetModel(engineTestModel{}).Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "parsed: INSERT INTO <test_schema>.<roles> (<connection_name>) VALUES (?)" || *poolCalls != 1 {
		t.Fatalf("existing Engine changed its snapshot: SQL=%q pool calls=%d", recorder.execQuery, *poolCalls)
	}
}

func TestConnectionConfigErrorFailsBeforeOpeningPool(t *testing.T) {
	for _, entry := range configuredTestEntrypoints() {
		t.Run(entry.name, func(t *testing.T) {
			sentinel := errors.New("config load failed")
			providerErr := fmt.Errorf("read connection config: %w", sentinel)
			reads := 0
			provider := connectionConfigProvider(func(any) error {
				reads++
				return providerErr
			})
			manager, _, poolCalls := newConfiguredTestManager(t, provider)
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				entry.open(t.Name(), manager)
			}()
			err, ok := recovered.(error)
			if !ok || err != providerErr || !errors.Is(err, sentinel) || reads != 1 || *poolCalls != 0 {
				t.Fatalf("config error=%v, reads=%d, pool calls=%d", recovered, reads, *poolCalls)
			}
		})
	}
}

func TestConnectDatabaseDoesNotReadConnectionConfig(t *testing.T) {
	reads := 0
	provider := connectionConfigProvider(func(any) error {
		reads++
		return errors.New("low-level constructor must not read config")
	})
	manager, _, poolCalls := newConfiguredTestManager(t, provider)
	if engine := ConnectDatabase(t.Name(), manager); engine == nil || reads != 0 || *poolCalls != 1 {
		t.Fatalf("low-level constructor read connection config: reads=%d, pool calls=%d", reads, *poolCalls)
	}
}

func TestConfigGetConfigCopiesStrategiesWithoutChangingSource(t *testing.T) {
	placeholder := &engineTestPlaceholderParser{}
	quoter := db.NewIdentifierQuoter("<", ">")
	column := precedenceColumnParser{columnName: "copied_name"}
	source := Config{EngineOptions: db.EngineOptions{
		PlaceholderParser: placeholder,
		IdentifierQuoter:  &quoter,
		ColumnParser:      &column,
	}}
	original := source
	for _, test := range []struct {
		name     string
		provider goi.DatabaseConfig
	}{
		{"value", source},
		{"pointer", &source},
	} {
		t.Run(test.name, func(t *testing.T) {
			var target Config
			if err := test.provider.GetConfig(&target); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(target, original) || target.EngineOptions.PlaceholderParser != placeholder ||
				target.EngineOptions.IdentifierQuoter != &quoter || target.EngineOptions.ColumnParser != &column {
				t.Fatalf("config copy lost strategy references: %#v", target)
			}
			target.EngineOptions = db.EngineOptions{}
			if !reflect.DeepEqual(source, original) {
				t.Fatalf("changing target modified source: %#v", source)
			}
		})
	}
}

func TestConfigGetConfigRejectsInvalidTargets(t *testing.T) {
	var typedNil *Config
	for _, test := range []struct {
		name   string
		target any
		isNil  bool
	}{
		{"nil", nil, true},
		{"typed_nil", typedNil, true},
		{"wrong_pointer", new(string), false},
		{"non_pointer", Config{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := (Config{}).GetConfig(test.target)
			if err == nil {
				t.Fatal("GetConfig accepted an invalid target")
			}
			want := i18n.T("db.config_type_is_not_match", map[string]any{
				"engine": driverName,
				"type":   fmt.Sprintf("%T", test.target),
			})
			if test.isNil {
				want = i18n.T("params.params_is_not_null", map[string]any{"name": "target"})
			}
			if err.Error() != want {
				t.Fatalf("GetConfig error=%q, want %q", err, want)
			}
		})
	}
}

type engineTestModel struct {
	ID     *int64  `field_name:"id" field_type:"BIGINT IDENTITY(1,1) PRIMARY KEY"`
	Name   *string `field_name:"name" field_type:"NVARCHAR(64) NOT NULL"`
	Status *string `field_name:"status" field_type:"NVARCHAR(32) DEFAULT 'enabled'"`
}

func TestColumnTypeCommentUsesParsedDefinition(t *testing.T) {
	field := reflect.StructField{Name: "Name"}
	if got := getComment(field, "VARCHAR(1) COMMENT '类型注释'"); got != "类型注释" {
		t.Fatalf("parsed type comment = %q", got)
	}
	field.Tag = `comment:"标签注释"`
	if got := getComment(field, "VARCHAR(1) COMMENT '类型注释'"); got != "标签注释" {
		t.Fatalf("comment tag = %q", got)
	}
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
	Name   *string `field_name:"name" field_type:"NVARCHAR(64)"`
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
	name *string `field_name:"name" field_type:"NVARCHAR(64) NOT NULL"`
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

const placeholderPrefix = "@p"

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
	return "[" + name + "]"
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
	tests := []struct {
		name     string
		query    string
		args     []any
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "scalar and multiple collections",
			query:    "tenant_id = " + placeholder + " AND id IN " + placeholder + " AND status NOT IN " + placeholder + " AND name = " + placeholder,
			args:     []any{7, []int64{1, 2}, [2]string{"disabled", "deleted"}, "admin"},
			wantSQL:  "tenant_id = @p1 AND id IN (@p2,@p3) AND status NOT IN (@p4,@p5) AND name = @p6",
			wantArgs: []any{7, int64(1), int64(2), "disabled", "deleted", "admin"},
		},
		{
			name:    "empty collection does not consume a number",
			query:   "tenant_id = " + placeholder + " AND id IN " + placeholder + " AND name = " + placeholder,
			args:    []any{7, []int64{}, "admin"},
			wantSQL: "tenant_id = @p1 AND id IN () AND name = @p2", wantArgs: []any{7, "admin"},
		},
		{
			name: "manual parentheses", query: "id IN (" + placeholder + "," + placeholder + ")",
			args: []any{1, 2}, wantSQL: "id IN (@p1,@p2)", wantArgs: []any{1, 2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalArgs := append([]any(nil), test.args...)
			query, args := parser.Parse(test.query, test.args...)
			if query != test.wantSQL || !reflect.DeepEqual(args, test.wantArgs) {
				t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", query, args, test.wantSQL, test.wantArgs)
			}
			if !reflect.DeepEqual(test.args, originalArgs) {
				t.Fatalf("input arguments were modified: %#v", test.args)
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
	Name *string `field_name:"name" field_type:"NVARCHAR(64)" column:"display_name" sql_type:"NVARCHAR(64)"`
	Skip *string `field_type:"-" sql_type:"-"`
}

func (alternateTagModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

type engineTestDBProvider struct {
	pool  *sql.DB
	calls int
}

func (provider *engineTestDBProvider) DB() *sql.DB {
	provider.calls++
	return provider.pool
}

func TestConnectDatabaseAcceptsBareProvider(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	provider := &engineTestDBProvider{pool: base.DB}
	engine := ConnectDatabase("bare", provider, db.EngineOptions{})
	if engine.DB != base.DB || engine.Name() != "bare" || provider.calls != 1 {
		t.Fatalf("provider was not reused: name=%q, calls=%d", engine.Name(), provider.calls)
	}
	if !reflect.DeepEqual(engine.placeholderParser, GetPlaceholderParser()) ||
		!reflect.DeepEqual(engine.identifierQuoter, GetIdentifierQuoter()) ||
		!reflect.DeepEqual(engine.columnParser, GetColumnParser()) {
		t.Fatal("nil options did not inherit package defaults")
	}
}

func TestConnectDatabaseMergesLocalStrategies(t *testing.T) {
	oldIdentifier, oldPlaceholder, oldColumn := GetIdentifierQuoter(), GetPlaceholderParser(), GetColumnParser()
	t.Cleanup(func() {
		SetIdentifierQuoter(oldIdentifier)
		SetPlaceholderParser(oldPlaceholder)
		SetColumnParser(oldColumn)
	})
	base, _ := newEngineTestEngine(t)
	provider := &engineTestDBProvider{pool: base.DB}
	first := db.EngineOptions{
		PlaceholderParser: namedPlaceholderParser{},
		IdentifierQuoter:  db.NewIdentifierQuoter("<", ">"),
		ColumnParser:      validationColumnParser{},
	}
	second := db.EngineOptions{
		PlaceholderParser: engineTestPlaceholderParser{},
		IdentifierQuoter:  db.NewIdentifierQuoter("{", "}"),
	}
	third := db.EngineOptions{ColumnParser: db.NewColumnParser("column", "sql_type")}
	engine := ConnectDatabase("local", provider, first, db.EngineOptions{}, second, db.EngineOptions{}, third)
	engine.SetModel(alternateTagModel{})
	if query, _ := engine.placeholderParser.Parse("SELECT ?", 1); engine.tableName != "{test_schema}.{roles}" ||
		!reflect.DeepEqual(engine.fieldSQL, []string{"role_id", "display_name"}) ||
		query != "parsed: SELECT ?" {
		t.Fatalf("local strategies not merged in order: table=%q, columns=%v", engine.tableName, engine.fieldSQL)
	}
	if !reflect.DeepEqual(GetIdentifierQuoter(), oldIdentifier) ||
		!reflect.DeepEqual(GetPlaceholderParser(), oldPlaceholder) ||
		!reflect.DeepEqual(GetColumnParser(), oldColumn) {
		t.Fatal("local strategies changed package defaults")
	}

	// 替换全局策略和调用方的选项值，不应改变已创建实例。
	second.IdentifierQuoter = oldIdentifier
	second.PlaceholderParser = oldPlaceholder
	third.ColumnParser = oldColumn
	SetIdentifierQuoter(db.NewIdentifierQuoter("<", ">"))
	SetPlaceholderParser(namedPlaceholderParser{})
	SetColumnParser(validationColumnParser{})
	engine.SetModel(alternateTagModel{})
	if query, _ := engine.placeholderParser.Parse("SELECT ?", 1); engine.tableName != "{test_schema}.{roles}" ||
		!reflect.DeepEqual(engine.fieldSQL, []string{"role_id", "display_name"}) ||
		query != "parsed: SELECT ?" {
		t.Fatal("created instance did not retain its strategy snapshot")
	}
	fresh := ConnectDatabase("fresh", provider)
	fresh.SetModel(alternateTagModel{})
	if fresh.tableName != "<test_schema>.<roles>" ||
		!reflect.DeepEqual(fresh.fieldSQL, []string{"custom_name"}) ||
		fresh.placeholderParser.Placeholder() != ":v" {
		t.Fatal("new instance did not inherit the latest defaults")
	}
}

func TestConnectRejectsDifferentEngine(t *testing.T) {
	name := t.Name()
	old, existed := goi.Settings.Databases[name]
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[name] = old
		} else {
			delete(goi.Settings.Databases, name)
		}
	})
	connected := false
	goi.Settings.Databases[name] = &goi.Database{
		Engine: "other",
		Connect: func(string) *sql.DB {
			connected = true
			return nil
		},
	}
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		Connect(name)
	}()
	if recovered == nil {
		t.Fatal("Connect accepted a different engine")
	}
	if connected {
		t.Fatal("Connect opened the pool before rejecting the engine")
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

func TestWhereInReportsCollectionExpansion(t *testing.T) {
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
		" ORDER BY " + quoteTestName("id") + " DESC OFFSET 2 ROWS FETCH NEXT 2 ROWS ONLY")
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

func TestWritesUseSelectedFieldsWithOptionalConditions(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	name := "renamed"
	if _, err := engine.Fields("Name").Insert(ctx, engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + ") VALUES (?)")
	if recorder.execQuery != want || len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != name {
		t.Fatalf("selected Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}

	// nil 是选中字段的 SQL NULL，不是省略字段的标志。
	if _, err := engine.Fields("Name").Insert(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != want || len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("selected NULL Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}

	// 不设置 Fields 时，全部映射字段都参与写入，即使所有指针都是 nil。
	if _, err := engine.Insert(ctx, &engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("id") + "," +
		quoteTestName("name") + "," + quoteTestName("status") + ") VALUES (?,?,?)")
	if recorder.execQuery != want || len(recorder.execArgs) != 3 {
		t.Fatalf("all-field NULL Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	for _, arg := range recorder.execArgs {
		if arg.Value != nil {
			t.Fatalf("all-field NULL Insert args = %#v", recorder.execArgs)
		}
	}

	// 模拟驱动只检查转交的 SQL，不表示真实数据库接受空列列表语法。
	if _, err := engine.Fields().Insert(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" () VALUES ()" ||
		len(recorder.execArgs) != 0 || recorder.execCount != 4 {
		t.Fatalf("empty selection Insert = %q, %#v; executions = %d", recorder.execQuery, recorder.execArgs, recorder.execCount)
	}
	if _, err := engine.Fields("Name").Update(ctx, engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?")
	if recorder.execQuery != want || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{name}) {
		t.Fatalf("unconditional selected Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "DELETE FROM "+engine.tableName || len(recorder.execArgs) != 0 {
		t.Fatalf("unconditional Delete = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	query := engine.Where("id = ?", int64(7))
	if _, err := query.Fields().Update(ctx, engineTestModel{}, false); err == nil {
		t.Fatal("empty selection Update was allowed")
	}
	if _, err := query.Fields("Name").Update(ctx, engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 2 ||
		recorder.execArgs[0].Value != name || recorder.execArgs[1].Value != int64(7) {
		t.Fatalf("selected Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := query.Fields("Name").Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != want || len(recorder.execArgs) != 2 ||
		recorder.execArgs[0].Value != nil || recorder.execArgs[1].Value != int64(7) {
		t.Fatalf("selected NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := query.Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
		quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 4 || recorder.execArgs[3].Value != int64(7) {
		t.Fatalf("all-field NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	for _, arg := range recorder.execArgs[:3] {
		if arg.Value != nil {
			t.Fatalf("all-field NULL Update args = %#v", recorder.execArgs)
		}
	}
	if _, err := query.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != wantDriverSQL("DELETE FROM "+engine.tableName+" WHERE id = ?") {
		t.Fatalf("Delete = %q", recorder.execQuery)
	}
	if _, err := engine.Insert(ctx, alternateTagModel{}, false); err == nil {
		t.Fatal("different operation model was accepted")
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
		quoteTestName("display_name") + " NVARCHAR(64)\n)"
	if recorder.execQuery != want {
		t.Fatalf("Migrate = %q, want %q", recorder.execQuery, want)
	}
	if recorder.queryQuery != wantDriverSQL("SELECT COUNT(1) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?") {
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
	Name *string `field_name:"name" field_type:"NVARCHAR(64)"`
}

func (defaultSchemaModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

func TestDefaultSchemaMatchesMigrationsAndQueries(t *testing.T) {
	engine, recorder := prepareMigrationTest(t)
	engine.SetModel(defaultSchemaModel{})
	if engine.tableName != quoteTestName("dbo")+"."+quoteTestName("roles") {
		t.Fatalf("default table = %q", engine.tableName)
	}
	engine.Migrate(context.Background(), defaultSchemaModel{})
	if recorder.queryQuery != wantDriverSQL("SELECT COUNT(1) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?") {
		t.Fatalf("default schema metadata = %q", recorder.queryQuery)
	}
}

type validationColumnParser struct{}

func (validationColumnParser) Parse(field reflect.StructField) (string, string, bool) {
	if field.Name == "Name" {
		return "custom_name", "NVARCHAR(64) NOT NULL", true
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

type engineZeroValueModel struct {
	Name  string `field_name:"name" field_type:"NVARCHAR(64)"`
	Count int64  `field_name:"count" field_type:"BIGINT"`
}

func (engineZeroValueModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestWritesIncludeSelectedZeroValues(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineZeroValueModel{})
	ctx := context.Background()
	if _, err := engine.Insert(ctx, engineZeroValueModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != "" || recorder.execArgs[1].Value != int64(0) {
		t.Fatalf("zero-value Insert args = %#v", recorder.execArgs)
	}
	if _, err := engine.Fields("Name").Where("id = ?", int64(7)).Update(ctx, engineZeroValueModel{}, false); err != nil {
		t.Fatal(err)
	}
	want := wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?")
	if recorder.execQuery != want || len(recorder.execArgs) != 2 ||
		recorder.execArgs[0].Value != "" || recorder.execArgs[1].Value != int64(7) {
		t.Fatalf("zero-value selected Update = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
}

type enginePointerValuer struct {
	text string
}

func (value *enginePointerValuer) Value() (driver.Value, error) {
	if value == nil {
		return "nil receiver must have been normalized to NULL", nil
	}
	return "converted:" + value.text, nil
}

type engineValuerModel struct {
	Name *enginePointerValuer `field_name:"name" field_type:"NVARCHAR(64)"`
}

func (engineValuerModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestWritesPreservePointerValuer(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineValuerModel{})
	ctx := context.Background()
	model := engineValuerModel{Name: &enginePointerValuer{text: "value"}}
	if _, err := engine.Insert(ctx, model, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != "converted:value" {
		t.Fatalf("pointer Valuer Insert args = %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, model, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != "converted:value" {
		t.Fatalf("pointer Valuer Update args = %#v", recorder.execArgs)
	}
	if _, err := engine.Insert(ctx, engineValuerModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("nil pointer Valuer Insert args = %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineValuerModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != nil {
		t.Fatalf("nil pointer Valuer Update args = %#v", recorder.execArgs)
	}
}

func recordedValues(args []driver.NamedValue) []any {
	values := make([]any, len(args))
	for index, arg := range args {
		values[index] = arg.Value
	}
	return values
}

func TestSkipNilControlsInsertAndUpdateParameters(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			ctx := context.Background()
			name := "selected"
			model := engineTestModel{Name: &name}
			if _, err := engine.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			columns := quoteTestName("id") + "," + quoteTestName("name") + "," + quoteTestName("status")
			insertSQL := "INSERT INTO " + engine.tableName + " (" + columns + ") VALUES (?,?,?)"
			wantInsertArgs := []any{nil, name, nil}
			updateSQL := "UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
				quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?"
			wantUpdateArgs := []any{nil, name, nil, int64(7)}
			if skipNil {
				insertSQL = "INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + ") VALUES (?)"
				wantInsertArgs = []any{name}
				updateSQL = "UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ? WHERE id = ?"
				wantUpdateArgs = []any{name, int64(7)}
			}
			if recorder.execQuery != wantDriverSQL(insertSQL) || !reflect.DeepEqual(recordedValues(recorder.execArgs), wantInsertArgs) {
				t.Fatalf("mixed Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
			}
			if _, err := engine.Where("id = ?", int64(7)).Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			if recorder.execQuery != wantDriverSQL(updateSQL) || !reflect.DeepEqual(recordedValues(recorder.execArgs), wantUpdateArgs) {
				t.Fatalf("mixed Update = %q, %#v", recorder.execQuery, recorder.execArgs)
			}

			// Fields 的选列及顺序先确定，skipNil 仅过滤这些选中字段。
			selected := engine.Fields("Status", "Name")
			if _, err := selected.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			if skipNil {
				if recorder.execQuery != wantDriverSQL(insertSQL) || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{name}) {
					t.Fatalf("selected Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
				}
			} else {
				want := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("status") + "," + quoteTestName("name") + ") VALUES (?,?)")
				if recorder.execQuery != want || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{nil, name}) {
					t.Fatalf("selected Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
				}
			}
			if _, err := selected.Where("id = ?", int64(7)).Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			if skipNil {
				if recorder.execQuery != wantDriverSQL(updateSQL) || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{name, int64(7)}) {
					t.Fatalf("selected Update = %q, %#v", recorder.execQuery, recorder.execArgs)
				}
			} else {
				want := wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("status") + " = ?," + quoteTestName("name") + " = ? WHERE id = ?")
				if recorder.execQuery != want || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{nil, name, int64(7)}) {
					t.Fatalf("selected Update = %q, %#v", recorder.execQuery, recorder.execArgs)
				}
			}
		})
	}
}

func TestSkipNilKeepsEmptyWriteBehavior(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	if _, err := engine.Insert(ctx, engineTestModel{}, true); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("all-skipped Insert = %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	executions := recorder.execCount
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineTestModel{}, true); err == nil {
		t.Fatal("all-skipped Update was allowed")
	}
	if recorder.execCount != executions {
		t.Fatal("all-skipped Update reached the SQL driver")
	}

	// 即使未选中字段有值，也不能因 skipNil 过滤后为空而扩大写入范围。
	name := "not selected"
	selected := engine.Fields("Status")
	if _, err := selected.Insert(ctx, engineTestModel{Name: &name}, true); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO "+engine.tableName+" () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("selection was expanded: %q, %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := selected.Where("id = ?", int64(7)).Update(ctx, engineTestModel{Name: &name}, true); err == nil {
		t.Fatal("unselected populated field participated in Update")
	}
}

type engineNullValuer struct {
	calls *int
}

func (value *engineNullValuer) Value() (driver.Value, error) {
	(*value.calls)++
	return nil, nil
}

type engineNullConversionModel struct {
	Name   sql.NullString    `field_name:"name" field_type:"NVARCHAR(64)"`
	Status *engineNullValuer `field_name:"status" field_type:"NVARCHAR(64)"`
}

func (engineNullConversionModel) ModelSet() *Settings {
	return &Settings{Schema: "test_schema", TableName: "roles"}
}

func TestSkipNilDoesNotEvaluateOrOmitNullValuers(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			engine.SetModel(engineNullConversionModel{})
			ctx := context.Background()
			calls := 0
			model := engineNullConversionModel{Status: &engineNullValuer{calls: &calls}}
			if _, err := engine.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			want := wantDriverSQL("INSERT INTO " + engine.tableName + " (" + quoteTestName("name") + "," + quoteTestName("status") + ") VALUES (?,?)")
			if recorder.execQuery != want || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{nil, nil}) || calls != 1 {
				t.Fatalf("NULL Valuer Insert = %q, %#v; calls=%d", recorder.execQuery, recorder.execArgs, calls)
			}
			if _, err := engine.Where("id = ?", int64(7)).Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			want = wantDriverSQL("UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?," + quoteTestName("status") + " = ? WHERE id = ?")
			if recorder.execQuery != want || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{nil, nil, int64(7)}) || calls != 2 {
				t.Fatalf("NULL Valuer Update = %q, %#v; calls=%d", recorder.execQuery, recorder.execArgs, calls)
			}
		})
	}
}

func TestSkipNilDoesNotOmitZeroValues(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineZeroValueModel{})
	ctx := context.Background()
	if _, err := engine.Insert(ctx, engineZeroValueModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{"", int64(0)}) {
		t.Fatalf("skipNil zero Insert args = %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineZeroValueModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{"", int64(0), int64(7)}) {
		t.Fatalf("skipNil zero Update args = %#v", recorder.execArgs)
	}
}

func TestUnconditionalUpdateRespectsSkipNil(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipNil=%v", skipNil), func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			ctx := context.Background()
			name := "all rows"
			if _, err := engine.Update(ctx, engineTestModel{Name: &name}, skipNil); err != nil {
				t.Fatal(err)
			}
			want := "UPDATE " + engine.tableName + " SET " + quoteTestName("id") + " = ?," +
				quoteTestName("name") + " = ?," + quoteTestName("status") + " = ?"
			wantArgs := []any{nil, name, nil}
			if skipNil {
				want = "UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?"
				wantArgs = []any{name}
			}
			if recorder.execQuery != wantDriverSQL(want) || !reflect.DeepEqual(recordedValues(recorder.execArgs), wantArgs) {
				t.Fatalf("unconditional Update = %q, %#v", recorder.execQuery, recorder.execArgs)
			}

			// 省略 WHERE 不改变 Fields 的选列规则。
			if _, err := engine.Fields("Name").Update(ctx, engineTestModel{}, false); err != nil {
				t.Fatal(err)
			}
			want = "UPDATE " + engine.tableName + " SET " + quoteTestName("name") + " = ?"
			if recorder.execQuery != wantDriverSQL(want) || !reflect.DeepEqual(recordedValues(recorder.execArgs), []any{nil}) {
				t.Fatalf("unconditional NULL Update = %q, %#v", recorder.execQuery, recorder.execArgs)
			}

			executions := recorder.execCount
			if _, err := engine.Fields().Update(ctx, engineTestModel{}, false); err == nil {
				t.Fatal("unconditional Update accepted an empty field selection")
			}
			if _, err := engine.Update(ctx, engineTestModel{}, true); err == nil {
				t.Fatal("unconditional Update accepted all-skipped fields")
			}
			if recorder.execCount != executions {
				t.Fatal("empty unconditional Update reached the SQL driver")
			}
		})
	}
}
