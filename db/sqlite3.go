package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/NeverStopDreamingWang/goi"
	"github.com/NeverStopDreamingWang/goi/internal/language"
	"github.com/NeverStopDreamingWang/goi/model/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type SQLite3DB struct {
	name        string               // 连接名称
	DB          *sql.DB              // 数据库连接对象
	transaction *sql.Tx              // 事务
	model       sqlite3.SQLite3Model // 模型
	fields      []string             // 模型字段
	field_sql   []string             // 表字段
	where_sql   []string             // 条件语句
	limit_sql   string               // 分页
	group_sql   string               // 分组
	order_sql   string               // 排序
	sql         string               // 执行 sql
	args        []interface{}        // 条件参数
}

// 连接 SQLite3 数据库
func SQLite3Connect(UseDataBases string) *SQLite3DB {
	database, ok := goi.Settings.DATABASES[UseDataBases]
	if ok == false {
		databasesNotErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.databases_not_error",
			TemplateData: map[string]interface{}{
				"name": UseDataBases,
			},
		})
		goi.Log.Error(databasesNotErrorMsg)
		panic(databasesNotErrorMsg)
	}
	return &SQLite3DB{
		name:      UseDataBases,
		DB:        database.DB(),
		model:     nil,
		fields:    nil,
		field_sql: nil,
		where_sql: nil,
		limit_sql: "",
		group_sql: "",
		order_sql: "",
		sql:       "",
		args:      nil,
	}
}

// 获取数据库别名
func (sqlite3DB SQLite3DB) Name() string {
	return sqlite3DB.name
}

// 获取SQL语句
func (sqlite3DB SQLite3DB) GetSQL() string {
	return sqlite3DB.sql
}

// 事务
func (sqlite3DB *SQLite3DB) WithTransaction(transactionFunc func(sqlite3DB *SQLite3DB, args ...interface{}) error, args ...interface{}) error {
	var err error
	if sqlite3DB.transaction != nil {
		transactionCannotBeNestedErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.transaction_cannot_be_nested_error",
		})
		return errors.New(transactionCannotBeNestedErrorMsg)
	}

	defer func() {
		sqlite3DB.transaction = nil
	}()

	sqlite3DB.transaction, err = sqlite3DB.DB.Begin()
	if err != nil {
		return err
	}
	err = transactionFunc(sqlite3DB, args...)
	if err != nil {
		_ = sqlite3DB.transaction.Rollback()
		return err
	}

	// 提交事务
	err = sqlite3DB.transaction.Commit()
	if err != nil {
		_ = sqlite3DB.transaction.Rollback()
		return err
	}
	return nil
}

// 执行语句
func (sqlite3DB *SQLite3DB) Execute(query string, args ...interface{}) (sql.Result, error) {
	if sqlite3DB.transaction != nil {
		return sqlite3DB.transaction.Exec(query, args...)
	} else {
		return sqlite3DB.DB.Exec(query, args...)
	}
}

// 查询语句
func (sqlite3DB *SQLite3DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := sqlite3DB.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, err
}

// 模型迁移
func (sqlite3DB *SQLite3DB) Migrate(db_name string, model sqlite3.SQLite3Model) {
	var err error
	modelSettings := model.ModelSet()

	row := sqlite3DB.DB.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name =?;", modelSettings.TABLE_NAME)

	var exists int
	err = row.Scan(&exists)
	if err != nil {
		selectErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.select_error",
			TemplateData: map[string]interface{}{
				"engine":  "SQLite3",
				"db_name": sqlite3DB.name,
				"err":     err,
			},
		})
		goi.Log.Error(selectErrorMsg)
		panic(selectErrorMsg)
	}

	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	tampFieldMap := make(map[string]bool)
	fieldSqlSlice := make([]string, modelType.NumField(), modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldName, ok := field.Tag.Lookup("field_name")
		if !ok {
			fieldName = strings.ToLower(field.Name)
		}
		fieldType, ok := field.Tag.Lookup("field_type")
		tampFieldMap[fieldName] = true
		fieldSqlSlice[i] = fmt.Sprintf("  `%v` %v", fieldName, fieldType)
	}
	createSql := fmt.Sprintf("CREATE TABLE `%v` (\n%v\n)", modelSettings.TABLE_NAME, strings.Join(fieldSqlSlice, ",\n"))

	if exists != 1 { // 创建表
		if modelSettings.MigrationsHandler.BeforeHandler != nil { // 迁移之前处理
			beforeMigrationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.before_migration",
			})
			goi.Log.Info(beforeMigrationMsg)
			err = modelSettings.MigrationsHandler.BeforeHandler()
			if err != nil {
				beforeMigrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "database.before_migration_error",
					TemplateData: map[string]interface{}{
						"err": err,
					},
				})
				goi.Log.Error(beforeMigrationErrorMsg)
				panic(beforeMigrationErrorMsg)
			}
		}
		migrationModelMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.migration",
			TemplateData: map[string]interface{}{
				"engine":  "SQLite3",
				"name":    sqlite3DB.name,
				"db_name": db_name,
				"tb_name": modelSettings.TABLE_NAME,
			},
		})
		goi.Log.Info(migrationModelMsg)
		_, err = sqlite3DB.Execute(createSql)
		if err != nil {
			migrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.migration_error",
				TemplateData: map[string]interface{}{
					"err": err,
				},
			})
			goi.Log.Error(migrationErrorMsg)
			panic(migrationErrorMsg)
		}

		if modelSettings.MigrationsHandler.AfterHandler != nil { // 迁移之后处理
			afterMigrationMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.after_migration",
			})
			goi.Log.Info(afterMigrationMsg)
			err = modelSettings.MigrationsHandler.AfterHandler()
			if err != nil {
				afterMigrationErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "database.after_migration_error",
					TemplateData: map[string]interface{}{
						"err": err,
					},
				})
				goi.Log.Error(afterMigrationErrorMsg)
				panic(afterMigrationErrorMsg)
			}
		}

	}
}

func (sqlite3DB *SQLite3DB) isSetModel() {
	if sqlite3DB.model == nil {
		notSetModelErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.not_SetModel_error",
		})
		goi.Log.Error(notSetModelErrorMsg)
		panic(notSetModelErrorMsg)
	}
}

// 设置使用模型  return 当前 *SQLite3DB 本身
func (sqlite3DB *SQLite3DB) SetModel(model sqlite3.SQLite3Model) *SQLite3DB {
	sqlite3DB.model = model
	ModelType := reflect.TypeOf(sqlite3DB.model)
	// 获取字段
	sqlite3DB.fields = make([]string, ModelType.NumField())
	sqlite3DB.field_sql = make([]string, ModelType.NumField())
	for i := 0; i < ModelType.NumField(); i++ {
		field := ModelType.Field(i)
		sqlite3DB.fields[i] = field.Name

		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		sqlite3DB.field_sql[i] = field_name
	}
	sqlite3DB.where_sql = nil
	sqlite3DB.limit_sql = ""
	sqlite3DB.group_sql = ""
	sqlite3DB.order_sql = ""
	sqlite3DB.sql = ""
	sqlite3DB.args = nil
	return sqlite3DB
}

// 设置查询结果字段  return 当前 *SQLite3DB 的副本指针
func (sqlite3DB SQLite3DB) Fields(fields ...string) *SQLite3DB {
	ModelType := reflect.TypeOf(sqlite3DB.model)
	// 获取字段
	sqlite3DB.fields = make([]string, len(fields))
	sqlite3DB.field_sql = make([]string, len(fields))
	for i, fieldName := range fields {
		field, ok := ModelType.FieldByName(fieldName)
		if ok == false {
			fieldIsNotErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.field_is_not_error",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			goi.Log.Error(fieldIsNotErrorMsg)
			panic(fieldIsNotErrorMsg)
		}
		sqlite3DB.fields[i] = field.Name

		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		sqlite3DB.field_sql[i] = field_name
	}
	return &sqlite3DB
}

// 插入数据库
func (sqlite3DB *SQLite3DB) Insert(ModelData sqlite3.SQLite3Model) (sql.Result, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}

	insertValues := make([]interface{}, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		field := ModelValue.FieldByName(fieldName)
		if field.Elem().IsValid() {
			insertValues[i] = field.Elem().Interface()
		} else {
			insertValues[i] = field.Interface()
		}
	}

	fieldsSQL := strings.Join(sqlite3DB.field_sql, "`,`")

	tempValues := make([]string, len(sqlite3DB.fields))
	for i := 0; i < len(sqlite3DB.fields); i++ {
		tempValues[i] = "?"
	}
	valuesSQL := strings.Join(tempValues, ",")

	sqlite3DB.sql = fmt.Sprintf("INSERT INTO `%v` (`%v`) VALUES (%v)", TableName, fieldsSQL, valuesSQL)
	return sqlite3DB.Execute(sqlite3DB.sql, insertValues...)
}

// 设置条件查询语句  return 当前 *SQLite3DB 的副本指针
func (sqlite3DB SQLite3DB) Where(query string, args ...interface{}) *SQLite3DB {
	queryParts := strings.Split(query, "?")
	if len(queryParts)-1 != len(args) {
		whereArgsPlaceholderErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.where_args_placeholder_error",
		})
		goi.Log.Error(whereArgsPlaceholderErrorMsg)
		panic(whereArgsPlaceholderErrorMsg)
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(queryParts[0])
	for i, param := range args {
		// 检查是否是切片
		paramValue := reflect.ValueOf(param)
		if paramValue.Kind() == reflect.Ptr {
			paramValue = paramValue.Elem()
		}

		if paramValue.Kind() == reflect.Slice {
			placeholders := "(" + strings.Repeat("?,", paramValue.Len()-1) + "?" + ")"
			queryBuilder.WriteString(placeholders)

			// 将切片元素加入 args
			for j := 0; j < paramValue.Len(); j++ {
				sqlite3DB.args = append(sqlite3DB.args, paramValue.Index(j).Interface())
			}
		} else {
			// 普通参数，直接添加占位符
			queryBuilder.WriteString("?")
			sqlite3DB.args = append(sqlite3DB.args, param)
		}
		// 添加切割后的下一部分
		queryBuilder.WriteString(queryParts[i+1])
	}
	sqlite3DB.where_sql = append(sqlite3DB.where_sql, queryBuilder.String())
	return &sqlite3DB
}

// 分组  return 当前 *SQLite3DB 的副本指针
func (sqlite3DB SQLite3DB) GroupBy(groups ...string) *SQLite3DB {
	var groupFields []string
	for _, group := range groups {
		group = strings.Trim(group, " `'\"")
		if group == "" {
			continue
		}
		groupFields = append(groupFields, fmt.Sprintf("`%s`", group))
	}
	if len(groupFields) > 0 {
		sqlite3DB.group_sql = fmt.Sprintf(" GROUP BY %s", strings.Join(groupFields, ", "))
	} else {
		sqlite3DB.group_sql = ""
	}
	return &sqlite3DB
}

// 排序  return 当前 *SQLite3DB 的副本指针
func (sqlite3DB SQLite3DB) OrderBy(orders ...string) *SQLite3DB {
	var orderFields []string
	for _, order := range orders {
		order = strings.Trim(order, " `'\"")
		if order == "" {
			continue
		}

		var sequence string
		var orderSQL string
		if order[0] == '-' {
			orderSQL = order[1:]
			sequence = "DESC"
		} else {
			orderSQL = order
			sequence = "ASC"
		}
		orderFields = append(orderFields, fmt.Sprintf("`%s` %s", orderSQL, sequence))
	}
	if len(orderFields) > 0 {
		sqlite3DB.order_sql = fmt.Sprintf(" ORDER BY %s", strings.Join(orderFields, ", "))
	} else {
		sqlite3DB.order_sql = ""
	}
	return &sqlite3DB
}

// 分页
func (sqlite3DB *SQLite3DB) Page(page int, pagesize int) (int, int, error) {
	if page <= 0 {
		page = 1
	}
	if pagesize <= 0 {
		pagesize = 10 // 默认分页大小
	}
	offset := (page - 1) * pagesize
	sqlite3DB.limit_sql = fmt.Sprintf(" LIMIT %d OFFSET %d", pagesize, offset)
	total, err := sqlite3DB.Count()
	totalPages := int(math.Ceil(float64(total) / float64(pagesize)))
	return total, totalPages, err
}

// 执行查询语句获取数据
func (sqlite3DB *SQLite3DB) Select(queryResult interface{}) error {
	sqlite3DB.isSetModel()

	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.limit_sql = ""
		sqlite3DB.group_sql = ""
		sqlite3DB.order_sql = ""
		sqlite3DB.args = nil
	}()
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	var (
		isPtr    bool
		ItemType reflect.Type
		result   = reflect.ValueOf(queryResult)
	)
	if result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotPtrErrorMsg)
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Slice {
		isNotSlicePtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_slice_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotSlicePtrErrorMsg)
	}

	ItemType = result.Type().Elem() // 获取切片中的元素
	if ItemType.Kind() == reflect.Ptr {
		isPtr = true
		ItemType = ItemType.Elem()
	}

	fieldsSQl := strings.Join(sqlite3DB.field_sql, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	if sqlite3DB.group_sql != "" {
		sqlite3DB.sql += sqlite3DB.group_sql
	}
	if sqlite3DB.order_sql != "" {
		sqlite3DB.sql += sqlite3DB.order_sql
	}
	if sqlite3DB.limit_sql != "" {
		sqlite3DB.sql += sqlite3DB.limit_sql
	}

	rows, err := sqlite3DB.DB.Query(sqlite3DB.sql, sqlite3DB.args...)
	defer rows.Close()
	if err != nil {
		return err
	} else if rows.Err() != nil {
		return rows.Err()
	}

	for rows.Next() {
		values := make([]interface{}, len(sqlite3DB.fields))

		item := reflect.New(ItemType).Elem()

		for i, fieldName := range sqlite3DB.fields {
			fieldValue := item.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				values[i] = new(interface{})
			} else {
				values[i] = fieldValue.Addr().Interface()
			}
		}

		err = rows.Scan(values...)
		if err != nil {
			return err
		}
		if isPtr == true {
			result.Set(reflect.Append(result, item.Addr()))
		} else {
			result.Set(reflect.Append(result, item))
		}
	}
	return nil
}

// 返回第一条数据
func (sqlite3DB *SQLite3DB) First(queryResult interface{}) error {
	sqlite3DB.isSetModel()

	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.limit_sql = ""
		sqlite3DB.group_sql = ""
		sqlite3DB.order_sql = ""
		sqlite3DB.args = nil
	}()
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Ptr {
		isNotPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotPtrErrorMsg)
	}
	result = result.Elem()

	if kind := result.Kind(); kind != reflect.Struct {
		isNotStructPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_struct_ptr",
			TemplateData: map[string]interface{}{
				"name": "queryResult",
			},
		})
		return errors.New(isNotStructPtrErrorMsg)
	}

	fieldsSQl := strings.Join(sqlite3DB.field_sql, "`,`")

	sqlite3DB.sql = fmt.Sprintf("SELECT `%v` FROM `%v`", fieldsSQl, TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}
	if sqlite3DB.group_sql != "" {
		sqlite3DB.sql += sqlite3DB.group_sql
	}
	if sqlite3DB.order_sql != "" {
		sqlite3DB.sql += sqlite3DB.order_sql
	}
	if sqlite3DB.limit_sql != "" {
		sqlite3DB.sql += sqlite3DB.limit_sql
	}
	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)

	values := make([]interface{}, len(sqlite3DB.fields))
	for i, fieldName := range sqlite3DB.fields {
		fieldValue := result.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			values[i] = new(interface{})
		} else {
			values[i] = fieldValue.Addr().Interface()
		}
	}

	err := row.Scan(values...)
	if err != nil {
		return err
	}
	return nil
}

// 返回查询条数数量
func (sqlite3DB *SQLite3DB) Count() (int, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	sqlite3DB.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 是否存在
func (sqlite3DB *SQLite3DB) Exists() (bool, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	sqlite3DB.sql = fmt.Sprintf("SELECT 1 FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)

	var exists int
	err := row.Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// 更新数据，返回操作条数
func (sqlite3DB *SQLite3DB) Update(ModelData sqlite3.SQLite3Model) (sql.Result, error) {
	sqlite3DB.isSetModel()

	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.args = nil
	}()
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	ModelValue := reflect.ValueOf(ModelData)
	if ModelValue.Kind() == reflect.Ptr {
		ModelValue = ModelValue.Elem()
	}

	// 字段
	updateFields := make([]string, 0)
	// 值
	updateValues := make([]interface{}, 0)
	for _, fieldName := range sqlite3DB.fields {
		field := ModelValue.FieldByName(fieldName)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.IsValid() {
			fieldValue := field.Interface()
			updateFields = append(updateFields, fmt.Sprintf("`%v`=?", strings.ToLower(fieldName)))
			updateValues = append(updateValues, fieldValue)
		}
	}
	fieldsSQl := strings.Join(updateFields, ",")

	sqlite3DB.sql = fmt.Sprintf("UPDATE `%v` SET %v", TableName, fieldsSQl)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	updateValues = append(updateValues, sqlite3DB.args...)
	return sqlite3DB.Execute(sqlite3DB.sql, updateValues...)
}

// 删除数据，返回操作条数
func (sqlite3DB *SQLite3DB) Delete() (sql.Result, error) {
	sqlite3DB.isSetModel()

	defer func() {
		sqlite3DB.where_sql = nil
		sqlite3DB.args = nil
	}()
	TableName := sqlite3DB.model.ModelSet().TABLE_NAME
	sqlite3DB.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	return sqlite3DB.Execute(sqlite3DB.sql, sqlite3DB.args...)
}
