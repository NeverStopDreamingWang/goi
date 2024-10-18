package goi

import (
	"database/sql"
)

// 数据库，使用时会自动连接，服务停止时自动关闭
type DataBase struct {
	ENGINE         string
	DataSourceName string
	db             *sql.DB
	Connect        func(ENGINE string, DataSourceName string) (*sql.DB, error) // 使用时自动连接
}

// 返回数据库连接对象
func (database *DataBase) DB() (*sql.DB, error) {
	var err error
	if database.db == nil {
		database.db, err = database.Connect(database.ENGINE, database.DataSourceName)
		if err != nil {
			return nil, err
		}
	}
	return database.db, nil
}

func (database *DataBase) Close() error {
	if database.db == nil {
		return nil
	}
	return database.db.Close()
}
