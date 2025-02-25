package db_test

import (
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/mysql"
)

func init() {
	// 初始化测试数据库
	mysqlDB := db.MySQLConnect("default") // 改为default,因为后续示例都用default

	// 迁移模型前清理旧表(如果存在)
	_, _ = mysqlDB.Execute("DROP TABLE IF EXISTS user_tb")

	// 迁移模型
	mysqlDB.Migrate("test_db", UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format("2006-01-02 15:04:05")

	user := UserModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
		Update_time: &now,
	}

	mysqlDB.SetModel(UserModel{})
	mysqlDB.Insert(user)
}

// 支持数据库 MySQL SQLite3
// 用户表
type UserModel struct {
	Id          *int64  `field_name:"id" field_type:"int NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT 'ID'" json:"id"`
	Username    *string `field_name:"username" field_type:"VARCHAR(255) NOT NULL COMMENT '用户名'" json:"username"`
	Password    *string `field_name:"password" field_type:"VARCHAR(255) NOT NULL COMMENT '密码'" json:"-"`
	Create_time *string `field_name:"create_time" field_type:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"DATETIME NOT NULL COMMENT '创建时间'" json:"update_time"`
}

// 设置表配置
func (userModel UserModel) ModelSet() *mysql.MySQLSettings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &mysql.MySQLSettings{
		MigrationsHandler: model.MigrationsHandler{ // 迁移时处理函数
			BeforeHandler: nil,      // 迁移之前处理函数
			AfterHandler:  InitUser, // 迁移之后处理函数
		},

		TABLE_NAME:      "user_tb",            // 设置表名
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
		COMMENT:         "用户表",             // 设置表注释

		// 自定义配置
		Settings: model.Settings{
			"encrypt_fields": encryptFields,
		},
	}

	return modelSettings
}

// 初始化用户
func InitUser() error {
	mysqlDB := db.MySQLConnect("default")

	var (
		id              int64  = 1
		username        string = "超级管理员"
		password        string = "admin"
		create_Datetime        = goi.GetTime().Format("2006-01-02 15:04:05")
		err             error
	)

	user := UserModel{
		Id:          &id,
		Username:    &username,
		Password:    &password,
		Create_time: &create_Datetime,
		Update_time: nil,
	}
	mysqlDB.SetModel(UserModel{})
	_, err = mysqlDB.Insert(&user)
	if err != nil {
		return err
	}

	return nil
}

func ExampleMySQLDB_Migrate() {
	// 连接数据库
	mysqlDB := db.MySQLConnect("default")

	// 迁移模型
	mysqlDB.Migrate("test_db", UserModel{})

	// Output:
}

func ExampleMySQLDB_Insert() {
	mysqlDB := db.MySQLConnect("default")

	// 准备测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format("2006-01-02 15:04:05")

	user := UserModel{
		Username:    &username,
		Password:    &password,
		Create_time: &now,
	}

	// 设置模型并插入数据
	mysqlDB.SetModel(UserModel{})
	result, err := mysqlDB.Insert(user)
	if err != nil {
		fmt.Println("插入错误:", err)
		return
	}

	id, _ := result.LastInsertId()
	fmt.Printf("插入成功, ID: %d\n", id)

	// Output:
	// 插入成功, ID: 1
}

func ExampleMySQLDB_Where() {
	mysqlDB := db.MySQLConnect("default")

	var users []*UserModel
	mysqlDB.SetModel(UserModel{})

	// Where 返回当前实例的副本指针
	// 简单条件
	mysqlDB = mysqlDB.Where("username = ?", "where_test1")

	// AND条件
	mysqlDB = mysqlDB.Where("create_time IS NULL")

	// 支持 IN 参数值为 Slice 或 Array
	testUsers := []string{"where_test1", "where_test2"}
	mysqlDB = mysqlDB.Where("username IN ?", testUsers)

	// LIKE条件
	mysqlDB = mysqlDB.Where("username LIKE ?", "where_test%")

	err := mysqlDB.Select(&users)
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

func ExampleMySQLDB_Select() {
	mysqlDB := db.MySQLConnect("default")

	// 查询多条记录 Select 支持 map 以及 结构体
	// var users []map[string]interface{}
	var users []*UserModel

	mysqlDB.SetModel(UserModel{})
	err := mysqlDB.Select(users)
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

func ExampleMySQLDB_Page() {
	mysqlDB := db.MySQLConnect("default")

	// 分页查询
	var users []*UserModel

	mysqlDB.SetModel(UserModel{})
	total, totalPages, err := mysqlDB.Page(2, 5) // 第2页,每页5条
	if err != nil {
		fmt.Println("分页错误:", err)
		return
	}

	err = mysqlDB.Select(&users)
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

func ExampleMySQLDB_First() {
	mysqlDB := db.MySQLConnect("default")

	// 查询单条记录 First 支持 map 以及 结构体
	// var user map[string]interface{}
	var user *UserModel

	mysqlDB.SetModel(UserModel{})
	mysqlDB = mysqlDB.Where("username = ?", "test_user")
	err := mysqlDB.First(user)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("用户名: %s\n", *user.Username)

	// Output:
	// 用户名: test_user
}

func ExampleMySQLDB_Fields() {
	mysqlDB := db.MySQLConnect("default")

	// 指定查询字段
	var user map[string]interface{}

	mysqlDB.SetModel(UserModel{})

	// Fields 返回当前实例的副本指针
	mysqlDB = mysqlDB.Fields("Username", "Create_time")
	err := mysqlDB.Select(&user)
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Println("用户名:", user["username"])

	// Output:
	// 用户名: user_1
}

func ExampleMySQLDB_Update() {
	mysqlDB := db.MySQLConnect("default")

	// 准备更新数据
	newPassword := "new_password"
	updateUser := UserModel{
		Password: &newPassword,
	}

	// 更新记录
	mysqlDB.SetModel(UserModel{})
	mysqlDB = mysqlDB.Where("username = ?", "test_user")
	result, err := mysqlDB.Update(updateUser)
	if err != nil {
		fmt.Println("更新错误:", err)
		return
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("更新成功, 影响行数: %d\n", rows)

	// Output:
	// 更新成功, 影响行数: 1
}

func ExampleMySQLDB_Delete() {
	mysqlDB := db.MySQLConnect("default")

	mysqlDB.SetModel(UserModel{})
	mysqlDB = mysqlDB.Where("username = ?", "test_user")

	// 删除记录
	result, err := mysqlDB.Delete()
	if err != nil {
		fmt.Println("删除错误:", err)
		return
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("删除成功, 影响行数: %d\n", rows)

	// Output:
	// 删除成功, 影响行数: 1
}

func ExampleMySQLDB_GroupBy() {
	mysqlDB := db.MySQLConnect("default")

	// 使用分组查询
	type Result struct {
		Date  *string `field_name:"date" json:"date"`
		Count *int    `field_name:"count" json:"count"`
	}

	var results []*Result
	mysqlDB.SetModel(UserModel{})
	// GroupBy 返回当前实例的副本指针
	mysqlDB = mysqlDB.GroupBy("DATE(create_time)")
	err := mysqlDB.Select(&results)
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

func ExampleMySQLDB_OrderBy() {
	mysqlDB := db.MySQLConnect("default")

	// 使用排序查询
	var users []UserModel
	mysqlDB.SetModel(UserModel{})

	// OrderBy 返回当前实例的副本指针
	mysqlDB = mysqlDB.OrderBy("-create_time")
	err := mysqlDB.Select(&users)
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

func ExampleMySQLDB_Count() {
	mysqlDB := db.MySQLConnect("default")

	// 统计记录数
	mysqlDB.SetModel(UserModel{})
	count, err := mysqlDB.Count()
	if err != nil {
		fmt.Println("统计错误:", err)
		return
	}

	fmt.Printf("总记录数: %d\n", count)

	// Output:
	// 总记录数: 1
}

func ExampleMySQLDB_Exists() {
	mysqlDB := db.MySQLConnect("default")

	mysqlDB.SetModel(UserModel{})
	// 检查记录是否存在
	exists, err := mysqlDB.Where("username = ?", "test_user").Exists()
	if err != nil {
		fmt.Println("查询错误:", err)
		return
	}

	fmt.Printf("记录是否存在: %v\n", exists)

	// Output:
	// 记录是否存在: true
}

func ExampleMySQLDB_WithTransaction() {
	mysqlDB := db.MySQLConnect("default")

	// 事务
	err := mysqlDB.WithTransaction(func(db *db.MySQLDB, args ...interface{}) error {
		// 准备测试数据
		username1 := "transaction_user1"
		username2 := "transaction_user2"
		password := "test123456"
		now := time.Now().Format("2006-01-02 15:04:05")

		// 插入第一个用户
		user1 := UserModel{
			Username:    &username1,
			Password:    &password,
			Create_time: &now,
		}
		db.SetModel(UserModel{})
		_, err := db.Insert(user1)
		if err != nil {
			return err
		}

		// 插入第二个用户
		user2 := UserModel{
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
