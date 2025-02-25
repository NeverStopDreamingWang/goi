# goi

基于 `net/http` 进行开发的 Web 框架

[详细示例：example](./example)

## goi 创建命令

使用 `go env GOMODCACHE` 获取 go 软件包路径：`mypath\Go\pkg\mod` + `github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe`

使用 `go env GOROOT` 获取 go 安装路径 `mypath\Go` + `bin`
将可执行文件复制到 Go\bin 目录下

**Windows**: `copy mypath\Go\pkg\mod\github.com\!never!stop!dreaming!wang\goi@v版本号\goi\goi.exe mypath\Go\bin\goi.exe`

编译

```cmd
go build -o goi.exe
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

* `Router.UrlPatterns(path string, desc string, view AsView)` 注册一个路由
    * `path` 路由
    * `desc` 描述
    * `view` 视图 `AsView` 类型
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

* `Router.StaticUrlPatterns(path string, desc string, FilePath string)` 注册静态文件路由
    * `path` 路由
    * `desc` 描述
    * `FilePath` 文件路径，支持`相对路径`和`绝对路径`

* `Router.StaticDirPatterns(path string, desc string, DirPath http.Dir)` 注册静态目录路由
    * `path` 路由
    * `desc` 描述
    * `DirPath` 静态映射路径，支持`相对路径`和`绝对路径`

* `Router.StaticFilePatternsFS(path string, desc string, FileFS embed.FS)` 注册 `embed.FS` 静态文件路由
    * `path` 路由
    * `desc` 描述
  * `FileFS` 嵌入文件路径，支持`相对路径`和`绝对路径`


* `Router.StaticDirPatternsFS(path string, desc string, DirFS embed.FS)` 注册 `embed.FS` 静态目录路由
    * `path` 路由
    * `desc` 描述
  * `DirFS` 嵌入目录路径，支持`相对路径`和`绝对路径`

## 中间件

* 请求中间件：`Server.MiddleWares.BeforeRequest(processRequest RequestMiddleware)` 注册请求中间件
    * `type RequestMiddleware func(request *goi.Request) interface{}`
        * `request` 请求对象


* 视图中间件：`Server.MiddleWares.BeforeView(processView ViewMiddleware)` 注册视图中间件
    * `type ViewMiddleware func(request *goi.Request) interface{}`
        * `request` 请求对象


* 响应中间件：`Server.MiddleWares.BeforeResponse(processResponse ResponseMiddleware)` 注册响应中间件
    * `type ResponseMiddleware func(request *goi.Request, viewResponse interface{}) interface{}`
        * `request` 请求对象
        * `viewResponse` 视图响应结果

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

`converter.go` 注册路由转换器

```go
package goi_test

import "github.com/NeverStopDreamingWang/goi"

func ExampleRegisterConverter() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter("my_phone", `(1[3456789]\d{9})`)

	// 邮箱
	goi.RegisterConverter("my_email", `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

	// URL
	goi.RegisterConverter("my_url", `(https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|www\.[a-zA-Z0-9][a-zA-Z0-9-]+[a-zA-Z0-9]\.[^\s]{2,}|https?:\/\/(?:www\.|(?!www))[a-zA-Z0-9]+\.[^\s]{2,}|www\.[a-zA-Z0-9]+\.[^\s]{2,})`)

	// 日期 (YYYY-MM-DD)
	goi.RegisterConverter("my_date", `(\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))`)

	// 时间 (HH:MM:SS)
	goi.RegisterConverter("my_time", `((?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d)`)

	// IP地址 (IPv4)
	goi.RegisterConverter("my_ipv4", `((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`)

	// 用户名 (字母开头，允许字母数字下划线，4-16位)
	goi.RegisterConverter("my_username", `([a-zA-Z][a-zA-Z0-9_]{3,15})`)
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
        testRouter.UrlPatterns("test_phone/<my_phone:phone>", "", goi.AsView{GET: TestPhone}) // 测试路由转换器
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

注册参数验证器

```go
package goi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/NeverStopDreamingWang/goi"
)

func init() {
    // 注册验证器
    // 手机号
    goi.RegisterValidate("phone", phoneValidate)
}

// phone 类型
func phoneValidate(value interface{}) goi.ValidationError {
	switch typeValue := value.(type) {
	case int:
		valueStr := strconv.Itoa(typeValue)
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(valueStr) == false {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	case string:
		var reStr = `^(1[3456789]\d{9})$`
		re := regexp.MustCompile(reStr)
		if re.MatchString(typeValue) == false {
			return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
		}
	default:
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数类型错误：%v", value))
	}

	return nil
}
```

使用

```go
package test

import (
	"fmt"
	"net/http"

	"github.com/NeverStopDreamingWang/goi"
)

// 参数验证
type testParamsValidParams struct {
    Username string            `name:"username" required:"string"`
    Password string            `name:"password" required:"string"`
    Age      string            `name:"age" required:"int"`
    Phone    string            `name:"phone" required:"phone"`
    Args     []string          `name:"args" optional:"slice"`
    Kwargs   map[string]string `name:"kwargs" optional:"map"`
}

// required 必传参数
// optional 可选
// 支持
// int *int []*int []... map[string]*int map[...]...
// ...

func TestParamsValid(request *goi.Request) interface{} {
	var params testParamsValidParams
	var bodyParams goi.BodyParamsValues
	var validationErr goi.ValidationError
	bodyParams = request.BodyParams() // Body 传参
	validationErr = bodyParams.ParseParams(&params)
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
			Code:    http.StatusOK,
			Message: "ok",
			Results: nil,
		},
	}
}

```

## ORM 模型

支持：MySQL、SQLite3

```go
package db_test

import (
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/sqlite3"
)

func init() {
	// 初始化测试数据库
	sqliteDB := db.SQLite3Connect("sqlite")

	// 迁移模型前清理旧表(如果存在)
	_, _ = sqliteDB.Execute("DROP TABLE IF EXISTS user_tb")

	// 迁移模型
	sqliteDB.Migrate("test_db", UserSqliteModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format("2006-01-02 15:04:05")

	user := UserSqliteModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
		Update_time: &now,
	}

	sqliteDB.SetModel(UserSqliteModel{})
	sqliteDB.Insert(user)
}

// 用户表模型
type UserSqliteModel struct {
	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field_name:"username" field_type:"TEXT NOT NULL" json:"username"`
	Password    *string `field_name:"password" field_type:"TEXT NOT NULL" json:"-"`
	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
}

// 设置表配置
func (userModel UserSqliteModel) ModelSet() *sqlite3.SQLite3Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &sqlite3.SQLite3Settings{
		MigrationsHandler: model.MigrationsHandler{
			BeforeHandler: nil,            // 迁移之前处理函数
			AfterHandler:  InitUserSqlite, // 迁移之后处理函数
		},

		TABLE_NAME: "user_tb", // 设置表名

		// 自定义配置
		Settings: model.Settings{
			"encrypt_fields": encryptFields,
		},
	}

	return modelSettings
}

// 初始化用户
func InitUserSqlite() error {
	sqliteDB := db.SQLite3Connect("sqlite")

	var (
		id              int64  = 1
		username        string = "超级管理员"
		password        string = "admin"
		create_Datetime        = goi.GetTime().Format("2006-01-02 15:04:05")
		err             error
	)

	user := UserSqliteModel{
		Id:          &id,
		Username:    &username,
		Password:    &password,
		Create_time: &create_Datetime,
		Update_time: nil,
	}
	sqliteDB.SetModel(UserSqliteModel{})
	_, err = sqliteDB.Insert(&user)
	if err != nil {
		return err
	}

	return nil
}

func ExampleSQLite3DB_Migrate() {
	// 连接数据库
	sqliteDB := db.SQLite3Connect("sqlite")

	// 迁移模型
	sqliteDB.Migrate("test_db", UserSqliteModel{})

	// Output:
}

func ExampleSQLite3DB_Insert() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 准备测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format("2006-01-02 15:04:05")

	user := UserSqliteModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
	}

	// 设置模型并插入数据
	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_Where() {
	sqliteDB := db.SQLite3Connect("sqlite")

	var users []*UserSqliteModel
	sqliteDB.SetModel(UserSqliteModel{})

	// Where 返回当前实例的副本指针
	// 简单条件
	sqliteDB = sqliteDB.Where("username = ?", "where_test1")

	// AND条件
	sqliteDB = sqliteDB.Where("create_time IS NULL")

	// 支持 IN 参数值为 Slice 或 Array
	testUsers := []string{"where_test1", "where_test2"}
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
	// 查询到 2 条记录
	// 用户名: where_test1
	// 用户名: where_test2
}

func ExampleSQLite3DB_Select() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 查询多条记录 Select 支持 map 以及 结构体
	// var users []map[string]interface{}
	var users []*UserSqliteModel

	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_Page() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 分页查询
	var users []*UserSqliteModel

	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_First() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 查询单条记录 First 支持 map 以及 结构体
	// var user map[string]interface{}
	var user *UserSqliteModel

	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_Fields() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 指定查询字段
	var user map[string]interface{}

	sqliteDB.SetModel(UserSqliteModel{})

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

func ExampleSQLite3DB_Update() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 准备更新数据
	newPassword := "new_password"
	updateUser := UserSqliteModel{
		Password: &newPassword,
	}

	// 更新记录
	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_Delete() {
	sqliteDB := db.SQLite3Connect("sqlite")

	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_GroupBy() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 使用分组查询
	type Result struct {
		Date  *string `field_name:"date" json:"date"`
		Count *int    `field_name:"count" json:"count"`
	}

	var results []*Result
	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_OrderBy() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 使用排序查询
	var users []UserSqliteModel
	sqliteDB.SetModel(UserSqliteModel{})

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

func ExampleSQLite3DB_Count() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 统计记录数
	sqliteDB.SetModel(UserSqliteModel{})
	count, err := sqliteDB.Count()
	if err != nil {
		fmt.Println("统计错误:", err)
		return
	}

	fmt.Printf("总记录数: %d\n", count)

	// Output:
	// 总记录数: 1
}

func ExampleSQLite3DB_Exists() {
	sqliteDB := db.SQLite3Connect("sqlite")

	sqliteDB.SetModel(UserSqliteModel{})
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

func ExampleSQLite3DB_WithTransaction() {
	sqliteDB := db.SQLite3Connect("sqlite")

	// 事务
	err := sqliteDB.WithTransaction(func(db *db.SQLite3DB, args ...interface{}) error {
		// 准备测试数据
		username1 := "transaction_user1"
		username2 := "transaction_user2"
		password := "test123456"
		now := time.Now().Format("2006-01-02 15:04:05")

		// 插入第一个用户
		user1 := UserSqliteModel{
			Username:    &username1,
			Password:    &password,
			Create_time: &now,
		}
		db.SetModel(UserSqliteModel{})
		_, err := db.Insert(user1)
		if err != nil {
			return err
		}

		// 插入第二个用户
		user2 := UserSqliteModel{
			Username:    &username2,
			Password:    &password,
			Create_time: &now,
		}
		_, err = db.Insert(user2)
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

* 生成 Token

```go
type Payloads struct {
    jwt.Payloads
    User_id  int64  `json:"user_id"`
    Username string `json:"username"`
}

header := jwt.Header{
    Alg: jwt.AlgHS256,
    Typ: jwt.TypJWT,
}

// 设置过期时间
twoHoursLater := goi.GetTime().Add(24 * 15 * time.Hour)

payloads := Payloads{ // 包含 jwt.Payloads
    Payloads: jwt.Payloads{
        Exp: jwt.ExpTime{twoHoursLater},
    },
    User_id:  *userInfo.Id,
    Username: *userInfo.Username,
}
token, err := jwt.NewJWT(header, payloads, goi.Settings.SECRET_KEY)
```

* 验证 Token

```go
payloads := &auth.Payloads{}
err := jwt.CkeckToken(token, goi.Settings.SECRET_KEY, payloads)
if jwt.JwtDecodeError(err) { // token 解码错误
    pass
} else if jwt.JwtExpiredSignatureError(err) { // token 已过期
    pass
}
```

##                  
