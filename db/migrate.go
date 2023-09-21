package db

import (
	"fmt"
	"github.com/NeverStopDreamingWang/hgee"
	"github.com/NeverStopDreamingWang/hgee/model"
	"strings"
)

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
			for _, Model := range Migrations.MODELS {
				createSql := model.MetaMysqlMigrate(Model)
				err = mysqlDB.Execute(createSql)
				if err != nil {
					panic(fmt.Sprintf("迁移错误: %v", err))
				}
				fmt.Printf("Migrate Mysql: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
			}
			defer mysqlDB.Close()
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
			fmt.Println("连接数据库失败！")
		}
		for _, Model := range Migrations.MODELS {
			createSql := model.MetaMysqlMigrate(Model)
			err = mysqlDB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("迁移错误: %v", err))
			}
			fmt.Printf("Migrate Mysql: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
		}
		defer mysqlDB.Close()
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
			defer sqlite3DB.Close()
			if err != nil {
				panic(fmt.Sprintf("连接 Sqlite3 [%v] 数据库 错误: %v", DBName, err))
			}
			for _, Model := range Migrations.MODELS {
				createSql := model.MetaSqlite3Migrate(Model)
				err = sqlite3DB.Execute(createSql)
				if err != nil {
					panic(fmt.Sprintf("迁移错误: %v", err))
				}
				fmt.Printf("Migrate Sqlite3: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
			}
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
			fmt.Println("连接数据库失败！")
		}
		for _, Model := range Migrations.MODELS {
			createSql := model.MetaSqlite3Migrate(Model)
			err = sqlite3DB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("迁移错误: %v", err))
			}
			fmt.Printf("Migrate Sqlite3: %v DataBase: %v Table: %v ...ok!\n", DBName, Database.NAME, Model.ModelSet().TableName)
		}
		defer sqlite3DB.Close()
	}
}
