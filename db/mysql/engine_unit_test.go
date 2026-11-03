package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

type engineTestDBProvider struct {
	database *sql.DB
	calls    int
}

func (provider *engineTestDBProvider) DB() *sql.DB {
	provider.calls++
	return provider.database
}

type engineOptionsTestModel struct {
	Name *string `field_name:"name" field_type:"VARCHAR(32)" column:"display_name" sql_type:"TEXT"`
}

func (engineOptionsTestModel) ModelSet() *Settings {
	return &Settings{TableName: "option_roles"}
}

func TestConnectDatabaseAcceptsDBProvider(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	provider := &engineTestDBProvider{database: base.DB}
	engine := ConnectDatabase("custom_connection", provider)
	if engine.DB != base.DB || engine.Name() != "custom_connection" || provider.calls != 1 {
		t.Fatalf("ConnectDatabase() did not preserve pool and name: DB=%p name=%q calls=%d",
			engine.DB, engine.Name(), provider.calls)
	}
	if !reflect.DeepEqual(engine.placeholderParser, GetPlaceholderParser()) ||
		!reflect.DeepEqual(engine.identifierQuoter, GetIdentifierQuoter()) ||
		!reflect.DeepEqual(engine.columnParser, GetColumnParser()) {
		t.Fatal("ConnectDatabase() did not inherit package strategies")
	}
}

func TestConnectDatabaseMergesLocalOptions(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	defaults := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	local := db.EngineOptions{
		PlaceholderParser: engineTestPlaceholderParser{},
		IdentifierQuoter:  db.NewIdentifierQuoter("[", "]"),
		ColumnParser:      db.NewColumnParser("column", "sql_type"),
	}
	lastQuoter := db.NewIdentifierQuoter("<", ">")
	tests := []struct {
		name    string
		options []db.EngineOptions
		want    db.EngineOptions
	}{
		{
			name:    "placeholder",
			options: []db.EngineOptions{{PlaceholderParser: local.PlaceholderParser}},
			want: db.EngineOptions{
				PlaceholderParser: local.PlaceholderParser,
				IdentifierQuoter:  defaults.IdentifierQuoter,
				ColumnParser:      defaults.ColumnParser,
			},
		},
		{
			name:    "identifier",
			options: []db.EngineOptions{{IdentifierQuoter: local.IdentifierQuoter}},
			want: db.EngineOptions{
				PlaceholderParser: defaults.PlaceholderParser,
				IdentifierQuoter:  local.IdentifierQuoter,
				ColumnParser:      defaults.ColumnParser,
			},
		},
		{
			name:    "column",
			options: []db.EngineOptions{{ColumnParser: local.ColumnParser}},
			want: db.EngineOptions{
				PlaceholderParser: defaults.PlaceholderParser,
				IdentifierQuoter:  defaults.IdentifierQuoter,
				ColumnParser:      local.ColumnParser,
			},
		},
		{
			name:    "ordered overrides and nil inheritance",
			options: []db.EngineOptions{local, {}, {IdentifierQuoter: lastQuoter}, {}},
			want: db.EngineOptions{
				PlaceholderParser: local.PlaceholderParser,
				IdentifierQuoter:  lastQuoter,
				ColumnParser:      local.ColumnParser,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := ConnectDatabase("local", &engineTestDBProvider{database: base.DB}, test.options...)
			got := db.EngineOptions{
				PlaceholderParser: engine.placeholderParser,
				IdentifierQuoter:  engine.identifierQuoter,
				ColumnParser:      engine.columnParser,
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ConnectDatabase() options = %#v, want %#v", got, test.want)
			}
		})
	}
	after := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	if !reflect.DeepEqual(after, defaults) {
		t.Fatal("local options changed package strategies")
	}
}

func TestConnectDatabaseSnapshotsStrategies(t *testing.T) {
	original := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	t.Cleanup(func() {
		SetPlaceholderParser(original.PlaceholderParser)
		SetIdentifierQuoter(original.IdentifierQuoter)
		SetColumnParser(original.ColumnParser)
	})
	SetPlaceholderParser(db.NewPlaceholderParser("?"))
	SetIdentifierQuoter(db.NewIdentifierQuoter("`", "`"))
	SetColumnParser(db.NewColumnParser("field_name", "field_type"))

	base, recorder := newEngineTestEngine(t)
	provider := &engineTestDBProvider{database: base.DB}
	first := ConnectDatabase("first", provider)
	SetPlaceholderParser(engineTestPlaceholderParser{})
	SetIdentifierQuoter(db.NewIdentifierQuoter("<", ">"))
	SetColumnParser(db.NewColumnParser("column", "sql_type"))
	second := ConnectDatabase("second", provider)

	first.SetModel(engineOptionsTestModel{})
	second.SetModel(engineOptionsTestModel{})
	if first.tableName != "`option_roles`" || second.tableName != "<option_roles>" {
		t.Fatalf("identifier snapshots changed: first=%q second=%q", first.tableName, second.tableName)
	}
	if !reflect.DeepEqual(first.fieldSQL, []string{"name"}) ||
		!reflect.DeepEqual(second.fieldSQL, []string{"display_name"}) {
		t.Fatalf("column snapshots changed: first=%v second=%v", first.fieldSQL, second.fieldSQL)
	}
	name := "admin"
	model := engineOptionsTestModel{Name: &name}
	if _, err := first.Insert(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `option_roles` (`name`) VALUES (?)" {
		t.Fatalf("existing Engine changed its strategies: %q", recorder.execQuery)
	}
	if _, err := second.Insert(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "parsed: INSERT INTO <option_roles> (<display_name>) VALUES (?)" {
		t.Fatalf("new Engine did not use updated strategies: %q", recorder.execQuery)
	}
}

func TestConnectRejectsMismatchedEngine(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	original := goi.Settings.Databases
	t.Cleanup(func() { goi.Settings.Databases = original })
	called := false
	goi.Settings.Databases = map[string]*goi.Database{
		"mismatch": {
			Engine: "other_engine",
			Connect: func(string) *sql.DB {
				called = true
				return base.DB
			},
		},
	}
	defer func() {
		if recover() == nil {
			t.Error("Connect() accepted a mismatched engine")
		}
		if called {
			t.Error("Connect() opened a pool before rejecting the engine")
		}
	}()
	Connect("mismatch")
}

type connectionConfigProvider func(any) error

func (provider connectionConfigProvider) GetConfig(target any) error { return provider(target) }

type configEntryPoint struct {
	name    string
	connect func(string, *goi.Database) *Engine
}

func configEntryPoints(t *testing.T) []configEntryPoint {
	t.Helper()
	original := goi.Settings.Databases
	goi.Settings.Databases = map[string]*goi.Database{}
	t.Cleanup(func() { goi.Settings.Databases = original })
	return []configEntryPoint{
		{"registered", db.ConnectDatabase[*Engine]},
		{"package", func(name string, database *goi.Database) *Engine {
			goi.Settings.Databases[name] = database
			return Connect(name)
		}},
	}
}

func TestConnectReadsConnectionConfig(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	defaults := db.EngineOptions{
		PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter:  GetIdentifierQuoter(),
		ColumnParser:      GetColumnParser(),
	}
	placeholder := db.NewPlaceholderParser(":v")
	quoter := db.NewIdentifierQuoter("<", ">")
	column := db.NewColumnParser("column", "sql_type")
	config := Config{EngineOptions: db.EngineOptions{
		PlaceholderParser: &placeholder,
		IdentifierQuoter:  &quoter,
		ColumnParser:      &column,
	}}
	var typedNil *Config
	providerCalls := 0
	provider := connectionConfigProvider(func(target any) error {
		providerCalls++
		return config.GetConfig(target)
	})
	for _, entry := range configEntryPoints(t) {
		t.Run(entry.name, func(t *testing.T) {
			for _, form := range []struct {
				name   string
				config goi.DatabaseConfig
				want   db.EngineOptions
			}{
				{"nil", nil, defaults},
				{"typed_nil", typedNil, defaults},
				{"empty", Config{}, defaults},
				{"value", config, config.EngineOptions},
				{"pointer", &config, config.EngineOptions},
				{"provider", provider, config.EngineOptions},
			} {
				t.Run(form.name, func(t *testing.T) {
					poolCalls, before := 0, providerCalls
					manager := &goi.Database{
						Engine: driverName,
						Config: form.config,
						Connect: func(string) *sql.DB {
							poolCalls++
							return base.DB
						},
					}
					engine := entry.connect(t.Name(), manager)
					got := db.EngineOptions{PlaceholderParser: engine.placeholderParser,
						IdentifierQuoter: engine.identifierQuoter, ColumnParser: engine.columnParser}
					if !reflect.DeepEqual(got, form.want) || poolCalls != 1 || engine.DB != base.DB {
						t.Fatalf("options=%#v, pool calls=%d", got, poolCalls)
					}
					if form.name == "provider" && providerCalls != before+1 {
						t.Fatalf("GetConfig calls=%d, want 1", providerCalls-before)
					}
				})
			}
		})
	}
	if !reflect.DeepEqual(defaults, db.EngineOptions{PlaceholderParser: GetPlaceholderParser(),
		IdentifierQuoter: GetIdentifierQuoter(), ColumnParser: GetColumnParser()}) {
		t.Fatal("connection config changed package strategies")
	}
}

func TestConnectionConfigErrorPrecedesPoolCreation(t *testing.T) {
	sentinel := errors.New("unavailable")
	want := fmt.Errorf("configuration: %w", sentinel)
	for _, entry := range configEntryPoints(t) {
		t.Run(entry.name, func(t *testing.T) {
			configCalls, poolCalls := 0, 0
			manager := &goi.Database{
				Engine: driverName,
				Config: connectionConfigProvider(func(any) error {
					configCalls++
					return want
				}),
				Connect: func(string) *sql.DB { poolCalls++; return nil },
			}
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				entry.connect(t.Name(), manager)
			}()
			err, ok := recovered.(error)
			if !ok || err != want || !errors.Is(err, sentinel) || configCalls != 1 || poolCalls != 0 {
				t.Fatalf("panic=%v, config calls=%d, pool calls=%d", recovered, configCalls, poolCalls)
			}
		})
	}
}

func TestLowLevelConnectDatabaseDoesNotReadConfig(t *testing.T) {
	base, _ := newEngineTestEngine(t)
	configCalls := 0
	manager := &goi.Database{
		Engine: driverName,
		Config: connectionConfigProvider(func(any) error {
			configCalls++
			return errors.New("must not read Config")
		}),
		Connect: func(string) *sql.DB { return base.DB },
	}
	engine := ConnectDatabase("direct", manager)
	if configCalls != 0 || engine.DB != base.DB {
		t.Fatalf("config calls=%d, DB=%p", configCalls, engine.DB)
	}
}

func TestConfigGetConfig(t *testing.T) {
	placeholder := db.NewPlaceholderParser(":v")
	quoter := db.NewIdentifierQuoter("<", ">")
	column := db.NewColumnParser("column", "sql_type")
	source := Config{EngineOptions: db.EngineOptions{PlaceholderParser: &placeholder,
		IdentifierQuoter: &quoter, ColumnParser: &column}}
	for _, provider := range []goi.DatabaseConfig{source, &source} {
		var target Config
		if err := provider.GetConfig(&target); err != nil {
			t.Fatal(err)
		}
		if target.EngineOptions.PlaceholderParser != &placeholder ||
			target.EngineOptions.IdentifierQuoter != &quoter || target.EngineOptions.ColumnParser != &column {
			t.Fatal("GetConfig did not preserve shared strategy references")
		}
		target.EngineOptions = db.EngineOptions{}
		if source.EngineOptions.ColumnParser != &column {
			t.Fatal("changing the target changed the source")
		}
	}
	var typedNil *Config
	for _, target := range []any{nil, typedNil, new(string), Config{}} {
		err := source.GetConfig(target)
		want := i18n.T("db.config_type_is_not_match", map[string]any{
			"engine": driverName, "type": fmt.Sprintf("%T", target),
		})
		if target == nil || target == typedNil {
			want = i18n.T("params.params_is_not_null", map[string]any{"name": "target"})
		}
		if err == nil || err.Error() != want {
			t.Fatalf("GetConfig(%T) = %v, want %q", target, err, want)
		}
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
	manager.Config = Config{EngineOptions: db.EngineOptions{ColumnParser: instanceOverride}}
	configuredEngine := db.ConnectDatabase[*Engine]("configured", manager)
	assertColumn(configuredEngine, "instance_name")
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
	assertColumn(configuredEngine, "instance_name")
	assertColumn(ConnectDatabase("new_shared", manager), "next_shared_name")
	if err := Validate(engineTestModel{}, false); err == nil || !strings.Contains(err.Error(), "next_shared_name") {
		t.Fatalf("Validate did not obtain the updated shared parser: %v", err)
	}
}

type engineTestModel struct {
	ID     *int64  `field_name:"id" field_type:"BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"`
	Name   *string `field_name:"name" field_type:"VARCHAR(64) NOT NULL"`
	Status *string `field_name:"status" field_type:"VARCHAR(32) DEFAULT 'enabled'"`
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

func (engineTestPlaceholderParser) Parse(query string, _ ...any) string {
	return "parsed: " + query
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

func TestExecutionMethodsRecordParsedSQL(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.placeholderParser = engineTestPlaceholderParser{}
	ctx := context.Background()

	const executeQuery = "UPDATE roles SET name = ?"
	if _, err := engine.ExecuteContext(ctx, executeQuery, "admin"); err != nil {
		t.Fatalf("ExecuteContext() returned an error: %v", err)
	}
	wantSQL := "parsed: " + executeQuery
	if engine.GetSQL() != wantSQL || recorder.execQuery != wantSQL {
		t.Fatalf("ExecuteContext() recorded SQL=%q driver SQL=%q, want %q", engine.GetSQL(), recorder.execQuery, wantSQL)
	}

	recorder.columns = []string{"value"}
	recorder.rows = [][]driver.Value{{int64(1)}}
	const queryRowQuery = "SELECT ?"
	var value int64
	if err := engine.QueryRowContext(ctx, queryRowQuery, int64(1)).Scan(&value); err != nil {
		t.Fatalf("QueryRowContext() returned an error: %v", err)
	}
	wantSQL = "parsed: " + queryRowQuery
	if engine.GetSQL() != wantSQL || recorder.queryQuery != wantSQL {
		t.Fatalf("QueryRowContext() recorded SQL=%q driver SQL=%q, want %q", engine.GetSQL(), recorder.queryQuery, wantSQL)
	}

	const query = "SELECT ? FROM roles"
	rows, err := engine.QueryContext(ctx, query, int64(1))
	if err != nil {
		t.Fatalf("QueryContext() returned an error: %v", err)
	}
	if err = rows.Close(); err != nil {
		t.Fatalf("Rows.Close() returned an error: %v", err)
	}
	wantSQL = "parsed: " + query
	if engine.GetSQL() != wantSQL || recorder.queryQuery != wantSQL {
		t.Fatalf("QueryContext() recorded SQL=%q driver SQL=%q, want %q", engine.GetSQL(), recorder.queryQuery, wantSQL)
	}
}

func TestSetModelKeepsLastExecutedSQL(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	if _, err := engine.ExecuteContext(context.Background(), "UPDATE roles SET name = ?", "admin"); err != nil {
		t.Fatalf("ExecuteContext() returned an error: %v", err)
	}
	lastSQL := engine.GetSQL()

	engine.SetModel(engineTestModel{})
	if engine.GetSQL() != lastSQL {
		t.Fatalf("SetModel() changed the last executed SQL from %q to %q", lastSQL, engine.GetSQL())
	}
}

func TestInsertIncludesSelectedNilFieldsAsNull(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	name := "admin"

	if _, err := engine.Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatalf("Insert() returned an error: %v", err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` (`id`,`name`,`status`) VALUES (?,?,?)" {
		t.Fatalf("unexpected INSERT SQL: %s", recorder.execQuery)
	}
	if len(recorder.execArgs) != 3 || recorder.execArgs[0].Value != nil || recorder.execArgs[1].Value != name || recorder.execArgs[2].Value != nil {
		t.Fatalf("unexpected INSERT args: %#v", recorder.execArgs)
	}
}

func TestMigrateUsesSettingsSchema(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"exists"}
	recorder.rows = [][]driver.Value{{int64(1)}}

	engine.Migrate(context.Background(), engineTestModel{})
	if recorder.queryQuery != "SELECT 1 FROM `information_schema`.`tables` WHERE `table_schema` = ? AND `table_name` = ?;" {
		t.Fatalf("unexpected migration lookup SQL: %s", recorder.queryQuery)
	}
	if len(recorder.queryArgs) != 2 || recorder.queryArgs[0].Value != "test_schema" || recorder.queryArgs[1].Value != "roles" {
		t.Fatalf("unexpected migration lookup args: %#v", recorder.queryArgs)
	}
}

func TestInsertWritesNullForAllSelectedNilFields(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)

	if _, err := engine.Insert(context.Background(), engineTestModel{}, false); err != nil {
		t.Fatalf("Insert() returned an error: %v", err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` (`id`,`name`,`status`) VALUES (?,?,?)" {
		t.Fatalf("unexpected all-nil INSERT SQL: %s", recorder.execQuery)
	}
	wantArgs := []driver.NamedValue{{Ordinal: 1, Value: nil}, {Ordinal: 2, Value: nil}, {Ordinal: 3, Value: nil}}
	if !reflect.DeepEqual(recorder.execArgs, wantArgs) {
		t.Fatalf("unexpected all-nil INSERT args: %#v", recorder.execArgs)
	}
}

func TestInsertSkipsUnreadableModelField(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineUnexportedModel{})
	name := "admin"

	if _, err := engine.Insert(context.Background(), engineUnexportedModel{name: &name}, false); err != nil {
		t.Fatal(err)
	}
	// 模拟驱动只检查不可读取字段未参与 SQL，不验证空字段 SQL 的方言合法性。
	if recorder.execQuery != "INSERT INTO `roles` () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("unreadable field reached INSERT: %q %#v", recorder.execQuery, recorder.execArgs)
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
		{name: "bytes", leftPart: "payload IN ", placeholder: "?", param: bytes},
		{name: "bytes pointer", leftPart: "payload IN ", placeholder: "?", param: &bytes},
		{name: "typed nil bytes", leftPart: "payload IN ", placeholder: "?", param: []byte(nil)},
		{name: "untyped nil", leftPart: "id IN ", placeholder: "?", param: nil},
		{name: "typed nil pointer", leftPart: "id IN ", placeholder: "?", param: nilPointer},
		{name: "short left part", leftPart: "i", placeholder: "?", param: values},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSQL, gotArgs, gotMatched := whereIn(test.leftPart, test.placeholder, test.param)
			if gotSQL != test.wantSQL || !reflect.DeepEqual(gotArgs, test.wantArgs) || gotMatched != test.wantMatched {
				t.Fatalf("whereIn() = (%q, %#v, %t), want (%q, %#v, %t)", gotSQL, gotArgs, gotMatched, test.wantSQL, test.wantArgs, test.wantMatched)
			}
		})
	}
}

func TestWhereExpandsInCollection(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	statuses := []string{"enabled", "disabled"}

	where := engine.Where("id = ? AND status IN ?", int64(1), statuses)
	wantSQL := []string{"id = ? AND status IN (?,?)"}
	wantArgs := []any{int64(1), "enabled", "disabled"}
	if !reflect.DeepEqual(where.whereSQL, wantSQL) || !reflect.DeepEqual(where.args, wantArgs) {
		t.Fatalf("Where() generated SQL=%#v args=%#v", where.whereSQL, where.args)
	}

	manual := engine.Where("status IN (?,?)", "enabled", "disabled")
	if !reflect.DeepEqual(manual.whereSQL, []string{"status IN (?,?)"}) ||
		!reflect.DeepEqual(manual.args, []any{"enabled", "disabled"}) {
		t.Fatalf("Where() with manual IN placeholders generated SQL=%#v args=%#v", manual.whereSQL, manual.args)
	}
}

func TestWhereHandlesEmptyCollectionAndBytes(t *testing.T) {
	engine, _ := newEngineTestEngine(t)

	empty := engine.Where("id IN ?", []int64{})
	if !reflect.DeepEqual(empty.whereSQL, []string{"id IN ()"}) || len(empty.args) != 0 {
		t.Fatalf("empty collection generated SQL=%#v args=%#v", empty.whereSQL, empty.args)
	}
	notIn := engine.Where("id NOT IN ?", []int64{})
	if !reflect.DeepEqual(notIn.whereSQL, []string{"id NOT IN ()"}) || len(notIn.args) != 0 {
		t.Fatalf("empty NOT IN generated SQL=%#v args=%#v", notIn.whereSQL, notIn.args)
	}

	blob := []byte{1, 2, 3}
	bytesCondition := engine.Where("payload = ?", blob)
	if !reflect.DeepEqual(bytesCondition.whereSQL, []string{"payload = ?"}) ||
		len(bytesCondition.args) != 1 || !reflect.DeepEqual(bytesCondition.args[0], blob) {
		t.Fatalf("[]byte generated SQL=%#v args=%#v", bytesCondition.whereSQL, bytesCondition.args)
	}
}

func TestWhereKeepsCollectionsWithoutInPlaceholder(t *testing.T) {
	tests := []struct {
		name  string
		query string
		arg   any
	}{
		{name: "slice outside IN", query: "payload = ?", arg: []string{"first", "second"}},
		{name: "slice in wrapped IN", query: "status IN (?)", arg: []string{"enabled", "disabled"}},
		{name: "array outside IN", query: "payload = ?", arg: [2]int{1, 2}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := newEngineTestEngine(t)
			where := engine.Where(test.query, test.arg)
			if !reflect.DeepEqual(where.whereSQL, []string{test.query}) ||
				!reflect.DeepEqual(where.args, []any{test.arg}) {
				t.Fatalf("Where(%q) changed an unmatched collection: whereSQL=%#v args=%#v", test.query, where.whereSQL, where.args)
			}
		})
	}
}

func TestWhereBranchesDoNotShareState(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	seed := engine.Where("id > ?", int64(0))
	seed.whereSQL = append(make([]string, 0, 4), seed.whereSQL...)
	seed.args = append(make([]any, 0, 4), seed.args...)
	left := seed.Where("name = ?", "left")
	right := seed.Where("status = ?", "right")

	if !reflect.DeepEqual(seed.whereSQL, []string{"id > ?"}) || !reflect.DeepEqual(seed.args, []any{int64(0)}) {
		t.Fatalf("seed query state was modified: whereSQL=%#v args=%#v", seed.whereSQL, seed.args)
	}
	if !reflect.DeepEqual(left.whereSQL, []string{"id > ?", "name = ?"}) ||
		!reflect.DeepEqual(left.args, []any{int64(0), "left"}) {
		t.Fatalf("left query state is incorrect: whereSQL=%#v args=%#v", left.whereSQL, left.args)
	}
	if !reflect.DeepEqual(right.whereSQL, []string{"id > ?", "status = ?"}) ||
		!reflect.DeepEqual(right.args, []any{int64(0), "right"}) {
		t.Fatalf("right query state is incorrect: whereSQL=%#v args=%#v", right.whereSQL, right.args)
	}
}

func TestSelectAcceptsPointerToNilSlice(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	var roles []engineTestModel

	if err := engine.Select(context.Background(), &roles); err != nil {
		t.Fatalf("Select() returned an error: %v", err)
	}
	if len(roles) != 1 || roles[0].ID == nil || *roles[0].ID != 1 {
		t.Fatalf("unexpected Select result: %#v", roles)
	}
	if recorder.queryQuery != "SELECT `id`,`name`,`status` FROM `test_schema`.`roles`" {
		t.Fatalf("unexpected SELECT SQL: %s", recorder.queryQuery)
	}
}

func TestSelectSupportsMapResults(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	var maps []map[string]any
	if err := engine.Select(context.Background(), &maps); err != nil {
		t.Fatalf("Select(map) returned an error: %v", err)
	}
	if len(maps) != 1 || maps[0]["Name"] == nil {
		t.Fatalf("unexpected map results: %#v", maps)
	}

	recorder.columns = []string{"name"}
	recorder.rows = [][]driver.Value{{"admin"}}
	var roles []map[string]*string
	if err := engine.Fields("Name").Select(context.Background(), &roles); err != nil {
		t.Fatalf("Select(typed map) returned an error: %v", err)
	}
	if len(roles) != 1 || roles[0]["Name"] == nil || *roles[0]["Name"] != "admin" {
		t.Fatalf("unexpected typed map results: %#v", roles)
	}

	recorder.columns = []string{"id"}
	recorder.rows = [][]driver.Value{{int64(1)}}
	var roleIDs []map[string]*int64
	if err := engine.Fields("ID").Select(context.Background(), &roleIDs); err != nil {
		t.Fatalf("Select(pointer value map) returned an error: %v", err)
	}
	if len(roleIDs) != 1 || roleIDs[0]["ID"] == nil || *roleIDs[0]["ID"] != 1 {
		t.Fatalf("unexpected pointer value map results: %#v", roleIDs)
	}
}

func TestSelectRejectsNilPointerAndArray(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	var rolesPointer *[]engineTestModel
	if err := engine.Select(context.Background(), rolesPointer); err == nil {
		t.Fatal("Select() accepted a nil slice pointer")
	}

	rolesArray := [1]engineTestModel{}
	if err := engine.Select(context.Background(), &rolesArray); err == nil {
		t.Fatal("Select() accepted an array target")
	}
}

func TestFirstSupportsMapAndRejectsMapPointer(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	direct := map[string]any{}
	if err := engine.First(context.Background(), direct); err != nil {
		t.Fatalf("First(map) returned an error: %v", err)
	}
	if _, ok := direct["Name"]; !ok {
		t.Fatalf("First(map) did not populate the map: %#v", direct)
	}

	var pointer map[string]any
	if err := engine.First(context.Background(), &pointer); err == nil {
		t.Fatal("First() accepted a map pointer")
	}
}

func TestFirstSupportsStructAndTypedMap(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	var nilMap map[string]any
	if err := engine.First(context.Background(), nilMap); err == nil {
		t.Fatal("First() accepted a direct nil map")
	}

	recorder.columns = []string{"name"}
	recorder.rows = [][]driver.Value{{"admin"}}
	typedMap := map[string]*string{}
	if err := engine.Fields("Name").First(context.Background(), typedMap); err != nil {
		t.Fatalf("First(typed map) returned an error: %v", err)
	}
	if typedMap["Name"] == nil || *typedMap["Name"] != "admin" {
		t.Fatalf("unexpected typed map result: %#v", typedMap)
	}
	var role engineTestModel
	if err := engine.Fields("Name").First(context.Background(), &role); err != nil {
		t.Fatalf("First(struct) returned an error: %v", err)
	}
	if role.Name == nil || *role.Name != "admin" {
		t.Fatalf("unexpected struct result: %#v", role)
	}
}

func TestUpdateAndDeleteAllowWithoutWhere(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	name := "updated"
	update := engineTestModel{Name: &name}

	if _, err := engine.Fields("Name").Update(context.Background(), update, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `name` = ?" ||
		!reflect.DeepEqual(recorder.execArgs, []driver.NamedValue{{Ordinal: 1, Value: name}}) {
		t.Fatalf("unconditional Update = %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "DELETE FROM `test_schema`.`roles`" || len(recorder.execArgs) != 0 || recorder.execCount != 2 {
		t.Fatalf("unconditional Delete = %q %#v, executions=%d", recorder.execQuery, recorder.execArgs, recorder.execCount)
	}
}

func TestUpdateRejectsEmptyFields(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	if _, err := engine.Fields().Update(context.Background(), engineTestModel{}, false); err == nil {
		t.Fatal("Update() accepted an empty selected field set")
	}
	if recorder.execCount != 0 {
		t.Fatal("empty Update reached the driver")
	}
}

func TestContextMethods(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := engine.ExecuteContext(canceled, "UPDATE roles SET name = ?", "name"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteContext() error = %v, want context.Canceled", err)
	}
	var roles []engineTestModel
	if err := engine.Select(canceled, &roles); !errors.Is(err, context.Canceled) {
		t.Fatalf("Select() error = %v, want context.Canceled", err)
	}
	name := "updated"
	if _, err := engine.Where("id = ?", int64(1)).Update(canceled, engineTestModel{Name: &name}, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("Update() error = %v, want context.Canceled", err)
	}

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "test")
	err := engine.WithTransaction(ctx, func(transaction *Engine, _ ...any) error {
		_, err := transaction.ExecuteContext(ctx, "UPDATE roles SET name = ?", "name")
		return err
	})
	if err != nil {
		t.Fatalf("WithTransaction() returned an error: %v", err)
	}
	if recorder.beginCount != 1 || recorder.commitCount != 1 || recorder.rollbackCount != 0 {
		t.Fatalf("unexpected transaction counts: begin=%d commit=%d rollback=%d", recorder.beginCount, recorder.commitCount, recorder.rollbackCount)
	}
	if recorder.execCtx == nil || recorder.execCtx.Value(contextKey("request")) != "test" {
		t.Fatal("ExecuteContext() did not pass the context to the driver")
	}
}

func TestWithTransactionReusesExistingTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		engine, recorder := newEngineTestEngine(t)
		err := engine.WithTransaction(context.Background(), func(outer *Engine, _ ...any) error {
			tx := outer.tx
			return outer.WithTransaction(context.Background(), func(inner *Engine, _ ...any) error {
				if inner.tx != tx {
					t.Fatal("nested transaction did not reuse the existing transaction")
				}
				_, err := inner.ExecuteContext(context.Background(), "UPDATE roles SET name = ?", "nested")
				return err
			})
		})
		if err != nil {
			t.Fatalf("WithTransaction() returned an error: %v", err)
		}
		if recorder.beginCount != 1 || recorder.commitCount != 1 || recorder.rollbackCount != 0 {
			t.Fatalf("unexpected transaction counts: begin=%d commit=%d rollback=%d", recorder.beginCount, recorder.commitCount, recorder.rollbackCount)
		}
	})

	t.Run("error", func(t *testing.T) {
		engine, recorder := newEngineTestEngine(t)
		nestedErr := errors.New("nested transaction error")
		err := engine.WithTransaction(context.Background(), func(outer *Engine, _ ...any) error {
			return outer.WithTransaction(context.Background(), func(inner *Engine, _ ...any) error {
				return nestedErr
			})
		})
		if !errors.Is(err, nestedErr) {
			t.Fatalf("WithTransaction() error = %v, want %v", err, nestedErr)
		}
		if recorder.beginCount != 1 || recorder.commitCount != 0 || recorder.rollbackCount != 1 {
			t.Fatalf("unexpected transaction counts: begin=%d commit=%d rollback=%d", recorder.beginCount, recorder.commitCount, recorder.rollbackCount)
		}
	})
}

func TestWithTransactionKeepsQueryStateIsolated(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	base := engine.Where("id = ?", int64(1))
	base.whereSQL = append(make([]string, 0, 4), base.whereSQL...)
	base.args = append(make([]any, 0, 4), base.args...)

	err := base.WithTransaction(context.Background(), func(transaction *Engine, _ ...any) error {
		branch := transaction.Where("name = ?", "transaction")
		if !reflect.DeepEqual(branch.whereSQL, []string{"id = ?", "name = ?"}) ||
			!reflect.DeepEqual(branch.args, []any{int64(1), "transaction"}) {
			t.Fatalf("transaction query state is incorrect: whereSQL=%#v args=%#v", branch.whereSQL, branch.args)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTransaction() returned an error: %v", err)
	}
	if !reflect.DeepEqual(base.whereSQL, []string{"id = ?"}) || !reflect.DeepEqual(base.args, []any{int64(1)}) {
		t.Fatalf("transaction modified the source query state: whereSQL=%#v args=%#v", base.whereSQL, base.args)
	}
}

func TestBasicMethodsKeepNonContextAPI(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	if _, err := engine.Execute("UPDATE roles SET name = ?", "admin"); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE roles SET name = ?" || recorder.execArgs[0].Value != "admin" {
		t.Fatalf("Execute() changed raw SQL or arguments: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	recorder.columns = []string{"id"}
	recorder.rows = [][]driver.Value{{int64(1)}}
	var id int64
	if err := engine.QueryRow("SELECT id FROM roles").Scan(&id); err != nil || id != 1 {
		t.Fatalf("QueryRow().Scan() = (%d, %v)", id, err)
	}
	rows, err := engine.Query("SELECT id FROM roles")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("Query() returned no row")
	}
	if err = rows.Scan(&id); err != nil || id != 1 {
		t.Fatalf("Query().Scan() = (%d, %v)", id, err)
	}
}

func TestPageCountsMatchingRows(t *testing.T) {
	tests := []struct {
		name     string
		page     int64
		pageSize int64
		limitSQL string
	}{
		{name: "second page", page: 2, pageSize: 10, limitSQL: " LIMIT 10 OFFSET 10"},
		{name: "default page and size", page: 0, pageSize: 0, limitSQL: " LIMIT 10 OFFSET 0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			recorder.columns = []string{"count"}
			recorder.rows = [][]driver.Value{{int64(25)}}
			query := engine.Where("status = ?", "enabled")
			total, pages, err := query.Page(context.Background(), test.page, test.pageSize)
			if err != nil || total != 25 || pages != 3 || query.limitSQL != test.limitSQL {
				t.Fatalf("Page() = (%d, %d, %v), limit = %q", total, pages, err, query.limitSQL)
			}
			if recorder.queryQuery != "SELECT count(*) FROM `test_schema`.`roles` WHERE status = ?" ||
				len(recorder.queryArgs) != 1 || recorder.queryArgs[0].Value != "enabled" {
				t.Fatalf("Page() changed Count SQL or arguments: %q %#v", recorder.queryQuery, recorder.queryArgs)
			}
		})
	}
}

func TestSelectKeepsQueryClauseOrder(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	query := engine.Fields("Status").Where("status = ?", "enabled").GroupBy("status").OrderBy("-status")
	query.limitSQL = " LIMIT 10 OFFSET 0"
	recorder.columns = []string{"status"}
	recorder.rows = [][]driver.Value{{"enabled"}}
	var roles []engineTestModel
	if err := query.Select(context.Background(), &roles); err != nil {
		t.Fatal(err)
	}
	wantSQL := "SELECT `status` FROM `test_schema`.`roles` WHERE status = ? GROUP BY `status` ORDER BY `status` DESC LIMIT 10 OFFSET 0"
	if recorder.queryQuery != wantSQL {
		t.Fatalf("Select() SQL = %q, want %q", recorder.queryQuery, wantSQL)
	}
	if len(roles) != 1 || roles[0].Status == nil || *roles[0].Status != "enabled" {
		t.Fatalf("Select() did not map the selected field: %#v", roles)
	}
}

func TestQueryMethodsHandleNoRows(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	recorder.rows = nil
	exists, err := engine.Exists(context.Background())
	if err != nil || exists {
		t.Fatalf("Exists() = (%v, %v), want (false, nil)", exists, err)
	}
	var role engineTestModel
	if err = engine.First(context.Background(), &role); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("First() error = %v, want sql.ErrNoRows", err)
	}
	var roles []engineTestModel
	if err = engine.Select(context.Background(), &roles); err != nil || len(roles) != 0 {
		t.Fatalf("Select() = (%#v, %v), want no rows and no error", roles, err)
	}
	recorder.columns = []string{"exists"}
	recorder.rows = [][]driver.Value{{int64(1)}}
	if exists, err = engine.Exists(context.Background()); err != nil || !exists {
		t.Fatalf("Exists() = (%v, %v), want (true, nil)", exists, err)
	}
}

func TestUpdateAndDeleteKeepWhereArguments(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	query := engine.Fields("Name").Where("id = ?", int64(7))
	name := "updated"
	if _, err := query.Update(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatal(err)
	}
	wantArgs := []driver.NamedValue{{Ordinal: 1, Value: name}, {Ordinal: 2, Value: int64(7)}}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `name` = ? WHERE id = ?" ||
		!reflect.DeepEqual(recorder.execArgs, wantArgs) {
		t.Fatalf("Update() changed SQL or argument order: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := query.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "DELETE FROM `test_schema`.`roles` WHERE id = ?" ||
		len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != int64(7) {
		t.Fatalf("Delete() changed SQL or arguments: %q %#v", recorder.execQuery, recorder.execArgs)
	}
}

type engineScalarTestModel struct {
	Name string `field_name:"name" field_type:"VARCHAR(64)"`
}

func (engineScalarTestModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

type engineNullableTestModel struct {
	Name sql.NullString `field_name:"name" field_type:"VARCHAR(64)"`
}

func (engineNullableTestModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

func TestWritesKeepZeroValuesAndExplicitNull(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(engineScalarTestModel{})
	if _, err := engine.Insert(context.Background(), engineScalarTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `roles` (`name`) VALUES (?)" ||
		len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != "" {
		t.Fatalf("Insert() discarded a zero value: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(context.Background(), engineScalarTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execArgs[0].Value != "" {
		t.Fatalf("Update() discarded a zero value: %#v", recorder.execArgs)
	}
	engine.SetModel(engineNullableTestModel{})
	if _, err := engine.Insert(context.Background(), engineNullableTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("Insert() did not preserve explicit SQL NULL: %#v", recorder.execArgs)
	}
}

func TestFieldsControlWritesIncludingNull(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	name, status := "admin", "enabled"
	if _, err := engine.Fields("Name").Insert(ctx, engineTestModel{Name: &name, Status: &status}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` (`name`) VALUES (?)" ||
		!reflect.DeepEqual(recorder.execArgs, []driver.NamedValue{{Ordinal: 1, Value: name}}) {
		t.Fatalf("unselected INSERT fields reached SQL: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Fields("Name").Where("id = ?", int64(7)).Update(ctx, engineTestModel{Status: &status}, false); err != nil {
		t.Fatal(err)
	}
	want := []driver.NamedValue{{Ordinal: 1, Value: nil}, {Ordinal: 2, Value: int64(7)}}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `name` = ? WHERE id = ?" ||
		!reflect.DeepEqual(recorder.execArgs, want) {
		t.Fatalf("selected nil was not written as NULL: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	want = []driver.NamedValue{
		{Ordinal: 1, Value: nil}, {Ordinal: 2, Value: nil}, {Ordinal: 3, Value: nil}, {Ordinal: 4, Value: int64(7)},
	}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `id` = ?,`name` = ?,`status` = ? WHERE id = ?" ||
		!reflect.DeepEqual(recorder.execArgs, want) {
		t.Fatalf("default selection did not include all fields: %q %#v", recorder.execQuery, recorder.execArgs)
	}
}

func TestInsertWithEmptyFieldsPassesSQLToDriver(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	if _, err := engine.Fields().Insert(context.Background(), engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	// 模拟驱动检查 SQL 转交，不代表所有数据库接受空字段语法。
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("empty selected field set was rewritten: %q %#v", recorder.execQuery, recorder.execArgs)
	}
}

type enginePointerValuer struct{ Text string }

func (value *enginePointerValuer) Value() (driver.Value, error) {
	if value == nil {
		return "custom nil", nil
	}
	return "encoded:" + value.Text, nil
}

type enginePointerValuerModel struct {
	Name *enginePointerValuer `field_name:"name" field_type:"VARCHAR(64)"`
}

func (enginePointerValuerModel) ModelSet() *Settings {
	return &Settings{TableName: "roles"}
}

func TestWritesPreservePointerValuerAndNormalizeNil(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	engine.SetModel(enginePointerValuerModel{})
	model := enginePointerValuerModel{Name: &enginePointerValuer{Text: "admin"}}
	if _, err := engine.Insert(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != "encoded:admin" {
		t.Fatalf("pointer Valuer was lost: %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(context.Background(), model, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execArgs[0].Value != "encoded:admin" {
		t.Fatalf("UPDATE did not use pointer Valuer: %#v", recorder.execArgs)
	}
	if _, err := engine.Insert(context.Background(), enginePointerValuerModel{}, false); err != nil {
		t.Fatal(err)
	}
	if len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("nil Valuer pointer did not become NULL: %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(context.Background(), enginePointerValuerModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execArgs[0].Value != nil {
		t.Fatalf("UPDATE did not normalize nil Valuer pointer: %#v", recorder.execArgs)
	}
}

func TestSkipNilControlsInsertAndUpdate(t *testing.T) {
	for _, skipNil := range []bool{false, true} {
		name := "NULL"
		if skipNil {
			name = "skip"
		}
		t.Run(name, func(t *testing.T) {
			engine, recorder := newEngineTestEngine(t)
			ctx := context.Background()
			value := "admin"
			model := engineTestModel{Name: &value}
			if _, err := engine.Insert(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL := "INSERT INTO `test_schema`.`roles` (`id`,`name`,`status`) VALUES (?,?,?)"
			wantArgs := []driver.NamedValue{{Ordinal: 1, Value: nil}, {Ordinal: 2, Value: value}, {Ordinal: 3, Value: nil}}
			if skipNil {
				wantSQL = "INSERT INTO `test_schema`.`roles` (`name`) VALUES (?)"
				wantArgs = []driver.NamedValue{{Ordinal: 1, Value: value}}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(recorder.execArgs, wantArgs) {
				t.Fatalf("Insert(skipNil=%t) = %q %#v", skipNil, recorder.execQuery, recorder.execArgs)
			}
			query := engine.Fields("Name", "Status").Where("id = ?", int64(7))
			if _, err := query.Update(ctx, model, skipNil); err != nil {
				t.Fatal(err)
			}
			wantSQL = "UPDATE `test_schema`.`roles` SET `name` = ?,`status` = ? WHERE id = ?"
			wantArgs = []driver.NamedValue{{Ordinal: 1, Value: value}, {Ordinal: 2, Value: nil}, {Ordinal: 3, Value: int64(7)}}
			if skipNil {
				wantSQL = "UPDATE `test_schema`.`roles` SET `name` = ? WHERE id = ?"
				wantArgs = []driver.NamedValue{{Ordinal: 1, Value: value}, {Ordinal: 2, Value: int64(7)}}
			}
			if recorder.execQuery != wantSQL || !reflect.DeepEqual(recorder.execArgs, wantArgs) {
				t.Fatalf("Update(skipNil=%t) = %q %#v", skipNil, recorder.execQuery, recorder.execArgs)
			}
		})
	}
}

func TestUnconditionalUpdateRespectsFieldsAndSkipNil(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	name := "updated"
	query := engine.Fields("Name", "Status")
	if _, err := query.Update(ctx, engineTestModel{Name: &name}, true); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `name` = ?" ||
		!reflect.DeepEqual(recorder.execArgs, []driver.NamedValue{{Ordinal: 1, Value: name}}) {
		t.Fatalf("unconditional skip-nil Update = %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Fields("Status").Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `status` = ?" ||
		!reflect.DeepEqual(recorder.execArgs, []driver.NamedValue{{Ordinal: 1, Value: nil}}) {
		t.Fatalf("unconditional NULL Update = %q %#v", recorder.execQuery, recorder.execArgs)
	}
	before := recorder.execCount
	if _, err := engine.Update(ctx, engineTestModel{}, true); err == nil {
		t.Fatal("all-nil skipped Update did not return the empty-fields error")
	}
	if recorder.execCount != before {
		t.Fatal("empty unconditional Update reached the driver")
	}
}

func TestSkipNilKeepsExistingEmptyWriteRules(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	if _, err := engine.Insert(ctx, engineTestModel{}, true); err != nil {
		t.Fatal(err)
	}
	// 模拟驱动只验证转交的 SQL，不自动补充默认行插入。
	if recorder.execQuery != "INSERT INTO `test_schema`.`roles` () VALUES ()" || len(recorder.execArgs) != 0 {
		t.Fatalf("all-nil Insert was rewritten: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	before := recorder.execCount
	query := engine.Fields("Status").Where("id = ?", int64(7))
	if _, err := query.Update(ctx, engineTestModel{}, true); err == nil {
		t.Fatal("all-nil skipped Update did not return the empty-fields error")
	}
	if recorder.execCount != before {
		t.Fatal("all-nil skipped Update reached the driver")
	}
	if _, err := query.Update(ctx, engineTestModel{}, false); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE `test_schema`.`roles` SET `status` = ? WHERE id = ?" ||
		len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != nil {
		t.Fatalf("explicit NULL Update = %q %#v", recorder.execQuery, recorder.execArgs)
	}
}

type engineSkipNilZeroModel struct {
	Name   string `field_name:"name" field_type:"VARCHAR(64)"`
	Count  int64  `field_name:"count" field_type:"BIGINT"`
	Active bool   `field_name:"active" field_type:"BOOLEAN"`
}

func (engineSkipNilZeroModel) ModelSet() *Settings { return &Settings{TableName: "roles"} }

func TestSkipNilDoesNotSkipZeroValuesOrValuerNull(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	ctx := context.Background()
	engine.SetModel(engineSkipNilZeroModel{})
	want := []driver.NamedValue{{Ordinal: 1, Value: ""}, {Ordinal: 2, Value: int64(0)}, {Ordinal: 3, Value: false}}
	if _, err := engine.Insert(ctx, engineSkipNilZeroModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recorder.execArgs, want) {
		t.Fatalf("skipNil discarded a zero value: %#v", recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineSkipNilZeroModel{}, true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recorder.execArgs[:3], want) {
		t.Fatalf("skipNil discarded an update zero value: %#v", recorder.execArgs)
	}
	engine.SetModel(engineNullableTestModel{})
	if _, err := engine.Insert(ctx, engineNullableTestModel{}, true); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "INSERT INTO `roles` (`name`) VALUES (?)" || len(recorder.execArgs) != 1 || recorder.execArgs[0].Value != nil {
		t.Fatalf("skipNil discarded a Valuer-produced NULL: %q %#v", recorder.execQuery, recorder.execArgs)
	}
	if _, err := engine.Where("id = ?", int64(7)).Update(ctx, engineNullableTestModel{}, true); err != nil {
		t.Fatal(err)
	}
	if recorder.execQuery != "UPDATE `roles` SET `name` = ? WHERE id = ?" || len(recorder.execArgs) != 2 || recorder.execArgs[0].Value != nil {
		t.Fatalf("skipNil discarded an update Valuer-produced NULL: %q %#v", recorder.execQuery, recorder.execArgs)
	}
}

func TestWithTransactionRollsBackBeforePropagatingPanic(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	wantPanic := errors.New("transaction callback panic")
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = engine.WithTransaction(context.Background(), func(*Engine, ...any) error {
			panic(wantPanic)
		})
	}()
	if recovered != wantPanic || recorder.rollbackCount != 1 || recorder.commitCount != 0 {
		t.Fatalf("transaction did not roll back and propagate the original panic: panic=%v rollback=%d commit=%d",
			recovered, recorder.rollbackCount, recorder.commitCount)
	}
}

func TestMapResultCallbacksKeepTargetsIsolated(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	var left, right []map[string]any
	writeLeft, err := db.PrepareSliceResult(engine.model, &left, engine.fields)
	if err != nil {
		t.Fatal(err)
	}
	writeRight, err := db.PrepareSliceResult(engine.model, &right, engine.fields)
	if err != nil {
		t.Fatal(err)
	}
	first, second := map[string]any{}, map[string]any{}
	writeFirst, err := db.PrepareSingleResult(engine.model, first, engine.fields)
	if err != nil {
		t.Fatal(err)
	}
	writeSecond, err := db.PrepareSingleResult(engine.model, second, engine.fields)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := engine.Query("SELECT id, name, status FROM roles")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected one row")
	}
	for _, writeResult := range []db.ResultWriter{writeLeft, writeRight, writeFirst, writeSecond} {
		if err := writeResult(rows); err != nil {
			t.Fatal(err)
		}
	}
	if len(left) != 1 || len(right) != 1 || len(first) != 3 || len(second) != 3 {
		t.Fatalf("query targets interfered: left=%#v right=%#v first=%#v second=%#v", left, right, first, second)
	}
}

func TestMapResultKeepsRowsIndependent(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"name"}
	recorder.rows = [][]driver.Value{{"first"}, {"second"}}
	var results []map[string]*string
	if err := engine.Fields("Name").Select(context.Background(), &results); err != nil {
		t.Fatalf("Select() returned an error: %v", err)
	}
	if len(results) != 2 || results[0]["Name"] == nil || results[1]["Name"] == nil {
		t.Fatalf("unexpected map results: %#v", results)
	}
	if *results[0]["Name"] != "first" || *results[1]["Name"] != "second" ||
		results[0]["Name"] == results[1]["Name"] {
		t.Fatalf("map rows share scan targets or contain incorrect values: %#v", results)
	}
}

func TestMapResultsUseStructFieldNames(t *testing.T) {
	original := GetColumnParser()
	t.Cleanup(func() { SetColumnParser(original) })
	SetColumnParser(db.NewColumnParser("column", "sql_type"))

	engine, recorder := newEngineTestEngine(t)
	query := engine.SetModel(columnParserTestModel{}).Fields("Name", "ID")
	recorder.columns = []string{"display_name", "role_id"}
	recorder.rows = [][]driver.Value{{"mapped", int64(7)}}

	var results []map[string]any
	if err := query.Select(context.Background(), &results); err != nil {
		t.Fatalf("Select() returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Select() returned %d rows, want 1", len(results))
	}
	first := map[string]any{}
	if err := query.First(context.Background(), first); err != nil {
		t.Fatalf("First() returned an error: %v", err)
	}
	for _, result := range []map[string]any{results[0], first} {
		name, nameOK := result["Name"].(*string)
		id, idOK := result["ID"].(*int64)
		if len(result) != 2 || !nameOK || name == nil || *name != "mapped" || !idOK || id == nil || *id != 7 {
			t.Fatalf("map did not use the selected struct field names: %#v", result)
		}
	}
	wantSQL := "SELECT `display_name`,`role_id` FROM `test_schema`.`field_parser_roles`"
	if recorder.queryQuery != wantSQL {
		t.Fatalf("query columns changed: got %q, want %q", recorder.queryQuery, wantSQL)
	}
}

func TestMapResultPreservesTargetsOnScanError(t *testing.T) {
	engine, recorder := newEngineTestEngine(t)
	recorder.columns = []string{"id"}
	recorder.rows = [][]driver.Value{{"invalid integer"}}
	var results []map[string]any
	if err := engine.Fields("ID").Select(context.Background(), &results); err == nil {
		t.Fatal("Select() accepted an invalid scan value")
	}
	if len(results) != 0 {
		t.Fatalf("Select() appended a result after scan failure: %#v", results)
	}
	result := map[string]any{"existing": true}
	if err := engine.Fields("ID").First(context.Background(), result); err == nil {
		t.Fatal("First() accepted an invalid scan value")
	}
	if len(result) != 1 || result["existing"] != true {
		t.Fatalf("First() wrote map entries after scan failure: %#v", result)
	}
}

func TestMapResultRejectsUnsupportedTargets(t *testing.T) {
	engine, _ := newEngineTestEngine(t)
	var mapPointers []*map[string]any
	if err := engine.Select(context.Background(), &mapPointers); err == nil {
		t.Fatal("Select() accepted map pointer elements")
	}
	var values []string
	if err := engine.Select(context.Background(), &values); err == nil {
		t.Fatal("Select() accepted unsupported scalar elements")
	}
	var value string
	if err := engine.First(context.Background(), &value); err == nil {
		t.Fatal("First() accepted an unsupported scalar pointer")
	}
}
