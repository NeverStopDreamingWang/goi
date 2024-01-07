package model

import (
	"fmt"
	"reflect"
	"strings"
)

// SQLite3 模型设置
type SQLite3Settings struct {
	TABLE_NAME string // 表名

	// 自定义配置
	MySettings MySettings
}

// 设置自定义配置
func (modelsettings *SQLite3Settings) Set(name string, value any) {
	modelsettings.MySettings[name] = value
}

// 获取自定义配置
func (modelsettings SQLite3Settings) Get(name string) any {
	return modelsettings.MySettings[name]
}

// 模型类
type SQLite3Model interface {
	ModelSet() *SQLite3Settings
}

// SQLite3 创建迁移
type SQLite3MakeMigrations struct {
	DATABASES []string
	MODELS    []SQLite3Model
}

// SQLite3 迁移
func MetaSQLite3Migrate(model SQLite3Model) string {
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
	createSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` (\n%v\n);", modelSetting.TABLE_NAME, strings.Join(field_slice, ",\n"))
	return createSql
}
