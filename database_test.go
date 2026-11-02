package goi

import (
	"errors"
	"fmt"
	"testing"
)

type databaseTestTarget struct {
	Name    string
	Default int
}

type databaseTestConfig struct {
	name   string
	target any
	err    error
}

func (config *databaseTestConfig) GetConfig(target any) error {
	config.target = target
	if config.err != nil {
		return config.err
	}
	target.(*databaseTestTarget).Name = config.name
	return nil
}

func TestDatabaseGetConfigDelegatesToProvider(t *testing.T) {
	config := &databaseTestConfig{name: "configured"}
	database := Database{Config: config}
	target := &databaseTestTarget{Name: "default", Default: 7}
	if err := database.GetConfig(target); err != nil {
		t.Fatal(err)
	}
	if config.target != target {
		t.Fatal("provider did not receive the original target")
	}
	if target.Name != "configured" || target.Default != 7 {
		t.Fatalf("target = %+v", target)
	}
}

func TestDatabaseGetConfigPreservesProviderError(t *testing.T) {
	sentinel := errors.New("configuration unavailable")
	want := fmt.Errorf("provider: %w", sentinel)
	config := &databaseTestConfig{err: want}
	database := Database{Config: config}
	target := "provider decides whether this target is valid"
	got := database.GetConfig(target)
	if got != want || !errors.Is(got, sentinel) {
		t.Fatalf("GetConfig() = %v, want original error %v", got, want)
	}
	if config.target != target {
		t.Fatal("GetConfig changed or rejected the target before delegating")
	}
}

type databaseNilPointerConfig struct{}

func (*databaseNilPointerConfig) GetConfig(target any) error {
	target.(*databaseTestTarget).Name = "called"
	return errors.New("nil provider must not be called")
}

func TestDatabaseGetConfigKeepsDefaultsForNilConfig(t *testing.T) {
	var typedNil *databaseNilPointerConfig
	for _, test := range []struct {
		name   string
		config DatabaseConfig
	}{
		{name: "nil"},
		{name: "typed_nil", config: typedNil},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := Database{Config: test.config}
			target := &databaseTestTarget{Name: "default", Default: 7}
			if err := database.GetConfig(target); err != nil {
				t.Fatal(err)
			}
			if target.Name != "default" || target.Default != 7 {
				t.Fatalf("nil config changed defaults: %+v", target)
			}
		})
	}
}
