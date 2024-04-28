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
	utils,
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

// model
var models = InitFile{
	Name: "models.go",
	Content: func() string {
		content := `package %s

import (
	// "github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	// "github.com/NeverStopDreamingWang/goi/model"
)

func init() {
	// MySQL 数据库
	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	if err != nil {
		panic(err)
	}
	defer mysqlObj.Close()
	// mysqlObj.Migrate(TestModel{})
	
	// sqlite 数据库
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	if err != nil {
		panic(err)
	}
	defer SQLite3DB.Close()
	// SQLite3DB.Migrate(TestSqliteModel{})
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
	"%s/%s"
	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 子路由
	%sRouter := %s.Server.Router.Include("/%s", "父路由")
	{
		// 注册一个路径
		%sRouter.UrlPatterns("/test_views", "测试接口", goi.AsView{GET: TestView})
	}
}
`
		return fmt.Sprintf(content, appName, projectName, projectName, appName, projectName, appName, appName)
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

func TestView(request *goi.Request) interface{} {

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

// 工具
var utils = InitFile{
	Name: "utils.go",
	Content: func() string {
		content := `package %s

`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return path.Join(baseDir, appName)
	},
}
