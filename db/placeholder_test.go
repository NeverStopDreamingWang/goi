package db

import (
	"reflect"
	"testing"
)

func TestDefaultPlaceholderParser(t *testing.T) {
	parser := NewPlaceholderParser("?")
	if got := parser.Placeholder(); got != "?" {
		t.Fatalf("Placeholder() = %q, want %q", got, "?")
	}

	query := "id = ? AND status = ?"
	args := []any{int64(1), "enabled"}
	if got, gotArgs := parser.Parse(query, args...); got != query || !reflect.DeepEqual(gotArgs, args) {
		t.Fatalf("Parse() = (%q, %#v), want (%q, %#v)", got, gotArgs, query, args)
	}
	query, args = "id IN ?", []any{[]int64{1, 2}}
	if got, gotArgs := parser.Parse(query, args...); got != query || !reflect.DeepEqual(gotArgs, args) {
		t.Fatalf("public default parser expanded a collection: (%q, %#v)", got, gotArgs)
	}
}
