# goi

基于 `net/http` 进行开发的 Web 框架

## 路由

### 多级路由

子父路由，内置路由转换器，自定义路由转换器

* `Router.Include(UrlPath)` 创建一个子路由
* `Router.UrlPatterns(UrlPath, goi.AsView{GET: Test1})` 注册一个路由

### 静态路由

静态路由映射，返回文件对象即响应该文件内容

* `Router.StaticUrlPatterns(UrlPath, StaticUrlPath)` 注册一个静态路由

## 中间件


* 请求中间件：`Server.MiddleWares.BeforeRequest(RequestMiddleWare)` 注册请求中间件
* 视图中间件：`Server.MiddleWares.BeforeView(ViewMiddleWare)` 注册视图中间件
* 响应中间件：`Server.MiddleWares.BeforeResponse(ResponseMiddleWare)` 注册响应中间件

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

## ORM 模型

支持：MySQL、SQLite3

#### **模型迁移**

* **MySQL**

  ```go
  // MySQL
  type UserModel struct {
  	Id          *int64  `field:"int NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户id'" json:"id"`
  	Username    *string `field:"varchar(255) DEFAULT NULL COMMENT '用户名'" json:"username"`
  	Password    *string `field:"varchar(255) DEFAULT NULL COMMENT '密码'" json:"password"`
  	Create_time *string `field:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"create_time"`
  	Update_time *string `field:"DATETIME DEFAULT NULL COMMENT '创建时间'" json:"update_time"`
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
  	Id          *int64  `field:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
  	Username    *string `field:"TEXT" json:"username"`
  	Password    *string `field:"TEXT" json:"password"`
  	Create_time *string `field:"TEXT NOT NULL" json:"create_time"`
  	Update_time *string `field:"TEXT" json:"update_time"`
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
func TestModelList(request *goi.Request) any {
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	
	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})                                                              // 设置操作表
	// resultSlice, err := mysqlObj.Fields("Id", "Username", "Password").Where("id>?", 1).Select()
	
	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	resultSlice, err := SQLite3DB.Fields("Id", "Username", "Password").Where("id>? limit 1, 2", 1).Select()
	
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	
	for i, item := range resultSlice {
		// user := item.(UserModel) // mysql 数据库
		user := item.(UserSqliteModel) // sqlite3 数据库
	}
    ...
}
```

### 查询单条数据

```go
func TestModelRetrieve(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)
	
	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}
	
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	
	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})                                                  // 设置操作表
	// result, err := mysqlObj.Fields("Id", "Username").Where("id=?", user_id).First()
	
	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Fields("Id", "Username").Where("id=?", user_id).First()
	
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// user := result.(UserModel)
	user := result.(UserSqliteModel)
	...
}
```

### 新增一条数据

```go
func TestModelCreate(request *goi.Request) any {
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	username := request.BodyParams.Get("username").(string)
	password := request.BodyParams.Get("password").(string)
	create_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	// user := &UserModel{
	// 	Username:    &username,
	// 	Password:    &password,
	// 	Create_time: &create_time,
	// }
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Insert(user)
	// result, err := mysqlObj.Fields("Id", "Password").Insert(user) // 指定插入字段
	
	// // sqlite3 数据库
	user := &UserSqliteModel{
		Username:    &username,
		Password:    &password,
		Create_time: &create_time,
	}
	SQLite3DB.SetModel(UserSqliteModel{})
	// result, err := SQLite3DB.Insert(user)
	result, err := SQLite3DB.Fields("Id", "Password", "Create_time").Insert(user)
	
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
func TestModelUpdate(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)
	username := request.BodyParams.Get("username").(string)
	
	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}
	
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	update_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
	// mysql 数据库
	// update_user := &UserModel{
	// 	Username:    &username,
	// 	Update_time: &update_time,
	// }
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Update(update_user)
	// result, err := mysqlObj.Fields("Username").Where("id=?", user_id).Update(update_user) // 更新指定字段
	
	// sqlite3 数据库
	update_user := &UserSqliteModel{
		Username:    &username,
		Update_time: &update_time,
	}
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Where("id=?", user_id).Update(update_user)
	
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
				Msg:    "修改失败！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)
    ...
}
```

### 删除数据

```go
func TestModelDelete(request *goi.Request) any {
	user_id := request.PathParams.Get("user_id").(int)
	
	if user_id == 0 {
		// 返回 goi.Response
		return goi.Response{
			Status: http.StatusNotFound, // Status 指定响应状态码
			Data:   nil,
		}
	}
	
	// mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	// defer mysqlObj.Close()
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return goi.Response{
			Status: http.StatusInternalServerError,
			Data:   err.Error(),
		}
	}
	// mysql 数据库
	// mysqlObj.SetModel(UserModel{})
	// result, err := mysqlObj.Where("id=?", user_id).Delete()
	
	// sqlite3 数据库
	SQLite3DB.SetModel(UserSqliteModel{})
	result, err := SQLite3DB.Where("id=?", user_id).Delete()
	
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
				Msg:    "已删除！",
				Data:   nil,
			},
		}
	}
	fmt.Println("rowNum", rowNum)
	...
}
```



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

## 内置 JWT Token

* 生成 Token

  ```go
  header := map[string]any{
      "alg": "SHA256",
      "typ": "JWT",
  }
   
  payloads := map[string]any{
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
