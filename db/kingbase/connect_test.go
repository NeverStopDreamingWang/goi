package kingbase

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v2"
	"github.com/NeverStopDreamingWang/goi/v2/db"
	"github.com/NeverStopDreamingWang/goi/v2/db/mysql"
	"github.com/NeverStopDreamingWang/goi/v2/db/oracle"
	"github.com/NeverStopDreamingWang/goi/v2/db/postgres"
	"github.com/NeverStopDreamingWang/goi/v2/db/sqlserver"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

type routeRecorder struct {
	pool         *sql.DB
	poolCalls    int
	driverEngine string
	execQuery    string
	execArgs     []driver.NamedValue
	query        string
	queryArgs    []driver.NamedValue
}

type routeConnector struct{ recorder *routeRecorder }
type routeDriver struct{}
type routeConn struct{ recorder *routeRecorder }
type routeRows struct{ read bool }

func (connector routeConnector) Connect(context.Context) (driver.Conn, error) {
	return &routeConn{recorder: connector.recorder}, nil
}
func (routeConnector) Driver() driver.Driver { return routeDriver{} }
func (routeDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use routeConnector")
}
func (*routeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("Prepare is not used")
}
func (*routeConn) Close() error { return nil }
func (*routeConn) Begin() (driver.Tx, error) {
	return nil, errors.New("Begin is not used")
}
func (connection *routeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	connection.recorder.execQuery = query
	connection.recorder.execArgs = append([]driver.NamedValue(nil), args...)
	return driver.RowsAffected(1), nil
}
func (connection *routeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	connection.recorder.query = query
	connection.recorder.queryArgs = append([]driver.NamedValue(nil), args...)
	return &routeRows{}, nil
}
func (*routeRows) Columns() []string { return []string{"value"} }
func (*routeRows) Close() error      { return nil }
func (rows *routeRows) Next(values []driver.Value) error {
	if rows.read {
		return io.EOF
	}
	rows.read = true
	values[0] = int64(7)
	return nil
}

func newRouteManager(t *testing.T, config goi.DatabaseConfig) (*goi.Database, *routeRecorder) {
	t.Helper()
	recorder := &routeRecorder{}
	recorder.pool = sql.OpenDB(routeConnector{recorder: recorder})
	t.Cleanup(func() { _ = recorder.pool.Close() })
	manager := &goi.Database{
		Engine: driverName,
		Config: config,
		Connect: func(engine string) *sql.DB {
			recorder.poolCalls++
			recorder.driverEngine = engine
			return recorder.pool
		},
	}
	return manager, recorder
}

func registerRouteManager(t *testing.T, name string, manager *goi.Database) {
	t.Helper()
	old, existed := goi.Settings.Databases[name]
	goi.Settings.Databases[name] = manager
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[name] = old
		} else {
			delete(goi.Settings.Databases, name)
		}
	})
}

func routePanic(t *testing.T, callback func()) any {
	t.Helper()
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		callback()
	}()
	if recovered == nil {
		t.Fatal("expected panic")
	}
	return recovered
}

type routeFields struct {
	ID   int64   `field_name:"id" field_type:"BIGINT" column:"record_id" sql_type:"BIGINT"`
	Name *string `field_name:"name" field_type:"VARCHAR(32)" column:"display_name" sql_type:"VARCHAR(32)"`
}
type mysqlRouteModel routeFields
type postgresRouteModel routeFields
type oracleRouteModel routeFields
type sqlserverRouteModel routeFields

func (mysqlRouteModel) ModelSet() *mysql.Settings {
	return &mysql.Settings{Schema: "app", TableName: "roles"}
}
func (postgresRouteModel) ModelSet() *postgres.Settings {
	return &postgres.Settings{Schema: "app", TableName: "roles"}
}
func (oracleRouteModel) ModelSet() *oracle.Settings {
	return &oracle.Settings{Schema: "app", TableName: "roles"}
}
func (sqlserverRouteModel) ModelSet() *sqlserver.Settings {
	return &sqlserver.Settings{Schema: "app", TableName: "roles"}
}

type routeCase struct {
	name    string
	mode    CompatibilityMode
	want    reflect.Type
	left    string
	right   string
	connect func(string) db.Engine
}

func routeCases() []routeCase {
	return []routeCase{
		{"postgres", ModePostgreSQL, reflect.TypeOf((*postgres.Engine)(nil)), "\"", "\"",
			func(name string) db.Engine { return db.Connect[*postgres.Engine](name) }},
		{"mysql", ModeMySQL, reflect.TypeOf((*mysql.Engine)(nil)), "`", "`",
			func(name string) db.Engine { return db.Connect[*mysql.Engine](name) }},
		{"oracle", ModeOracle, reflect.TypeOf((*oracle.Engine)(nil)), "\"", "\"",
			func(name string) db.Engine { return db.Connect[*oracle.Engine](name) }},
		{"sqlserver", ModeSQLServer, reflect.TypeOf((*sqlserver.Engine)(nil)), "[", "]",
			func(name string) db.Engine { return db.Connect[*sqlserver.Engine](name) }},
	}
}

func routePool(engine db.Engine) *sql.DB {
	switch value := engine.(type) {
	case *mysql.Engine:
		return value.DB
	case *postgres.Engine:
		return value.DB
	case *oracle.Engine:
		return value.DB
	case *sqlserver.Engine:
		return value.DB
	default:
		return nil
	}
}

func writeRoute(engine db.Engine, placeholder string) error {
	ctx := context.Background()
	name := "admin"
	where := "id = " + placeholder
	var err error
	switch value := engine.(type) {
	case *mysql.Engine:
		_, err = value.SetModel(mysqlRouteModel{}).Fields("Name").Where(where, int64(7)).
			Update(ctx, mysqlRouteModel{Name: &name}, false)
	case *postgres.Engine:
		_, err = value.SetModel(postgresRouteModel{}).Fields("Name").Where(where, int64(7)).
			Update(ctx, postgresRouteModel{Name: &name}, false)
	case *oracle.Engine:
		_, err = value.SetModel(oracleRouteModel{}).Fields("Name").Where(where, int64(7)).
			Update(ctx, oracleRouteModel{Name: &name}, false)
	case *sqlserver.Engine:
		_, err = value.SetModel(sqlserverRouteModel{}).Fields("Name").Where(where, int64(7)).
			Update(ctx, sqlserverRouteModel{Name: &name}, false)
	default:
		return fmt.Errorf("unexpected engine %T", engine)
	}
	return err
}

func TestCompatibilityRoutesReusePoolAndKeepKingbaseDriver(t *testing.T) {
	for _, test := range routeCases() {
		t.Run(test.name, func(t *testing.T) {
			manager, recorder := newRouteManager(t, Config{Mode: test.mode})
			name := t.Name()
			registerRouteManager(t, name, manager)
			first, second := test.connect(name), Connect(name)
			if reflect.TypeOf(first) != test.want || reflect.TypeOf(second) != test.want || first == second {
				t.Fatalf("routes returned %T and %T, want distinct %v", first, second, test.want)
			}
			if first.Name() != name || routePool(first) != recorder.pool || routePool(second) != recorder.pool ||
				recorder.poolCalls != 1 || recorder.driverEngine != driverName || manager.Engine != driverName {
				t.Fatalf("physical pool changed: calls=%d, driver=%q", recorder.poolCalls, recorder.driverEngine)
			}
			if err := writeRoute(first, "?"); err != nil {
				t.Fatal(err)
			}
			quote := func(value string) string { return test.left + value + test.right }
			wantSQL := "UPDATE " + quote("app") + "." + quote("roles") +
				" SET " + quote("name") + " = $1 WHERE id = $2"
			if recorder.execQuery != wantSQL || len(recorder.execArgs) != 2 ||
				recorder.execArgs[0].Value != "admin" || recorder.execArgs[1].Value != int64(7) {
				t.Fatalf("write SQL=%q, args=%v", recorder.execQuery, recorder.execArgs)
			}
			var value int64
			if err := second.QueryRow("SELECT ?", int64(7)).Scan(&value); err != nil {
				t.Fatal(err)
			}
			if value != 7 || recorder.query != "SELECT $1" || len(recorder.queryArgs) != 1 {
				t.Fatalf("query SQL=%q, args=%v, value=%d", recorder.query, recorder.queryArgs, value)
			}
		})
	}
}

func TestCompatibilityModeValues(t *testing.T) {
	for _, test := range []struct {
		mode CompatibilityMode
		want string
	}{
		{ModePostgreSQL, "postgres"},
		{ModeOracle, "oracle"},
		{ModeMySQL, "mysql"},
		{ModeSQLServer, "sqlserver"},
	} {
		if string(test.mode) != test.want {
			t.Errorf("mode = %q, want driver name %q", test.mode, test.want)
		}
	}
}

func TestConfigFormsAndDefaultMode(t *testing.T) {
	var typedNil *Config
	tests := []struct {
		name   string
		config goi.DatabaseConfig
		want   reflect.Type
	}{
		{"nil", nil, reflect.TypeOf((*postgres.Engine)(nil))},
		{"value", Config{}, reflect.TypeOf((*postgres.Engine)(nil))},
		{"postgres_string", Config{Mode: "postgres"}, reflect.TypeOf((*postgres.Engine)(nil))},
		{"pointer", &Config{}, reflect.TypeOf((*postgres.Engine)(nil))},
		{"typed_nil", typedNil, reflect.TypeOf((*postgres.Engine)(nil))},
		{"pointer_mode", &Config{Mode: ModeOracle}, reflect.TypeOf((*oracle.Engine)(nil))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, recorder := newRouteManager(t, test.config)
			engine := ConnectDatabase("config", manager)
			if reflect.TypeOf(engine) != test.want || recorder.poolCalls != 1 {
				t.Fatalf("engine=%T, calls=%d", engine, recorder.poolCalls)
			}
		})
	}
}

type routePlaceholderParser struct{}

func (routePlaceholderParser) Placeholder() string { return ":v" }
func (routePlaceholderParser) Parse(query string, args ...any) string {
	for index := range args {
		query = strings.Replace(query, ":v", fmt.Sprintf("@%d", index+1), 1)
	}
	return query
}

func nativeRouteOptions() []db.EngineOptions {
	return []db.EngineOptions{
		{PlaceholderParser: mysql.GetPlaceholderParser(), IdentifierQuoter: mysql.GetIdentifierQuoter(), ColumnParser: mysql.GetColumnParser()},
		{PlaceholderParser: postgres.GetPlaceholderParser(), IdentifierQuoter: postgres.GetIdentifierQuoter(), ColumnParser: postgres.GetColumnParser()},
		{PlaceholderParser: oracle.GetPlaceholderParser(), IdentifierQuoter: oracle.GetIdentifierQuoter(), ColumnParser: oracle.GetColumnParser()},
		{PlaceholderParser: sqlserver.GetPlaceholderParser(), IdentifierQuoter: sqlserver.GetIdentifierQuoter(), ColumnParser: sqlserver.GetColumnParser()},
	}
}

func TestConfigStrategiesOverrideLocallyAndAreSnapshotted(t *testing.T) {
	defaults := nativeRouteOptions()
	for _, test := range routeCases() {
		t.Run(test.name, func(t *testing.T) {
			config := &Config{
				Mode: test.mode,
				EngineOptions: db.EngineOptions{
					PlaceholderParser: routePlaceholderParser{},
					IdentifierQuoter:  db.NewIdentifierQuoter("<", ">"),
					ColumnParser:      db.NewColumnParser("column", "sql_type"),
				},
			}
			manager, recorder := newRouteManager(t, config)
			engine := ConnectDatabase("custom", manager)
			config.EngineOptions = db.EngineOptions{}
			if err := writeRoute(engine, ":v"); err != nil {
				t.Fatal(err)
			}
			if recorder.execQuery != "UPDATE <app>.<roles> SET <display_name> = @1 WHERE id = @2" {
				t.Fatalf("custom SQL=%q", recorder.execQuery)
			}
			fresh := ConnectDatabase("default", manager)
			if err := writeRoute(fresh, "?"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(recorder.execQuery, " = $1 WHERE id = $2") ||
				strings.Contains(recorder.execQuery, "<display_name>") || recorder.poolCalls != 1 {
				t.Fatalf("fresh SQL=%q, pool calls=%d", recorder.execQuery, recorder.poolCalls)
			}
			if !reflect.DeepEqual(nativeRouteOptions(), defaults) {
				t.Fatal("Kingbase options changed native package defaults")
			}
		})
	}
}

func TestCompatibilityRoutesInheritSharedColumnParser(t *testing.T) {
	old := db.GetColumnParser()
	t.Cleanup(func() { db.SetColumnParser(old) })
	for _, test := range routeCases() {
		t.Run(test.name, func(t *testing.T) {
			db.SetColumnParser(db.NewColumnParser("column", "sql_type"))
			manager, recorder := newRouteManager(t, Config{Mode: test.mode})
			engine := ConnectDatabase("shared", manager)
			quote := func(value string) string { return test.left + value + test.right }
			checkColumn := func(engine db.Engine, column string) {
				t.Helper()
				if err := writeRoute(engine, "?"); err != nil {
					t.Fatal(err)
				}
				want := "UPDATE " + quote("app") + "." + quote("roles") +
					" SET " + quote(column) + " = $1 WHERE id = $2"
				if recorder.execQuery != want {
					t.Fatalf("SQL=%q, want %q", recorder.execQuery, want)
				}
			}
			checkColumn(engine, "display_name")
			db.SetColumnParser(nil)
			checkColumn(engine, "display_name")
			checkColumn(ConnectDatabase("unchanged", manager), "display_name")
			db.SetColumnParser(db.NewColumnParser("field_name", "field_type"))
			checkColumn(engine, "display_name")
			checkColumn(ConnectDatabase("default", manager), "name")
		})
	}
}

type routeConfigProvider func(target any) error

func (provider routeConfigProvider) GetConfig(target any) error { return provider(target) }

func TestCustomConfigProviderSelectsEachCompatibilityMode(t *testing.T) {
	for _, test := range routeCases() {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			provider := routeConfigProvider(func(target any) error {
				calls++
				config, ok := target.(*Config)
				if !ok || config == nil {
					return fmt.Errorf("unexpected config target %T", target)
				}
				*config = Config{Mode: test.mode}
				return nil
			})
			manager, recorder := newRouteManager(t, provider)
			engine := ConnectDatabase("custom_provider", manager)
			if reflect.TypeOf(engine) != test.want || calls != 1 || recorder.poolCalls != 1 {
				t.Fatalf("engine=%T, config calls=%d, pool calls=%d", engine, calls, recorder.poolCalls)
			}
		})
	}
}

func TestConfigProviderErrorFailsBeforeOpeningPool(t *testing.T) {
	sentinel := errors.New("config provider failed")
	calls := 0
	provider := routeConfigProvider(func(any) error {
		calls++
		return fmt.Errorf("load Kingbase config: %w", sentinel)
	})
	manager, recorder := newRouteManager(t, provider)
	recovered := routePanic(t, func() { ConnectDatabase("invalid", manager) })
	err, ok := recovered.(error)
	if !ok || !errors.Is(err, sentinel) {
		t.Fatalf("provider error was not preserved: %v", recovered)
	}
	if calls != 1 || recorder.poolCalls != 0 {
		t.Fatalf("config calls=%d, pool calls=%d", calls, recorder.poolCalls)
	}
}

func TestUnknownModeFailsBeforeOpeningPool(t *testing.T) {
	manager, recorder := newRouteManager(t, Config{Mode: "unknown"})
	recovered := routePanic(t, func() { ConnectDatabase("invalid", manager) })
	if !strings.Contains(fmt.Sprint(recovered), "unknown") || recorder.poolCalls != 0 {
		t.Fatalf("panic=%v, pool calls=%d", recovered, recorder.poolCalls)
	}
}

func TestConfigGetConfigCopiesWithoutChangingSource(t *testing.T) {
	placeholder := &routePlaceholderParser{}
	quoter := db.NewIdentifierQuoter("<", ">")
	column := db.NewColumnParser("column", "sql_type")
	source := Config{
		Mode: ModeMySQL,
		EngineOptions: db.EngineOptions{
			PlaceholderParser: placeholder,
			IdentifierQuoter:  &quoter,
			ColumnParser:      &column,
		},
	}
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
				t.Fatalf("config copy lost values or strategy references: %#v", target)
			}
			target.Mode = ModeOracle
			target.EngineOptions = db.EngineOptions{}
			if !reflect.DeepEqual(source, original) {
				t.Fatalf("changing target modified the source: %#v", source)
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
			err := (Config{Mode: ModeMySQL}).GetConfig(test.target)
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

func TestConnectRequiresMatchingRegisteredConfig(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		name := t.Name()
		old, existed := goi.Settings.Databases[name]
		delete(goi.Settings.Databases, name)
		t.Cleanup(func() {
			if existed {
				goi.Settings.Databases[name] = old
			}
		})
		routePanic(t, func() { Connect(name) })
	})
	t.Run("mismatch", func(t *testing.T) {
		manager, recorder := newRouteManager(t, nil)
		manager.Engine = "mysql"
		registerRouteManager(t, t.Name(), manager)
		routePanic(t, func() { Connect(t.Name()) })
		if recorder.poolCalls != 0 {
			t.Fatal("mismatched Connect opened the pool")
		}
	})
	t.Run("direct_provider", func(t *testing.T) {
		manager, recorder := newRouteManager(t, nil)
		manager.Engine = "custom-provider"
		if _, ok := ConnectDatabase("direct", manager).(*postgres.Engine); !ok ||
			recorder.driverEngine != "custom-provider" {
			t.Fatal("direct constructor checked the provider engine")
		}
	})
}

func TestGenericConnectRejectsWrongReturnedType(t *testing.T) {
	manager, _ := newRouteManager(t, Config{Mode: ModeMySQL})
	registerRouteManager(t, t.Name(), manager)
	recovered := routePanic(t, func() { db.Connect[*postgres.Engine](t.Name()) })
	if message := fmt.Sprint(recovered); !strings.Contains(message, "postgres.Engine") ||
		!strings.Contains(message, "mysql.Engine") {
		t.Fatalf("unexpected type assertion panic: %v", recovered)
	}
}

func TestGokbPlaceholderParserBoundaries(t *testing.T) {
	parser := gokbPlaceholderParser{}
	if parser.Placeholder() != "?" {
		t.Fatalf("ORM placeholder=%q", parser.Placeholder())
	}
	tests := []struct {
		name  string
		query string
		args  []any
		want  string
	}{
		{"empty", "", []any{1}, ""},
		{"no_args", "SELECT ?", nil, "SELECT ?"},
		{"native", "SELECT $1", []any{1}, "SELECT $1"},
		{"sequential", "SELECT ? WHERE id = ?", []any{"admin", 7}, "SELECT $1 WHERE id = $2"},
		{"less_args", "SELECT ?, ?", []any{1}, "SELECT $1, ?"},
		{"more_args", "SELECT ?", []any{1, 2}, "SELECT $1"},
		{"adjacent", "??", []any{1, 2}, "$1$2"},
		{"unicode", "SELECT '中文', ?", []any{1}, "SELECT '中文', $1"},
		{"raw_text", "SELECT '?' -- ?\nWHERE id = ?", []any{1, 2, 3}, "SELECT '$1' -- $2\nWHERE id = $3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parser.Parse(test.query, test.args...); got != test.want {
				t.Fatalf("Parse()=%q, want %q", got, test.want)
			}
		})
	}
}
