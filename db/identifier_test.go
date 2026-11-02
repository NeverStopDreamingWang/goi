package db

import "testing"

func TestIdentifierQuoter(t *testing.T) {
	var zeroValue DefaultIdentifierQuoter
	tests := []struct {
		name   string
		quoter IdentifierQuoter
		input  string
		want   string
	}{
		{name: "zero value", quoter: zeroValue, input: ` user"name `, want: `user"name`},
		{name: "ansi", quoter: NewIdentifierQuoter(`"`, `"`), input: `user"name`, want: `"user"name"`},
		{name: "mysql", quoter: NewIdentifierQuoter("`", "`"), input: "user`name", want: "`user`name`"},
		{name: "sql server", quoter: NewIdentifierQuoter("[", "]"), input: "user]name", want: "[user]name]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.quoter.QuoteIdentifier(test.input); got != test.want {
				t.Fatalf("QuoteIdentifier() = %q, want %q", got, test.want)
			}
		})
	}

	mysqlQuoter := NewIdentifierQuoter("`", "`")
	if got := mysqlQuoter.QuoteIdentifiers([]string{"id", "name"}); got != "`id`,`name`" {
		t.Fatalf("QuoteIdentifiers() = %q", got)
	}
	if got := mysqlQuoter.QuoteIdentifiers([]string{"", " id ", "   ", "name", ""}); got != "`id`,`name`" {
		t.Fatalf("QuoteIdentifiers() with empty identifiers = %q", got)
	}
}

func TestIdentifierQuoterAcceptsConfiguredDelimiter(t *testing.T) {
	tests := []struct {
		name   string
		quoter IdentifierQuoter
		input  string
		want   string
	}{
		{name: "mysql raw", quoter: NewIdentifierQuoter("`", "`"), input: "create_time", want: "`create_time`"},
		{name: "mysql quoted", quoter: NewIdentifierQuoter("`", "`"), input: " `create_time` ", want: "`create_time`"},
		{name: "mysql escaped", quoter: NewIdentifierQuoter("`", "`"), input: "`user``name`", want: "`user``name`"},
		{name: "ansi quoted", quoter: NewIdentifierQuoter(`"`, `"`), input: `"create_time"`, want: `"create_time"`},
		{name: "sql server quoted", quoter: NewIdentifierQuoter("[", "]"), input: "[create_time]", want: "[create_time]"},
		{name: "different delimiter is content", quoter: NewIdentifierQuoter("`", "`"), input: `"create_time"`, want: "`\"create_time\"`"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.quoter.QuoteIdentifier(test.input); got != test.want {
				t.Fatalf("QuoteIdentifier(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestDefaultIdentifierQuoterNormalizesDelimiters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "without delimiters", input: "name", want: "[name]"},
		{name: "missing right delimiter", input: "[name", want: "[name]"},
		{name: "missing left delimiter", input: "name]", want: "[name]"},
		{name: "both delimiters", input: "[name]", want: "[name]"},
		{name: "single delimiter", input: "[", want: "[]"},
		{name: "inner delimiter remains unchanged", input: "[user]name", want: "[user]name]"},
		{name: "escaped inner delimiter", input: "[user]]name]", want: "[user]]name]"},
		{name: "repeated outer delimiters", input: "[[name]]]", want: "[name]"},
		{name: "name ending with delimiter is not preserved", input: "[name]]]", want: "[name]"},
	}
	quoter := NewIdentifierQuoter("[", "]")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := quoter.QuoteIdentifier(test.input); got != test.want {
				t.Fatalf("QuoteIdentifier(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
