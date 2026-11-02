package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestDatabaseTemplatesUseContextAPI(t *testing.T) {
	oldName := appName
	appName = "demo"
	t.Cleanup(func() { appName = oldName })

	for _, template := range []InitFile{models, serializer} {
		content := template.Content()
		if _, err := parser.ParseFile(token.NewFileSet(), template.Name, content, parser.AllErrors); err != nil {
			t.Fatalf("%s is not valid Go: %v", template.Name, err)
		}
		if !strings.Contains(content, `"context"`) {
			t.Fatalf("%s does not import context", template.Name)
		}
	}

	modelSource := models.Content()
	if !strings.Contains(modelSource, "Migrate(context.Background(), MyModel{})") ||
		!strings.Contains(modelSource, `field_type:"INTEGER PRIMARY KEY AUTOINCREMENT"`) ||
		strings.Contains(modelSource, "AUTO_INCREMENT") || strings.Contains(modelSource, " COMMENT ") {
		t.Fatal("model template does not use the SQLite migration API and SQL types")
	}
	serializerSource := serializer.Content()
	for _, call := range []string{
		"Count(context.Background())",
		`Fields("Name", "CreateTime", "UpdateTime").Insert(context.Background(), self, true)`,
		`Fields("Name", "UpdateTime").Where(`,
		"Insert(context.Background(), self, true)",
		"Update(context.Background(), validated_data, false)",
		"Delete(context.Background())",
	} {
		if !strings.Contains(serializerSource, call) {
			t.Fatalf("serializer template is missing %s", call)
		}
	}
}
