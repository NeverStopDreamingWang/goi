package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
)

func testCreateConfig(t *testing.T) createConfig {
	t.Helper()
	return createConfig{
		RootDir:        filepath.Join(t.TempDir(), "demo"),
		ProjectName:    "demo",
		ProjectPath:    "backend/demo",
		AppName:        "records",
		AppPath:        "backend/permission/records",
		ModulePath:     "example.com/team/demo",
		SettingsImport: "example.com/team/demo/backend/demo",
		SecretKey:      "test-secret",
		PrivateKey:     "test-private-key",
		PublicKey:      "test-public-key",
	}
}

func TestDatabaseTemplatesUseContextAPI(t *testing.T) {
	config := testCreateConfig(t)

	for _, template := range []InitFile{models, serializer} {
		content := template.Content(config)
		if _, err := parser.ParseFile(token.NewFileSet(), template.Name, content, parser.AllErrors); err != nil {
			t.Fatalf("%s is not valid Go: %v", template.Name, err)
		}
		if !strings.Contains(content, `"context"`) {
			t.Fatalf("%s does not import context", template.Name)
		}
	}

	modelSource := models.Content(config)
	if !strings.Contains(modelSource, "func Migrate(ctx context.Context)") ||
		!strings.Contains(modelSource, "Migrate(ctx, MyModel{})") ||
		strings.Contains(modelSource, "func init()") ||
		!strings.Contains(modelSource, `TableName: "records_my_model"`) ||
		!strings.Contains(modelSource, `field_type:"INTEGER PRIMARY KEY AUTOINCREMENT"`) ||
		strings.Contains(modelSource, "AUTO_INCREMENT") || strings.Contains(modelSource, " COMMENT ") {
		t.Fatal("model template does not use the SQLite migration API and SQL types")
	}
	serializerSource := serializer.Content(config)
	for _, call := range []string{
		"Count(ctx)",
		`Fields("Name", "CreateTime", "UpdateTime").Insert(ctx, self, true)`,
		`Fields("Name", "UpdateTime").Where(`,
		"Insert(ctx, self, true)",
		"Update(ctx, validatedData, true)",
		"Delete(ctx)",
	} {
		if !strings.Contains(serializerSource, call) {
			t.Fatalf("serializer template is missing %s", call)
		}
	}
}

func TestSerializerMethodsPassContextToORM(t *testing.T) {
	content := serializer.Content(testCreateConfig(t))
	file, err := parser.ParseFile(token.NewFileSet(), serializer.Name, content, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "context.Background()") || strings.Contains(content, "context.TODO()") {
		t.Fatal("serializer must use the caller's context")
	}
	methods := make(map[string]*ast.FuncDecl)
	for _, declaration := range file.Decls {
		if method, ok := declaration.(*ast.FuncDecl); ok && method.Recv != nil {
			methods[method.Name.Name] = method
		}
	}
	for _, test := range []struct {
		method    string
		operation string
	}{
		{"Validate", "Count"},
		{"Create", "Insert"},
		{"Update", "Update"},
		{"Delete", "Delete"},
	} {
		t.Run(test.method, func(t *testing.T) {
			method := methods[test.method]
			if method == nil || method.Type.Params.NumFields() == 0 {
				t.Fatal("method is missing a context parameter")
			}
			parameter := method.Type.Params.List[0]
			parameterType, ok := parameter.Type.(*ast.SelectorExpr)
			if !ok || len(parameter.Names) != 1 || parameter.Names[0].Name != "ctx" || parameterType.Sel.Name != "Context" {
				t.Fatal("first parameter must be ctx context.Context")
			}
			if qualifier, ok := parameterType.X.(*ast.Ident); !ok || qualifier.Name != "context" {
				t.Fatal("context parameter must use context.Context")
			}

			calls := 0
			ast.Inspect(method.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != test.operation {
					return true
				}
				calls++
				if len(call.Args) == 0 {
					t.Errorf("%s call is missing ctx", test.operation)
				} else if argument, ok := call.Args[0].(*ast.Ident); !ok || argument.Name != "ctx" {
					t.Errorf("%s must receive the caller's ctx", test.operation)
				}
				return true
			})
			if calls != 1 {
				t.Fatalf("found %d %s calls, want 1", calls, test.operation)
			}
		})
	}
}

func TestTemplatesUseCurrentModuleAndVersion(t *testing.T) {
	config := testCreateConfig(t)

	for _, group := range []struct {
		name      string
		templates []InitFile
	}{
		{"project", MainAppFileList},
		{"app", AppFileList},
	} {
		for _, template := range group.templates {
			t.Run(group.name+"/"+template.Name, func(t *testing.T) {
				content := template.Content(config)
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
	if !strings.Contains(goMod.Content(config), wantDependency) {
		t.Fatalf("go.mod template does not require %s", wantDependency)
	}
}

func TestProjectURLsOnlyDeclaresConfigurationPackage(t *testing.T) {
	config := testCreateConfig(t)
	assertPackageOnly(t, projectURLs.Name, []byte(projectURLs.Content(config)), config.ProjectName)
}

func assertPackageOnly(t *testing.T, filename string, content []byte, packageName string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filename, content, parser.AllErrors|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name.Name != packageName || len(file.Imports) != 0 || len(file.Decls) != 0 || len(file.Comments) != 0 {
		t.Fatalf("%s must contain only package %s", filename, packageName)
	}
}

func TestProjectAndAppTemplatesUseSameDatabase(t *testing.T) {
	config := testCreateConfig(t)
	settingsSource := settings.Content(config)
	if !strings.Contains(settingsSource, `Databases["default"]`) ||
		!strings.Contains(settingsSource, `Engine: "sqlite3"`) ||
		!strings.Contains(settingsSource, `_ "github.com/mattn/go-sqlite3"`) ||
		strings.Contains(settingsSource, `Engine: "mysql"`) {
		t.Fatal("project template must register the default SQLite connection and physical driver")
	}
	urlSource := urls.Content(config)
	if !strings.Contains(urlSource, `"example.com/team/demo/backend/demo"`) {
		t.Fatal("app routes do not import the actual project settings package")
	}
}

func TestGeneratedValidatorImplementsCurrentMethodContract(t *testing.T) {
	content := validator.Content(testCreateConfig(t))
	file, err := parser.ParseFile(token.NewFileSet(), validator.Name, content, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	methods := make(map[string]*ast.FuncDecl)
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil {
			continue
		}
		if receiver, ok := method.Recv.List[0].Type.(*ast.Ident); ok && receiver.Name == "phoneValidator" {
			methods[method.Name.Name] = method
		}
	}
	contract := reflect.TypeFor[goi.Validator]()
	for index := 0; index < contract.NumMethod(); index++ {
		required := contract.Method(index)
		method := methods[required.Name]
		if method == nil {
			t.Fatalf("phoneValidator is missing interface method %s", required.Name)
		}
		if method.Type.Params.NumFields() != required.Type.NumIn() || method.Type.Results.NumFields() != required.Type.NumOut() {
			t.Fatalf("phoneValidator.%s has the wrong signature", required.Name)
		}
		if parameter, ok := method.Type.Params.List[0].Type.(*ast.Ident); !ok || parameter.Name != "any" {
			t.Fatalf("phoneValidator.%s must accept any", required.Name)
		}
		lastResult := method.Type.Results.List[len(method.Type.Results.List)-1].Type
		result, ok := lastResult.(*ast.SelectorExpr)
		if !ok || result.Sel.Name != "ValidationError" {
			t.Fatalf("phoneValidator.%s must return goi.ValidationError", required.Name)
		}
		if qualifier, ok := result.X.(*ast.Ident); !ok || qualifier.Name != "goi" {
			t.Fatalf("phoneValidator.%s has the wrong error type", required.Name)
		}
	}
}

func TestTemplatesUseActualSettingsPackageNameWithoutFixedAlias(t *testing.T) {
	for _, packageName := range []string{"settingspkg", "goi"} {
		config := testCreateConfig(t)
		config.ProjectName = packageName
		config.ProjectPath = "backend/configuration"
		config.SettingsImport = config.ModulePath + "/" + config.ProjectPath
		for _, template := range []InitFile{mainGo, urls} {
			t.Run(packageName+"/"+template.Name, func(t *testing.T) {
				assertSettingsImportBindings(t, template, config)
			})
		}
	}
}

func assertSettingsImportBindings(t *testing.T, template InitFile, config createConfig) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), template.Name, template.Content(config), parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	importNames := make(map[string]bool)
	settingsFound, frameworkName := false, ""
	for _, item := range file.Imports {
		importPath, err := strconv.Unquote(item.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := "goi"
		if importPath == config.SettingsImport {
			settingsFound = true
			name = config.ProjectName
			if item.Name != nil {
				t.Fatal("settings package must use its declared name, not a fixed import alias")
			}
		} else if item.Name != nil {
			name = item.Name.Name
		}
		if importNames[name] {
			t.Fatalf("duplicate package import name %q", name)
		}
		importNames[name] = true
		if importPath == "github.com/NeverStopDreamingWang/goi/v3" {
			frameworkName = name
			if config.ProjectName == "goi" && frameworkName != "goiFramework" {
				t.Fatal("framework import must avoid the settings package named goi")
			}
		}
	}
	if !settingsFound {
		t.Fatal("template does not import the actual settings path")
	}
	serverReferences, frameworkReferences := 0, 0
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch selector.Sel.Name {
		case "Server":
			serverReferences++
			if qualifier, ok := selector.X.(*ast.Ident); !ok || qualifier.Name != config.ProjectName {
				t.Error("Server reference does not use the settings package's declared name")
			}
		case "ViewSet":
			frameworkReferences++
			if qualifier, ok := selector.X.(*ast.Ident); !ok || qualifier.Name != frameworkName {
				t.Error("ViewSet reference does not use the framework import name")
			}
		}
		return true
	})
	if serverReferences == 0 {
		t.Fatal("template does not reference the project Server")
	}
	if template.Name == urls.Name && (frameworkName == "" || frameworkReferences == 0) {
		t.Fatal("route template does not reference the framework ViewSet")
	}
}
