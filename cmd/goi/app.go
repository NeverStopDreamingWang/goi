package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var CreateApp = &cobra.Command{
	Use:   "create-app [path]<app-name>",
	Short: "创建应用",
	Long: "在项目根目录创建应用。当前目录必须存在 go.mod，不会向上查找。\n" +
		"[path] 为可选的相对父路径前缀，包含当前系统的目录分隔符；<app-name> 为必填应用包名。\n" +
		"方括号和尖括号仅用于说明，不需要输入。\n" +
		"例如 myapp 或 path/myapp。",
	Example: "  mkdir project\n  cd project\n  goi create-project path/myproject\n  goi create-app path/myapp",
	Args:    cobra.ExactArgs(1),
	RunE:    GoiCreateApp,
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
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, modulePath, settingsImport, settingsPackage, err := readProject(cwd)
	if err != nil {
		return err
	}
	appPath, appName, err := parseCreationPath(args[0])
	if err != nil {
		return err
	}
	config := createConfig{
		RootDir:        root,
		ProjectName:    settingsPackage,
		AppName:        appName,
		AppPath:        appPath,
		ModulePath:     modulePath,
		SettingsImport: settingsImport,
	}
	if err := ensureNewDirectory(filepath.Join(root, filepath.FromSlash(config.AppPath))); err != nil {
		return err
	}
	return generateFiles(config, AppFileList)
}

// model
var models = InitFile{
	Name: "models.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"context"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlite3"
)

// Migrate 在数据库配置完成后显式调用，创建当前应用的数据表。
func Migrate(ctx context.Context) {
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.Migrate(ctx, MyModel{})
}

// 模型
type MyModel struct {
	Id         *int64  ` + "`" + `field_name:"id" field_type:"INTEGER PRIMARY KEY AUTOINCREMENT" comment:"ID" json:"id"` + "`" + `
	Name       *string ` + "`" + `field_name:"name" field_type:"TEXT NOT NULL UNIQUE" comment:"名称" json:"name"` + "`" + `
	CreateTime *string ` + "`" + `field_name:"create_time" field_type:"DATETIME NOT NULL" comment:"创建时间" json:"create_time"` + "`" + `
	UpdateTime *string ` + "`" + `field_name:"update_time" field_type:"DATETIME DEFAULT NULL" comment:"更新时间" json:"update_time"` + "`" + `
}

// 设置表配置
func (self MyModel) ModelSet() *sqlite3.Settings {
	modelSettings := &sqlite3.Settings{
		MigrationsHandler: sqlite3.MigrationsHandler{ // 迁移时处理函数
			BeforeHandler: nil, // 迁移之前处理函数
			AfterHandler:  nil, // 迁移之后处理函数
		},

		TableName: "%s", // 设置表名

		Settings: goi.Params{},
	}

	return modelSettings
}
`
		return fmt.Sprintf(content, config.AppName, config.AppName+"_my_model")
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.AppPath))
	},
}

// 序列化器
var serializer = InitFile{
	Name: "serializer.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/sqlite3"
)

func (self MyModel) Validate(ctx context.Context) error {
	// 自定义验证
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)

	if self.Id != nil {
		sqlite3DB = sqlite3DB.Where("` + "`" + `id` + "`" + ` != ?", self.Id)
	}

	if self.Name != nil {
		flag, err := sqlite3DB.Where("` + "`" + `name` + "`" + ` = ?", self.Name).Count(ctx)
		if err != nil {
			return fmt.Errorf("查询数据库失败: %%w", err)
		}
		if flag > 0 {
			return errors.New("名称重复")
		}
	}
	return nil
}

func (self *MyModel) Create(ctx context.Context) error {
	if self == nil {
		return errors.New("待创建的模型不能为空")
	}
	if self.CreateTime == nil {
		CreateTime := goi.GetTime().Format(time.DateTime)
		self.CreateTime = &CreateTime
	}

	err := sqlite3.Validate(self, false)
	if err != nil {
		return err
	}
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	// 显式选择插入列，省略自增主键；skipNil=true 让 nil 字段使用数据库默认处理。
	result, err := sqlite3DB.Fields("Name", "CreateTime", "UpdateTime").Insert(ctx, self, true)
	if err != nil {
		return fmt.Errorf("添加失败: %%w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取新增记录 ID 失败: %%w", err)
	}
	self.Id = &id
	return nil
}

func (self *MyModel) Update(ctx context.Context, validatedData *MyModel) error {
	if self == nil {
		return errors.New("待更新的模型不能为空")
	}
	if validatedData == nil {
		return errors.New("更新数据不能为空")
	}
	if self.Id == nil {
		return errors.New("待更新模型的 ID 不能为空")
	}
	UpdateTime := goi.GetTime().Format(time.DateTime)
	validatedData.UpdateTime = &UpdateTime

	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	// 部分更新：Name 为 nil 时保留原值，不覆盖 NOT NULL 列。
	_, err := sqlite3DB.Fields("Name", "UpdateTime").Where("` + "`" + `id` + "`" + ` = ?", self.Id).Update(ctx, validatedData, true)
	if err != nil {
		return fmt.Errorf("修改失败: %%w", err)
	}

	// 更新 instance
	return nil
}

func (self MyModel) Delete(ctx context.Context) error {
	if self.Id == nil {
		return errors.New("待删除模型的 ID 不能为空")
	}
	sqlite3DB := db.Connect[*sqlite3.Engine]("default")
	sqlite3DB.SetModel(self)
	_, err := sqlite3DB.Where("` + "`" + `id` + "`" + ` = ?", self.Id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("删除失败: %%w", err)
	}
	return nil
}

`
		return fmt.Sprintf(content, config.AppName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.AppPath))
	},
}

// 路由
var urls = InitFile{
	Name: "urls.go",
	Content: func(config createConfig) string {
		frameworkName, frameworkAlias := "goi", ""
		if config.ProjectName == frameworkName {
			// 保留配置包的真实名称，仅在重名时为框架包设置别名。
			frameworkName, frameworkAlias = "goiFramework", "goiFramework "
		}
		content := `package %[1]s

import (
	"%[2]s"
	%[3]s"github.com/NeverStopDreamingWang/goi/v3"
)

func init() {
	%[1]sRouter := %[4]s.Server.Router.Include("%[1]s/", "父路由")
	{
		%[1]sRouter.Path("", "查询列表/新增", %[5]s.ViewSet{GET: listView, POST: createView})
		%[1]sRouter.Path("<int:pk>", "详情/修改/删除", %[5]s.ViewSet{GET: retrieveView, PUT: updateView, DELETE: deleteView})

	}
}
`
		return fmt.Sprintf(content, config.AppName, config.SettingsImport, frameworkAlias, config.ProjectName, frameworkName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.AppPath))
	},
}

// 序列化器
var views = InitFile{
	Name: "views.go",
	Content: func(config createConfig) string {
		content := `package %s

import (
	"net/http"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/response"
)

// 参数验证
type listValidParams struct {
	Page     int64 ` + "`" + `name:"page" type:"int" required:"true"` + "`" + `
	PageSize int64 ` + "`" + `name:"page_size" type:"int" required:"true"` + "`" + `
}

func listView(request *goi.Request) any {
	var params listValidParams
	var queryParams goi.Params
	var validationErr goi.ValidationError

	queryParams = request.QueryParams() // Query 传参
	validationErr = queryParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	return response.Data{
		Code:    http.StatusOK,
		Message: "",
		Data: nil,
	}
}

// 参数验证
type createValidParams struct {
	Name *string ` + "`" + `name:"name" type:"string" required:"true"` + "`" + `
}

func createView(request *goi.Request) any {
	var params createValidParams
	var bodyParams goi.Params
	var validationErr goi.ValidationError

	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}

	return response.Data{
		Code:    http.StatusOK,
		Message: "创建成功",
		Data: "",
	}
}

func retrieveView(request *goi.Request) any {
	var pk int64
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	return response.Data{
		Code:    http.StatusOK,
		Message: "",
		Data: "",
	}
}

// 参数验证
type updateValidParams struct {
	Name *string ` + "`" + `name:"name" type:"string"` + "`" + `
}

func updateView(request *goi.Request) any {
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

	return response.Data{
		Code:    http.StatusOK,
		Message: "修改成功",
		Data: "",
	}
}

func deleteView(request *goi.Request) any {
	var pk int64
	var validationErr goi.ValidationError

	validationErr = request.PathParams.Get("pk", &pk) // 路由传参
	if validationErr != nil {
		return validationErr.Response()
	}

	return response.Data{
		Code:    http.StatusOK,
		Message: "删除成功",
		Data: nil,
	}
}

`
		return fmt.Sprintf(content, config.AppName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.AppPath))
	},
}

// 工具
var utils = InitFile{
	Name: "utils.go",
	Content: func(config createConfig) string {
		content := `package %s

`
		return fmt.Sprintf(content, config.AppName)
	},
	Path: func(config createConfig) string {
		return filepath.Join(config.RootDir, filepath.FromSlash(config.AppPath))
	},
}
