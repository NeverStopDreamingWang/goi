package db

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"github.com/NeverStopDreamingWang/hgee/model"
	"strings"
)

// 判断表是否存在
func TableIsExists(TableName string, TabelSlice []string) bool {
	for _, str := range TabelSlice {
		if str == TableName {
			return true
		}
	}
	return false
}

// Mysql 迁移
func MysqlMigrate(Migrations model.MysqlMakeMigrations) {
	if len(Migrations.DATABASES) == 0 {
		panic("请指定迁移数据库！")
	}
	// 迁移所有
	if strings.ToLower(Migrations.DATABASES[0]) == "all" {
		for DBName, Database := range hgee.Settings.DATABASES {
			if strings.ToLower(Database.ENGINE) != "mysql" {
				continue
			}
			mysqlDB, err := MysqlConnect(DBName)
			if err != nil {
				panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
			}

			// 获取所有表
			TabelSclie := make([]string, 0)
			rows, err := mysqlDB.Query("SHOW TABLES;")
			if err != nil {
				panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
			}
			for rows.Next() {
				var tableData string
				err = rows.Scan(&tableData)
				if err != nil {
					panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
				}
				TabelSclie = append(TabelSclie, tableData)
			}
			_ = rows.Close()

			for _, Model := range Migrations.MODELS {
				TableName := Model.ModelSet().TableName
				if TableIsExists(TableName, TabelSclie) { // 判断是否存在
					continue
				}
				createSql := model.MetaMysqlMigrate(Model)
				_, err = mysqlDB.Execute(createSql)
				if err != nil {
					panic(fmt.Sprintf("迁移错误: %v", err))
				}
				fmt.Printf("Migrate Mysql: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
			}
			_ = mysqlDB.Close()
		}
		return
	}
	fmt.Println("migrations DATABASES", Migrations.DATABASES)
	// 迁移到指定数据库
	for _, DBName := range Migrations.DATABASES {
		Database, ok := hgee.Settings.DATABASES[DBName]
		if ok == false {
			panic(fmt.Sprintf("配置数据库: %v 不存在！", DBName))
		}
		if strings.ToLower(Database.ENGINE) != "mysql" {
			continue
		}
		mysqlDB, err := MysqlConnect(DBName)
		if err != nil {
			panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
		}

		// 获取所有表
		TabelSclie := make([]string, 0)
		rows, err := mysqlDB.Query("SHOW TABLES;")
		if err != nil {
			panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
		}
		for rows.Next() {
			var tableData string
			err = rows.Scan(&tableData)
			if err != nil {
				panic(fmt.Sprintf("连接 Mysql [%v] 数据库 错误: %v", DBName, err))
			}
			TabelSclie = append(TabelSclie, tableData)
		}
		_ = rows.Close()

		for _, Model := range Migrations.MODELS {
			TableName := Model.ModelSet().TableName
			if TableIsExists(TableName, TabelSclie) { // 判断是否存在
				continue
			}
			createSql := model.MetaMysqlMigrate(Model)
			_, err = mysqlDB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("迁移错误: %v", err))
			}
			fmt.Printf("Migrate Mysql: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
		}
		_ = mysqlDB.Close()
	}
}

// Sqlite3 迁移
func Sqlite3Migrate(Migrations model.Sqlite3MakeMigrations) {
	if len(Migrations.DATABASES) == 0 {
		panic("请指定迁移数据库！")
	}
	// 迁移所有
	if strings.ToLower(Migrations.DATABASES[0]) == "all" {
		for DBName, Database := range hgee.Settings.DATABASES {
			if strings.ToLower(Database.ENGINE) != "sqlite3" {
				continue
			}
			sqlite3DB, err := Sqlite3Connect(DBName)
			if err != nil {
				panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
			}

			// 获取所有表
			TabelSclie := make([]string, 0)
			rows, err := sqlite3DB.Query("SELECT name FROM sqlite_master WHERE type='table';")
			if err != nil {
				panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
			}
			for rows.Next() {
				var tableData string
				err = rows.Scan(&tableData)
				if err != nil {
					panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
				}
				TabelSclie = append(TabelSclie, tableData)
			}
			_ = rows.Close()

			for _, Model := range Migrations.MODELS {
				TableName := Model.ModelSet().TableName
				if TableIsExists(TableName, TabelSclie) { // 判断是否存在
					continue
				}
				createSql := model.MetaSqlite3Migrate(Model)
				_, err = sqlite3DB.Execute(createSql)
				if err != nil {
					panic(fmt.Sprintf("迁移错误: %v", err))
				}
				fmt.Printf("Migrate Sqlite3: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
			}
			_ = sqlite3DB.Close()
		}
		return
	}
	fmt.Println("migrations DATABASES", Migrations.DATABASES)
	// 迁移到指定数据库
	for _, DBName := range Migrations.DATABASES {
		Database, ok := hgee.Settings.DATABASES[DBName]
		if ok == false {
			panic(fmt.Sprintf("配置数据库: %v 不存在！", DBName))
		}
		if strings.ToLower(Database.ENGINE) != "mysql" {
			continue
		}
		sqlite3DB, err := Sqlite3Connect(DBName)
		if err != nil {
			panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
		}

		// 获取所有表
		TabelSclie := make([]string, 0)
		rows, err := sqlite3DB.Query("SELECT name FROM sqlite_master WHERE type='table';")
		if err != nil {
			panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
		}
		for rows.Next() {
			var tableData string
			err = rows.Scan(&tableData)
			if err != nil {
				panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
			}
			TabelSclie = append(TabelSclie, tableData)
		}
		_ = rows.Close()

		for _, Model := range Migrations.MODELS {
			TableName := Model.ModelSet().TableName
			if TableIsExists(TableName, TabelSclie) { // 判断是否存在
				continue
			}
			createSql := model.MetaSqlite3Migrate(Model)
			_, err = sqlite3DB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("迁移错误: %v", err))
			}
			fmt.Printf("Migrate Sqlite3: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
		}
		_ = sqlite3DB.Close()
	}
}
