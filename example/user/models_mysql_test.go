package user_test

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/auth"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"github.com/NeverStopDreamingWang/goi/model/mysql"
)

func init() {
	// MySQL 数据库
	mysqlDB := db.MySQLConnect("default")
	mysqlDB.Migrate("test_goi", UserModel{})
}

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

var InitUserList = [][]any{
	{int64(1), "admin", "admin"},
}

// 初始化数据
func InitUser() error {
	mysqlDB := db.MySQLConnect("default")
	for _, item := range InitUserList {
		var (
			Id              = item[0].(int64)
			Username        = item[1].(string)
			Password        = item[2].(string)
			Create_Time     = goi.GetTime().Format("2006-01-02 15:04:05")
			encryptPassword string
		)
		encryptPassword, err := auth.MakePassword(Password)
		if err != nil {
			return err
		}
		user := &UserModel{
			Id:          &Id,
			Username:    &Username,
			Password:    &encryptPassword,
			Create_time: &Create_Time,
		}
		mysqlDB.SetModel(UserModel{})
		_, err = mysqlDB.Insert(user)
		if err != nil {
			return err
		}
	}
	return nil
}
