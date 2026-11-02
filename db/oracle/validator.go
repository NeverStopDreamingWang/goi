package oracle

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/NeverStopDreamingWang/goi/v2/db"
	"github.com/NeverStopDreamingWang/goi/v2/internal/i18n"
)

// commentRe 从列类型定义中提取注释。
var commentRe = regexp.MustCompile(`(?i)COMMENT\s+['"]([^'"]+)['"]`)

// Validate 按模型列定义校验数据
//
// 参数:
//   - attrs: Model 待校验模型，支持结构体或非 nil 结构体指针
//   - partial: bool true 时跳过 nil 指针字段，仅控制校验，不控制写入
//
// 返回:
//   - error: 模型或字段校验错误
//
// 说明:
//   - 每次使用当前全局 ColumnParser，不使用已有 Engine 的解析器
//   - 校验必填指针和字符串长度；字段范围与列定义由 ColumnParser 决定
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

// getComment 优先读取 comment 标签，缺省时从列类型中提取注释
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

// validateField 校验解析器选中的字段，跳过 id 列并为错误添加字段说明
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

// validateFieldType 校验 nil 指针的必填约束和字符串长度
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
		// VARCHAR2 区分 BYTE/CHAR，NVARCHAR2 按字符数校验。
		err = varcharValidate(fieldType, fieldValue.String())
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// isNotNullValidate 判断列是否必填，自动生成列和带默认值的列除外
func isNotNullValidate(fieldType string) bool {
	var re *regexp.Regexp

	// IDENTITY 列由数据库生成值。
	var identityRe = `(?i)GENERATED\s+ALWAYS\s+AS\s+IDENTITY|IDENTITY`
	re = regexp.MustCompile(identityRe)
	if re.MatchString(fieldType) {
		return false
	}

	// 带 DEFAULT 的列允许调用方省略。
	var defaultRe = `(?i)DEFAULT`
	re = regexp.MustCompile(defaultRe)
	if re.MatchString(fieldType) {
		return false
	}

	var notNullRe = `(?i)NOT\s+NULL`
	re = regexp.MustCompile(notNullRe)
	return re.MatchString(fieldType)
}

// varcharValidate 校验 VARCHAR2/NVARCHAR2 长度；默认按字节，CHAR 或 NVARCHAR2 按字符
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
