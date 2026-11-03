//go:build integration

package kingbase_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi/v3"
	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/db/kingbase"
	"github.com/NeverStopDreamingWang/goi/v3/db/postgres"
)

// 运行前需导入官方驱动 _ "kingbase.com/gokb"，启动 PostgreSQL 兼容模式的 Kingbase，并配置 DSN 与权限。
func init() {
	goi.Settings.Databases["default"] = &goi.Database{
		Engine: "kingbase",
		Config: kingbase.Config{Mode: kingbase.ModePostgreSQL},
		Connect: func(Engine string) *sql.DB {
			var DB *sql.DB
			var err error
			// 示例 DSN，请根据实际环境修改:
			// host=127.0.0.1 port=54321 user=system password=xxx dbname=test_goi sslmode=disable
			dataSourceName := "host=127.0.0.1 port=54321 user=system password=xxx dbname=test_goi sslmode=disable"
			DB, err = sql.Open(Engine, dataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return DB
		},
	}

	// 初始化测试数据库（示例）
	kbDB := db.Connect[*postgres.Engine]("default")

	// 迁移模型前清理旧表(如果存在)
	_, _ = kbDB.Execute(`DROP TABLE IF EXISTS "user_tb"`)

	// 迁移模型
	kbDB.Migrate(context.Background(), UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	kbDB.SetModel(UserModel{})
	// PostgreSQL 兼容模式下使用 RETURNING 子句获取主键
	row := kbDB.QueryRow(`INSERT INTO "user_tb" ("username", "password", "create_time") VALUES (?, ?, ?) RETURNING id`, username, password, now)
	var id int64
	if err := row.Scan(&id); err != nil {
		fmt.Println("插入错误:", err)
		return
	}
	user.ID = &id
}

type UserModel struct {
	ID         *int64  `field_name:"id" field_type:"BIGSERIAL PRIMARY KEY" json:"id"`
	Username   *string `field_name:"username" field_type:"VARCHAR(255) NOT NULL" json:"username"`
	Password   *string `field_name:"password" field_type:"VARCHAR(255) NOT NULL" json:"-"`
	CreateTime *string `field_name:"create_time" field_type:"TIMESTAMP WITH TIME ZONE DEFAULT NOW()" json:"create_time"`
	UpdateTime *string `field_name:"update_time" field_type:"TIMESTAMP WITH TIME ZONE" json:"update_time"`
}

func (userModel UserModel) ModelSet() *postgres.Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &postgres.Settings{
		MigrationsHandler: postgres.MigrationsHandler{
			BeforeHandler: nil,
			AfterHandler:  nil,
		},
		TableName: "user_tb",
		Settings: goi.Params{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

func ExampleConnect_migrate() {
	kbDB := db.Connect[*postgres.Engine]("default")
	kbDB.Migrate(context.Background(), UserModel{})
	// Output:
}

func ExampleConnect_insert() {
	kbDB := db.Connect[*postgres.Engine]("default")

	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	kbDB.SetModel(UserModel{})
	_, err := kbDB.Fields("Username", "Password", "CreateTime").Insert(context.Background(), user, false)
	if err != nil {
		fmt.Println("插入错误:", err)
		return
	}
	fmt.Printf("插入成功")
	// Output:
}
