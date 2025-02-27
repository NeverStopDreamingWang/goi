package user

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/sqlite3"
)

func init() {
	// sqlite 数据库
	sqlite3DB := db.SQLite3Connect("sqlite")
	sqlite3DB.Migrate("test_goi", UserModel{})
}

// sqlite 数据库
// 用户表
type UserModel struct {
	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field_name:"username" field_type:"TEXT NOT NULL" json:"username"`
	Password    *string `field_name:"password" field_type:"TEXT NOT NULL" json:"password"`
	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
}

// 设置表配置
func (userSqliteModel UserModel) ModelSet() *sqlite3.SQLite3Settings {
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
		Settings: model.Settings{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

// 初始化数据
func InitUserData() error {
	sqlite3DB := db.SQLite3Connect("sqlite")

	userData := [][]string{
		{"张三", "123"},
		{"李四", "456"},
	}

	for i, item := range userData {
		id := int64(i + 1)
		create_time := goi.GetTime().Format("2006-01-02 15:04:05")

		user := &UserModel{
			Id:          &id,
			Username:    &item[0],
			Password:    &item[1],
			Create_time: &create_time,
		}
		sqlite3DB.SetModel(UserModel{})
		_, err := sqlite3DB.Insert(user)
		if err != nil {
			return err
		}
	}
	return nil
}
