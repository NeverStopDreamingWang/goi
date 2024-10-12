# goi

基于 `net/http` 进行开发的 Web 框架

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

* `Router.StaticFilePatternsFs(path string, desc string, FileFs embed.FS)` 注册 `embed.FS` 静态文件路由
    * `path` 路由
    * `desc` 描述
    * `FileFs` 嵌入文件路径，支持`相对路径`和`绝对路径`


* `Router.StaticDirPatternsFs(path string, desc string, DirFs embed.FS)` 注册 `embed.FS` 静态目录路由
    * `path` 路由
    * `desc` 描述
    * `DirFs` 嵌入目录路径，支持`相对路径`和`绝对路径`

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
func TestAuth(t *testing.T) {
// 原始密码
password := "goi123456"

// 生成密码的哈希值
hashedPassword, err := MakePassword(password, bcrypt.DefaultCost)
if err != nil {
fmt.Println("密码加密失败:", err)
return
}

// 输出加密后的密码
fmt.Println("加密后的密码:", hashedPassword)

// 验证密码
isValid := CheckPassword(password, hashedPassword)
if isValid {
fmt.Println("密码验证成功")
} else {
fmt.Println("密码验证失败")
}
}
```

## 内置 Converter 路由转换器

注册路由转换器

```go
func init() {
	// 注册路由转换器

	// 手机号
	goi.RegisterConverter("my_phone", `(1[3456789]\d{9})`)
}

```

使用

```go
func init() {
	// 创建一个子路由
	testRouter := example.Server.Router.Include("/test", "测试路由")
    {
        // 注册一个路径
		// 类型 my_phone 
		// 名称 phone 
		testRouter.UrlPatterns("/test_phone/<my_phone:phone>", "", goi.AsView{GET: TestPhone}) // 测试路由转换器
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
func init() {
	// 注册验证器
	// 手机号
	goi.RegisterValidate("phone", phoneValidate)
}

// phone 类型
func phoneValidate(value string) goi.ValidationError {
	var IntRe = `^(1[3456789]\d{9})$`
	re := regexp.MustCompile(IntRe)
	if re.MatchString(value) == false {
		return goi.NewValidationError(http.StatusBadRequest, fmt.Sprintf("参数错误：%v", value))
	}
	return nil
}
```

使用

```go
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

```

## ORM 模型

支持：MySQL、SQLite3

#### **模型迁移**

* **MySQL**

  ```go
  // MySQL
  type UserModel struct {
  	Id          *int64  `field_name:"id" field_type:"int NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户id'" json:"id"`
  	Username    *string `field_name:"username" field_type:"varchar(255) DEFAULT NULL COMMENT '用户名'" json:"username"`
  	Password    *string `field_name:"password" field_type:"varchar(255) DEFAULT NULL COMMENT '密码'" json:"password"`
  	Create_time *string `field_name:"create_time" field_type:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"create_time"`
  	Update_time *string `field_name:"update_time" field_type:"DATETIME DEFAULT NULL COMMENT '创建时间'" json:"update_time"`
  }
  
  // 设置表配置
  func (UserModel) ModelSet() *model.MySQLSettings {
  	encryptFields := []string{
  		"username",
  		"password",
  	}
  
  	modelSettings := &model.MySQLSettings{
  		TABLE_NAME:      "user_tb",            // 设置表名
  		ENGINE:          "InnoDB",             // 设置存储引擎，默认: InnoDB
  		AUTO_INCREMENT:  2,                    // 设置自增长起始值
  		COMMENT:         "用户表",                // 设置表注释
  		DEFAULT_CHARSET: "utf8mb4",            // 设置默认字符集，如: utf8mb4
  		COLLATE:         "utf8mb4_0900_ai_ci", // 设置校对规则，如 utf8mb4_0900_ai_ci;
  		ROW_FORMAT:      "",                   // 设置行的存储格式，如 DYNAMIC, COMPACT, FULL.
  		DATA_DIRECTORY:  "",                   // 设置数据存储目录
  		INDEX_DIRECTORY: "",                   // 设置索引存储目录
  		STORAGE:         "",                   // 设置存储类型，如 DISK、MEMORY、CSV
  		CHECKSUM:        0,                    // 表格的校验和算法，如 1 开启校验和
  		DELAY_KEY_WRITE: 0,                    // 控制非唯一索引的写延迟，如 1
  		MAX_ROWS:        0,                    // 设置最大行数
  		MIN_ROWS:        0,                    // 设置最小行数
  		PARTITION_BY:    "",                   // 定义分区方式，如 RANGE、HASH、LIST
  
  		// 自定义配置
  		MySettings: model.MySettings{
  			"encrypt_fields": encryptFields,
  		},
  	}
  
  	return modelSettings
  }
  ```

* **SQLite**

  ```go
  // 用户表
  type UserSqliteModel struct {
  	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
  	Username    *string `field_name:"username" field_type:"TEXT" json:"username"`
  	Password    *string `field_name:"password" field_type:"TEXT" json:"password"`
  	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
  	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
  	Test        *string `field_name:"test_txt" field_type:"TEXT" json:"txt"` // 更新表字段
  }
  
  // 设置表配置
  func (UserSqliteModel) ModelSet() *model.SQLite3Settings {
  	encryptFields := []string{
  		"username",
  		"password",
  	}
  
  	modelSettings := &model.SQLite3Settings{
  		TABLE_NAME: "user_tb", // 设置表名
  
  		// 自定义配置
  		MySettings: model.MySettings{
  			"encrypt_fields": encryptFields,
  		},
  	}
  	return modelSettings
  }
  
  ```

### 查询多条数据

```go
// 参数验证
type testModelListParams struct {
	Page     int `name:"page" required:"int"`
	Pagesize int `name:"pagesize" required:"int"`
}

// 读取多条数据到 Model
func TestModelList(request *goi.Request) interface{} {
	var params testModelListParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	defer mysqlObj.Close()
	// SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	// defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	// mysql 数据库
	var userSlice []UserModel
	mysqlObj.SetModel(UserModel{}) // 设置操作表
	mysqlObj.Fields("Id", "Username", "Password").Where("id>?", 1).OrderBy("-id")
	total, page_number, err := mysqlObj.Page(params.Page, params.Pagesize)
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	err = mysqlObj.Select(&userSlice)

	// sqlite3 数据库
	// var userSlice []UserSqliteModel
	// SQLite3DB.SetModel(UserSqliteModel{})
	// err = SQLite3DB.Fields("Id", "Username", "Password").Where("id>?", 1).OrderBy("-id")
	// total, page_number, err := SQLite3DB.Page(params.Page, params.Pagesize)
	// if err != nil {
	// 	return goi.Response{
	// 		Status: http.StatusInternalServerError,
	// 		Data:   err.Error(),
	// 	}
	// }
	// err = SQLite3DB.Select(&userSlice)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	...
}
```

### 查询单条数据

```go
// 参数验证
type testModelRetrieveParams struct {
	User_id int `name:"user_id" required:"int"`
}

// 读取第一条数据到 Model
func TestModelRetrieve(request *goi.Request) interface{} {
	var params testModelRetrieveParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	if params.User_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	defer mysqlObj.Close()
	// SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	// defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}

	// mysql 数据库
	user := UserModel{}
	mysqlObj.SetModel(UserModel{}) // 设置操作表
	err = mysqlObj.Fields("Id", "Username").Where("id=?", params.User_id).First(&user)

	// sqlite3 数据库
	// user := UserSqliteModel{}
	// SQLite3DB.SetModel(UserSqliteModel{})
	// err = SQLite3DB.Fields("Id", "Username").Where("id=?", params.User_id).First(&user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	...
}
```

### 新增一条数据

```go
// 参数验证
type testModelCreateParams struct {
	Username string `name:"username" required:"string"`
	Password string `name:"password" required:"string"`
}

// 添加一条数据到 Model
func TestModelCreate(request *goi.Request) interface{} {
	var params testModelCreateParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	defer mysqlObj.Close()
	// SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	// defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	create_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	user := &UserModel{
		Username:    &params.Username,
		Password:    &params.Password,
		Create_time: &create_time,
	}
	mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Insert(user)
	result, err := mysqlObj.Fields("Id", "Password").Insert(user) // 指定插入字段

	// // sqlite3 数据库
	// user := &UserSqliteModel{
	// 	Username:    &params.Username,
	// 	Password:    &params.Password,
	// 	Create_time: &create_time,
	// }
	// SQLite3DB.SetModel(UserSqliteModel{})
	// // result, err := SQLite3DB.Insert(user)
	// result, err := SQLite3DB.Fields("Id", "Password", "Create_time").Insert(user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	id, err := result.LastInsertId()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	fmt.Println("id", id)

	user.Id = &id
	...
}
```

### 更新数据

```go
// 参数验证
type testModelUpdateParams struct {
	User_id  int    `name:"user_id" required:"int"`
	Username string `name:"username" optional:"string"`
	Password string `name:"password" optional:"string"`
}

// 修改 Model
func TestModelUpdate(request *goi.Request) interface{} {
	var params testModelUpdateParams
	var validationErr goi.ValidationError
	validationErr = request.BodyParams.ParseParams(&params)
	if validationErr != nil {
		// 验证器返回
		return validationErr.Response()

	}

	if params.User_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	defer mysqlObj.Close()
	// SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	// defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	update_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	update_user := &UserModel{
		Username:    nil,
		Password:    nil,
		Update_time: &update_time,
	}
	if params.Username != ""{
		update_user.Username = &params.Username
	}
	if params.Password != ""{
		update_user.Password = &params.Password
	}
	mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Update(update_user)
	result, err := mysqlObj.Fields("Username").Where("id=?", params.User_id).Update(update_user) // 更新指定字段

	// sqlite3 数据库
	// update_user := &UserSqliteModel{
	// 	Username:    nil,
	// 	Password:    nil,
	// 	Update_time: &update_time,
	// }
	// if params.Username != ""{
	// 	update_user.Username = &params.Username
	// }
	// if params.Password != ""{
	// 	update_user.Password = &params.Password
	// }
	// SQLite3DB.SetModel(UserSqliteModel{})
	// result, err := SQLite3DB.Where("id=?", params.User_id).Update(update_user)

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusInternalServerError,
				Message:    "修改失败！",
Results:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)
    ...
}
```

### 删除数据

```go
// 删除 Model
func TestModelDelete(request *goi.Request) interface{} {
	var user_id int
	var validationErr goi.ValidationError
	validationErr = request.PathParams.Get("user_id", &user_id)
	if validationErr != nil {
		return validationErr.Response()
	}

	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}

	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	defer mysqlObj.Close()
	// SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	// defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// mysql 数据库
	mysqlObj.SetModel(UserModel{})
	result, err := mysqlObj.Where("id=?", user_id).Delete()

	// sqlite3 数据库
	// SQLite3DB.SetModel(UserSqliteModel{})
	// result, err := SQLite3DB.Where("id=?", user_id).Delete()

	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	rowNum, err := result.RowsAffected()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	if rowNum == 0 {
		return goi.Response{
			Status: http.StatusOK,
			Data: goi.Data{
				Status: http.StatusOK,
				Message:    "已删除！",
Results:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)
	...
}
```

## 内置 JWT Token

* 生成 Token

```go
header := map[string]interface{}{
  "alg": "SHA256",
  "typ": "JWT",
}

payloads := map[string]interface{}{
  "user_id":  1,
  "username": "wjh123",
  "exp":      expTime,
}
token, err := jwt.NewJWT(header, payloads, "xxxxxx")
```

* 验证 Token

```go
payloads, err := jwt.CkeckToken(token, "xxxxxx")
if jwt.JwtDecodeError(err) { // token 解码错误！
 pass
}else if jwt.JwtExpiredSignatureError(err) { // token 已过期
 pass
}
```

##                  

[详细示例：example](./example)
