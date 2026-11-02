package db

import (
	"reflect"
	"testing"
)

func TestEngineOptionsMerge(t *testing.T) {
	defaults := EngineOptions{
		PlaceholderParser: NewPlaceholderParser("?"),
		IdentifierQuoter:  NewIdentifierQuoter("`", "`"),
		ColumnParser:      NewColumnParser("field_name", "field_type"),
	}
	first := EngineOptions{PlaceholderParser: NewPlaceholderParser(":v")}
	last := EngineOptions{
		PlaceholderParser: NewPlaceholderParser("$"),
		IdentifierQuoter:  NewIdentifierQuoter("[", "]"),
	}
	merged := defaults.Merge(first, EngineOptions{}, last)
	if merged.PlaceholderParser.Placeholder() != "$" {
		t.Fatal("later non-nil parser did not override the earlier parser")
	}
	if got := merged.IdentifierQuoter.QuoteIdentifier("name"); got != "[name]" {
		t.Fatalf("identifier = %q", got)
	}
	if !reflect.DeepEqual(merged.ColumnParser, defaults.ColumnParser) {
		t.Fatal("nil override replaced the default column parser")
	}
	if defaults.PlaceholderParser.Placeholder() != "?" || defaults.IdentifierQuoter.QuoteIdentifier("name") != "`name`" {
		t.Fatal("Merge changed the original options")
	}
	if !reflect.DeepEqual(defaults.Merge(), defaults) {
		t.Fatal("Merge without overrides changed the options")
	}
}
