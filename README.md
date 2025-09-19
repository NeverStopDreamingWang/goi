# goi

基于 `net/http` 开发的一个 Web 框架，语法与 `Django` 类似，简单易用，快速上手，详情请查看示例

[详细示例：example](./example)

## goi 创建命令

使用 `go env GOMODCACHE` 获取 go 软件包路径：`mypath\Go\pkg\mod` + `github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe`

使用 `go env GOROOT` 获取 go 安装路径 `mypath\Go` + `bin`
将可执行文件复制到 Go\bin 目录下

**Windows**: `copy mypath\Go\pkg\mod\github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe mypath\Go\bin\goi.exe`

编译

```cmd
go build -p=1 -o goi.exe
```

**Linux**: `cp mypath\go\pkg\mod\github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi mypath\go\bin\goi`

编译

```cmd
go build -o goi
```

```shel
> goi

Usage（用法）:                                         
        goi <command> [arguments]                      
The commands are（命令如下）:                          
        create-project  myproject   创建项目       
        create-app      myapp       创建app

```

示例

```shell
# 创建项目
> goi create-project example

# 新建应用 app
> cd example
> goi create-app myapp

```

## 路由

### 多级路由

子父路由，内置路由转换器，自定义路由转换器

* `Router.Include(path string, desc string)` 创建一个子路由
    * `path` 路由
    * `desc` 描述

* `Router.Path(path string, desc string, view ViewSet)` 注册一个路由
    * `path` 路由
    * `desc` 描述
  * `view` 视图 `ViewSet` 类型
        * `GET` `HandlerFunc`
        * `HEAD`    `HandlerFunc`
        * `POST`    `HandlerFunc`
        * `PUT`     `HandlerFunc`
        * `PATCH`   `HandlerFunc`
        * `DELETE`  `HandlerFunc`
        * `CONNECT` `HandlerFunc`
        * `OPTIONS` `HandlerFunc`
        * `TRACE`   `HandlerFunc`

### 静态路由

静态路由映射，返回文件对象即响应该文件内容

静态路由映射可通过两种方式注册：

方式一：Router.* 辅助方法（内部基于公共 ViewSet 生成器）

* `Router.StaticFile(path string, desc string, filePath string)` 注册静态文件路由
    * `path` 路由
    * `desc` 描述
  * `filePath` 文件路径，支持`相对路径`和`绝对路径`

* `Router.StaticDir(path string, desc string, dirPath http.Dir)` 注册静态目录路由
    * `path` 路由
    * `desc` 描述
  * `dirPath` 静态映射路径，支持`相对路径`和`绝对路径`

* `Router.StaticFileFS(path string, desc string, fileFS embed.FS, defaultPath string)` 注册 `embed.FS` 静态文件路由
    * `path` 路由
    * `desc` 描述
  * `fileFS` 嵌入文件路径，支持`相对路径`和`绝对路径`
  * `defaultPath` 默认文件路径


* `Router.StaticDirFS(path string, desc string, dirFS embed.FS)` 注册 `embed.FS` 静态目录路由
    * `path` 路由
    * `desc` 描述
  * `dirFS` 嵌入目录路径，支持`相对路径`和`绝对路径`

### 静态 View 生成器

提供四个公共方法生成静态资源的 HandlerFunc，配合 `Router.Path` 使用：

- `goi.StaticFileView(filePath string) HandlerFunc`
- `goi.StaticDirView(dirPath http.Dir) HandlerFunc`
- `goi.StaticFileFSView(fs embed.FS) HandlerFunc`
- `goi.StaticDirFSView(fs embed.FS) HandlerFunc`

示例：

```go
// 单文件
router.Path("index.html", "首页", goi.ViewSet{GET: goi.StaticFileView("template/html/index.html"),})

// 目录（带路由参数 <path:fileName>）
router.Path("static/<path:fileName>", "静态目录", goi.ViewSet{GET: goi.StaticDirView(http.Dir("public")), })

// 嵌入式单文件 / 目录
router.Path("logo.svg", "静态FS文件", goi.ViewSet{GET: goi.StaticFileFSView(assets.LogoFS, "logo.svg"), })
router.Path("assets/", "静态FS目录", goi.ViewSet{GET: goi.StaticDirFSView(assets.AssetsFS), })
```

## 中间件

框架采用接口式中间件，按职责可实现以下任意方法（可选实现）：

```go
// 中间件
// MiddleWare 定义四个阶段钩子：
// - ProcessRequest：请求阶段（顺序执行）；返回非 nil 则短路，内层不再进入
// - ProcessView：路由解析后、视图执行前（顺序执行）；返回非 nil 则短路，但响应阶段仍会全链执行
// - ProcessException：异常阶段（逆序执行）；
// - ProcessResponse：响应阶段（逆序执行、全链）；常用于追加/修改响应头
type MiddleWare interface {
ProcessRequest(*Request) interface{}                // 请求中间件
ProcessView(*Request) interface{}                   // 视图中间件
ProcessException(*Request, interface{}) interface{} // 异常中间件
ProcessResponse(*Request, *Response)                // 响应中间件
}
```

注册：

```go
func init() {
example.Server.MiddleWare = append(example.Server.MiddleWare, &MyMiddleware{})
}
```

执行顺序：

- 请求阶段（ProcessRequest）：正序执行；返回 nil 则继续执行，否则返回结果
- 视图阶段（ProcessView）：正序执行；返回 nil 则继续执行，否则返回结果
- 异常阶段（ProcessException）：逆序执行；返回 nil 则继续执行，否则返回结果
- 响应阶段（ProcessResponse）：逆序执行；可对视图结果进行包装/替换

异常处理：

- 若所有异常中间件均返回 nil，将由框架统一返回 500 响应

## 日志模块

* **支持三种日志**
    * 信息日志：`Log.INFO_OUT_PATH`
    * 访问日志：`Log.ACCESS_OUT_PATH`
    * 错误日志：`Log.ERROR_OUT_PATH ERROR`
* **日志等级**：是否开启 `Log.DEBUG = true`
    * DEBUG
        * 使用：`goi.Debug()`
        * 输出：Console（终端）、INFO 信息日志、ACCESS 访问日志、ERROR 错误日志
    * INFO
        * 使用：`goi.Info()`
        * 输出：Console（终端）、INFO 信息日志、ACCESS 访问日志
    * WARNING
        * 使用：`goi.Warning()`
        * 输出：Console（终端）、INFO 信息日志
    * ERROR
        * 使用：`goi.Error()`
        * 输出：Console（终端）、INFO 信息日志、ERROR 错误日志
    * MeteLog
        * 使用：`goi.MetaLog()`
        * 输出：Console（终端）、INFO 信息日志、ACCESS 访问日志、ERROR 错误日志
* **日志切割**：两种可同时设置满足任意一种即切割
    * 按照日志大小：`Log.SplitSize`
        * 例：`Log.SplitSize = 1024 * 1024`
    * 按照日期格式：`Log.SplitTime = "2006-01-02"`
        * 例：`Log.SplitTime = "2006-01-02"` 设置日期格式

## 缓存

* 支持过期策略，默认惰性删除
    * PERIODIC（定期删除：默认每隔 1s 就随机抽取一些设置了过期时间的 key，检查其是否过期）
    * SCHEDULED（定时删除：某个设置了过期时间的 key，到期后立即删除）
* 支持内存淘汰策略
    * NOEVICTION（直接抛出错误）
    * ALLKEYS_RANDOM（随机删除-所有键）
    * ALLKEYS_LRU（删除最近最少使用-所有键）
    * ALLKEYS_LFU（删除最近最不频繁使用-所有键）
    * VOLATILE_RANDOM（随机删除-设置过期时间的键）
    * VOLATILE_LRU（删除最近最少使用-设置过期时间的键）
    * VOLATILE_LFU（删除最近最不频繁使用-设置过期时间的键）
    * VOLATILE_TTL（删除即将过期的键-设置过期时间的键）

## 内置 auth 密码生成与验证

```go
package auth_test

import (
	"fmt"

	"github.com/NeverStopDreamingWang/goi/auth"
)

func ExampleMakePassword() {
	// 原始密码
	password := "goi123456"

	// 生成密码的哈希值
	hashedPassword, err := auth.MakePassword(password)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("原始密码: %s\n", password)

	// 验证密码
	isValid := auth.CheckPassword(password, hashedPassword)
	fmt.Printf("密码验证: %v\n", isValid)

	// Output:
	//
	// 原始密码: goi123456
	// 密码验证: true
}
```

## 内置 Converter 路由转换器

注册路由转换器

* 包含参数验证
* 包含参数解析

功能：

实现 `goi.Converter` 结构体

```go
package goi_test

import (
	"github.com/NeverStopDreamingWang/goi"
)

// 手机号
var phoneConverter = goi.Converter{
	Regex: `(1[3456789]\d{9})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 邮箱
var emailConverter = goi.Converter{
	Regex: `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// URL
var urlConverter = goi.Converter{
	Regex: `(https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|www\.[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9]+\.[^\s]{2,}|www\.[a-zA-Z0-9]+\.[^\s]{2,})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 日期 (YYYY-MM-DD)
var dateConverter = goi.Converter{
	Regex: `(\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 时间 (HH:MM:SS)
var timeConverter = goi.Converter{
	Regex: `((?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d)`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// IP地址 (IPv4)
var ipv4Converter = goi.Converter{
	Regex: `((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

// 用户名 (字母开头，允许字母数字下划位)
var usernameConverter = goi.Converter{
	Regex: `([a-zA-Z][a-zA-Z0-9_]{3,15})`,
	ToGo:  func(value string) (interface{}, error) { return value, nil },
}

func ExampleRegisterConverter() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter("my_phone", phoneConverter)

	// 邮箱
	goi.RegisterConverter("my_email", emailConverter)

	// URL
	goi.RegisterConverter("my_url", urlConverter)

	// 日期 (YYYY-MM-DD)
	goi.RegisterConverter("my_date", dateConverter)

	// 时间 (HH:MM:SS)
	goi.RegisterConverter("my_time", timeConverter)

	// IP地址 (IPv4)
	goi.RegisterConverter("my_ipv4", ipv4Converter)

	// 用户名 (字母开头，允许字母数字下划位)
	goi.RegisterConverter("my_username", usernameConverter)
}
```

使用

```go
package goi_test

import (
	"goi_test/goi_test"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
	// 创建一个子路由
	testRouter := goi_test.Server.Router.Include("test/", "测试路由")
	{
		// 注册一个路径 
		// 类型 my_phone
		// 名称 phone
		testRouter.Path("test_phone/<my_phone:phone>", "", goi.ViewSet{GET: TestPhone}) // 测试路由转换器
	}
}

// 测试手机号路由转换器
func TestPhone(request *goi.Request) interface{} {
	var phone string
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("phone", &phone)
	if validationErr != nil {
		return validationErr.Response()
	}
	resp := map[string]interface{}{
		"status": http.StatusOK,
		"msg":    phone,
		"data":   "OK",
	}
	return resp
}

```

## 内置 Validator 参数验证器

* 包含参数验证
* 包含参数解析

实现 `goi.Validate` 接口

```go
package goi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/NeverStopDreamingWang/goi"
)

// phoneValidator 手机号验证器示例
type phoneValidator struct{}

func (validator phoneValidator) Validate(value interface{}) goi.ValidationError {
	switch typeValue := value.(type) {
	case int:
		valueStr := strconv.Itoa(typeValue)
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if !re.MatchString(valueStr) {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	case string:
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if !re.MatchString(typeValue) {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	default:
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}
	return nil
}

func (validator phoneValidator) ToGo(value interface{}) (interface{}, goi.ValidationError) {
	switch typeValue := value.(type) {
	case int:
		return strconv.Itoa(typeValue), nil
	case string:
		return typeValue, nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// emailValidator 邮箱验证器示例
type emailValidator struct{}

func (validator emailValidator) Validate(value interface{}) goi.ValidationError {
	switch v := value.(type) {
	case string:
		re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !re.MatchString(v) {
			return goi.NewValidationError(http.StatusBadRequest, "invalid email format")
		}
		return nil
	default:
		return goi.NewValidationError(http.StatusBadRequest, "email must be string")
	}
}

// 当解析参数值类型与结构体字段类型不一致，且结构体字段类型为自定义类型时，需要实现 ToGo 方法
func (validator emailValidator) ToGo(value interface{}) (interface{}, goi.ValidationError) {
	switch v := value.(type) {
	case string:
		// 转换类型，并返回
		return Email{Address: v}, nil
	default:
		// Validate 会拦截返回
		return nil, nil
	}
}

// 支持自定义类型
type Email struct {
	Address string `json:"address"`
}
type testParamsValidParams struct {
	Phone string `name:"phone" type:"phone" required:"true"`
	Email *Email `name:"email" type:"email" required:"true"`
}

// ExampleRegisterValidate 展示如何注册和使用自定义验证器
func ExampleRegisterValidate() {
	// 注册手机号验证器
	goi.RegisterValidate("phone", phoneValidator{})

	// 注册邮箱验证器
	goi.RegisterValidate("email", emailValidator{})

	var validationErr goi.ValidationError

	var params testParamsValidParams
	bodyParams := goi.Params{
		"phone": "13800000000",
		"email": "test@example.com",
	}
	validationErr = bodyParams.ParseParams(&params)
	if validationErr != nil {
		return
	}
	fmt.Printf("Phone %+v %T\n", params.Phone, params.Phone)
	fmt.Printf("Email %+v %T\n", params.Email, params.Email)
}

func TestRegisterValidate(t *testing.T) {
	ExampleRegisterValidate()
}

```

## ORM 模型

支持：MySQL、SQLite3

```go
package sqlite3_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/sqlite3"
)

func init() {
	goi.Settings.DATABASES["default"] = &goi.DataBase{
		ENGINE: "sqlite3",
		Connect: func(ENGINE string) *sql.DB {
			var DB *sql.DB
			var err error
			DataSourceName := filepath.Join(goi.Settings.BASE_DIR, "sqlite3.db")
			DB, err = sql.Open(ENGINE, DataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return DB
		},
	}
	// 初始化测试数据库
	// 使用泛型通用连接
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 使用专用连接
	// sqliteDB := sqlite3.Connect("default")

	// 迁移模型前清理旧表(如果存在)
	_, _ = sqliteDB.Execute("DROP TABLE IF EXISTS user_tb")

	// 迁移模型
	sqliteDB.Migrate("test_db", UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
		Update_time: &now,
	}

	sqliteDB.SetModel(UserModel{})
	result, err := sqliteDB.Insert(user)
	if err != nil {
		fmt.Println("插入错误:", err)
		return
	}
	id, _ := result.LastInsertId()
	user.Id = &id
}

// 用户表模型
type UserModel struct {
	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field_name:"username" field_type:"TEXT NOT NULL" json:"username"`
	Password    *string `field_name:"password" field_type:"TEXT NOT NULL" json:"-"`
	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
}

// 设置表配置
func (userModel UserModel) ModelSet() *sqlite3.Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &sqlite3.Settings{
		MigrationsHandler: sqlite3.MigrationsHandler{
			BeforeHandler: nil,      // 迁移之前处理函数
			AfterHandler:  InitUser, // 迁移之后处理函数
		},

		TABLE_NAME: "user_tb", // 设置表名

		// 自定义配置
		Settings: goi.Params{
			"encrypt_fields": encryptFields,
		},
	}

	return modelSettings
}

// 初始化用户
func InitUser() error {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	var (
		id              int64 = 1
		username              = "超级管理员"
		password              = "admin"
		create_Datetime       = goi.GetTime().Format(time.DateTime)
		err             error
	)

	user := UserModel{
		Id:          &id,
		Username:    &username,
		Password:    &password,
		Create_time: &create_Datetime,
		Update_time: nil,
	}
	sqliteDB.SetModel(UserModel{})
	_, err = sqliteDB.Insert(&user)
	if err != nil {
		return err
	}

	return nil
}

func ExampleEngine_Migrate() {
	// 连接数据库
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 迁移模型
	sqliteDB.Migrate("test_db", UserModel{})

	// Output:
}

func ExampleEngine_Insert() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 准备测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
	}

	// 设置模型并插入数据
	sqliteDB.SetModel(UserModel{})
	result, err := sqliteDB.Insert(user)
	if err != nil {
		fmt.Println("插入错误:", err)
		return
	}

	id, _ := result.LastInsertId()
	fmt.Printf("插入成功, ID: %d\n", id)

	// Output:
	// 插入成功, ID: 1
}

func ExampleEngine_Where() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	var users []*UserModel
	sqliteDB.SetModel(UserModel{})

	// Where 返回当前实例的副本指针
	// 简单条件
	sqliteDB = sqliteDB.Where("username = ?", "test_user")

	// AND条件
	sqliteDB = sqliteDB.Where("create_time IS NULL")

	// 支持 IN 参数值为 Slice 或 Array
	testUsers := []string{"test_user", "test_user_1"}
	sqliteDB = sqliteDB.Where("username IN ?", testUsers)

	// LIKE条件
	sqliteDB = sqliteDB.Where("username LIKE ?", "where_test%")

	err := sqliteDB.Select(&users)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("查询到 %d 条记录\n", len(users))
	for _, user := range users {
		fmt.Printf("用户名: %s\n", *user.Username)
	}

	// Output:
	// 查询到 1 条记录
	// 用户名: test_user
}

func ExampleEngine_Select() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 查询多条记录 Select 支持 map 以及 结构体
	// var users []map[string]interface{}
	var users []*UserModel

	sqliteDB.SetModel(UserModel{})
	err := sqliteDB.Select(users)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	for _, user := range users {
		fmt.Printf("用户名: %s\n", *user.Username)
	}

	// Output:
	// 用户名: test_user
}

func ExampleEngine_Page() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 分页查询
	var users []*UserModel

	sqliteDB.SetModel(UserModel{})
	total, totalPages, err := sqliteDB.Page(2, 5) // 第2页,每页5条
	if err != nil {
		fmt.Println("分页错误:", err)
		return
	}

	err = sqliteDB.Select(&users)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("总记录数: %d\n", total)
	fmt.Printf("总页数: %d\n", totalPages)
	fmt.Printf("当前页记录数: %d\n", len(users))

	// Output:
	// 总记录数: 15
	// 总页数: 3
	// 当前页记录数: 5
}

func ExampleEngine_First() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 查询单条记录 First 支持 map 以及 结构体
	// var user map[string]interface{}
	var user *UserModel

	sqliteDB.SetModel(UserModel{})
	sqliteDB = sqliteDB.Where("username = ?", "test_user")
	err := sqliteDB.First(user)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("用户名: %s\n", *user.Username)

	// Output:
	// 用户名: test_user
}

func ExampleEngine_Fields() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 指定查询字段
	var user map[string]interface{}

	sqliteDB.SetModel(UserModel{})

	// Fields 返回当前实例的副本指针
	sqliteDB = sqliteDB.Fields("Username", "Create_time")
	err := sqliteDB.Select(&user)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Println("用户名:", user["username"])

	// Output:
	// 用户名: user_1
}

func ExampleEngine_Update() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 准备更新数据
	newPassword := "new_password"
	updateUser := UserModel{
		Password: &newPassword,
	}

	// 更新记录
	sqliteDB.SetModel(UserModel{})
	sqliteDB = sqliteDB.Where("username = ?", "test_user")
	result, err := sqliteDB.Update(updateUser)
	if err != nil {
		fmt.Println("更新错误:", err)
		return
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("更新成功, 影响行数: %d\n", rows)

	// Output:
	// 更新成功, 影响行数: 1
}

func ExampleEngine_Delete() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	sqliteDB.SetModel(UserModel{})
	sqliteDB = sqliteDB.Where("username = ?", "test_user")

	// 删除记录
	result, err := sqliteDB.Delete()
	if err != nil {
		fmt.Println("删除错误:", err)
		return
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("删除成功, 影响行数: %d\n", rows)

	// Output:
	// 删除成功, 影响行数: 1
}

func ExampleEngine_GroupBy() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 使用分组查询
	type Result struct {
		Date  *string `field_name:"date" json:"date"`
		Count *int    `field_name:"count" json:"count"`
	}

	var results []*Result
	sqliteDB.SetModel(UserModel{})
	// GroupBy 返回当前实例的副本指针
	sqliteDB = sqliteDB.GroupBy("DATE(create_time)")
	err := sqliteDB.Select(&results)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	for _, result := range results {
		fmt.Printf("日期: %s, 注册人数: %d\n", *result.Date, *result.Count)
	}

	// Output:
	// 日期: 2024-03-21, 注册人数: 15
}

func ExampleEngine_OrderBy() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 使用排序查询
	var users []UserModel
	sqliteDB.SetModel(UserModel{})

	// OrderBy 返回当前实例的副本指针
	sqliteDB = sqliteDB.OrderBy("-create_time")
	err := sqliteDB.Select(&users)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	for _, user := range users {
		fmt.Println("用户名:", *user.Username)
	}

	// Output:
	// 用户名: test_user
}

func ExampleEngine_Count() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 统计记录数
	sqliteDB.SetModel(UserModel{})
	count, err := sqliteDB.Count()
	if err != nil {
		fmt.Println("统计错误:", err)
		return
	}

	fmt.Printf("总记录数: %d\n", count)

	// Output:
	// 总记录数: 1
}

func ExampleEngine_Exists() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	sqliteDB.SetModel(UserModel{})
	// 检查记录是否存在
	exists, err := sqliteDB.Where("username = ?", "test_user").Exists()
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("记录是否存在: %v\n", exists)

	// Output:
	// 记录是否存在: true
}

func ExampleEngine_WithTransaction() {
	sqliteDB := db.Connect[*sqlite3.Engine]("default")

	// 事务
	err := sqliteDB.WithTransaction(func(engine *sqlite3.Engine, args ...interface{}) error {
		// 准备测试数据
		username1 := "transaction_user1"
		username2 := "transaction_user2"
		password := "test123456"
		now := time.Now().Format(time.DateTime)

		// 插入第一个用户
		user1 := UserModel{
			Username:    &username1,
			Password:    &password,
			Create_time: &now,
		}
		engine.SetModel(UserModel{})
		_, err := engine.Insert(user1)
		if err != nil {
			return err
		}

		// 插入第二个用户
		user2 := UserModel{
			Username:    &username2,
			Password:    &password,
			Create_time: &now,
		}
		_, err = engine.Insert(user2)
		if err != nil {
			return err
		}

		return nil
	})
	// 事务执行的错误，发生错误时会自动回滚
	if err != nil {
		fmt.Println("事务执行错误:", err)
		return
	}

	fmt.Println("事务执行完成")

	// Output:
	// 事务执行完成
}

```

## 内置 JWT Token

```go
package jwt_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/jwt"
)

// 自定义负载结构体
type CustomPayloads struct {
	jwt.Payloads        // 嵌入基础 Payloads 结构体
	User_id      int    `json:"user_id"`
	Username     string `json:"username"`
}

func ExampleNewJWT() {
	// 创建头部信息
	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	// 创建自定义负载
	payload := CustomPayloads{
		Payloads: jwt.Payloads{
			// 设置过期时间为2小时后
			Exp: jwt.ExpTime{Time: goi.GetTime().Add(time.Hour * 2)},
		},
		User_id:  1,
		Username: "test_user",
	}

	// 签名密钥
	key := "your-secret-key"

	// 生成 JWT 令牌
	token, err := jwt.NewJWT(header, payload, key)
	if err != nil {
		fmt.Println("生成令牌错误:", err)
		return
	}
	fmt.Println("生成的令牌成功")

	// 验证并解析令牌
	var decodedPayload CustomPayloads
	err = jwt.CkeckToken(token, key, &decodedPayload)
	if err != nil {
		fmt.Println("验证令牌错误:", err)
		return
	}

	fmt.Println("解析的用户ID:", decodedPayload.User_id)
	fmt.Println("解析的用户名:", decodedPayload.Username)

	// Output:
	// 生成的令牌成功
	// 解析的用户ID: 1
	// 解析的用户名: test_user
}

func ExampleCkeckToken() {
	// 签名密钥
	key := "your-secret-key"

	// 创建一个已过期的令牌
	header := jwt.Header{
		Alg: jwt.AlgHS256,
		Typ: jwt.TypJWT,
	}

	payload := CustomPayloads{
		Payloads: jwt.Payloads{
			// 设置过期时间为1小时前
			Exp: jwt.ExpTime{Time: goi.GetTime().Add(-time.Hour)},
		},
		User_id:  1,
		Username: "test_user",
	}

	// 生成过期的令牌
	expiredToken, _ := jwt.NewJWT(header, payload, key)

	// 尝试验证过期的令牌
	var decodedPayload CustomPayloads
	err := jwt.CkeckToken(expiredToken, key, &decodedPayload)
	if errors.Is(err, jwt.ErrExpiredSignature) {
		fmt.Println("令牌已过期")
	} else if errors.Is(err, jwt.ErrDecode) {
		fmt.Println("令牌格式错误")
	} else if err != nil {
		fmt.Println("其他错误:", err)
	}

	// 使用错误的密钥验证令牌
	wrongKey := "wrong-secret-key"
	err = jwt.CkeckToken(expiredToken, wrongKey, &decodedPayload)
	if err == jwt.ErrDecode {
		fmt.Println("签名验证失败")
	}

	// Output:
	// 令牌已过期
	// 签名验证失败
}

```

##                             
