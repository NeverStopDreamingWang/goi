package goi

import (
	"database/sql"
)

// DataBase 数据库连接管理器
//
// 字段:
//   - ENGINE string: 数据库引擎类型
//   - DataSourceName string: 数据库连接字符串
//   - db *sql.DB: 数据库连接实例
//   - Connect func: 数据库连接函数，用于创建连接
type DataBase struct {
	ENGINE         string
	DataSourceName string
	db             *sql.DB
	Connect        func(ENGINE string, DataSourceName string) *sql.DB // 使用时自动连接
}

// DB 获取数据库连接实例
//
// 返回:
//   - *sql.DB: 数据库连接对象，如果连接不存在则自动创建
func (database *DataBase) DB() *sql.DB {
	if database.db == nil {
		database.db = database.Connect(database.ENGINE, database.DataSourceName)
	}
	return database.db
}

// Close 关闭数据库连接
//
// 返回:
//   - error: 关闭过程中的错误信息，如果连接不存在则返回nil
func (database *DataBase) Close() error {
	if database.db == nil {
		return nil
	}
	return database.db.Close()
}
