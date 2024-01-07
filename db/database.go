package db

import (
	"errors"
	"github.com/NeverStopDreamingWang/goi"
	"strings"
)

// 连接 MySQL 数据库
func MySQLConnect(UseDataBases ...string) (*MySQLDB, error) {
	if len(UseDataBases) == 0 { // 使用默认数据库
		UseDataBases = append(UseDataBases, "default")
	}

	for _, Name := range UseDataBases {
		database, ok := goi.Settings.DATABASES[Name]
		if ok == true && strings.ToLower(database.ENGINE) == "mysql" {
			databaseObject, err := MetaMySQLConnect(database)
			if err != nil {
				continue
			}
			databaseObject.Name = Name
			return databaseObject, err
		}
	}
	return nil, errors.New("连接 MySQL 错误！")
}

// 连接 Sqlite3 数据库
func Sqlite3Connect(UseDataBases ...string) (*Sqlite3DB, error) {
	if len(UseDataBases) == 0 { // 使用默认数据库
		UseDataBases = append(UseDataBases, "default")
	}

	for _, Name := range UseDataBases {
		Database, ok := goi.Settings.DATABASES[Name]
		if ok == true && strings.ToLower(Database.ENGINE) == "sqlite3" {
			databaseObject, err := MetaSqlite3Connect(Database)
			if err != nil {
				continue
			}
			databaseObject.Name = Name
			return databaseObject, err
		}
	}
	return nil, errors.New("连接 Sqlite3 错误！")
}
