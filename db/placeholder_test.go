package db

import "testing"

func TestDefaultPlaceholderParser(t *testing.T) {
	parser := NewPlaceholderParser("?")
	if got := parser.Placeholder(); got != "?" {
		t.Fatalf("Placeholder() = %q, want %q", got, "?")
	}

	query := "id = ? AND status = ?"
	if got := parser.Parse(query, int64(1), "enabled"); got != query {
		t.Fatalf("Parse() = %q, want %q", got, query)
	}
}
