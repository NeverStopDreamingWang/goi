package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path"
	"path/filepath"
	"reflect"
)

var CreateApp = &cobra.Command{
	Use:   "create-app",
	Short: "创建应用",
	RunE:  GoiCreateApp,
}

func init() {
	GoiCmd.AddCommand(CreateApp) // 创建应用
}

var AppFileList = []InitFile{
	models,
	serializer,
	urls,
	views,
}

// 创建项目
func GoiCreateApp(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("请输入应用名称")
	}
	appName = args[0]

	baseDir, _ := os.Getwd()

	// 创建项目目录
	appDir := path.Join(baseDir, appName)
	_, err := os.Stat(appDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(appDir, 0644)
		if err != nil {
			return err
		}
	}

	projectName = filepath.Base(baseDir)

	for _, ItemFile := range AppFileList {
		// 创建路由转换器文件
		filePath := path.Join(appDir, ItemFile.Name)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)

		for i, arg := range ItemFile.Args {
			ItemFile.Args[i] = reflect.ValueOf(arg).Elem().Interface()
		}
		Content := fmt.Sprintf(ItemFile.Content, ItemFile.Args...)
		_, err = file.WriteString(Content)
		if err != nil {
			return err
		}
	}

	return nil
}

// ORM
var models = InitFile{
	Name: "models.go",
	Content: `package %s

import (
	"github.com/NeverStopDreamingWang/goi/migrate"
	"github.com/NeverStopDreamingWang/goi/model"
)

func init() {
	// MySQL 数据库
	MySQLMigrations := model.MySQLMakeMigrations{
		DATABASES: []string{"default"},
		MODELS: []model.MySQLModel{
			
		},
	}
	migrate.MySQLMigrate(MySQLMigrations)

	// sqlite 数据库
	SQLite3Migrations := model.SQLite3MakeMigrations{
		DATABASES: []string{"sqlite_1"},
		MODELS: []model.SQLite3Model{
			
		},
	}
	migrate.SQLite3Migrate(SQLite3Migrations)
}
`,
	Args: []any{&appName},
}

// 序列化器
var serializer = InitFile{
	Name: "serializer.go",
	Content: `package %s

`,
	Args: []any{&appName},
}

// 路由
var urls = InitFile{
	Name: "urls.go",
	Content: `package %s

import (
	"%s"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 子路由
	%sRouter := %s.Server.Router.Include("/%s")
	{
		// 注册一个路径
		// testRouter.UrlPatterns("/test_views", goi.AsView{GET: TestView})
	}
}
`,
	Args: []any{&appName, &projectName, &appName, &projectName, &appName},
}

// 序列化器
var views = InitFile{
	Name: "views.go",
	Content: `package %s

import (
	"github.com/NeverStopDreamingWang/goi"
	"net/http"
)

func TestView(request *goi.Request) any {

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Msg:    "ok",
			Data:   nil,
		},
	}
}
`,
	Args: []any{&appName},
}
