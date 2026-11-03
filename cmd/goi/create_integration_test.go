//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 需要 Go、CGO/C 编译器，以及下载生成项目依赖的网络或模块缓存。
func TestGeneratedProjectBuildsAndRunsSQLite(t *testing.T) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go SDK is required")
	}
	if output, err := exec.Command(goCommand, "env", "CGO_ENABLED").Output(); err != nil || strings.TrimSpace(string(output)) != "1" {
		t.Skip("generated SQLite application requires CGO_ENABLED=1 and a C compiler")
	}
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(t.TempDir(), "goi")
	if runtime.GOOS == "windows" {
		cli += ".exe"
	}
	runCreationCommand(t, repository, goCommand, "build", "-o", cli, "./cmd/goi")
	for _, test := range []struct {
		name   string
		module string
	}{
		{"default_root_module", ""},
		{"custom_module", "example.com/team/demo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifyGeneratedProject(t, goCommand, cli, repository, test.module)
		})
	}
}

func verifyGeneratedProject(t *testing.T, goCommand string, cli string, repository string, moduleOverride string) {
	t.Helper()
	work := filepath.Join(t.TempDir(), "project-root")
	if err := os.MkdirAll(work, 0755); err != nil {
		t.Fatal(err)
	}
	missingModuleOutput, err := creationCommand(work, cli, "create-app", "backend/permission/records").CombinedOutput()
	if err == nil || !strings.Contains(string(missingModuleOutput), "goi help create-app") {
		t.Fatalf("missing go.mod error=%v, output=%s", err, missingModuleOutput)
	}
	help, err := creationCommand(work, cli, "help", "create-app").CombinedOutput()
	if err != nil || !strings.Contains(string(help), "go.mod") || !strings.Contains(string(help), "create-app") {
		t.Fatalf("create-app help does not explain the root requirement: %v\n%s", err, help)
	}
	if _, err := creationCommand(work, cli, "create-project", "项目", "--module", "example.com/demo").CombinedOutput(); err == nil {
		t.Fatal("invalid project name was accepted")
	}
	if _, err := os.Stat(filepath.Join(work, "项目")); !os.IsNotExist(err) {
		t.Fatalf("invalid project name created files: %v", err)
	}
	for _, relative := range []string{"go.mod", "backend"} {
		if _, err := os.Stat(filepath.Join(work, relative)); !os.IsNotExist(err) {
			t.Fatalf("rejected command created %s: %v", relative, err)
		}
	}
	projectArguments := []string{"create-project", "backend/goi_example"}
	expectedModule := filepath.Base(work)
	if moduleOverride != "" {
		projectArguments = append(projectArguments, "--module", moduleOverride)
		expectedModule = moduleOverride
	}
	runCreationCommand(t, work, cli, projectArguments...)
	project := work
	moduleContent, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil || !strings.Contains(string(moduleContent), "module "+expectedModule+"\n") {
		t.Fatalf("generated module=%q, error=%v", moduleContent, err)
	}
	settingsPath := filepath.Join(project, "backend", "goi_example", "settings.go")
	for _, relative := range []string{"go.mod", "server/main.go", "backend/goi_example/urls.go"} {
		if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("missing root project file %s: %v", relative, err)
		}
	}
	urlsPath := filepath.Join(project, "backend", "goi_example", "urls.go")
	urlsContent, err := os.ReadFile(urlsPath)
	if err != nil {
		t.Fatal(err)
	}
	assertPackageOnly(t, urlsPath, urlsContent, "goi_example")
	if _, err := os.Stat(filepath.Join(project, "backend", "goi_example", "go.mod")); !os.IsNotExist(err) {
		t.Fatalf("created an unexpected nested module: %v", err)
	}
	settingsBefore, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := creationCommand(work, cli, "create-project", "backend/goi_example").CombinedOutput(); err == nil {
		t.Fatal("repeated project creation succeeded")
	}
	if after, err := os.ReadFile(settingsPath); err != nil || !bytes.Equal(after, settingsBefore) {
		t.Fatal("repeated project creation changed existing settings or keys")
	}

	// create-app 仅从项目根目录执行，不向上搜索；嵌套参数决定输出目录和末级包名。
	if _, err := creationCommand(filepath.Join(project, "backend", "goi_example"), cli, "create-app", "blocked").CombinedOutput(); err == nil {
		t.Fatal("app creation from a project subdirectory succeeded")
	}
	for _, directory := range []string{filepath.Join(project, "blocked"), filepath.Join(project, "backend", "goi_example", "blocked")} {
		if _, err := os.Stat(directory); !os.IsNotExist(err) {
			t.Fatalf("rejected command created %s: %v", directory, err)
		}
	}
	runCreationCommand(t, project, cli, "create-app", "backend/permission/records")
	runCreationCommand(t, project, cli, "create-app", "backend/permission/other")
	serializerPath := filepath.Join(project, "backend", "permission", "records", "serializer.go")
	serializerBefore, err := os.ReadFile(serializerPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := creationCommand(project, cli, "create-app", "backend/permission/records").CombinedOutput(); err == nil {
		t.Fatal("repeated app creation succeeded")
	}
	if after, err := os.ReadFile(serializerPath); err != nil || !bytes.Equal(after, serializerBefore) {
		t.Fatal("repeated app creation changed existing code")
	}
	runCreationCommand(t, project, goCommand, "mod", "edit", "-replace=github.com/NeverStopDreamingWang/goi/v3="+repository)
	if err := os.WriteFile(filepath.Join(project, "backend", "permission", "records", "generated_test.go"), []byte(generatedSerializerTest), 0644); err != nil {
		t.Fatal(err)
	}
	runCreationCommand(t, project, goCommand, "build", "-mod=mod", "./...")
	importPath, err := creationCommand(project, goCommand, "list", "-mod=mod", "-f", "{{.ImportPath}}", "./backend/permission/records").Output()
	if err != nil || strings.TrimSpace(string(importPath)) != expectedModule+"/backend/permission/records" {
		t.Fatalf("generated app import path=%q, error=%v", importPath, err)
	}
	runCreationCommand(t, project, goCommand, "test", "-mod=mod", "-count=1", "./backend/permission/records")
}

func creationCommand(directory string, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd
}

func runCreationCommand(t *testing.T, directory string, command string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", filepath.Base(command), args, err, output)
	}
}

const generatedSerializerTest = `package records

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlite3"
)

func TestGeneratedSerializer(t *testing.T) {
	t.Cleanup(func() {
		for _, database := range goi.Settings.Databases { _ = database.Close() }
		_ = goi.Log.Close()
	})
	engine := db.Connect[*sqlite3.Engine]("default")
	var exists int
	err := engine.QueryRow("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", (MyModel{}).ModelSet().TableName).Scan(&exists)
	if !errors.Is(err, sql.ErrNoRows) { t.Fatalf("app import unexpectedly migrated its table: %v", err) }
	Migrate(context.Background())
	name := "original"
	model := &MyModel{Name: &name}
	if err := model.Create(context.Background()); err != nil { t.Fatal(err) }
	if model.Id == nil { t.Fatal("insert did not return an ID") }
	if err := model.Update(context.Background(), nil); err == nil { t.Fatal("nil update was accepted") }
	if err := model.Update(context.Background(), &MyModel{}); err != nil { t.Fatal(err) }
	var saved MyModel
	if err := engine.SetModel(MyModel{}).Where("id = ?", model.Id).First(context.Background(), &saved); err != nil { t.Fatal(err) }
	if saved.Name == nil || *saved.Name != name { t.Fatal("partial update overwrote the omitted NOT NULL field") }
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for operation, run := range map[string]func() error{
		"validate": func() error { return model.Validate(canceled) },
		"create": func() error { return (&MyModel{Name: &name}).Create(canceled) },
		"update": func() error { return model.Update(canceled, &MyModel{Name: &name}) },
		"delete": func() error { return model.Delete(canceled) },
	} {
		if err := run(); !errors.Is(err, context.Canceled) { t.Errorf("%s lost context cancellation: %v", operation, err) }
	}
}
`
