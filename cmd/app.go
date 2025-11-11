package main

import (
	"fmt"
	"os"
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
			err = os.MkdirAll(itemPath, 0755)
			if err != nil {
				return err
			}
		}

		// 创建文件
		filePath := filepath.Join(itemPath, itemFile.Name)
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
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/sqlite3"
)

func init() {
	// sqlite 数据库
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.Migrate("%s", MyModel{})
}


// 模型
type MyModel struct {
	Id          *int64  ` + "`" + `field_name:"id" field_type:"BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT 'ID'" json:"id"` + "`" + `
	Name        *string ` + "`" + `field_name:"name" field_type:"VARCHAR(255) NOT NULL UNIQUE COMMENT '名称'" json:"name"` + "`" + `
	Create_time *string ` + "`" + `field_name:"create_time" field_type:"DATETIME NOT NULL COMMENT '创建时间'" json:"create_time"` + "`" + `
	Update_time *string ` + "`" + `field_name:"update_time" field_type:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"update_time"` + "`" + `
}


// 设置表配置
func (self MyModel) ModelSet() *sqlite3.Settings {
	modelSettings := &sqlite3.Settings{
		MigrationsHandler: sqlite3.MigrationsHandler{ // 迁移时处理函数
			BeforeHandler: nil,  // 迁移之前处理函数
			AfterHandler:  nil,  // 迁移之后处理函数
		},

		TABLE_NAME: "user_tb", // 设置表名

		Settings: goi.Params{},
	}

	return modelSettings
}
`
		return fmt.Sprintf(content, appName, appName)
	},
	Path: func() string {
		return filepath.Join(baseDir, appName)
	},
}

// 序列化器
var serializer = InitFile{
	Name: "serializer.go",
	Content: func() string {
		content := `package %s

import (
	"errors"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/sqlite3"
)

func (self MyModel) Validate() error {
	// 自定义验证
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)

	if self.Id != nil {
		sqlite3DB = sqlite3DB.Where("` + "`" + `id` + "`" + ` != ?", self.Id)
	}

	if self.Name != nil {
		flag, err := sqlite3DB.Where("` + "`" + `name` + "`" + ` = ?", self.Name).Count()
		if err != nil {
			return errors.New("查询数据库错误")
		}
		if flag > 0 {
			return errors.New("名称重复")
		}
	}
	return nil
}

func (self *MyModel) Create() error {
	if self.Create_time == nil {
		Create_time := goi.GetTime().Format(time.DateTime)
		self.Create_time = &Create_time
	}

	err := sqlite3.Validate(self, true)
	if err != nil {
		return err
	}
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	result, err := sqlite3DB.Insert(self)
	if err != nil {
		return errors.New("添加失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return errors.New("添加失败")
	}
	self.Id = &id
	return nil
}

func (self *MyModel) Update(validated_data *MyModel) error {
	Update_time := goi.GetTime().Format(time.DateTime)
	validated_data.Update_time = &Update_time

	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	_, err := sqlite3DB.Where("` + "`" + `id` + "`" + ` = ?", self.Id).Update(validated_data)
	if err != nil {
		return errors.New("修改失败")
	}

	// 更新 instance
	return nil
}

`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return filepath.Join(baseDir, appName)
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
	%sRouter := %s.Server.Router.Include("%s/", "父路由")
	{
		%sRouter.Path("", "查询列表/新增", goi.ViewSet{GET: listView, POST: createView})
		%sRouter.Path("<int:pk>", "详情/修改/删除", goi.ViewSet{GET: retrieveView, PUT: updateView, DELETE: deleteView})

	}
}
`
		return fmt.Sprintf(content, appName, projectName, projectName, appName, projectName, appName, appName, appName)
	},
	Path: func() string {
		return filepath.Join(baseDir, appName)
	},
}

// 序列化器
var views = InitFile{
	Name: "views.go",
	Content: func() string {
		content := `package %s

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
)

// 参数验证
type listValidParams struct {
	Page      int ` + "`" + `name:"page" type:"int" required:"true"` + "`" + `
	Page_Size int ` + "`" + `name:"page_size" type:"int" required:"true"` + "`" + `
}

func listView(request *goi.Request) interface{} {
	var params listValidParams
	var queryParams goi.Params
	var validationErr goi.ValidationError

	queryParams = request.QueryParams() // Query 传参
	validationErr = queryParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "",
		Results: nil,
	}
}

// 参数验证
type createValidParams struct {
	Name string ` + "`" + `name:"name" type:"string" required:"true"` + "`" + `
}

func createView(request *goi.Request) interface{} {
	var params createValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "创建成功",
		Results: "",
	}
}

func retrieveView(request *goi.Request) interface{} {
	var pk int64
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "",
		Results: "",
	}
}

// 参数验证
type updateValidParams struct {
	Name *string ` + "`" + `name:"name" type:"string"` + "`" + `
}

func updateView(request *goi.Request) interface{} {
	var pk int64
	var params updateValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	validationErr = request.PathParams.Get("pk", &pk)
	if validationErr != nil {
		return validationErr.Response()
	}

	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "修改成功",
		Results: "",
	}
}

func deleteView(request *goi.Request) interface{} {
	var pk int64
	var validationErr goi.ValidationError

	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	return goi.Data{
		Code:    http.StatusOK,
		Message: "删除成功",
		Results: nil,
	}
}

`
		return fmt.Sprintf(content, appName)
	},
	Path: func() string {
		return filepath.Join(baseDir, appName)
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
		return filepath.Join(baseDir, appName)
	},
}
