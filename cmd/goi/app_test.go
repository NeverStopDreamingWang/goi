package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
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

func TestTemplatesUseCurrentModuleAndVersion(t *testing.T) {
	oldProject, oldApp := projectName, appName
	oldSecret, oldPrivate, oldPublic := secretKey, privateKey, publicKey
	projectName, appName = "demo", "example"
	secretKey, privateKey, publicKey = "test-secret", "test-private-key", "test-public-key"
	t.Cleanup(func() {
		projectName, appName = oldProject, oldApp
		secretKey, privateKey, publicKey = oldSecret, oldPrivate, oldPublic
	})

	for _, group := range []struct {
		name      string
		templates []InitFile
	}{
		{"project", MainAppFileList},
		{"app", AppFileList},
	} {
		for _, template := range group.templates {
			t.Run(group.name+"/"+template.Name, func(t *testing.T) {
				content := template.Content()
				if strings.Contains(content, "github.com/NeverStopDreamingWang/goi/v2") {
					t.Fatal("template still references the v2 module")
				}
				if strings.HasSuffix(template.Name, ".go") {
					if _, err := parser.ParseFile(token.NewFileSet(), template.Name, content, parser.AllErrors); err != nil {
						t.Fatalf("template is not valid Go: %v", err)
					}
				}
			})
		}
	}

	wantDependency := "github.com/NeverStopDreamingWang/goi/v3 " + goi.Version()
	if !strings.Contains(goMod.Content(), wantDependency) {
		t.Fatalf("go.mod template does not require %s", wantDependency)
	}
}
