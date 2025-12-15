package sqlserver

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// Validate SQL Server Engine Validate
// attrs 校验数据
// partial 是否仅验证提供的非零值字段，零值字段跳过不验证必填项
func Validate(attrs Model, partial bool) error {
	validatedDataValue := reflect.ValueOf(attrs)
	if validatedDataValue.Kind() == reflect.Ptr {
		validatedDataValue = validatedDataValue.Elem()
	}
	modelType := validatedDataValue.Type()

	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	if modelType.Kind() != reflect.Struct {
		isNotStructPtrErrorMsg := i18n.T("serializer.is_not_struct_ptr", map[string]any{
			"name": "ModelSerializer.model",
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

func getComment(field reflect.StructField) string {
	fieldComment, ok := field.Tag.Lookup("comment")
	if ok {
		return fieldComment
	}
	fieldType, ok := field.Tag.Lookup("field_type")
	if ok {
		var commentRe = `(?i)COMMENT\s+['"]([^'"]+)['"]`
		re := regexp.MustCompile(commentRe)
		matches := re.FindStringSubmatch(fieldType)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func validateField(field reflect.StructField, fieldValue reflect.Value) error {
	field_name, ok := field.Tag.Lookup("field_name")
	if !ok {
		field_name = strings.ToLower(field.Name)
	}
	fieldType, ok := field.Tag.Lookup("field_type")
	if !ok { // 无 field_type 标签，跳过验证
		return nil
	}

	if field_name == "id" {
		return nil
	}

	err := validateFieldType(fieldType, fieldValue)
	if err != nil {
		comment := getComment(field)
		if comment == "" {
			comment = field_name
		}
		return fmt.Errorf(`"%v" %v`, comment, err.Error())
	}
	return nil
}

// 根据 field_type 校验字段值
func validateFieldType(fieldType string, fieldValue reflect.Value) error {
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() && isNotNullValidate(fieldType) {
			fieldIsNotNullErrorMsg := i18n.T("serializer.field_is_not_null_error")
			return errors.New(fieldIsNotNullErrorMsg)
		}
		fieldValue = fieldValue.Elem()
	}

	var err error
	switch fieldValue.Kind() {
	case reflect.String:
		// NVARCHAR/VARCHAR 长度校验
		err = varcharValidate(fieldType, fieldValue.Interface())
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// 不允许为空
func isNotNullValidate(fieldType string) bool {
	var re *regexp.Regexp

	// IDENTITY 自增列不做 NOT NULL 校验
	var identityRe = `(?i)IDENTITY\b`
	re = regexp.MustCompile(identityRe)
	if re.MatchString(fieldType) {
		return false
	}

	// DEFAULT 默认值
	var defaultRe = `(?i)DEFAULT`
	re = regexp.MustCompile(defaultRe)
	if re.MatchString(fieldType) {
		return false
	}

	var notNullRe = `(?i)NOT\s+NULL`
	re = regexp.MustCompile(notNullRe)
	return re.MatchString(fieldType)
}

// NVARCHAR/VARCHAR 长度校验
func varcharValidate(fieldType string, value any) error {
	var VarcharRe = `(?i)(N?VARCHAR)\((\d+)\)`
	re := regexp.MustCompile(VarcharRe)
	matches := re.FindStringSubmatch(fieldType)
	if len(matches) < 3 {
		return nil
	}

	length, err := strconv.Atoi(matches[2])
	if err != nil {
		panic(err)
	}

	strValue, ok := value.(string)
	if !ok {
		fieldTypeErrorMsg := i18n.T("serializer.field_type_string_error")
		return errors.New(fieldTypeErrorMsg)
	}

	if len(strValue) > length {
		varcharLenErrorMsg := i18n.T("serializer.varchar_len_error", map[string]any{
			"length": length,
		})
		return errors.New(varcharLenErrorMsg)
	}
	return nil
}
