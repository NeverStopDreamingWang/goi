package user

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/sqlite3"
)

func init() {
	// sqlite 数据库
	sqlite3DB := db.SQLite3Connect("default")
	sqlite3DB.Migrate("test_goi", UserModel{})
}

// sqlite 数据库
// 用户表
type UserModel struct {
	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field_name:"username" field_type:"TEXT NOT NULL" json:"username"`
	Password    *string `field_name:"password" field_type:"TEXT NOT NULL" json:"-"`
	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
}

// 设置表配置
func (self UserModel) ModelSet() *sqlite3.SQLite3Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &sqlite3.SQLite3Settings{
		MigrationsHandler: model.MigrationsHandler{ // 迁移时处理函数
			BeforeHandler: nil,          // 迁移之前处理函数
			AfterHandler:  InitUserData, // 迁移之后处理函数
		},

		TABLE_NAME: "user_tb", // 设置表名

		// 自定义配置
		Settings: goi.Params{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

var initUserList = [][]any{
	{"张三", "123"},
	{"李四", "456"},
}

// 初始化数据
func InitUserData() error {
	var err error

	sqlite3DB := db.SQLite3Connect("default")
	sqlite3DB.SetModel(UserModel{})
	total, err := sqlite3DB.Count()
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	for _, item := range initUserList {
		var (
			username = item[0].(string)
			password = item[1].(string)
		)

		user := &UserModel{
			Username: &username,
			Password: &password,
		}
		err = user.Validate()
		if err != nil {
			return err
		}
		err = user.Create()
		if err != nil {
			return err
		}
	}
	return nil
}
