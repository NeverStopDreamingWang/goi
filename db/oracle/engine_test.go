package oracle_test

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/db/oracle"
)

// 注意: 要运行这些示例测试，需要本地已配置好 Oracle 数据库和监听，并修改 DSN 与权限
func init() {
	goi.Settings.DATABASES["oracle_default"] = &goi.DataBase{
		ENGINE: "oracle",
		Connect: func(ENGINE string) *sql.DB {
			var DB *sql.DB
			var err error
			// 示例 DSN，请根据实际环境修改，例如:
			// user="scott" password="tiger" connectString="127.0.0.1:1521/orclpdb1"
			// 具体格式请参考 godror 文档
			dataSourceName := "user=scott password=tiger connectString=127.0.0.1:1521/orclpdb1"
			DB, err = sql.Open(ENGINE, dataSourceName)
			if err != nil {
				goi.Log.Error(err)
				panic(err)
			}
			return DB
		},
	}

	// 初始化测试数据库（示例）
	oraDB := db.Connect[*oracle.Engine]("oracle_default")

	// 迁移模型前清理旧表(如果存在)
	_, _ = oraDB.Execute("BEGIN EXECUTE IMMEDIATE 'DROP TABLE \"USER_TB\"'; EXCEPTION WHEN OTHERS THEN NULL; END;")

	// 迁移模型
	oraDB.Migrate("", UserModel{})

	// 插入初始测试数据
	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	oraDB.SetModel(UserModel{})
	// Oracle 推荐使用 RETURNING INTO 获取主键，这里仅演示 ORM Insert 调用
	if _, err := oraDB.Insert(user); err != nil {
		fmt.Println("插入错误:", err)
		return
	}
}

type UserModel struct {
	ID         *int64  `field_name:"id" field_type:"NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY" json:"id"`
	Username   *string `field_name:"username" field_type:"VARCHAR2(255 BYTE) NOT NULL" json:"username"`
	Password   *string `field_name:"password" field_type:"VARCHAR2(255 BYTE) NOT NULL" json:"-"`
	CreateTime *string `field_name:"create_time" field_type:"TIMESTAMP DEFAULT CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime *string `field_name:"update_time" field_type:"TIMESTAMP" json:"update_time"`
}

func (userModel UserModel) ModelSet() *oracle.Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &oracle.Settings{
		MigrationsHandler: oracle.MigrationsHandler{
			BeforeHandler: nil,
			AfterHandler:  nil,
		},
		TABLE_NAME: "USER_TB",
		Settings: goi.Params{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

func ExampleEngine_Migrate() {
	oraDB := db.Connect[*oracle.Engine]("oracle_default")
	oraDB.Migrate("", UserModel{})
	// Output:
}

func ExampleEngine_Insert() {
	oraDB := db.Connect[*oracle.Engine]("oracle_default")

	username := "test_user"
	password := "test123456"
	now := time.Now().Format(time.DateTime)

	user := UserModel{
		Username:   &username,
		Password:   &password,
		CreateTime: &now,
	}

	oraDB.SetModel(UserModel{})
	if _, err := oraDB.Insert(user); err != nil {
		fmt.Println("插入错误:", err)
		return
	}

	fmt.Println("插入成功")
	// Output:
}
