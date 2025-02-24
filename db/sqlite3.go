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

// SQLite3DB 结构体用于管理SQLite3数据库连接和操作
type SQLite3DB struct {
	name        string               // 数据库连接名称
	DB          *sql.DB              // 数据库连接对象
	transaction *sql.Tx              // 当前事务对象
	model       sqlite3.SQLite3Model // 当前操作的数据模型
	fields      []string             // 模型结构体字段名
	field_sql   []string             // 数据库表字段名
	where_sql   []string             // WHERE条件语句
	limit_sql   string               // LIMIT分页语句
	group_sql   string               // GROUP BY分组语句
	order_sql   string               // ORDER BY排序语句
	sql         string               // 最终执行的SQL语句
	args        []interface{}        // SQL语句的参数值
}

// SQLite3Connect 连接SQLite3数据库
//
// 参数:
//   - UseDataBases: string 数据库配置名称
//
// 返回:
//   - *SQLite3DB: SQLite3数据库操作实例
//
// 说明:
//   - 从全局配置中获取数据库连接信息
//   - 如果配置不存在或连接失败则panic
func SQLite3Connect(UseDataBases string) *SQLite3DB {
	database, ok := goi.Settings.DATABASES[UseDataBases]
	if !ok {
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

// Name 获取数据库连接名称
//
// 返回:
//   - string: 当前数据库连接名称
func (sqlite3DB SQLite3DB) Name() string {
	return sqlite3DB.name
}

// GetSQL 获取最近一次执行的SQL语句
//
// 返回:
//   - string: 最近一次执行的SQL语句
func (sqlite3DB SQLite3DB) GetSQL() string {
	return sqlite3DB.sql
}

// WithTransaction 在事务中执行指定的函数
//
// 参数:
//   - transactionFunc: func(sqlite3DB *SQLite3DB, args ...interface{}) error 事务执行函数
//   - args: ...interface{} 传递给事务函数的参数列表
//
// 返回:
//   - error: 事务执行的错误，发生错误时会自动回滚
//
// 说明:
//   - 不支持嵌套事务，如果当前已在事务中则返回错误
//   - 事务函数执行失败时自动回滚
func (sqlite3DB SQLite3DB) WithTransaction(transactionFunc func(sqlite3DB *SQLite3DB, args ...interface{}) error, args ...interface{}) error {
	var err error
	if sqlite3DB.transaction != nil {
		transactionCannotBeNestedErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.transaction_cannot_be_nested_error",
		})
		return errors.New(transactionCannotBeNestedErrorMsg)
	}

	sqlite3DB.transaction, err = sqlite3DB.DB.Begin()
	if err != nil {
		return err
	}
	err = transactionFunc(&sqlite3DB, args...)
	if err != nil {
		_ = sqlite3DB.transaction.Rollback()
		return err
	}

	// 提交事务
	return sqlite3DB.transaction.Commit()
}

// Execute 执行SQL语句
//
// 参数:
//   - query: string SQL语句
//   - args: ...interface{} SQL参数值列表
//
// 返回:
//   - sql.Result: 执行操作的结果
//   - error: 执行过程中的错误
//
// 说明:
//   - 支持在事务中使用
func (sqlite3DB *SQLite3DB) Execute(query string, args ...interface{}) (sql.Result, error) {
	if sqlite3DB.transaction != nil {
		return sqlite3DB.transaction.Exec(query, args...)
	} else {
		return sqlite3DB.DB.Exec(query, args...)
	}
}

// Query 执行查询SQL语句
//
// 参数:
//   - query: string SQL查询语句
//   - args: ...interface{} SQL参数值列表
//
// 返回:
//   - *sql.Rows: 查询结果集
//   - error: 查询过程中的错误
//
// 说明:
//   - 返回的结果集需要调用方手动关闭
//   - 查询失败时返回nil和错误信息
//   - 支持在事务中使用
func (sqlite3DB *SQLite3DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if sqlite3DB.transaction != nil {
		return sqlite3DB.transaction.Query(query, args...)
	} else {
		return sqlite3DB.DB.Query(query, args...)
	}
}

// Migrate 根据模型创建数据库表
//
// 参数:
//   - db_name: string 数据库名称
//   - model: sqlite3.SQLite3Model 数据模型
//
// 说明:
//   - 检查表是否已存在，存在则跳过创建
//   - 解析模型结构体的字段标签
//   - 支持自定义迁移前后处理函数
//   - 创建失败时会panic
func (sqlite3DB *SQLite3DB) Migrate(db_name string, model sqlite3.SQLite3Model) {
	var err error
	modelSettings := model.ModelSet()

	row := sqlite3DB.DB.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name =?;", modelSettings.TABLE_NAME)

	var exists int
	err = row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) == false && err != nil {
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

	var fieldSqlSlice []string
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldType, ok := field.Tag.Lookup("field_type")
		if !ok {
			continue
		}
		fieldName, ok := field.Tag.Lookup("field_name")
		if !ok {
			fieldName = strings.ToLower(field.Name)
		}

		fieldSqlSlice = append(fieldSqlSlice, fmt.Sprintf("  `%v` %v", fieldName, fieldType))
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

// isSetModel 检查是否已设置数据模型
//
// 说明:
//   - 内部方法，用于验证模型是否已设置
//   - 如果未设置模型则panic
func (sqlite3DB *SQLite3DB) isSetModel() {
	if sqlite3DB.model == nil {
		notSetModelErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.not_SetModel_error",
		})
		goi.Log.Error(notSetModelErrorMsg)
		panic(notSetModelErrorMsg)
	}
}

// SetModel 设置当前操作的数据模型
//
// 参数:
//   - model: sqlite3.SQLite3Model 数据模型实例
//
// 返回:
//   - *SQLite3DB: 当前实例指针，支持链式调用
//
// 说明:
//   - 设置后会重置所有查询条件
//   - 解析模型的字段信息用于后续操作
func (sqlite3DB *SQLite3DB) SetModel(model sqlite3.SQLite3Model) *SQLite3DB {
	sqlite3DB.model = model
	ModelType := reflect.TypeOf(sqlite3DB.model)
	// 获取字段
	sqlite3DB.fields = nil
	sqlite3DB.field_sql = nil
	sqlite3DB.where_sql = nil
	sqlite3DB.limit_sql = ""
	sqlite3DB.group_sql = ""
	sqlite3DB.order_sql = ""
	sqlite3DB.sql = ""
	sqlite3DB.args = nil

	for i := 0; i < ModelType.NumField(); i++ {
		field := ModelType.Field(i)

		_, ok := field.Tag.Lookup("field_type")
		if !ok {
			continue
		}

		sqlite3DB.fields = append(sqlite3DB.fields, field.Name)
		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		sqlite3DB.field_sql = append(sqlite3DB.field_sql, field_name)
	}
	return sqlite3DB
}

// Fields 指定查询要返回的字段
//
// 参数:
//   - fields: ...string 要查询的字段名列表
//
// 返回:
//   - *SQLite3DB: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名必须是模型结构体中已定义的字段
//   - 只返回带有field_type标签的字段
//   - 字段名大小写不敏感，会自动转换为数据库字段名
//   - 如果字段不存在则会panic
func (sqlite3DB SQLite3DB) Fields(fields ...string) *SQLite3DB {
	ModelType := reflect.TypeOf(sqlite3DB.model)
	// 获取字段
	sqlite3DB.fields = nil
	sqlite3DB.field_sql = nil

	for _, fieldName := range fields {
		field, ok := ModelType.FieldByName(fieldName)
		if !ok {
			fieldIsNotErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "database.field_is_not_error",
				TemplateData: map[string]interface{}{
					"name": fieldName,
				},
			})
			goi.Log.Error(fieldIsNotErrorMsg)
			panic(fieldIsNotErrorMsg)
		}

		// 检查字段是否有field_type标签
		_, ok = field.Tag.Lookup("field_type")
		if !ok {
			continue
		}

		sqlite3DB.fields = append(sqlite3DB.fields, field.Name)

		// 获取数据库字段名，如果没有field_name标签则使用字段名小写
		field_name, ok := field.Tag.Lookup("field_name")
		if !ok {
			field_name = strings.ToLower(field.Name)
		}
		sqlite3DB.field_sql = append(sqlite3DB.field_sql, field_name)
	}
	return &sqlite3DB
}

// Insert 向数据库插入一条记录
//
// 参数:
//   - ModelData: sqlite3.SQLite3Model 要插入的数据模型实例
//
// 返回:
//   - sql.Result: 插入操作的结果
//   - error: 插入过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 支持指针和非指针类型的字段值
//   - 只插入带有field_type标签的字段
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

// Where 设置查询条件
//
// 参数:
//   - query: string WHERE条件语句
//   - args: ...interface{} 条件参数值列表
//
// 返回:
//   - *SQLite3DB: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 支持多次调用，条件之间使用AND连接
//   - 支持参数占位符?自动替换
//   - 支持 in 切片类型的参数
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

		if (paramValue.Kind() == reflect.Slice || paramValue.Kind() == reflect.Array) && paramValue.Len() != 0 {
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

// GroupBy 设置分组条件
//
// 参数:
//   - groups: ...string 分组字段名列表
//
// 返回:
//   - *SQLite3DB: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名会自动去除首尾的空格和引号字符
//   - 字段名会自动添加反引号
//   - 空字段名会被忽略
//   - 不传参数时清空分组条件
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

// OrderBy 设置排序条件
//
// 参数:
//   - orders: ...string 排序规则列表，字段名前加"-"表示降序，否则为升序
//
// 返回:
//   - *SQLite3DB: 当前实例的副本指针，支持链式调用
//
// 说明:
//   - 字段名会自动去除首尾的空格和引号字符
//   - 字段名会自动添加反引号
//   - 空字段名会被忽略
//   - 不传参数时清空排序条件
func (sqlite3DB SQLite3DB) OrderBy(orders ...string) *SQLite3DB {
	var orderFields []string
	for _, order := range orders {
		// 去除字段名首尾的空格和引号
		order = strings.Trim(order, " `'\"")
		if order == "" {
			continue
		}

		var sequence string
		var orderSQL string
		// 处理排序方向
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

// Page 设置分页参数
//
// 参数:
//   - page: int 页码，从1开始
//   - pagesize: int 每页记录数
//
// 返回:
//   - int: 总记录数
//   - int: 总页数
//   - error: 查询过程中的错误
//
// 说明:
//   - 页码小于等于0时自动设为1
//   - 每页记录数小于等于0时设为默认值10
//   - 总页数根据总记录数和每页记录数计算
//   - 会自动执行一次COUNT查询获取总记录数
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

// Select 执行查询并将结果扫描到切片中
//
// 参数:
//   - queryResult: []*struct{} 或 []map[string]interface{} 用于接收结果的切片指针
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入结构体指针切片或map切片
//   - 支持指针和非指针类型的字段
//   - 自动映射数据库字段到结构体字段
//   - 查询失败时返回错误信息
func (sqlite3DB *SQLite3DB) Select(queryResult interface{}) error {
	sqlite3DB.isSetModel()

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

	kind := result.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		isNotSlicePtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_slice_or_array",
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
	if kind := ItemType.Kind(); kind != reflect.Struct && kind != reflect.Map {
		isNotStructPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_slice_struct_ptr_or_map",
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

	rows, err := sqlite3DB.DB.Query(sqlite3DB.sql, sqlite3DB.args...)
	defer rows.Close()
	if err != nil {
		return err
	} else if rows.Err() != nil {
		return rows.Err()
	}

	for rows.Next() {
		values := make([]interface{}, len(sqlite3DB.fields))

		var item reflect.Value
		if ItemType.Kind() == reflect.Struct {
			// 如果是结构体类型，使用 New 创建
			item = reflect.New(ItemType).Elem()

			for i, fieldName := range sqlite3DB.fields {
				fieldValue := item.FieldByName(fieldName)
				if !fieldValue.IsValid() {
					values[i] = new(interface{})
				} else {
					values[i] = fieldValue.Addr().Interface()
				}
			}
		} else {
			ModelType := reflect.TypeOf(sqlite3DB.model)

			for i, fieldName := range sqlite3DB.fields {
				field, _ := ModelType.FieldByName(fieldName)
				val := reflect.New(field.Type) // 获取模型字段类型
				values[i] = val.Interface()
			}
		}

		err = rows.Scan(values...)
		if err != nil {
			return err
		}

		// 设置 map 值
		if ItemType.Kind() == reflect.Map {
			item = reflect.MakeMap(ItemType)
			for i, field_name := range sqlite3DB.field_sql {
				val := reflect.ValueOf(values[i])
				item.SetMapIndex(reflect.ValueOf(field_name), val.Elem())
			}
		}

		if isPtr == true {
			result.Set(reflect.Append(result, item.Addr()))
		} else {
			result.Set(reflect.Append(result, item))
		}
	}
	return nil
}

// First 获取查询结果的第一条记录
//
// 参数:
//   - queryResult: *struct{} 或 map[string]interface{} 用于接收结果的结构体指针或map
//
// 返回:
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须传入结构体指针或map
//   - 自动映射数据库字段到结构体字段
//   - 无记录时返回sql.ErrNoRows
//   - 查询失败时返回错误信息
func (sqlite3DB *SQLite3DB) First(queryResult interface{}) error {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	result := reflect.ValueOf(queryResult)
	if result.Kind() != reflect.Map {
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
	}

	if kind := result.Kind(); kind != reflect.Struct && kind != reflect.Map {
		isNotStructPtrErrorMsg := language.I18n.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "database.is_not_struct_ptr_or_map",
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
	err := row.Err()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(sqlite3DB.fields))
	if result.Kind() == reflect.Struct {
		for i, fieldName := range sqlite3DB.fields {
			fieldValue := result.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				values[i] = new(interface{})
			} else {
				values[i] = fieldValue.Addr().Interface()
			}
		}
	} else {
		ModelType := reflect.TypeOf(sqlite3DB.model)

		for i, fieldName := range sqlite3DB.fields {
			field, _ := ModelType.FieldByName(fieldName)
			val := reflect.New(field.Type) // 获取模型字段类型
			values[i] = val.Interface()
		}
	}

	err = row.Scan(values...)
	if err != nil {
		return err
	}

	// 更新实际扫描到的值
	if result.Kind() == reflect.Map {
		for i, field_name := range sqlite3DB.field_sql {
			val := reflect.ValueOf(values[i])
			result.SetMapIndex(reflect.ValueOf(field_name), val.Elem())
		}
	}

	return nil
}

// Count 获取符合当前条件的记录总数
//
// 返回:
//   - int: 记录总数
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过SetModel设置数据模型
//   - 会考虑当前设置的WHERE条件
//   - 查询失败时返回0和错误信息
func (sqlite3DB *SQLite3DB) Count() (int, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	sqlite3DB.sql = fmt.Sprintf("SELECT count(*) FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)
	err := row.Err()
	if err != nil {
		return 0, err
	}

	var count int
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Exists 检查是否存在符合条件的记录
//
// 返回:
//   - bool: 是否存在记录
//   - error: 查询过程中的错误
//
// 说明:
//   - 必须先通过SetModel设置数据模型
//   - 会考虑当前设置的WHERE条件
//   - 使用SELECT 1优化查询性能
//   - 查询出错时返回false和错误信息
//   - 无记录时返回false和nil
func (sqlite3DB *SQLite3DB) Exists() (bool, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME

	sqlite3DB.sql = fmt.Sprintf("SELECT 1 FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	row := sqlite3DB.DB.QueryRow(sqlite3DB.sql, sqlite3DB.args...)
	err := row.Err()
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	var exists int
	err = row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// Update 更新符合条件的记录
//
// 参数:
//   - ModelData: sqlite3.SQLite3Model 包含更新数据的模型实例
//
// 返回:
//   - sql.Result: 更新操作的结果
//   - error: 更新过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 只更新非空字段
//   - 支持指针和非指针类型的字段值
//   - 会考虑当前设置的WHERE条件
func (sqlite3DB *SQLite3DB) Update(ModelData sqlite3.SQLite3Model) (sql.Result, error) {
	sqlite3DB.isSetModel()

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

// Delete 删除符合条件的记录
//
// 返回:
//   - sql.Result: 删除操作的结果
//   - error: 删除过程中的错误
//
// 说明:
//   - 调用前必须先通过SetModel设置数据模型
//   - 根据当前设置的WHERE条件删除记录
func (sqlite3DB *SQLite3DB) Delete() (sql.Result, error) {
	sqlite3DB.isSetModel()

	TableName := sqlite3DB.model.ModelSet().TABLE_NAME
	sqlite3DB.sql = fmt.Sprintf("DELETE FROM `%v`", TableName)
	if len(sqlite3DB.where_sql) > 0 {
		sqlite3DB.sql += fmt.Sprintf(" WHERE %v", strings.Join(sqlite3DB.where_sql, " AND "))
	}

	return sqlite3DB.Execute(sqlite3DB.sql, sqlite3DB.args...)
}
