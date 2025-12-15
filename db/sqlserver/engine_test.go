package sqlserver_test

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/sqlserver"
)

// 注意: 要运行这些示例测试，需要本地已配置好 SQL Server 数据库，并修改 DSN 与权限
func init() {
	goi.Settings.DATABASES["sqlserver_default"] = &goi.DataBase{
		ENGINE: "sqlserver",
		Connect: func(ENGINE string) *sql.DB {
			var DB *sql.DB
			var err error
			// 示例 DSN，请根据实际环境修改，例如:
			// sqlserver://username:password@localhost:1433?database=TestDB
			dataSourceName := "sqlserver://sa:yourStrong(!)Password@localhost:1433?database=TestDB"
			DB, err = sql.Open(ENGINE, dataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return DB
		},
	}

	// 初始化测试数据库（示例）
	mssqlDB := db.Connect[*sqlserver.Engine]("sqlserver_default")
	// 如果表已存在先删除
	_, _ = mssqlDB.Execute("IF OBJECT_ID('dbo.user_tb', 'U') IS NOT NULL DROP TABLE dbo.user_tb;")

	// 迁移模型
	mssqlDB.Migrate("", UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	mssqlDB.SetModel(UserModel{})
	if _, err := mssqlDB.Insert(user); err != nil {
		fmt.Println("插入错误:", err)
		return
	}
}

type UserModel struct {
	ID         *int64  `field_name:"id" field_type:"BIGINT IDENTITY(1,1) PRIMARY KEY" json:"id"`
	Username   *string `field_name:"username" field_type:"NVARCHAR(255) NOT NULL" json:"username"`
	Password   *string `field_name:"password" field_type:"NVARCHAR(255) NOT NULL" json:"-"`
	CreateTime *string `field_name:"create_time" field_type:"DATETIME2 DEFAULT SYSDATETIME()" json:"create_time"`
	UpdateTime *string `field_name:"update_time" field_type:"DATETIME2" json:"update_time"`
}

func (userModel UserModel) ModelSet() *sqlserver.Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &sqlserver.Settings{
		MigrationsHandler: sqlserver.MigrationsHandler{
			BeforeHandler: nil,
			AfterHandler:  nil,
		},
		TABLE_NAME: "user_tb",
		Settings: goi.Params{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

func ExampleEngine_Migrate() {
	mssqlDB := db.Connect[*sqlserver.Engine]("sqlserver_default")
	mssqlDB.Migrate("dbo", UserModel{})
	// Output:
}

func ExampleEngine_Insert() {
	mssqlDB := db.Connect[*sqlserver.Engine]("sqlserver_default")

	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	mssqlDB.SetModel(UserModel{})
	if _, err := mssqlDB.Insert(user); err != nil {
		fmt.Println("插入错误:", err)
		return
	}

	fmt.Println("插入成功")
	// Output:
}
