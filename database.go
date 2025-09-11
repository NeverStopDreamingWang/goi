package goi

import (
	"database/sql"
)

type DBConnectFunc func() *sql.DB

// DataBase 数据库连接管理器
//
// 字段:
//   - ENGINE string: 数据库引擎名称
//   - Connect func: 数据库连接函数，使用时才会连接，且仅会调用一次
//   - db *sql.DB: 数据库连接实例
type DataBase struct {
	ENGINE  string
	Connect func(ENGINE string) *sql.DB // 使用时自动连接
	db      *sql.DB
}

// DB 获取数据库连接实例
//
// 返回:
//   - *sql.DB: 数据库连接对象，如果连接不存在则自动创建
func (database *DataBase) DB() *sql.DB {
	if database.db == nil {
		database.db = database.Connect(database.ENGINE)
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
