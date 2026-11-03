package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// createConfig 保存一次创建所需的参数，不在多次命令或模板之间共享可变状态。
type createConfig struct {
	RootDir        string
	ProjectName    string
	ProjectPath    string
	AppName        string
	AppPath        string
	ModulePath     string
	SettingsImport string
	SecretKey      string
	PrivateKey     string
	PublicKey      string
}

// validateName 验证可作为目录及导入路径使用的 ASCII Go 包名。
func validateName(name string) error {
	valid := token.IsIdentifier(name)
	for _, character := range name {
		valid = valid && character <= 127
	}
	if !valid || name == "_" || name == "main" ||
		!filepath.IsLocal(name) || filepath.Base(name) != name {
		return fmt.Errorf("无效名称 %q：请使用 ASCII Go 包名，且不能包含路径", name)
	}
	return nil
}

// parseCreationPath 解析项目根目录下的创建路径，末级目录名作为 Go 包名。
// 路径可用正斜杠或反斜杠分隔，不接受绝对路径、空段或 .、.. 路径段。
func parseCreationPath(value string) (relativePath, packageName string, err error) {
	relativePath = strings.ReplaceAll(value, "\\", "/")
	if !filepath.IsLocal(filepath.FromSlash(relativePath)) || path.Clean(relativePath) != relativePath {
		return "", "", fmt.Errorf("无效创建路径 %q：请使用当前目录内的相对路径，例如 backend/user", value)
	}
	if pathErr := validateModulePath(relativePath); pathErr != nil {
		return "", "", fmt.Errorf("无效创建路径 %q: %w", value, pathErr)
	}
	packageName = path.Base(relativePath)
	if nameErr := validateName(packageName); nameErr != nil {
		return "", "", nameErr
	}
	return relativePath, packageName, nil
}

// validateModulePath 验证 module 路径，不要求它与项目目录同名。
func validateModulePath(modulePath string) error {
	if modulePath == "" || path.Clean(modulePath) != modulePath || strings.HasPrefix(modulePath, "/") {
		return fmt.Errorf("无效 module 路径 %q", modulePath)
	}
	for _, character := range modulePath {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || strings.ContainsRune("/.-_~", character) {
			continue
		}
		return fmt.Errorf("无效 module 路径 %q", modulePath)
	}
	for _, part := range strings.Split(modulePath, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") {
			return fmt.Errorf("无效 module 路径 %q", modulePath)
		}
	}
	return nil
}

// ensureNewDirectory 在渲染或生成密钥前拒绝已有目标，不覆盖现有项目或应用。
func ensureNewDirectory(directory string) error {
	_, err := os.Lstat(directory)
	if err == nil {
		return fmt.Errorf("目标已存在，不会覆盖 %s: %w", directory, os.ErrExist)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查目标 %s: %w", directory, err)
	}
	return nil
}

// generateFiles 先渲染、格式化并检查全部目标，再以排他创建方式写入，绝不截断旧文件。
// 写入失败时返回具体路径和原始错误，不删除可能已由用户修改的目录或文件。
func generateFiles(config createConfig, templates []InitFile) error {
	type generatedFile struct {
		path    string
		content []byte
	}
	if config.RootDir == "" {
		return fmt.Errorf("项目目录不能为空")
	}
	root, err := filepath.Abs(config.RootDir)
	if err != nil {
		return err
	}
	files := make([]generatedFile, 0, len(templates))
	seen := make(map[string]bool, len(templates))
	for _, template := range templates {
		target, err := filepath.Abs(filepath.Join(template.Path(config), template.Name))
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == "." || !filepath.IsLocal(relative) {
			return fmt.Errorf("生成路径超出项目目录: %s", target)
		}
		if seen[target] {
			return fmt.Errorf("重复的生成路径: %s", target)
		}
		seen[target] = true
		content := []byte(template.Content(config))
		if filepath.Ext(template.Name) == ".go" {
			content, err = format.Source(content)
			if err != nil {
				return fmt.Errorf("格式化模板 %s: %w", target, err)
			}
		}
		if err := ensureNewDirectory(target); err != nil {
			return err
		}
		// 不沿项目目录下已有的符号链接写入其它位置。
		for parent := filepath.Dir(target); ; parent = filepath.Dir(parent) {
			relative, err := filepath.Rel(root, parent)
			if err != nil {
				return err
			}
			if relative == "." {
				break
			}
			info, statErr := os.Lstat(parent)
			if statErr == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
				return fmt.Errorf("生成目录不是普通目录: %s", parent)
			}
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return fmt.Errorf("检查生成目录 %s: %w", parent, statErr)
			}
		}
		files = append(files, generatedFile{path: target, content: content})
	}
	// 计划中的文件不能同时充当其它文件的父目录，也不能指向同一目标。
	for i, left := range files {
		for j, right := range files {
			if i == j {
				continue
			}
			relative, err := filepath.Rel(left.path, right.path)
			if err == nil && filepath.IsLocal(relative) {
				return fmt.Errorf("生成路径冲突: %s 与 %s", left.path, right.path)
			}
		}
	}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.path), 0755); err != nil {
			return fmt.Errorf("创建目录 %s: %w", filepath.Dir(file.path), err)
		}
		destination, err := os.OpenFile(file.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("创建文件 %s: %w", file.path, err)
		}
		_, writeErr := destination.Write(file.content)
		closeErr := destination.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return fmt.Errorf("写入或关闭文件 %s: %w", file.path, err)
		}
	}
	return nil
}

// readProject 读取当前项目根目录的 go.mod，查找声明 Server 的配置包。
// 不向上搜索，create-app 必须在 create-project 生成的项目根目录执行。
func readProject(directory string) (root, modulePath, settingsImport, settingsPackage string, err error) {
	root, err = filepath.Abs(directory)
	if err != nil {
		return
	}
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("当前目录不存在 go.mod；create-app 必须在项目根目录执行，请在该目录运行命令，使用 goi help create-app 查看流程: %w", err)
		}
		return
	}
	modulePath, err = readModulePath(content)
	if err != nil {
		err = fmt.Errorf("读取 %s: %w", filepath.Join(root, "go.mod"), err)
		return
	}
	err = filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			if filename != root {
				if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || entry.Name() == "node_modules" {
					return filepath.SkipDir
				}
				if _, statErr := os.Stat(filepath.Join(filename, "go.mod")); statErr == nil {
					return filepath.SkipDir
				} else if !errors.Is(statErr, os.ErrNotExist) {
					return statErr
				}
			}
			return nil
		}
		if entry.Name() != "settings.go" {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), filename, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("读取配置文件 %s: %w", filename, parseErr)
		}
		if file.Name.Name == "main" {
			return nil
		}
		found := false
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, specification := range group.Specs {
				values := specification.(*ast.ValueSpec)
				for _, name := range values.Names {
					found = found || name.Name == "Server"
				}
			}
		}
		if !found {
			return nil
		}
		if nameErr := validateName(file.Name.Name); nameErr != nil {
			return fmt.Errorf("配置包 %s: %w", filename, nameErr)
		}
		if settingsPackage != "" {
			return fmt.Errorf("存在多个声明 Server 的 settings.go，无法确定项目配置包")
		}
		settingsPackage = file.Name.Name
		relative, _ := filepath.Rel(root, filepath.Dir(filename))
		settingsImport = modulePath
		if relative != "." {
			settingsImport += "/" + filepath.ToSlash(relative)
		}
		if importErr := validateModulePath(settingsImport); importErr != nil {
			return fmt.Errorf("配置包导入路径无效: %w", importErr)
		}
		return nil
	})
	if err != nil {
		return
	}
	if settingsPackage == "" {
		err = fmt.Errorf("未找到声明 Server 的项目配置包 settings.go")
	}
	return
}

func readModulePath(content []byte) (string, error) {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		modulePath := fields[1]
		if strings.HasPrefix(modulePath, "\"") || strings.HasPrefix(modulePath, "`") {
			var err error
			modulePath, err = strconv.Unquote(modulePath)
			if err != nil {
				return "", fmt.Errorf("无效的 module 指令: %w", err)
			}
		}
		if err := validateModulePath(modulePath); err != nil {
			return "", err
		}
		return modulePath, nil
	}
	return "", fmt.Errorf("go.mod 缺少 module 指令")
}
