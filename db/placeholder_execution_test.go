package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/mysql"
	"github.com/NeverStopDreamingWang/goi/v3/db/oracle"
	"github.com/NeverStopDreamingWang/goi/v3/db/postgres"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlite3"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlserver"
)

type placeholderExecutionEngine interface {
	db.Engine
	ExecuteContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	GetSQL() string
}

type placeholderExecutionProvider struct{ pool *sql.DB }

func (provider placeholderExecutionProvider) DB() *sql.DB { return provider.pool }

const placeholderReturnedSQL = "SELECT ?, ?, ?"

type replacingPlaceholderParser struct {
	calls int
	query string
	args  []any
}

func (*replacingPlaceholderParser) Placeholder() string { return ":v" }
func (parser *replacingPlaceholderParser) Parse(query string, args ...any) (string, []any) {
	parser.calls++
	parser.query, parser.args = query, append([]any(nil), args...)
	return placeholderReturnedSQL, []any{args[1], args[0], "added"}
}

func TestExecutionEntrypointsUseReturnedPlaceholderSQLAndArguments(t *testing.T) {
	factories := []struct {
		name string
		new  func(db.DBProvider, db.EngineOptions) placeholderExecutionEngine
	}{
		{"mysql", func(provider db.DBProvider, options db.EngineOptions) placeholderExecutionEngine {
			return mysql.ConnectDatabase("test", provider, options)
		}},
		{"postgres", func(provider db.DBProvider, options db.EngineOptions) placeholderExecutionEngine {
			return postgres.ConnectDatabase("test", provider, options)
		}},
		{"oracle", func(provider db.DBProvider, options db.EngineOptions) placeholderExecutionEngine {
			return oracle.ConnectDatabase("test", provider, options)
		}},
		{"sqlserver", func(provider db.DBProvider, options db.EngineOptions) placeholderExecutionEngine {
			return sqlserver.ConnectDatabase("test", provider, options)
		}},
		{"sqlite3", func(provider db.DBProvider, options db.EngineOptions) placeholderExecutionEngine {
			return sqlite3.ConnectDatabase("test", provider, options)
		}},
	}
	operations := []struct {
		name       string
		useContext bool
		run        func(placeholderExecutionEngine, context.Context, string, ...any) error
	}{
		{"Execute", false, func(engine placeholderExecutionEngine, _ context.Context, query string, args ...any) error {
			_, err := engine.Execute(query, args...)
			return err
		}},
		{"ExecuteContext", true, func(engine placeholderExecutionEngine, ctx context.Context, query string, args ...any) error {
			_, err := engine.ExecuteContext(ctx, query, args...)
			return err
		}},
		{"Query", false, func(engine placeholderExecutionEngine, _ context.Context, query string, args ...any) error {
			return closePlaceholderRows(engine.Query(query, args...))
		}},
		{"QueryContext", true, func(engine placeholderExecutionEngine, ctx context.Context, query string, args ...any) error {
			return closePlaceholderRows(engine.QueryContext(ctx, query, args...))
		}},
		{"QueryRow", false, func(engine placeholderExecutionEngine, _ context.Context, query string, args ...any) error {
			var value int64
			return engine.QueryRow(query, args...).Scan(&value)
		}},
		{"QueryRowContext", true, func(engine placeholderExecutionEngine, ctx context.Context, query string, args ...any) error {
			var value int64
			return engine.QueryRowContext(ctx, query, args...).Scan(&value)
		}},
	}
	for _, factory := range factories {
		for _, operation := range operations {
			for _, transaction := range []bool{false, true} {
				path := "pool"
				if transaction {
					path = "transaction"
				}
				t.Run(factory.name+"/"+operation.name+"/"+path, func(t *testing.T) {
					recorder := &placeholderExecutionRecorder{}
					pool := sql.OpenDB(placeholderExecutionConnector{recorder})
					t.Cleanup(func() { _ = pool.Close() })
					parser := &replacingPlaceholderParser{}
					engine := factory.new(placeholderExecutionProvider{pool}, db.EngineOptions{PlaceholderParser: parser})
					type contextKey struct{}
					ctx := context.WithValue(context.Background(), contextKey{}, "request")
					const query = "SELECT :v, :v"
					args := []any{"first", int64(7)}
					originalArgs := append([]any(nil), args...)
					var parsedSQL string
					run := func(current placeholderExecutionEngine) error {
						err := operation.run(current, ctx, query, args...)
						parsedSQL = current.GetSQL()
						return err
					}
					var err error
					if transaction {
						err = withPlaceholderTransaction(ctx, engine, run)
					} else {
						err = run(engine)
					}
					if err != nil {
						t.Fatal(err)
					}
					if parser.calls != 1 || parser.query != query || !reflect.DeepEqual(parser.args, originalArgs) || !reflect.DeepEqual(args, originalArgs) {
						t.Fatalf("parser calls=%d, query=%q, parsed args=%#v, original args=%#v", parser.calls, parser.query, parser.args, args)
					}
					wantArgs := []driver.NamedValue{{Ordinal: 1, Value: int64(7)}, {Ordinal: 2, Value: "first"}, {Ordinal: 3, Value: "added"}}
					if recorder.calls != 1 || recorder.query != placeholderReturnedSQL || parsedSQL != placeholderReturnedSQL || !reflect.DeepEqual(recorder.args, wantArgs) {
						t.Fatalf("driver calls=%d SQL=%q args=%#v, GetSQL=%q", recorder.calls, recorder.query, recorder.args, parsedSQL)
					}
					wantContext := context.Background()
					if operation.useContext {
						wantContext = ctx
					}
					if recorder.ctx != wantContext || recorder.inTransaction != transaction || recorder.connections != 1 {
						t.Fatalf("context=%v, transaction=%t, connections=%d", recorder.ctx, recorder.inTransaction, recorder.connections)
					}
					if transaction && (recorder.beginContext != ctx || recorder.commits != 1 || recorder.rollbacks != 0) {
						t.Fatalf("begin context=%v, commits=%d, rollbacks=%d", recorder.beginContext, recorder.commits, recorder.rollbacks)
					}
				})
			}
		}
	}
}

func closePlaceholderRows(rows *sql.Rows, err error) error {
	if err != nil {
		return err
	}
	return rows.Close()
}

func withPlaceholderTransaction(ctx context.Context, engine placeholderExecutionEngine, run func(placeholderExecutionEngine) error) error {
	switch current := engine.(type) {
	case *mysql.Engine:
		return current.WithTransaction(ctx, func(tx *mysql.Engine, _ ...any) error { return run(tx) })
	case *postgres.Engine:
		return current.WithTransaction(ctx, func(tx *postgres.Engine, _ ...any) error { return run(tx) })
	case *oracle.Engine:
		return current.WithTransaction(ctx, func(tx *oracle.Engine, _ ...any) error { return run(tx) })
	case *sqlserver.Engine:
		return current.WithTransaction(ctx, func(tx *sqlserver.Engine, _ ...any) error { return run(tx) })
	case *sqlite3.Engine:
		return current.WithTransaction(ctx, func(tx *sqlite3.Engine, _ ...any) error { return run(tx) })
	default:
		return errors.New("unexpected placeholder execution engine")
	}
}

type placeholderExecutionRecorder struct {
	calls, connections, commits, rollbacks int
	query                                  string
	args                                   []driver.NamedValue
	ctx, beginContext                      context.Context
	inTransaction                          bool
}

type placeholderExecutionConnector struct{ recorder *placeholderExecutionRecorder }
type placeholderExecutionDriver struct{}
type placeholderExecutionConn struct {
	recorder      *placeholderExecutionRecorder
	inTransaction bool
}
type placeholderExecutionRows struct{ read bool }

func (connector placeholderExecutionConnector) Connect(context.Context) (driver.Conn, error) {
	connector.recorder.connections++
	return &placeholderExecutionConn{recorder: connector.recorder}, nil
}
func (placeholderExecutionConnector) Driver() driver.Driver { return placeholderExecutionDriver{} }
func (placeholderExecutionDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use the placeholder execution connector")
}
func (*placeholderExecutionConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}
func (*placeholderExecutionConn) Close() error { return nil }
func (connection *placeholderExecutionConn) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}
func (connection *placeholderExecutionConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	connection.inTransaction = true
	connection.recorder.beginContext = ctx
	return connection, nil
}
func (connection *placeholderExecutionConn) Commit() error {
	connection.recorder.commits++
	connection.inTransaction = false
	return nil
}
func (connection *placeholderExecutionConn) Rollback() error {
	connection.recorder.rollbacks++
	connection.inTransaction = false
	return nil
}
func (connection *placeholderExecutionConn) record(ctx context.Context, query string, args []driver.NamedValue) {
	connection.recorder.calls++
	connection.recorder.query, connection.recorder.ctx = query, ctx
	connection.recorder.args = append([]driver.NamedValue(nil), args...)
	connection.recorder.inTransaction = connection.inTransaction
}
func (connection *placeholderExecutionConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	connection.record(ctx, query, args)
	return driver.RowsAffected(1), nil
}
func (connection *placeholderExecutionConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	connection.record(ctx, query, args)
	return &placeholderExecutionRows{}, nil
}
func (*placeholderExecutionRows) Columns() []string { return []string{"value"} }
func (*placeholderExecutionRows) Close() error      { return nil }
func (rows *placeholderExecutionRows) Next(values []driver.Value) error {
	if rows.read {
		return io.EOF
	}
	rows.read, values[0] = true, int64(1)
	return nil
}
