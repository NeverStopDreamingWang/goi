package main

import (
	"bytes"
	"errors"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCreationNamesAndModules(t *testing.T) {
	for _, name := range []string{"demo", "my_app", "app2"} {
		if err := validateName(name); err != nil {
			t.Errorf("valid name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", ".", "..", "../outside", `..\outside`, "my-app", "package", "main", "_", "2app", "项目", "café"} {
		if err := validateName(name); err == nil {
			t.Errorf("accepted invalid name %q", name)
		}
	}
	for _, module := range []string{"demo", "example.com/team/demo", "example.com/team/demo/v3"} {
		if err := validateModulePath(module); err != nil {
			t.Errorf("valid module %q: %v", module, err)
		}
	}
	for _, module := range []string{"", "../demo", "example.com/../demo", "/absolute", "example.com//demo", "example.com/demo/", "example.com/demo with spaces", "https://example.com/demo", "demo\nrequire bad"} {
		if err := validateModulePath(module); err == nil {
			t.Errorf("accepted invalid module %q", module)
		}
	}
}

func TestParseCreationPath(t *testing.T) {
	for _, test := range []struct {
		input string
		path  string
		name  string
	}{
		{"user", "user", "user"},
		{"xxx/user", "xxx/user", "user"},
		{`xxx\user`, "xxx/user", "user"},
		{"api-v1/v2/user", "api-v1/v2/user", "user"},
	} {
		path, name, err := parseCreationPath(test.input)
		if err != nil || path != test.path || name != test.name {
			t.Errorf("parseCreationPath(%q)=(%q,%q,%v), want (%q,%q,nil)", test.input, path, name, err, test.path, test.name)
		}
	}
	for _, input := range []string{
		"", ".", "..", "../user", "./user", "xxx/../user", "xxx/./user", "xxx//user", "xxx/user/",
		"/xxx/user", `C:\xxx\user`, `\\server\share\user`, `xxx\..\user`,
		"xxx/my-app", "xxx/main", "xxx/_", "xxx/package", "xxx/2user", "xxx/用户", "bad parent/user", ".hidden/user",
	} {
		if _, _, err := parseCreationPath(input); err == nil {
			t.Errorf("accepted invalid creation path %q", input)
		}
	}
}

func TestEnsureNewDirectoryDoesNotReuseExistingTargets(t *testing.T) {
	root := t.TempDir()
	if err := ensureNewDirectory(filepath.Join(root, "new")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, filepath.Join(root, "file")} {
		if path != root {
			if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if err := ensureNewDirectory(path); !errors.Is(err, os.ErrExist) {
			t.Errorf("existing target %s: %v, want os.ErrExist", path, err)
		}
	}
}

func sourceTemplate(name string, source string, relativeDir string) InitFile {
	return InitFile{
		Name:    name,
		Content: func(createConfig) string { return source },
		Path:    func(config createConfig) string { return filepath.Join(config.RootDir, relativeDir) },
	}
}

func TestGenerateFilesFormatsGoAndRefusesOverwrite(t *testing.T) {
	config := testCreateConfig(t)
	first := sourceTemplate("first.go", "package demo\nvar Value=1\n", "")
	second := sourceTemplate("second.go", "package demo\nvar Other=2\n", "")
	if err := generateFiles(config, []InitFile{first, second}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(config.RootDir, "first.go")
	want, err := format.Source([]byte(first.Content(config)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("generated file=%q, error=%v, want %q", got, err, want)
	}
	// Windows 上已关闭的文件可立即改名，避免生成命令遗留打开的文件句柄。
	if err := os.Rename(path, path+".closed"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".closed", path); err != nil {
		t.Fatal(err)
	}
	if err := generateFiles(config, []InitFile{sourceTemplate("fresh.go", "package demo\n", ""), first}); err == nil {
		t.Fatal("generation overwrote an existing file")
	}
	if _, err := os.Stat(filepath.Join(config.RootDir, "fresh.go")); !os.IsNotExist(err) {
		t.Fatalf("conflicting plan partially wrote fresh.go: %v", err)
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, want) {
		t.Fatalf("existing file changed: %q, %v", after, err)
	}
}

func TestGenerateFilesDoesNotFollowExistingSymlink(t *testing.T) {
	config := testCreateConfig(t)
	if err := os.MkdirAll(config.RootDir, 0755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(config.RootDir, "linked")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	if err := generateFiles(config, []InitFile{sourceTemplate("escape.go", "package demo\n", "linked")}); err == nil {
		t.Fatal("generation followed an existing symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "escape.go")); !os.IsNotExist(err) {
		t.Fatalf("generation wrote outside its root: %v", err)
	}
}

func TestGenerateFilesRejectsInvalidPlansBeforeWriting(t *testing.T) {
	for _, test := range []struct {
		name      string
		templates []InitFile
	}{
		{"invalid_go", []InitFile{sourceTemplate("good.go", "package demo\n", ""), sourceTemplate("bad.go", "package {", "")}},
		{"outside_directory", []InitFile{sourceTemplate("escape.go", "package demo\n", "..")}},
		{"outside_filename", []InitFile{sourceTemplate("../escape.go", "package demo\n", "")}},
		{"duplicate", []InitFile{sourceTemplate("same.go", "package demo\n", ""), sourceTemplate("same.go", "package demo\n", "")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testCreateConfig(t)
			if err := generateFiles(config, test.templates); err == nil {
				t.Fatal("accepted an invalid generation plan")
			}
			if _, err := os.Stat(config.RootDir); !os.IsNotExist(err) {
				t.Fatalf("invalid plan created its target: %v", err)
			}
		})
	}
}

func TestGenerateFilesRejectsEmptyRoot(t *testing.T) {
	if err := generateFiles(createConfig{}, nil); err == nil {
		t.Fatal("generation accepted an empty root directory")
	}
}

func TestGenerateFilesRejectsCaseEquivalentTargetsOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-equivalent paths use Windows path semantics")
	}
	config := testCreateConfig(t)
	templates := []InitFile{
		sourceTemplate("same.go", "package demo\n", ""),
		sourceTemplate("SAME.GO", "package demo\n", ""),
	}
	if err := generateFiles(config, templates); err == nil || !strings.Contains(err.Error(), "生成路径冲突") {
		t.Fatalf("case-equivalent targets returned %v, want a planned path conflict", err)
	}
	if _, err := os.Stat(config.RootDir); !os.IsNotExist(err) {
		t.Fatalf("conflicting plan created its target: %v", err)
	}
}

func writeProjectFixture(t *testing.T, root string, module string, configDir string, packageName string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, configDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+module+"\n\ngo 1.24.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, configDir, "settings.go"), []byte("package "+packageName+"\nvar Server any\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReadProjectUsesModuleAndSettingsPackageNotDirectoryName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "renamed-checkout")
	writeProjectFixture(t, root, `"example.com/team/original"`, "backend/configuration", "settingspkg")
	gotRoot, modulePath, settingsImport, settingsPackage, err := readProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != root || modulePath != "example.com/team/original" || settingsImport != "example.com/team/original/backend/configuration" || settingsPackage != "settingspkg" {
		t.Fatalf("project=(%q, %q, %q, %q)", gotRoot, modulePath, settingsImport, settingsPackage)
	}
	config := testCreateConfig(t)
	config.RootDir, config.ModulePath = gotRoot, modulePath
	config.SettingsImport, config.ProjectName = settingsImport, settingsPackage
	assertSettingsImportBindings(t, urls, config)
}

func TestReadProjectDoesNotSearchParentDirectories(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root, "example.com/demo", "configuration", "settings")
	nested := filepath.Join(root, "internal", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := readProject(nested); err == nil || !strings.Contains(err.Error(), "goi help create-app") {
		t.Fatalf("subdirectory error=%v, want project-root requirement and help command", err)
	}
}

func TestReadProjectSkipsForeignTrees(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root, "example.com/team/demo", "backend/demo", "demo")
	for _, directory := range []string{".hidden/config", "vendor/example/config", "node_modules/example/config"} {
		writeProjectFixture(t, root, "example.com/team/demo", directory, "ignored")
	}
	writeProjectFixture(t, filepath.Join(root, "nested_module"), "example.com/foreign", "config", "foreign")
	outside := t.TempDir()
	writeProjectFixture(t, outside, "example.com/outside", "config", "outside")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Logf("symlink coverage unavailable: %v", err)
	}
	gotRoot, modulePath, settingsImport, settingsPackage, err := readProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != root || modulePath != "example.com/team/demo" || settingsImport != "example.com/team/demo/backend/demo" || settingsPackage != "demo" {
		t.Fatalf("selected a foreign settings package: (%q,%q,%q,%q)", gotRoot, modulePath, settingsImport, settingsPackage)
	}
}

func projectCommand(modulePath string) *cobra.Command {
	command := &cobra.Command{}
	command.Flags().String("module", modulePath, "")
	return command
}

func TestCreateProjectUsesCurrentDirectoryAsRoot(t *testing.T) {
	for _, test := range []struct {
		name     string
		rootName string
		module   string
		want     string
	}{
		{"default_module", "project-root", "", "project-root"},
		{"custom_module_overrides_invalid_root_name", "my project", "example.com/team/demo", "example.com/team/demo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), test.rootName)
			if err := os.MkdirAll(root, 0755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			if err := GoiCreateProject(projectCommand(test.module), []string{"backend/goi_example"}); err != nil {
				t.Fatal(err)
			}
			for _, relative := range []string{"go.mod", "server/main.go", "backend/goi_example/settings.go", "backend/goi_example/urls.go"} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
					t.Fatalf("missing %s: %v", relative, err)
				}
			}
			urlsPath := filepath.Join(root, "backend", "goi_example", "urls.go")
			urlsContent, err := os.ReadFile(urlsPath)
			if err != nil {
				t.Fatal(err)
			}
			assertPackageOnly(t, urlsPath, urlsContent, "goi_example")
			content, err := os.ReadFile(filepath.Join(root, "go.mod"))
			if err != nil || !strings.Contains(string(content), "module "+test.want+"\n") {
				t.Fatalf("root module=%q, error=%v", content, err)
			}
			if _, err := os.Stat(filepath.Join(root, "backend", "goi_example", "go.mod")); !os.IsNotExist(err) {
				t.Fatalf("created an extra project module layer: %v", err)
			}
		})
	}
}

func TestCreateProjectRejectsInvalidDefaultModuleWithoutWriting(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my project")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := GoiCreateProject(projectCommand(""), []string{"backend/goi_example"}); err == nil {
		t.Fatal("invalid default module path was accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected project creation wrote files: entries=%v, error=%v", entries, err)
	}
}

func TestCreateProjectRejectsPlannedFileAsDirectoryWithoutWriting(t *testing.T) {
	for _, projectPath := range []string{"go.mod/demo", "server/main.go/demo", "GO.MOD/demo", "SERVER/MAIN.GO/demo"} {
		t.Run(projectPath, func(t *testing.T) {
			if projectPath != strings.ToLower(projectPath) && runtime.GOOS != "windows" {
				t.Skip("case-equivalent paths use Windows path semantics")
			}
			root := t.TempDir()
			t.Chdir(root)
			err := GoiCreateProject(projectCommand("example.com/team/demo"), []string{projectPath})
			if err == nil || !strings.Contains(err.Error(), "生成路径冲突") {
				t.Fatalf("create-project %q returned %v, want a planned path conflict", projectPath, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("rejected project creation wrote files: entries=%v, error=%v", entries, err)
			}
		})
	}
}

func TestCreateProjectRejectsExistingTargetsWithoutPartialOutput(t *testing.T) {
	for _, target := range []string{"go.mod", "server/main.go", "backend/goi_example"} {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(target))
			if target == "backend/goi_example" {
				path = filepath.Join(path, "existing.txt")
			}
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("keep existing"), 0644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			if err := GoiCreateProject(projectCommand("example.com/team/demo"), []string{"backend/goi_example"}); !errors.Is(err, os.ErrExist) {
				t.Fatalf("existing target returned %v, want os.ErrExist", err)
			}
			if content, err := os.ReadFile(path); err != nil || string(content) != "keep existing" {
				t.Fatal("existing target changed")
			}
			if target != "go.mod" {
				if _, err := os.Stat(filepath.Join(root, "go.mod")); !os.IsNotExist(err) {
					t.Fatalf("rejected project creation wrote go.mod: %v", err)
				}
			}
		})
	}
}

func TestCreateAppRequiresProjectRootAndDoesNotWriteFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root, "example.com/demo", "configuration", "settings")
	subdirectory := filepath.Join(root, "configuration")
	t.Chdir(subdirectory)
	if err := GoiCreateApp(nil, []string{"xxx/user"}); err == nil {
		t.Fatal("app creation from a subdirectory succeeded")
	}
	for _, directory := range []string{filepath.Join(root, "xxx"), filepath.Join(subdirectory, "xxx")} {
		if _, err := os.Stat(directory); !os.IsNotExist(err) {
			t.Fatalf("rejected command created %s: %v", directory, err)
		}
	}
}

func TestCreateNestedAppsCreatesAndReusesParentDirectories(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root, "example.com/team/demo", "backend/configuration", "settings")
	t.Chdir(root)
	for _, appPath := range []string{"backend/permission/user", "backend/permission/role"} {
		if err := GoiCreateApp(nil, []string{appPath}); err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(appPath)
		for _, template := range AppFileList {
			filename := filepath.Join(root, filepath.FromSlash(appPath), template.Name)
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(content, []byte("package "+name+"\n")) {
				t.Fatalf("%s does not use the leaf package name %q", filename, name)
			}
		}
	}
	filename := filepath.Join(root, "backend", "permission", "user", "serializer.go")
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := GoiCreateApp(nil, []string{"backend/permission/user"}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("existing app returned %v, want os.ErrExist", err)
	}
	if after, err := os.ReadFile(filename); err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing app code changed")
	}
}

func TestCreateNestedAppRejectsFileOrSymlinkParent(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFixture(t, root, "example.com/demo", "configuration", "settings")
			parent := filepath.Join(root, "xxx")
			outside := t.TempDir()
			if kind == "file" {
				if err := os.WriteFile(parent, []byte("existing parent"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(outside, parent); err != nil {
				t.Skipf("symlink creation is unavailable: %v", err)
			}
			t.Chdir(root)
			if err := GoiCreateApp(nil, []string{"xxx/user"}); err == nil {
				t.Fatal("unsafe parent was accepted")
			}
			if _, err := os.Stat(filepath.Join(outside, "user")); !os.IsNotExist(err) {
				t.Fatalf("app creation escaped its root: %v", err)
			}
			if kind == "file" {
				if content, err := os.ReadFile(parent); err != nil || string(content) != "existing parent" {
					t.Fatal("existing parent file changed")
				}
			}
		})
	}
}

func TestReadProjectRejectsMissingOrAmbiguousSettings(t *testing.T) {
	for _, test := range []struct {
		name string
		set  func(*testing.T, string)
	}{
		{"no_module", func(*testing.T, string) {}},
		{"no_settings", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"main_settings", func(t *testing.T, root string) { writeProjectFixture(t, root, "example.com/demo", "", "main") }},
		{"invalid_import_directory", func(t *testing.T, root string) {
			writeProjectFixture(t, root, "example.com/demo", "配置", "settings")
		}},
		{"ambiguous", func(t *testing.T, root string) {
			writeProjectFixture(t, root, "example.com/demo", "backend/first", "first")
			writeProjectFixture(t, root, "example.com/demo", "other/second", "second")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			test.set(t, root)
			if _, _, _, _, err := readProject(root); err == nil {
				t.Fatal("accepted an unsupported project")
			}
		})
	}
}
