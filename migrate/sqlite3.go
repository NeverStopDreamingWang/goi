package migrate

import (
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"reflect"
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

// SQLite3 迁移
func SQLite3Migrate(Migrations model.SQLite3MakeMigrations) {
	if len(Migrations.DATABASES) == 0 {
		panic("请指定迁移数据库！")
	}
	// 迁移数据库
	for _, DBName := range Migrations.DATABASES {
		Database, ok := goi.Settings.DATABASES[DBName]
		if ok == false {
			panic(fmt.Sprintf("配置数据库: %v 不存在！", DBName))
		}
		if strings.ToLower(Database.ENGINE) != "sqlite3" {
			continue
		}
		sqlite3DB, err := db.SQLite3Connect(DBName)
		if err != nil {
			panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
		}

		// 获取所有表
		TabelSclie := make([]string, 0)
		rows, err := sqlite3DB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name not in ('sqlite_master','sqlite_sequence');")
		if err != nil {
			panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
		}
		for rows.Next() {
			var tableData string
			err = rows.Scan(&tableData)
			if err != nil {
				panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
			}
			TabelSclie = append(TabelSclie, tableData)
		}
		_ = rows.Close()

		for _, Model := range Migrations.MODELS {
			TableName := Model.ModelSet().TABLE_NAME

			modelType := reflect.TypeOf(Model)
			if modelType.Kind() == reflect.Ptr {
				modelType = modelType.Elem()
			}
			tampFieldMap := make(map[string]*int8)
			fieldSqlSlice := make([]string, modelType.NumField(), modelType.NumField())
			for i := 0; i < modelType.NumField(); i++ {
				field := modelType.Field(i)
				fieldName, ok := field.Tag.Lookup("field_name")
				if !ok {
					fieldName = strings.ToLower(field.Name)
				}
				fieldType, ok := field.Tag.Lookup("field_type")
				tampFieldMap[fieldName] = nil
				fieldSqlSlice[i] = fmt.Sprintf("`%v` %v", fieldName, fieldType)
			}
			createSql := fmt.Sprintf("CREATE TABLE `%v` (\n%v\n)", TableName, strings.Join(fieldSqlSlice, ",\n"))

			if TableIsExists(TableName, TabelSclie) { // 更新表
				sql := fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%v';", TableName)
				rows, err = sqlite3DB.Query(sql)
				if err != nil {
					panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
				}
				var old_createSql string
				for rows.Next() {
					err = rows.Scan(&old_createSql)
					if err != nil {
						panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
					}
				}
				_ = rows.Close()
				if createSql != old_createSql { // 创建表语句不一样
					goi.Log.MetaLog(fmt.Sprintf("正在更新 SQLite3: %v 数据库: %v 表: %v ", DBName, Database.NAME, TableName))

					oldTableName := fmt.Sprintf("%v_%v", TableName, goi.Version())
					goi.Log.MetaLog(fmt.Sprintf("- 重命名旧表：%v...", oldTableName))
					_, err = sqlite3DB.Execute(fmt.Sprintf("ALTER TABLE `%v` RENAME TO `%v`;", TableName, oldTableName))
					if err != nil {
						panic(fmt.Sprintf("重命名旧表错误: %v", err))
					}

					goi.Log.MetaLog(fmt.Sprintf("- 创建新表：%v...", oldTableName))
					_, err = sqlite3DB.Execute(createSql)
					if err != nil {
						panic(fmt.Sprintf("创建新表错误: %v", err))
					}

					TabelFieldSlice := make([]string, 0)
					rows, err = sqlite3DB.Query(fmt.Sprintf("PRAGMA table_info(`%v`);", oldTableName))
					if err != nil {
						panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
					}
					for rows.Next() {
						var (
							cid        int
							name       string
							dataType   string
							notNull    int
							dflt_value *string
							pk         int
						)
						err = rows.Scan(&cid, &name, &dataType, &notNull, &dflt_value, &pk)
						if err != nil {
							panic(fmt.Sprintf("连接 SQLite3 [%v] 数据库 错误: %v", DBName, err))
						}
						if _, exists := tampFieldMap[name]; exists {
							TabelFieldSlice = append(TabelFieldSlice, name)
						}
					}
					_ = rows.Close()

					goi.Log.MetaLog("- 迁移数据...")
					migrateFieldSql := strings.Join(TabelFieldSlice, ",")
					migrateSql := fmt.Sprintf("INSERT INTO `%v` (%v) SELECT %v FROM `%v`;", TableName, migrateFieldSql, migrateFieldSql, oldTableName)
					_, err = sqlite3DB.Execute(migrateSql)
					if err != nil {
						panic(fmt.Sprintf("迁移表数据错误: %v", err))
					}

					goi.Log.MetaLog("更新完成!\n")
				}

			} else { // 创建表
				goi.Log.MetaLog(fmt.Sprintf("正在迁移 SQLite3: %v 数据库: %v 表: %v ...", DBName, Database.NAME, TableName))
				_, err = sqlite3DB.Execute(createSql)
				if err != nil {
					panic(fmt.Sprintf("迁移错误: %v", err))
				}
				goi.Log.MetaLog("迁移完成!\n")
			}
		}
		_ = sqlite3DB.Close()
	}
}
