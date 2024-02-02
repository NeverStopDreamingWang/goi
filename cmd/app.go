package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path"
	"path/filepath"
)

var CreateApp = &cobra.Command{
	Use:   "create-app",
	Short: "创建应用",
	RunE:  GoiCreateApp,
	Args:  cobra.ExactArgs(1),
}

func init() {
	GoiCmd.AddCommand(CreateApp) // 创建应用
}

// 创建-应用
var AppFileList = []InitFile{
	models,
	serializer,
	urls,
	views,
}

// 创建应用
func GoiCreateApp(cmd *cobra.Command, args []string) error {
	appName = args[0]

	projectName = filepath.Base(baseDir)

	for _, itemFile := range AppFileList {
		itemPath := itemFile.Path()
		_, err := os.Stat(itemPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(itemPath, 0755)
			if err != nil {
				return err
			}
		}

		// 创建文件
		filePath := path.Join(itemPath, itemFile.Name)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)

		Content := itemFile.Content()
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
	Content: func() string {
		content := `package %s

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
`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return path.Join(baseDir, appName)
	},
}

// 序列化器
var serializer = InitFile{
	Name: "serializer.go",
	Content: func() string {
		content := `package %s

`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return path.Join(baseDir, appName)
	},
}

// 路由
var urls = InitFile{
	Name: "urls.go",
	Content: func() string {
		content := `package %s

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
`
		return fmt.Sprintf(content, appName, projectName, appName, projectName, appName)
	},
	Path: func() string {
		return path.Join(baseDir, appName)
	},
}

// 序列化器
var views = InitFile{
	Name: "views.go",
	Content: func() string {
		content := `package %s

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
`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return path.Join(baseDir, appName)
	},
}
