package sqlite3

import (
	"errors"
	"reflect"

	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/NeverStopDreamingWang/goi/model/sqlite3"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 模型类
type ModelSerializer struct {
	Instance sqlite3.SQLite3Model
}

// sqlite3DB *db.SQLite3DB 数据库对象
// attrs 验证数据
// partial 是否仅验证提供的非零值字段，零值字段跳过不验证必填选项
func (modelSerializer ModelSerializer) Validate(sqlite3DB *db.SQLite3DB, attrs sqlite3.SQLite3Model, partial bool) error {
	validatedDataValue := reflect.ValueOf(attrs)
	modelType := validatedDataValue.Type()

	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	if modelType.Kind() != reflect.Struct {
		isNotStructPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "serializer.is_not_struct_ptr",
			TemplateData: map[string]interface{}{
				"name": "ModelSerializer.model",
			},
		})
		return errors.New(isNotStructPtrErrorMsg)
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldValue := validatedDataValue.Field(i)

		if partial == true && fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			continue
		}

		// 根据标签校验字段值
		if err := validateField(field, fieldValue); err != nil {
			return err
		}
	}
	return nil
}

func (modelSerializer ModelSerializer) Update(instance sqlite3.SQLite3Model, validated_data sqlite3.SQLite3Model) {
	instanceValue := reflect.ValueOf(instance)
	validatedDataValue := reflect.ValueOf(validated_data)

	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}
	if validatedDataValue.Kind() == reflect.Ptr {
		validatedDataValue = validatedDataValue.Elem()
	}
	instanceType := instanceValue.Type()

	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)
		instanceField := instanceValue.Field(i)
		validatedField := validatedDataValue.FieldByName(field.Name)

		if validatedField.Kind() == reflect.Ptr && validatedField.IsNil() {
			continue
		}
		if !instanceField.CanSet() {
			continue
		}
		instanceField.Set(validatedField)
	}
}
