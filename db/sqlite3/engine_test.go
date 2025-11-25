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
		id              int64  = 1
		username        string = "超级管理员"
		password        string = "admin"
		create_Datetime        = goi.GetTime().Format(time.DateTime)
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
	sqliteDB.Migrate(UserModel{})

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
	// var users []map[string]any
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
	// var user map[string]any
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
	var user map[string]any

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
	err := sqliteDB.WithTransaction(func(engine *sqlite3.Engine, args ...any) error {
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
