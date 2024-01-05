package user

import (
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
)

func init() {
	// Mysql 数据库
	MysqlMigrations := model.MysqlMakeMigrations{
		DATABASES: []string{"all"},
		MODELS: []model.MysqlModel{
			UserModel{},
		},
	}
	db.MysqlMigrate(MysqlMigrations)

	// sqlite 数据库
	Sqlite3Migrations := model.Sqlite3MakeMigrations{
		DATABASES: []string{"all"},
		MODELS: []model.Sqlite3Model{
			UserSqliteModel{},
		},
	}
	db.Sqlite3Migrate(Sqlite3Migrations)
}

// 用户表
type UserModel struct {
	Id          *int64  `field:"int NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户id'" json:"id"`
	Username    *string `field:"varchar(255) DEFAULT NULL COMMENT '用户名'" json:"username"`
	Password    *string `field:"varchar(255) DEFAULT NULL COMMENT '密码'" json:"password"`
	Create_time *string `field:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"create_time"`
	Update_time *string `field:"DATETIME DEFAULT NULL COMMENT '创建时间'" json:"update_time"`
}

// 设置表配置
func (UserModel) ModelSet() *model.MysqlSettings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &model.MysqlSettings{
		TableName:       "user_tb",            // 设置表名
		ENGINE:          "InnoDB",             // 设置存储引擎，默认: InnoDB
		AUTO_INCREMENT:  2,                    // 设置自增长起始值
		COMMENT:         "用户表",                // 设置表注释
		DEFAULT_CHARSET: "utf8mb4",            // 设置默认字符集，如: utf8mb4
		COLLATE:         "utf8mb4_0900_ai_ci", // 设置校对规则，如 utf8mb4_0900_ai_ci;
		ROW_FORMAT:      "",                   // 设置行的存储格式，如 DYNAMIC, COMPACT, FULL.
		DATA_DIRECTORY:  "",                   // 设置数据存储目录
		INDEX_DIRECTORY: "",                   // 设置索引存储目录
		STORAGE:         "",                   // 设置存储类型，如 DISK、MEMORY、CSV
		CHECKSUM:        0,                    // 表格的校验和算法，如 1 开启校验和
		DELAY_KEY_WRITE: 0,                    // 控制非唯一索引的写延迟，如 1
		MAX_ROWS:        0,                    // 设置最大行数
		MIN_ROWS:        0,                    // 设置最小行数
		PARTITION_BY:    "",                   // 定义分区方式，如 RANGE、HASH、LIST

		// 自定义配置
		MySettings: model.MySettings{
			"encrypt_fields": encryptFields,
		},
	}

	return modelSettings
}

// sqlite 数据库
// 用户表
type UserSqliteModel struct {
	Id          *int64  `field:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field:"TEXT" json:"username"`
	Password    *string `field:"TEXT" json:"password"`
	Create_time *string `field:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field:"TEXT" json:"update_time"`
}

// 设置表配置
func (UserSqliteModel) ModelSet() *model.Sqlite3Settings {
	encryptFields := []string{
		"username",
		"password",
	}

	modelSettings := &model.Sqlite3Settings{
		TableName: "user_tb", // 设置表名

		// 自定义配置
		MySettings: model.MySettings{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}
