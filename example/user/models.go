package user

import (
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"time"
)

func init() {
	// MySQL 数据库
	mysqlObj, err := db.MySQLConnect("default", "mysql_slave_2")
	if err != nil {
		panic(err)
	}
	defer mysqlObj.Close()
	mysqlObj.Migrate(UserModel{})
	
	// sqlite 数据库
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	if err != nil {
		panic(err)
	}
	defer SQLite3DB.Close()
	SQLite3DB.Migrate(UserSqliteModel{})
}

// 用户表
type UserModel struct {
	Id          *int64  `field_name:"id" field_type:"int NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户id'" json:"id"`
	Username    *string `field_name:"username" field_type:"VARCHAR(255) DEFAULT NULL COMMENT '用户名'" json:"username"`
	Password    *string `field_name:"password" field_type:"VARCHAR(255) DEFAULT NULL COMMENT '密码'" json:"password"`
	Create_time *string `field_name:"create_time" field_type:"DATETIME DEFAULT NULL COMMENT '更新时间'" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"DATETIME DEFAULT NULL COMMENT '创建时间'" json:"update_time"`
	Test *string `field_name:"test_txt" field_type:"VARCHAR(255)" json:"txt"` // 更新表字段
}

// 设置表配置
func (userModel UserModel) ModelSet() *model.MySQLSettings {
	encryptFields := []string{
		"username",
		"password",
	}
	
	modelSettings := &model.MySQLSettings{
		MigrationsHandler: model.MigrationsHandler{ // 迁移时处理函数
			BeforeFunc: nil, // 迁移之前处理函数
			AfterFunc:  nil, // 迁移之后处理函数
		},
		
		TABLE_NAME:      "user_tb",            // 设置表名
		ENGINE:          "InnoDB",             // 设置存储引擎，默认: InnoDB
		AUTO_INCREMENT:  2,                    // 设置自增长起始值
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
		MySettings: model.MySettings{
			"encrypt_fields": encryptFields,
		},
	}
	
	return modelSettings
}

// sqlite 数据库
// 用户表
type UserSqliteModel struct {
	Id          *int64  `field_name:"id" field_type:"INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT" json:"id"`
	Username    *string `field_name:"username" field_type:"TEXT" json:"username"`
	Password    *string `field_name:"password" field_type:"TEXT" json:"password"`
	Create_time *string `field_name:"create_time" field_type:"TEXT NOT NULL" json:"create_time"`
	Update_time *string `field_name:"update_time" field_type:"TEXT" json:"update_time"`
	Test *string `field_name:"test_txt" field_type:"TEXT" json:"txt"` // 更新表字段
}

// 设置表配置
func (userSqliteModel UserSqliteModel) ModelSet() *model.SQLite3Settings {
	encryptFields := []string{
		"username",
		"password",
	}
	
	modelSettings := &model.SQLite3Settings{
		MigrationsHandler: model.MigrationsHandler{ // 迁移时处理函数
			BeforeFunc: nil,          // 迁移之前处理函数
			AfterFunc:  InitUserData, // 迁移之后处理函数
		},
		
		TABLE_NAME: "user_tb", // 设置表名
		
		// 自定义配置
		MySettings: model.MySettings{
			"encrypt_fields": encryptFields,
		},
	}
	return modelSettings
}

// 初始化数据
func InitUserData() error {
	SQLite3DB, err := db.SQLite3Connect("sqlite_1")
	defer SQLite3DB.Close()
	if err != nil {
		return err
	}
	
	userData := [][]string{
		{"张三", "123"},
		{"李四", "456"},
	}
	
	for i, item := range userData {
		id := int64(i + 1)
		create_time := time.Now().In(goi.Settings.LOCATION).Format("2006-01-02 15:04:05")
		
		user := &UserSqliteModel{
			Id:          &id,
			Username:    &item[0],
			Password:    &item[1],
			Create_time: &create_time,
		}
		SQLite3DB.SetModel(UserSqliteModel{})
		_, err = SQLite3DB.Insert(user)
		if err != nil {
			return err
		}
	}
	return nil
}
