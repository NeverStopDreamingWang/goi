package seaboxsql

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

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/mysql"
	"github.com/NeverStopDreamingWang/goi/v3/db/oracle"
	"github.com/NeverStopDreamingWang/goi/v3/db/postgres"
)

type routeRecorder struct {
	pool         *sql.DB
	poolCalls    int
	driverEngine string
	query        string
	args         []driver.NamedValue
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
	connection.recorder.query = query
	connection.recorder.args = append([]driver.NamedValue(nil), args...)
	return driver.RowsAffected(1), nil
}
func (connection *routeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	connection.recorder.query = query
	connection.recorder.args = append([]driver.NamedValue(nil), args...)
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
	return &goi.Database{
		Engine: driverName,
		Config: config,
		Connect: func(engine string) *sql.DB {
			recorder.poolCalls++
			recorder.driverEngine = engine
			return recorder.pool
		},
	}, recorder
}

func registerRouteManager(t *testing.T, manager *goi.Database) {
	t.Helper()
	old, existed := goi.Settings.Databases[t.Name()]
	goi.Settings.Databases[t.Name()] = manager
	t.Cleanup(func() {
		if existed {
			goi.Settings.Databases[t.Name()] = old
		} else {
			delete(goi.Settings.Databases, t.Name())
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

func (mysqlRouteModel) ModelSet() *mysql.Settings {
	return &mysql.Settings{Schema: "app", TableName: "roles"}
}
func (postgresRouteModel) ModelSet() *postgres.Settings {
	return &postgres.Settings{Schema: "app", TableName: "roles"}
}
func (oracleRouteModel) ModelSet() *oracle.Settings {
	return &oracle.Settings{Schema: "app", TableName: "roles"}
}

func writeRoute(engine db.Engine, placeholder string) error {
	name := "admin"
	ctx, where := context.Background(), "id = "+placeholder
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
	default:
		return fmt.Errorf("unexpected engine %T", engine)
	}
	return err
}

type routeCase struct {
	mode    CompatibilityMode
	want    reflect.Type
	quote   string
	connect func(string) db.Engine
}

func routeDriverPlaceholder(mode CompatibilityMode, index int) string {
	switch mode {
	case ModeMySQL:
		return "?"
	case ModeOracle:
		return fmt.Sprintf(":%d", index)
	default:
		return fmt.Sprintf("$%d", index)
	}
}

func routeCases() []routeCase {
	return []routeCase{
		{ModePostgreSQL, reflect.TypeOf((*postgres.Engine)(nil)), `"`,
			func(name string) db.Engine { return db.Connect[*postgres.Engine](name) }},
		{ModeMySQL, reflect.TypeOf((*mysql.Engine)(nil)), "`",
			func(name string) db.Engine { return db.Connect[*mysql.Engine](name) }},
		{ModeOracle, reflect.TypeOf((*oracle.Engine)(nil)), `"`,
			func(name string) db.Engine { return db.Connect[*oracle.Engine](name) }},
	}
}

type routeConfigProvider func(any) error

func (provider routeConfigProvider) GetConfig(target any) error { return provider(target) }

func TestCompatibilityRoutesKeepPhysicalDriverAndPool(t *testing.T) {
	for _, test := range routeCases() {
		t.Run(string(test.mode), func(t *testing.T) {
			configCalls := 0
			manager, recorder := newRouteManager(t, routeConfigProvider(func(target any) error {
				configCalls++
				return (Config{Mode: test.mode}).GetConfig(target)
			}))
			registerRouteManager(t, manager)
			first, second := test.connect(t.Name()), Connect(t.Name())
			if reflect.TypeOf(first) != test.want || reflect.TypeOf(second) != test.want || first == second {
				t.Fatalf("returned %T and %T, want distinct %v", first, second, test.want)
			}
			if first.Name() != t.Name() || configCalls != 2 || recorder.poolCalls != 1 ||
				recorder.driverEngine != driverName || manager.Engine != driverName {
				t.Fatalf("config calls=%d, pool calls=%d, driver=%q", configCalls, recorder.poolCalls, recorder.driverEngine)
			}
			if err := writeRoute(first, "?"); err != nil {
				t.Fatal(err)
			}
			quote := func(value string) string { return test.quote + value + test.quote }
			want := "UPDATE " + quote("app") + "." + quote("roles") + " SET " + quote("name") + " = " + routeDriverPlaceholder(test.mode, 1) + " WHERE id = " + routeDriverPlaceholder(test.mode, 2)
			if recorder.query != want || len(recorder.args) != 2 ||
				recorder.args[0].Value != "admin" || recorder.args[1].Value != int64(7) {
				t.Fatalf("SQL=%q, args=%v", recorder.query, recorder.args)
			}
			var value int64
			if err := second.QueryRow("SELECT ?", int64(7)).Scan(&value); err != nil {
				t.Fatal(err)
			}
			if recorder.query != "SELECT "+routeDriverPlaceholder(test.mode, 1) || value != 7 {
				t.Fatalf("query=%q, value=%d", recorder.query, value)
			}
		})
	}
}

func TestMySQLCompatibilityExpandsWhereInParameters(t *testing.T) {
	manager, recorder := newRouteManager(t, Config{Mode: ModeMySQL})
	registerRouteManager(t, manager)
	engine := db.Connect[*mysql.Engine](t.Name())
	name := "admin"
	_, err := engine.SetModel(mysqlRouteModel{}).Fields("Name").Where("id IN ?", []int64{7, 8}).
		Update(context.Background(), mysqlRouteModel{Name: &name}, false)
	if err != nil {
		t.Fatal(err)
	}
	const wantSQL = "UPDATE `app`.`roles` SET `name` = ? WHERE id IN (?,?)"
	wantArgs := []driver.NamedValue{
		{Ordinal: 1, Value: "admin"}, {Ordinal: 2, Value: int64(7)}, {Ordinal: 3, Value: int64(8)},
	}
	if recorder.query != wantSQL || !reflect.DeepEqual(recorder.args, wantArgs) {
		t.Fatalf("write SQL=%q, args=%#v, want SQL=%q args=%#v", recorder.query, recorder.args, wantSQL, wantArgs)
	}
}

func TestConfigFormsAndDefaultMode(t *testing.T) {
	var typedNil *Config
	for _, config := range []goi.DatabaseConfig{nil, Config{}, &Config{}, typedNil} {
		manager, recorder := newRouteManager(t, config)
		if engine := ConnectDatabase("default", manager); reflect.TypeOf(engine) != reflect.TypeOf((*postgres.Engine)(nil)) || recorder.poolCalls != 1 {
			t.Fatalf("config %T returned %T, pool calls=%d", config, engine, recorder.poolCalls)
		}
	}
	manager, _ := newRouteManager(t, &Config{Mode: ModeOracle})
	if _, ok := ConnectDatabase("pointer", manager).(*oracle.Engine); !ok {
		t.Fatal("pointer config did not select Oracle")
	}
}

type routePlaceholderParser struct{}

func (routePlaceholderParser) Placeholder() string { return ":v" }
func (routePlaceholderParser) Parse(query string, args ...any) (string, []any) {
	for index := range args {
		query = strings.Replace(query, ":v", fmt.Sprintf("@%d", index+1), 1)
	}
	return query, args
}

func nativeRouteOptions() []db.EngineOptions {
	return []db.EngineOptions{
		{PlaceholderParser: mysql.GetPlaceholderParser(), IdentifierQuoter: mysql.GetIdentifierQuoter(), ColumnParser: mysql.GetColumnParser()},
		{PlaceholderParser: postgres.GetPlaceholderParser(), IdentifierQuoter: postgres.GetIdentifierQuoter(), ColumnParser: postgres.GetColumnParser()},
		{PlaceholderParser: oracle.GetPlaceholderParser(), IdentifierQuoter: oracle.GetIdentifierQuoter(), ColumnParser: oracle.GetColumnParser()},
	}
}

func TestConfigStrategiesOverrideLocallyAndAreSnapshotted(t *testing.T) {
	defaults := nativeRouteOptions()
	for _, test := range routeCases() {
		t.Run(string(test.mode), func(t *testing.T) {
			config := &Config{Mode: test.mode, EngineOptions: db.EngineOptions{
				PlaceholderParser: routePlaceholderParser{},
				IdentifierQuoter:  db.NewIdentifierQuoter("<", ">"),
				ColumnParser:      db.NewColumnParser("column", "sql_type"),
			}}
			manager, recorder := newRouteManager(t, config)
			engine := ConnectDatabase("custom", manager)
			config.EngineOptions = db.EngineOptions{}
			if err := writeRoute(engine, ":v"); err != nil {
				t.Fatal(err)
			}
			if recorder.query != "UPDATE <app>.<roles> SET <display_name> = @1 WHERE id = @2" {
				t.Fatalf("custom SQL=%q", recorder.query)
			}
			if err := writeRoute(ConnectDatabase("fresh", manager), "?"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(recorder.query, " = "+routeDriverPlaceholder(test.mode, 1)+" WHERE id = "+routeDriverPlaceholder(test.mode, 2)) || strings.Contains(recorder.query, "display_name") ||
				recorder.poolCalls != 1 || !reflect.DeepEqual(nativeRouteOptions(), defaults) {
				t.Fatalf("defaults changed: SQL=%q, pool calls=%d", recorder.query, recorder.poolCalls)
			}
		})
	}
}

func setRouteDialectDefaults(mode CompatibilityMode, parser db.PlaceholderParser, quoter db.IdentifierQuoter) {
	switch mode {
	case ModePostgreSQL:
		postgres.SetPlaceholderParser(parser)
		postgres.SetIdentifierQuoter(quoter)
	case ModeMySQL:
		mysql.SetPlaceholderParser(parser)
		mysql.SetIdentifierQuoter(quoter)
	case ModeOracle:
		oracle.SetPlaceholderParser(parser)
		oracle.SetIdentifierQuoter(quoter)
	}
}

func TestDialectAndSharedOptionsAreInheritedAndSnapshotted(t *testing.T) {
	defaults := nativeRouteOptions()
	byMode := map[CompatibilityMode]db.EngineOptions{
		ModeMySQL: defaults[0], ModePostgreSQL: defaults[1], ModeOracle: defaults[2],
	}
	oldColumn := db.GetColumnParser()
	t.Cleanup(func() {
		for mode, options := range byMode {
			setRouteDialectDefaults(mode, options.PlaceholderParser, options.IdentifierQuoter)
		}
		db.SetColumnParser(oldColumn)
	})
	for _, test := range routeCases() {
		t.Run(string(test.mode), func(t *testing.T) {
			old := byMode[test.mode]
			setRouteDialectDefaults(test.mode, routePlaceholderParser{}, db.NewIdentifierQuoter("[", "]"))
			db.SetColumnParser(db.NewColumnParser("column", "sql_type"))
			manager, recorder := newRouteManager(t, Config{Mode: test.mode})
			engine := ConnectDatabase("inherited", manager)
			manager.Config = Config{Mode: test.mode, EngineOptions: db.EngineOptions{
				PlaceholderParser: db.NewPlaceholderParser("?"),
			}}
			overridden := ConnectDatabase("connection_override", manager)
			setRouteDialectDefaults(test.mode, old.PlaceholderParser, old.IdentifierQuoter)
			db.SetColumnParser(oldColumn)

			if err := writeRoute(engine, ":v"); err != nil {
				t.Fatal(err)
			}
			if recorder.query != "UPDATE [app].[roles] SET [display_name] = @1 WHERE id = @2" {
				t.Fatalf("inherited SQL=%q", recorder.query)
			}
			if err := writeRoute(overridden, "?"); err != nil {
				t.Fatal(err)
			}
			if recorder.query != "UPDATE [app].[roles] SET [display_name] = ? WHERE id = ?" {
				t.Fatalf("connection parser did not override dialect parser: %q", recorder.query)
			}
			manager.Config = Config{Mode: test.mode}
			if err := writeRoute(ConnectDatabase("fresh", manager), "?"); err != nil {
				t.Fatal(err)
			}
			quote := func(value string) string { return test.quote + value + test.quote }
			wantSQL := "UPDATE " + quote("app") + "." + quote("roles") +
				" SET " + quote("name") + " = " + routeDriverPlaceholder(test.mode, 1) +
				" WHERE id = " + routeDriverPlaceholder(test.mode, 2)
			if recorder.query != wantSQL || !reflect.DeepEqual(nativeRouteOptions(), defaults) {
				t.Fatalf("fresh SQL=%q, want %q; native package defaults changed", recorder.query, wantSQL)
			}
		})
	}
}

func TestCompatibilityModeValues(t *testing.T) {
	for mode, want := range map[CompatibilityMode]string{
		ModePostgreSQL: "postgres",
		ModeMySQL:      "mysql",
		ModeOracle:     "oracle",
	} {
		if string(mode) != want {
			t.Errorf("mode=%q, want %q", mode, want)
		}
	}
}

func TestInvalidConfigFailsBeforeOpeningPool(t *testing.T) {
	sentinel := errors.New("config unavailable")
	providerErr := fmt.Errorf("load config: %w", sentinel)
	manager, recorder := newRouteManager(t, routeConfigProvider(func(any) error { return providerErr }))
	recovered := routePanic(t, func() { ConnectDatabase("error", manager) })
	if err, ok := recovered.(error); !ok || err != providerErr || !errors.Is(err, sentinel) || recorder.poolCalls != 0 {
		t.Fatalf("panic=%v, pool calls=%d", recovered, recorder.poolCalls)
	}
	manager, recorder = newRouteManager(t, Config{Mode: "unknown"})
	recovered = routePanic(t, func() { ConnectDatabase("unknown", manager) })
	if !strings.Contains(fmt.Sprint(recovered), "unknown") || recorder.poolCalls != 0 {
		t.Fatalf("panic=%v, pool calls=%d", recovered, recorder.poolCalls)
	}
}

func TestConnectEntryValidation(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		registerRouteManager(t, nil)
		delete(goi.Settings.Databases, t.Name())
		routePanic(t, func() { Connect(t.Name()) })
	})
	t.Run("mismatched_engine", func(t *testing.T) {
		manager, recorder := newRouteManager(t, nil)
		manager.Engine = "custom-driver"
		registerRouteManager(t, manager)
		routePanic(t, func() { Connect(t.Name()) })
		if recorder.poolCalls != 0 {
			t.Fatal("mismatched Connect opened a pool")
		}
		if _, ok := ConnectDatabase("direct", manager).(*postgres.Engine); !ok || recorder.driverEngine != "custom-driver" {
			t.Fatal("direct constructor changed or rejected the physical driver")
		}
	})
	t.Run("generic_type", func(t *testing.T) {
		manager, _ := newRouteManager(t, Config{Mode: ModeMySQL})
		registerRouteManager(t, manager)
		recovered := routePanic(t, func() { db.Connect[*postgres.Engine](t.Name()) })
		if message := fmt.Sprint(recovered); !strings.Contains(message, "mysql.Engine") || !strings.Contains(message, "postgres.Engine") {
			t.Fatalf("unexpected type assertion panic: %v", recovered)
		}
		if _, ok := db.ConnectDatabase[db.Engine](t.Name(), manager).(*mysql.Engine); !ok {
			t.Fatal("generic interface route returned the wrong type")
		}
	})
}

func TestConfigGetConfigValidatesTargetAndCopiesStrategyReferences(t *testing.T) {
	for _, target := range []any{nil, (*Config)(nil), Config{}, new(string)} {
		if err := (Config{}).GetConfig(target); err == nil {
			t.Fatalf("accepted target %T", target)
		}
	}
	parser, quoter, column := &routePlaceholderParser{}, db.NewIdentifierQuoter("<", ">"), db.NewColumnParser("column", "sql_type")
	source := Config{Mode: ModeMySQL, EngineOptions: db.EngineOptions{
		PlaceholderParser: parser, IdentifierQuoter: &quoter, ColumnParser: &column,
	}}
	for _, provider := range []goi.DatabaseConfig{source, &source} {
		var target Config
		if err := provider.GetConfig(&target); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(target, source) || target.EngineOptions.PlaceholderParser != parser ||
			target.EngineOptions.IdentifierQuoter != &quoter || target.EngineOptions.ColumnParser != &column {
			t.Fatalf("copy lost values or strategy references: %#v", target)
		}
		target.Mode, target.EngineOptions = ModeOracle, db.EngineOptions{}
		if source.Mode != ModeMySQL || source.EngineOptions.PlaceholderParser != parser {
			t.Fatal("changing the target changed its source")
		}
	}
}
