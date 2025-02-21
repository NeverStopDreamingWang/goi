package sqlite3

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func getComment(field reflect.StructField) string {
	fieldComment, ok := field.Tag.Lookup("comment")
	if ok {
		return fieldComment
	}
	return ""
}

func validateField(field reflect.StructField, fieldValue reflect.Value) error {
	field_name, ok := field.Tag.Lookup("field_name")
	if !ok {
		field_name = strings.ToLower(field.Name)
	}
	fieldType, ok := field.Tag.Lookup("field_type")
	if !ok {
		fieldTagFieldTypeErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "serializer.field_tag_field_type_error",
			TemplateData: map[string]interface{}{
				"name": field_name,
			},
		})
		panic(fieldTagFieldTypeErrorMsg)
	}

	err := validateFieldType(fieldType, fieldValue)
	if err != nil {
		comment := getComment(field)
		if comment == "" {
			comment = field_name
		}
		return errors.New(comment + err.Error())
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
	return nil
}

// 不允许为空
func isNotNullValidate(fieldType string) bool {
	var re *regexp.Regexp
	var autoIncrementRe = `(?i)AUTOINCREMENT`
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
