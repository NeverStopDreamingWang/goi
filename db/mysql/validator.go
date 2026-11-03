package mysql

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/NeverStopDreamingWang/goi/v3/db"
	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// 预编译的正则表达式
var (
	commentRe       = regexp.MustCompile(`(?i)COMMENT\s+['"]([^'"]+)['"]`)
	autoIncrementRe = regexp.MustCompile(`(?i)AUTO_INCREMENT`)
	defaultRe       = regexp.MustCompile(`(?i)DEFAULT`)
	notNullRe       = regexp.MustCompile(`(?i)NOT\s+NULL`)
	varcharRe       = regexp.MustCompile(`(?i)VARCHAR\((\d+)\)`)
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
//   - 检查 nil 指针的非空约束和 VARCHAR 字符数，其他约束由数据库处理
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

// getComment 优先获取 comment 标签，否则从解析器返回的列类型定义中提取注释。
func getComment(field reflect.StructField, columnType string) string {
	fieldComment, ok := field.Tag.Lookup("comment")
	if ok {
		return fieldComment
	}
	matches := commentRe.FindStringSubmatch(columnType)
	if len(matches) > 1 {
		return matches[1]
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
		comment := getComment(field, columnType)
		if comment == "" {
			comment = columnName
		}
		return fmt.Errorf(`"%v" %v`, comment, err.Error())
	}
	return nil
}

// validateFieldType 校验 nil 指针的非空约束和字符串的 VARCHAR 长度。
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
		err = varcharValidate(fieldType, fieldValue.String())
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// isNotNullValidate 判断是否须校验 NOT NULL，带自增或默认值时不校验。
func isNotNullValidate(fieldType string) bool {
	if autoIncrementRe.MatchString(fieldType) {
		return false
	}
	if defaultRe.MatchString(fieldType) {
		return false
	}
	return notNullRe.MatchString(fieldType)
}

// varcharValidate 校验 VARCHAR 的字符数，非 VARCHAR 定义不处理。
func varcharValidate(fieldType string, strValue string) error {
	matches := varcharRe.FindStringSubmatch(fieldType)
	if len(matches) < 2 {
		return nil
	}
	// 提取括号中的数字并转换为整数
	length, err := strconv.Atoi(matches[1])
	if err != nil {
		panic(err)
	}

	if utf8.RuneCountInString(strValue) > length {
		varcharLenErrorMsg := i18n.T("serializer.varchar_len_error", map[string]any{
			"length": length,
		})
		return errors.New(varcharLenErrorMsg)
	}
	return nil
}
