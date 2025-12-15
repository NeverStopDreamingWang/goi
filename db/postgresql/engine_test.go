package postgresql_test

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/postgresql"
)

// 注意: 要运行这些示例测试，需要本地已启动 PostgreSQL，且修改 DSN 与权限
func init() {
	goi.Settings.DATABASES["default"] = &goi.DataBase{
		ENGINE: "postgres",
		Connect: func(ENGINE string) *sql.DB {
			var DB *sql.DB
			var err error
			// 示例 DSN，请根据实际环境修改:
			// host=127.0.0.1 port=5432 user=postgres password=xxx dbname=test_goi sslmode=disable
			dataSourceName := "host=127.0.0.1 port=5432 user=postgres password=xxx dbname=test_goi sslmode=disable"
			DB, err = sql.Open(ENGINE, dataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return DB
		},
	}

	// 初始化测试数据库（示例）
	pgDB := db.Connect[*postgresql.Engine]("default")

	// 迁移模型前清理旧表(如果存在)
	_, _ = pgDB.Execute("DROP TABLE IF EXISTS \"user_tb\"")

	// 迁移模型
	pgDB.Migrate("public", UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	pgDB.SetModel(UserModel{})
	// PostgreSQL 推荐使用 RETURNING 子句获取主键
	row := pgDB.QueryRow("INSERT INTO \"user_tb\" (\"username\", \"password\", \"create_time\") VALUES (?, ?, ?) RETURNING id", username, password, now)
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

func (userModel UserModel) ModelSet() *postgresql.Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &postgresql.Settings{
		MigrationsHandler: postgresql.MigrationsHandler{
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
	pgDB := db.Connect[*postgresql.Engine]("default")
	pgDB.Migrate("public", UserModel{})
	// Output:
}

func ExampleEngine_Insert() {
	pgDB := db.Connect[*postgresql.Engine]("default")

	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	pgDB.SetModel(UserModel{})
	_, err := pgDB.Insert(user)
	if err != nil {
		fmt.Println("插入错误:", err)
		return
	}
	fmt.Printf("插入成功")
	// Output:
}
