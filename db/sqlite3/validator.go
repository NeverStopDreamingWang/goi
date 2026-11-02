package sqlite3

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/NeverStopDreamingWang/goi/v2/db"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// Validate 根据模型字段定义校验数据。
//
// 参数:
//   - attrs: Model 待校验的模型数据
//   - partial: bool true 时跳过 nil 指针，false 时按列约束校验；不影响 Insert/Update 的写入字段
//
// 返回:
//   - error: 模型无效或字段校验失败时返回错误
//
// 说明:
//   - 使用调用开始时的包级 ColumnParser，不读取已有 Engine 的配置
//   - 仅校验解析器选中的字段，列名为 id 的字段跳过
//   - 仅检查 nil 指针的非空约束，其他约束由数据库处理
func Validate(attrs Model, partial bool) error {
	parser := GetColumnParser()
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

		// 根据当前解析器返回的字段定义校验字段值。
		if err := validateField(field, fieldValue, parser); err != nil {
			return err
		}
	}
	return nil
}

// getComment 获取字段的 comment 标签，未设置时返回空字符串。
func getComment(field reflect.StructField) string {
	fieldComment, ok := field.Tag.Lookup("comment")
	if ok {
		return fieldComment
	}
	return ""
}

// validateField 按解析后的列定义校验字段，并为错误补充字段说明。
func validateField(field reflect.StructField, fieldValue reflect.Value, parser db.ColumnParser) error {
	columnName, columnType, ok := parser.Parse(field)
	if !ok {
		return nil
	}

	if columnName == "id" {
		return nil
	}

	err := validateFieldType(columnType, fieldValue)
	if err != nil {
		comment := getComment(field)
		if comment == "" {
			comment = columnName
		}
		return fmt.Errorf(`"%v" %v`, comment, err.Error())
	}
	return nil
}

// validateFieldType 校验 nil 指针的非空约束。
func validateFieldType(fieldType string, fieldValue reflect.Value) error {
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() && isNotNullValidate(fieldType) {
			fieldIsNotNullErrorMsg := i18n.T("serializer.field_is_not_null_error")
			return errors.New(fieldIsNotNullErrorMsg)
		}
		fieldValue = fieldValue.Elem()
	}
	return nil
}

// isNotNullValidate 判断是否须校验 NOT NULL，带自增或默认值时不校验。
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
