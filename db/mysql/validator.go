package mysql

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// MySQL Engine Validate 验证器
// attrs 验证数据
// partial 是否仅验证提供的非零值字段，零值字段跳过不验证必填选
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

func getComment(field reflect.StructField) string {
	fieldComment, ok := field.Tag.Lookup("comment")
	if ok {
		return fieldComment
	}
	fieldType, ok := field.Tag.Lookup("field_type")
	if !ok {
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
			fieldIsNotNullErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "serializer.field_is_not_null_error",
			})
			return errors.New(fieldIsNotNullErrorMsg)
		}
		fieldValue = fieldValue.Elem()
	}
	var err error
	switch fieldValue.Kind() {
	case reflect.String:
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
	var autoIncrementRe = `(?i)AUTO_INCREMENT`
	re = regexp.MustCompile(autoIncrementRe)
	if re.MatchString(fieldType) {
		return false
	}
	var defaultRe = `(?i)DEFAULT`
	re = regexp.MustCompile(defaultRe)
	if re.MatchString(fieldType) {
		return false
	}

	var notNullRe = `(?i)NOT\s+NULL`
	re = regexp.MustCompile(notNullRe)
	return re.MatchString(fieldType)
}

// varchar 类型
func varcharValidate(fieldType string, value interface{}) error {
	var VarcharRe = `(?i)VARCHAR\((\d+)\)`
	re := regexp.MustCompile(VarcharRe)
	matches := re.FindStringSubmatch(fieldType)
	if len(matches) < 2 {
		return nil
	}
	// 提取括号中的数字并转换为整数
	length, err := strconv.Atoi(matches[1])
	if err != nil {
		panic(err)
	}

	// 检查 value 类型
	strValue, ok := value.(string)
	if !ok {
		fieldTypeErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "serializer.field_type_string_error",
		})
		return errors.New(fieldTypeErrorMsg)
	}

	if len(strValue) > length {
		varcharLenErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "serializer.varchar_len_error",
			TemplateData: map[string]interface{}{
				"length": length,
			},
		})
		return errors.New(varcharLenErrorMsg)
	}
	return nil
}
