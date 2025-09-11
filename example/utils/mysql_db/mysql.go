package mysql_db

import (
	"database/sql"
	"time"

	"github.com/NeverStopDreamingWang/goi"
)

// MySQL 配置
type ConfigModel struct {
	Uri string `yaml:"uri"`
}

var Config = &ConfigModel{}

func Connect(ENGINE string) *sql.DB {
	var err error
	MySQLDB, err := sql.Open(ENGINE, Config.Uri)
	if err != nil {
		goi.Log.Error(err)
		panic(err)
	}
	// 设置连接池参数
	MySQLDB.SetMaxOpenConns(10)           // 设置最大打开连接数
	MySQLDB.SetMaxIdleConns(5)            // 设置最大空闲连接数
	MySQLDB.SetConnMaxLifetime(time.Hour) // 设置连接的最大存活时间
	return MySQLDB
}
