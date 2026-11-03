package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

type genericTestEngine struct{}

func (*genericTestEngine) Name() string { return "generic_test" }

func (*genericTestEngine) Execute(string, ...any) (sql.Result, error) { return nil, nil }

func (*genericTestEngine) QueryRow(string, ...any) *sql.Row { return nil }

func (*genericTestEngine) Query(string, ...any) (*sql.Rows, error) { return nil, nil }

type otherGenericTestEngine struct{ genericTestEngine }

type constrainedTestEngine interface {
	Engine
	ExtraMethod()
}

func registerGenericTestEngine(t *testing.T, engine Engine) *goi.Database {
	t.Helper()
	name := t.Name()
	Register(name, func(string, *goi.Database) Engine { return engine })
	t.Cleanup(func() {
		engineMu.Lock()
		defer engineMu.Unlock()
		delete(engines, name)
	})
	return &goi.Database{Engine: name}
}

func requireEngineTypeMismatch(t *testing.T, engine Engine, wantType reflect.Type, connect func()) {
	t.Helper()
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		connect()
	}()
	if recovered == nil {
		t.Fatal("ConnectDatabase() accepted a mismatched engine type")
	}
	want := i18n.T("db.engine_type_is_not_match", map[string]any{
		"want_type": wantType.String(),
		"got_type":  reflect.TypeOf(engine).String(),
	})
	if got := fmt.Sprint(recovered); got != want {
		t.Fatalf("ConnectDatabase() panic = %q, want localized mismatch %q", got, want)
	}
}

func TestConnectDatabaseReturnsMatchingPointerType(t *testing.T) {
	want := &genericTestEngine{}
	database := registerGenericTestEngine(t, want)
	if got := ConnectDatabase[*genericTestEngine]("test", database); got != want {
		t.Fatalf("ConnectDatabase() = %p, want %p", got, want)
	}
}

func TestConnectDatabaseRejectsMismatchedPointerType(t *testing.T) {
	engine := &genericTestEngine{}
	database := registerGenericTestEngine(t, engine)
	requireEngineTypeMismatch(t, engine, reflect.TypeFor[*otherGenericTestEngine](), func() {
		ConnectDatabase[*otherGenericTestEngine]("test", database)
	})
}

func TestConnectDatabaseRejectsConstrainedInterfaceWithLocalizedError(t *testing.T) {
	engine := &genericTestEngine{}
	database := registerGenericTestEngine(t, engine)
	requireEngineTypeMismatch(t, engine, reflect.TypeFor[constrainedTestEngine](), func() {
		ConnectDatabase[constrainedTestEngine]("test", database)
	})
}

func TestConnectionErrorsPreserveLocalizedMessages(t *testing.T) {
	for _, test := range []struct {
		name string
		want func(string) string
		run  func(string)
	}{
		{
			name: "nil_factory",
			want: func(string) string { return i18n.T("db.register_engine_is_not_nil") },
			run:  func(name string) { Register(name, nil) },
		},
		{
			name: "duplicate_factory",
			want: func(name string) string {
				return i18n.T("db.register_engine_twice", map[string]any{"name": name})
			},
			run: func(name string) {
				factory := func(string, *goi.Database) Engine { return &genericTestEngine{} }
				Register(name, factory)
				Register(name, factory)
			},
		},
		{
			name: "missing_connection",
			want: func(name string) string {
				return i18n.T("db.databases_not_error", map[string]any{"name": name})
			},
			run: func(name string) { Connect[*genericTestEngine](name) },
		},
		{
			name: "unregistered_engine",
			want: func(name string) string {
				return i18n.T("db.engine_not_registered", map[string]any{"engine": name})
			},
			run: func(name string) {
				ConnectDatabase[*genericTestEngine](name, &goi.Database{Engine: name})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// 配置名称中的 % 应原样保留，不能再次作为格式化指令解析。
			name := t.Name() + "_%s%"
			t.Cleanup(func() {
				engineMu.Lock()
				defer engineMu.Unlock()
				delete(engines, name)
			})
			defer func() {
				recovered := recover()
				err, ok := recovered.(error)
				if !ok || err.Error() != test.want(name) {
					t.Errorf("panic=%v, want error %q", recovered, test.want(name))
				}
			}()
			test.run(name)
			t.Fatal("expected an error panic")
		})
	}
}
