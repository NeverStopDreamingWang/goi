package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
)

var CreateApp = &cobra.Command{
	Use:   "create-app",
	Short: "创建应用",
	Args:  cobra.ExactArgs(1),
	RunE:  GoiCreateApp,
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
			err = os.MkdirAll(itemPath, 0666)
			if err != nil {
				return err
			}
		}

		// 创建文件
		filePath := path.Join(itemPath, itemFile.Name)
		file, err := os.Create(filePath)

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

		// 请求传参
		%sRouter.UrlPatterns("/params/<string:name>", "传参", goi.AsView{GET: TestParams})

		// 参数验证
		%sRouter.UrlPatterns("/params_valid", "参数验证", goi.AsView{POST: TestParamsValid})

	}
}
`
		return fmt.Sprintf(content, appName, projectName, projectName, appName, projectName, appName, appName, appName, appName)
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
	"fmt"
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
)

func TestView(request *goi.Request) interface{} {

	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status: http.StatusOK,
			Message:    "ok",
			Results:   nil,
		},
	}
}

// 请求传参
func TestParams(request *goi.Request) interface{} {
	var name string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("name", &name) // 路由传参
	// validationErr = request.QueryParams.Get("name", &name) // Query 传参
	// validationErr = request.BodyParams.Get("name", &name) // Body 传参
	if validationErr != nil {
		return validationErr.Response()
	}
	msg := fmt.Sprintf("参数: %v 参数类型:  %T", name, name)
	fmt.Println(msg)
	return goi.Response{
		Status: http.StatusCreated,               // 返回指定响应状态码 404
		Data:   goi.Data{http.StatusOK, msg, ""}, // 响应数据 null
	}
}

// 参数验证
type testParamsValidParams struct {
	Username string            ` + "`" + `name:"username" required:"string"` + "`" + `
	Password string            ` + "`" + `name:"password" required:"string"` + "`" + `
	Age      string            ` + "`" + `name:"age" required:"int"` + "`" + `
	Phone    string            ` + "`" + `name:"phone" required:"phone"` + "`" + `
	Args     []string          ` + "`" + `name:"args" optional:"slice"` + "`" + `
	Kwargs   map[string]string ` + "`" + `name:"kwargs" optional:"map"` + "`" + `
}

// required 必传参数
// optional 可选
// 支持
// int *int []*int []... map[string]*int map[...]...
// ...

func TestParamsValid(request *goi.Request) interface{} {
	var params testParamsValidParams
	var validationErr goi.ValidationError
	// validationErr = request.PathParams.ParseParams(&params) // 路由传参
	// validationErr = request.QueryParams.ParseParams(&params) // Query 传参
	validationErr = request.BodyParams.ParseParams(&params) // Body 传参
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

		// 自定义返回
		// return goi.Response{
		// 	Status: http.StatusOK,
		// 	Data: goi.Data{
		// 		Status: http.StatusBadRequest,
		// 		Message:    "参数错误",
		// 		Results:   nil,
		// 	},
		// }
	}
	fmt.Println(params)
	return goi.Response{
		Status: http.StatusOK,
		Data: goi.Data{
			Status:  http.StatusOK,
			Message: "ok",
			Results:    nil,
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
