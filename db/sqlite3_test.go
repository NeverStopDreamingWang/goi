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
	now := time.Now().Format(time.DateTime)

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
		Settings: goi.Params{
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
		create_Datetime        = goi.GetTime().Format(time.DateTime)
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
	now := time.Now().Format(time.DateTime)

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
		now := time.Now().Format(time.DateTime)

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
