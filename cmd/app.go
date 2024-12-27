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
	// 	"github.com/NeverStopDreamingWang/goi"
	// 	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/mysql"
)

func init() {
	// MySQL 数据库
	// mysqlDB := db.MySQLConnect("default")
	// mysqlDB.Migrate("test_goi", TestModel{})
	
	// sqlite 数据库
	// sqlite3DB := db.SQLite3Connect("sqlite")
	// SQLite3DB.Migrate("test_goi", TestSqliteModel{})
}


// 模型
type MyModel struct {
	Id          *int64  ` + "`" + `field_name:"id" field_type:"BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT 'ID'" json:"id"` + "`" + `
	Name        *string ` + "`" + `field_name:"name" field_type:"VARCHAR(255) NOT NULL UNIQUE COMMENT '名称'" json:"name"` + "`" + `
	Create_time *string ` + "`" + `field_name:"create_time" field_type:"DATETIME NOT NULL COMMENT '创建时间'" json:"create_time"` + "`" + `
	Update_time *string ` + "`" + `field_name:"update_time" field_type:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"update_time"` + "`" + `
}


// 设置表配置
func (myModel MyModel) ModelSet() *mysql.MySQLSettings {
	modelSettings := &mysql.MySQLSettings{
		MigrationsHandler: model.MigrationsHandler{ // 迁移时处理函数
			BeforeHandler: nil,  // 迁移之前处理函数
			AfterHandler:  nil,  // 迁移之后处理函数
		},

		TABLE_NAME:      "tb_",                // 设置表名
		ENGINE:          "InnoDB",             // 设置存储引擎，默认: InnoDB
		AUTO_INCREMENT:  1,                    // 设置自增长起始值
		DEFAULT_CHARSET: "utf8mb4",            // 设置默认字符集，如: utf8mb4
		COLLATE:         "utf8mb4_0900_ai_ci", // 设置校对规则，如 utf8mb4_0900_ai_ci;
		MIN_ROWS:        0,                    // 设置最小行数
		MAX_ROWS:        0,                    // 设置最大行数
		CHECKSUM:        0,                    // 表格的校验和算法，如 1 开启校验和
		DELAY_KEY_WRITE: 0,                    // 控制非唯一索引的写延迟，如 1
		ROW_FORMAT:      "",                   // 设置行的存储格式，如 DYNAMIC, COMPACT, FULL.
		DATA_DIRECTORY:  "",                   // 设置数据存储目录
		INDEX_DIRECTORY: "",                   // 设置数据存储目录
		PARTITION_BY:    "",                   // 定义分区方式，如 RANGE、HASH、LIST
		COMMENT:         "表",                 // 设置表注释

		// 自定义配置
		Settings: model.Settings{},
	}

	return modelSettings
}
`
		return fmt.Sprintf(content, appName)
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
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/serializer/mysql"
)

type MyModelSerializer struct {
	mysql.ModelSerializer

	// 实例
	Instance *MyModel
}

func (serializer MyModelSerializer) Validate(mysqlDB *db.MySQLDB, attrs MyModel, partial bool) goi.ValidationError {
	// 调用 ModelSerializer.Validate
	validationErr := serializer.ModelSerializer.Validate(mysqlDB, attrs, partial)
	if validationErr != nil {
		return validationErr
	}

	// 自定义验证
	mysqlDB.SetModel(attrs)

	if serializer.Instance != nil {
		mysqlDB.Where("` + "`" + `id` + "`" + ` != ?", serializer.Instance.Id)
	}

	if attrs.Name != nil {
		flag, err := mysqlDB.Where("` + "`" + `name` + "`" + ` = ?", attrs.Name).Count()
		if err != nil {
			return goi.NewValidationError(http.StatusBadRequest, "查询数据库错误")
		}
		if flag > 0 {
			return goi.NewValidationError(http.StatusBadRequest, "名称重复")
		}
	}
	return nil
}

func (serializer MyModelSerializer) Create(mysqlDB *db.MySQLDB, validated_data *MyModel) goi.ValidationError {
	// 调用 serializer.Validate
	validationErr := serializer.Validate(mysqlDB, *validated_data, false)
	if validationErr != nil {
		return validationErr
	}

	mysqlDB.SetModel(*validated_data)
	result, err := mysqlDB.Insert(*validated_data)
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "添加失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "添加失败")
	}
	validated_data.Id = &id
	return nil
}

func (serializer MyModelSerializer) Update(mysqlDB *db.MySQLDB, instance *MyModel, validated_data *MyModel) goi.ValidationError {
	var err error

	// 设置当前实例
	serializer.Instance = instance

	// 调用 serializer.Validate
	validationErr := serializer.Validate(mysqlDB, *validated_data, true)
	if validationErr != nil {
		return validationErr
	}

	mysqlDB.SetModel(*instance)

	_, err = mysqlDB.Where("` + "`" + `id` + "`" + ` = ?", serializer.Instance.Id).Update(validated_data)
	if err != nil {
		return goi.NewValidationError(http.StatusInternalServerError, "修改失败")
	}

	// 更新 instance
	serializer.ModelSerializer.Update(instance, validated_data)
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
	// 子路由
	%sRouter := %s.Server.Router.Include("%s/", "父路由")
	{
		// 注册一个路径
		%sRouter.UrlPatterns("", "获取列表/创建", goi.AsView{GET: listView, POST: createView})
		%sRouter.UrlPatterns("<int:pk>", "详情/更新/删除", goi.AsView{GET: retrieveView, PUT: updateView, DELETE: deleteView})

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
	Page int            ` + "`" + `name:"page" required:"int"` + "`" + `
	Page_Size int            ` + "`" + `name:"page_size" required:"int"` + "`" + `
}
func listView(request *goi.Request) interface{} {
	var params listValidParams
	var queryParams goi.ParamsValues
	var validationErr goi.ValidationError

	queryParams = request.QueryParams() // Query 传参
	validationErr = queryParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}
	
	return goi.Data{
		Code: http.StatusOK,
		Message:    "",
		Results:   nil,
	}
}

// 参数验证
type createValidParams struct {
	Name string            ` + "`" + `name:"name" required:"string"` + "`" + `
}
func createView(request *goi.Request) interface{} {
	var params createValidParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	
	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return validationErr.Response()
	}
	
	return goi.Data{
		Code:  http.StatusOK,
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
		Code:  http.StatusOK,
		Message: "",
		Results: "",
	}
}

// 参数验证
type updateValidParams struct {
	Name *string            ` + "`" + `name:"name" optional:"string"` + "`" + `
}
func updateView(request *goi.Request) interface{} {
	var pk int64
	var params updateValidParams
	var bodyParams goi.BodyParamsValues
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
		Code:  http.StatusOK,
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
		Code:  http.StatusOK,
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
