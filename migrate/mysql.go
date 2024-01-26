package migrate

import (
	"fmt"
	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/model"
	"reflect"
	"strings"
)

// MySQL 迁移
func MySQLMigrate(Migrations model.MySQLMakeMigrations) {
	if len(Migrations.DATABASES) == 0 {
		panic("请指定迁移数据库！")
	}
	// 迁移数据库
	for _, DBName := range Migrations.DATABASES {
		Database, ok := goi.Settings.DATABASES[DBName]
		if ok == false {
			panic(fmt.Sprintf("配置数据库: %v 不存在！", DBName))
		}
		if strings.ToLower(Database.ENGINE) != "mysql" {
			continue
		}
		mysqlDB, err := db.MySQLConnect(DBName)
		if err != nil {
			panic(fmt.Sprintf("连接 MySQL [%v] 数据库 错误: %v", DBName, err))
		}

		// 获取所有表
		TabelSclie := make([]string, 0)
		rows, err := mysqlDB.Query("SHOW TABLES;")
		if err != nil {
			panic(fmt.Sprintf("连接 MySQL [%v] 数据库 错误: %v", DBName, err))
		}
		for rows.Next() {
			var tableData string
			err = rows.Scan(&tableData)
			if err != nil {
				panic(fmt.Sprintf("连接 MySQL [%v] 数据库 错误: %v", DBName, err))
			}
			TabelSclie = append(TabelSclie, tableData)
		}
		_ = rows.Close()

		for _, Model := range Migrations.MODELS {
			TableName := Model.ModelSet().TABLE_NAME
			createSql := createTableSql(Model)

			if TableIsExists(TableName, TabelSclie) { // 判断是否存在
				continue
			}

			_, err = mysqlDB.Execute(createSql)
			if err != nil {
				panic(fmt.Sprintf("迁移错误: %v", err))
			}
			goi.Log.MetaLog(fmt.Sprintf("Migrate MySQL: %v DataBase: %v Table: %v ...ok!", DBName, Database.NAME, Model.ModelSet().TABLE_NAME))
		}
		_ = mysqlDB.Close()
	}
}

// MySQL 创建表语句
func createTableSql(model model.MySQLModel) string {
	modelSetting := model.ModelSet()

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	field_slice := make([]string, modelType.NumField(), modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldName := strings.ToLower(field.Name)
		fieldType := field.Tag.Get("field")
		field_slice[i] = fmt.Sprintf("`%v` %v", fieldName, fieldType)
	}
	createSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` (\n%v\n)", modelSetting.TABLE_NAME, strings.Join(field_slice, ",\n"))
	if modelSetting.ENGINE != "" { // 设置存储引擎
		createSql += fmt.Sprintf("\nENGINE=%v", modelSetting.ENGINE)
	}
	if modelSetting.AUTO_INCREMENT != 0 { // 设置自增长起始值
		createSql += fmt.Sprintf("\nAUTO_INCREMENT=%v", modelSetting.AUTO_INCREMENT)
	}
	if modelSetting.COMMENT != "" { // 设置表注释
		createSql += fmt.Sprintf("\nCOMMENT='%v'", modelSetting.COMMENT)
	}
	if modelSetting.DEFAULT_CHARSET != "" { // 设置默认字符集
		createSql += fmt.Sprintf("\nDEFAULT CHARSET=%v", modelSetting.DEFAULT_CHARSET)
	}
	if modelSetting.COLLATE != "" { // 设置校对规则
		createSql += fmt.Sprintf("\nCOLLATE=%v", modelSetting.COLLATE)
	}
	if modelSetting.ROW_FORMAT != "" { // 设置行的存储格式
		createSql += fmt.Sprintf("\nROW_FORMAT=%v", modelSetting.ROW_FORMAT)
	}
	if modelSetting.DATA_DIRECTORY != "" { // 设置数据存储目录
		createSql += fmt.Sprintf("\nDATA DIRECTORY='%v'", modelSetting.DATA_DIRECTORY)
	}
	if modelSetting.INDEX_DIRECTORY != "" { // 设置索引存储目录
		createSql += fmt.Sprintf("\nINDEX DIRECTORY='%v'", modelSetting.INDEX_DIRECTORY)
	}
	if modelSetting.STORAGE != "" { // 设置存储类型
		createSql += fmt.Sprintf("\nSTORAGE=%v", modelSetting.STORAGE)
	}
	if modelSetting.CHECKSUM != 0 { // 表格的校验和算法
		createSql += fmt.Sprintf("\nCHECKSUM=%v", modelSetting.CHECKSUM)
	}
	if modelSetting.DELAY_KEY_WRITE != 0 { // 控制非唯一索引的写延迟
		createSql += fmt.Sprintf("\nDELAY_KEY_WRITE=%v", modelSetting.DELAY_KEY_WRITE)
	}
	if modelSetting.MAX_ROWS != 0 { // 设置最大行数
		createSql += fmt.Sprintf("\nMAX_ROWS=%v", modelSetting.MAX_ROWS)
	}
	if modelSetting.MIN_ROWS != 0 { // 设置最小行数
		createSql += fmt.Sprintf("\nMIN_ROWS=%v", modelSetting.MIN_ROWS)
	}
	if modelSetting.PARTITION_BY != "" { // 定义分区方式
		createSql += fmt.Sprintf("\nPARTITION_BY %v", modelSetting.PARTITION_BY)
	}
	createSql += ";"
	return createSql
}
