package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3/db"
)

type customIdentifierQuoter struct{}

func (customIdentifierQuoter) QuoteIdentifier(identifier string) string {
	return "<" + strings.TrimSpace(identifier) + ">"
}

func (quoter customIdentifierQuoter) QuoteIdentifiers(identifiers []string) string {
	quoted := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			continue
		}
		quoted = append(quoted, quoter.QuoteIdentifier(identifier))
	}
	return strings.Join(quoted, ",")
}

func TestSetIdentifierQuoter(t *testing.T) {
	SetIdentifierQuoter(customIdentifierQuoter{})
	t.Cleanup(func() {
		SetIdentifierQuoter(db.NewIdentifierQuoter("`", "`"))
	})
	SetIdentifierQuoter(nil)
	engine, recorder := newEngineTestEngine(t)
	name := "admin"

	if _, err := engine.Fields("Name").Insert(context.Background(), engineTestModel{Name: &name}, false); err != nil {
		t.Fatalf("Insert() returned an error: %v", err)
	}
	if recorder.execQuery != `INSERT INTO <test_schema>.<roles> (<name>) VALUES (?)` {
		t.Fatalf("SetIdentifierQuoter() was not applied: %s", recorder.execQuery)
	}
}

func TestGetIdentifierQuoter(t *testing.T) {
	t.Cleanup(func() {
		SetIdentifierQuoter(db.NewIdentifierQuoter("`", "`"))
	})

	SetIdentifierQuoter(db.NewIdentifierQuoter(`"`, `"`))
	if got := GetIdentifierQuoter().QuoteIdentifier(`user"name`); got != `"user"name"` {
		t.Fatalf("QuoteIdentifier() = %q", got)
	}
	if got := GetIdentifierQuoter().QuoteIdentifiers([]string{"id", "name"}); got != `"id","name"` {
		t.Fatalf("QuoteIdentifiers() = %q", got)
	}

	customQuoter := db.NewIdentifierQuoter("[", "]")
	if got := customQuoter.QuoteIdentifier("user]name"); got != "[user]name]" {
		t.Fatalf("custom IdentifierQuoter generated %q", got)
	}
	if got := GetIdentifierQuoter().QuoteIdentifier("name"); got != `"name"` {
		t.Fatalf("custom IdentifierQuoter changed the global quoter: %q", got)
	}

	SetIdentifierQuoter(customQuoter)
	if got := GetIdentifierQuoter().QuoteIdentifier("name"); got != "[name]" {
		t.Fatalf("SetIdentifierQuoter() generated %q", got)
	}
}

func TestGroupByAndOrderByAcceptQuotedIdentifiers(t *testing.T) {
	t.Cleanup(func() {
		SetIdentifierQuoter(db.NewIdentifierQuoter("`", "`"))
	})

	SetIdentifierQuoter(db.NewIdentifierQuoter("`", "`"))
	engine, _ := newEngineTestEngine(t)
	if grouped := engine.GroupBy("`create_time`"); grouped.groupSQL != " GROUP BY `create_time`" {
		t.Errorf("GroupBy() generated %q", grouped.groupSQL)
	}
	if grouped := engine.GroupBy("`create_time"); grouped.groupSQL != " GROUP BY `create_time`" {
		t.Errorf("GroupBy() with missing right delimiter generated %q", grouped.groupSQL)
	}
	if grouped := engine.GroupBy("create_time`"); grouped.groupSQL != " GROUP BY `create_time`" {
		t.Errorf("GroupBy() with missing left delimiter generated %q", grouped.groupSQL)
	}
	if grouped := engine.GroupBy("", " id ", "   ", "name"); grouped.groupSQL != " GROUP BY `id`,`name`" {
		t.Errorf("GroupBy() with multiple fields generated %q", grouped.groupSQL)
	}
	if ordered := engine.OrderBy("-`create_time`"); ordered.orderSQL != " ORDER BY `create_time` DESC" {
		t.Errorf("OrderBy() generated %q", ordered.orderSQL)
	}
	if ordered := engine.OrderBy("-`create_time"); ordered.orderSQL != " ORDER BY `create_time` DESC" {
		t.Errorf("OrderBy() with missing right delimiter generated %q", ordered.orderSQL)
	}

	SetIdentifierQuoter(db.NewIdentifierQuoter(`"`, `"`))
	doubleQuoteEngine, _ := newEngineTestEngine(t)
	if grouped := doubleQuoteEngine.GroupBy(`"create_time"`); grouped.groupSQL != ` GROUP BY "create_time"` {
		t.Errorf("GroupBy() with double quote generated %q", grouped.groupSQL)
	}
	if grouped := engine.GroupBy("create_time"); grouped.groupSQL != " GROUP BY `create_time`" {
		t.Errorf("existing Engine did not preserve its identifier quoter: %q", grouped.groupSQL)
	}

	SetIdentifierQuoter(db.NewIdentifierQuoter("[", "]"))
	bracketEngine, _ := newEngineTestEngine(t)
	if grouped := bracketEngine.GroupBy("[create_time]"); grouped.groupSQL != " GROUP BY [create_time]" {
		t.Errorf("GroupBy() with brackets generated %q", grouped.groupSQL)
	}
}
