package model

import (
	"fmt"
	"reflect"
	"strings"
)

// Mysql 模型设置
type MysqlSettings struct {
	TableName       string // 设置表名
	ENGINE          string // 设置存储引擎，默认: InnoDB
	AUTO_INCREMENT  int    // 设置自增长起始值
	COMMENT         string // 设置表注释
	DEFAULT_CHARSET string // 设置默认字符集，如: utf8mb4
	COLLATE         string // 设置校对规则，如 utf8mb4_0900_ai_ci;
	ROW_FORMAT      string // 设置行的存储格式，如 DYNAMIC, COMPACT, FULL.
	DATA_DIRECTORY  string // 设置数据存储目录
	INDEX_DIRECTORY string // 设置索引存储目录
	STORAGE         string // 设置存储类型，如 DISK、MEMORY、CSV
	CHECKSUM        int    // 表格的校验和算法，如 1 开启校验和
	DELAY_KEY_WRITE int    // 控制非唯一索引的写延迟，如 1
	MAX_ROWS        int    // 设置最大行数
	MIN_ROWS        int    // 设置最小行数
	PARTITION_BY    string // 定义分区方式，如 RANGE、HASH、LIST

	// 自定义配置
	MySettings MySettings
}

// 设置自定义配置
func (mysqlmodelsettings *MysqlSettings) Set(name string, value any) {
	mysqlmodelsettings.MySettings[name] = value
}

// 获取自定义配置
func (mysqlmodelsettings MysqlSettings) Get(name string) any {
	return mysqlmodelsettings.MySettings[name]
}

// 模型类
type MysqlModel interface {
	ModelSet() *MysqlSettings
}

// Mysql 创建迁移
type MysqlMakeMigrations struct {
	DATABASES []string
	MODELS    []MysqlModel
}

// Mysql 迁移
func MetaMysqlMigrate(model MysqlModel) string {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	modelSetting := model.ModelSet()

	field_slice := make([]string, modelType.NumField(), modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldName := strings.ToLower(field.Name)
		fieldType := field.Tag.Get("field")
		field_slice[i] = fmt.Sprintf("`%v` %v", fieldName, fieldType)
	}
	createSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` (\n%v\n)", modelSetting.TableName, strings.Join(field_slice, ",\n"))
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
