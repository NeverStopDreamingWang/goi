package mysql

import (
	"net/http"
	"reflect"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/db"
	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/NeverStopDreamingWang/goi/model/mysql"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// 模型类
type ModelSerializer struct {
	Instance mysql.MySQLModel
}

func (modelSerializer ModelSerializer) Validate(mysqlDB *db.MySQLDB, attrs mysql.MySQLModel, partial bool) goi.ValidationError {
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
		return goi.NewValidationError(http.StatusBadRequest, isNotStructPtrErrorMsg)
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

func (modelSerializer ModelSerializer) Update(instance mysql.MySQLModel, validated_data mysql.MySQLModel) {
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
