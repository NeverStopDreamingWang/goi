package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v2"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
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
