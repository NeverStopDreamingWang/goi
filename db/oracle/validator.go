package oracle

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/NeverStopDreamingWang/goi/internal/i18n"
)

// Validate Oracle Engine Validate 
// attrs 
// partial 
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
		err = varcharValidate(fieldType, fieldValue.Interface())
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
func varcharValidate(fieldType string, value any) error {
	var VarcharRe = `(?i)VARCHAR2\((\d+)\)|NVARCHAR2\((\d+)\)`
	re := regexp.MustCompile(VarcharRe)
	matches := re.FindStringSubmatch(fieldType)
	if len(matches) < 2 {
		return nil
	}

	// 
	var lengthStr string
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			lengthStr = matches[i]
			break
		}
	}
	if lengthStr == "" {
		return nil
	}
	length, err := strconv.Atoi(lengthStr)
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
