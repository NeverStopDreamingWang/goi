package oracle

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// Validate Oracle Engine Validate 
// attrs 
// partial 
func Validate(attrs Model, partial bool) error {
	validatedDataValue := reflect.ValueOf(attrs)
	if !validatedDataValue.IsValid() || (validatedDataValue.Kind() == reflect.Ptr && validatedDataValue.IsNil()) {
		isNotStructPtrErrorMsg := i18n.T("serializer.is_not_struct_ptr", map[string]any{
			"name": "ModelSerializer.model",
		})
		return errors.New(isNotStructPtrErrorMsg)
	}
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

		// 
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
	if !ok { //  field_type 
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

//  field_type 
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
		// VARCHAR2 / NVARCHAR2 
		err = varcharValidate(fieldType, fieldValue.String())
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// 
func isNotNullValidate(fieldType string) bool {
	var re *regexp.Regexp

	//  IDENTITY 
	var identityRe = `(?i)GENERATED\s+ALWAYS\s+AS\s+IDENTITY|IDENTITY`
	re = regexp.MustCompile(identityRe)
	if re.MatchString(fieldType) {
		return false
	}

	// DEFAULT 
	var defaultRe = `(?i)DEFAULT`
	re = regexp.MustCompile(defaultRe)
	if re.MatchString(fieldType) {
		return false
	}

	var notNullRe = `(?i)NOT\s+NULL`
	re = regexp.MustCompile(notNullRe)
	return re.MatchString(fieldType)
}

// VARCHAR2 / NVARCHAR2 
func varcharValidate(fieldType string, strValue string) error {
	var VarcharRe = `(?i)(N?VARCHAR2)\((\d+)(?:\s+(BYTE|CHAR))?\)`
	re := regexp.MustCompile(VarcharRe)
	matches := re.FindStringSubmatch(fieldType)
	if len(matches) < 3 {
		return nil
	}

	length, err := strconv.Atoi(matches[2])
	if err != nil {
		panic(err)
	}

	valueLength := len(strValue)
	if strings.EqualFold(matches[1], "NVARCHAR2") || strings.EqualFold(matches[3], "CHAR") {
		valueLength = utf8.RuneCountInString(strValue)
	}

	if valueLength > length {
		varcharLenErrorMsg := i18n.T("serializer.varchar_len_error", map[string]any{
			"length": length,
		})
		return errors.New(varcharLenErrorMsg)
	}
	return nil
}
